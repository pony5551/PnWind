package pntask

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// schedule 解析好的 cron 表达式。每一段是一张位图：第 n 位是 1，表示这一段可以取 n。
type schedule struct {
	second, minute, hour, day, month, week uint64
	dayStar, weekStar                      bool // "日""周"这一段是不是以 * 开头（* 或者 */2 这样带步长的），决定两段怎么配合，见 dayMatches
}

// field 表达式里的一段：名字和取值范围。
type field struct {
	name     string
	min, max uint
}

// fields 表达式的六段，依次是：秒 分 时 日 月 周。周的 0 和 7 都是周日。
var fields = [6]field{
	{"秒", 0, 59},
	{"分", 0, 59},
	{"时", 0, 23},
	{"日", 1, 31},
	{"月", 1, 12},
	{"周", 0, 7},
}

// parse 解析六段 cron 表达式：秒 分 时 日 月 周，空白分隔。
//
// 每一段可以写：
//
//   - 任意值
//     5        单个值
//     1-5      范围
//     1,3,5    列表，列表里的每一项可以是这里的任一种
//     */15     步长：从最小值起每 15 一步
//     10-40/10 范围加步长
//     5/15     从 5 起到这一段的最大值，每 15 一步
//
// 例子："0 0 3 * * *" 每天 3 点；"*/10 * * * * *" 每 10 秒；"0 30 9 * * 1-5" 周一到周五 9 点半。
func parse(spec string) (*schedule, error) {
	parts := strings.Fields(spec)
	if len(parts) != len(fields) {
		return nil, fmt.Errorf("cron 表达式 %q 应当有 6 段（秒 分 时 日 月 周），实际有 %d 段", spec, len(parts))
	}
	var bits [6]uint64
	for i, part := range parts {
		b, err := parseField(part, fields[i])
		if err != nil {
			return nil, fmt.Errorf("cron 表达式 %q 的第 %d 段（%s）%w", spec, i+1, fields[i].name, err)
		}
		bits[i] = b
	}
	if bits[5]&(1<<7) != 0 { // 周日写成 7 的，归到 0
		bits[5] = bits[5]&^(1<<7) | 1
	}
	return &schedule{
		second: bits[0], minute: bits[1], hour: bits[2], day: bits[3], month: bits[4], week: bits[5],
		dayStar: strings.HasPrefix(parts[3], "*"), weekStar: strings.HasPrefix(parts[5], "*"),
	}, nil
}

// parseField 解析一段，返回位图。
func parseField(expr string, f field) (uint64, error) {
	var bits uint64
	for item := range strings.SplitSeq(expr, ",") {
		rangeExpr, stepExpr, hasStep := strings.Cut(item, "/")
		step := uint(1)
		if hasStep {
			n, err := strconv.ParseUint(stepExpr, 10, 8)
			if err != nil || n == 0 {
				return 0, fmt.Errorf("%q 不合法：步长 %q 要是大于 0 的整数", item, stepExpr)
			}
			step = uint(n)
		}

		lo, hi := f.min, f.max
		if rangeExpr != "*" {
			loExpr, hiExpr, isRange := strings.Cut(rangeExpr, "-")
			n, err := strconv.ParseUint(loExpr, 10, 8)
			if err != nil {
				return 0, fmt.Errorf("%q 不合法：%q 不是数字", item, loExpr)
			}
			lo = uint(n)
			switch {
			case isRange:
				if n, err = strconv.ParseUint(hiExpr, 10, 8); err != nil {
					return 0, fmt.Errorf("%q 不合法：%q 不是数字", item, hiExpr)
				}
				hi = uint(n)
			case !hasStep: // 单个值；带步长的 5/15 表示从 5 到最大值
				hi = lo
			}
		}
		if lo < f.min || hi > f.max || lo > hi {
			return 0, fmt.Errorf("%q 不合法：取值范围是 %d 到 %d", item, f.min, f.max)
		}
		for v := lo; v <= hi; v += step {
			bits |= 1 << v
		}
	}
	return bits, nil
}

// next 返回严格晚于 t 的最早一个匹配时刻，时区和 t 相同。五年之内没有匹配的时刻（例如 2 月 30 日）返回零值。
//
// 做法：从下一秒开始，哪一段对不上，就跳到这一段的下一个单位的开头，再从头检查。
func (s *schedule) next(t time.Time) time.Time {
	loc := t.Location()
	limit := t.Year() + 5
	t = t.Truncate(time.Second).Add(time.Second)
	for t.Year() <= limit {
		year, month, day := t.Date()
		hour, minute, second := t.Clock()
		var n time.Time
		switch {
		case s.month&(1<<uint(month)) == 0: // 下个月 1 日 0 点
			n = time.Date(year, month+1, 1, 0, 0, 0, 0, loc)
		case !s.dayMatches(t): // 明天 0 点
			n = time.Date(year, month, day+1, 0, 0, 0, 0, loc)
		case s.hour&(1<<uint(hour)) == 0: // 下一个整点
			n = t.Add(time.Hour - time.Duration(minute)*time.Minute - time.Duration(second)*time.Second)
		case s.minute&(1<<uint(minute)) == 0: // 下一分钟的 0 秒
			n = t.Add(time.Minute - time.Duration(second)*time.Second)
		case s.second&(1<<uint(second)) == 0:
			n = t.Add(time.Second)
		default:
			return t
		}
		if !n.After(t) {
			// 有夏令时的时区，在切换的那一刻 time.Date 算出来的时间可能不往前走。这里保证每一步都前进，不会原地打转。
			n = t.Add(time.Hour)
		}
		t = n
	}
	return time.Time{}
}

// dayMatches 判断 t 这一天对不对得上"日"和"周"两段。
// 和标准 cron 一样：两段都有限制时，任意一段对上就算；其中一段以 * 开头时，两段要同时对上。
// 以 * 开头的有两种：单独一个 *（它什么都对得上，等于只看另一段），和 */2 这样带步长的——
// "0 0 9 */2 * 1-5" 是"单数日，并且是周一到周五"。
func (s *schedule) dayMatches(t time.Time) bool {
	dayOK := s.day&(1<<uint(t.Day())) != 0
	weekOK := s.week&(1<<uint(t.Weekday())) != 0
	if s.dayStar || s.weekStar {
		return dayOK && weekOK
	}
	return dayOK || weekOK
}
