package pnlogrus

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// capture 把全局日志换成只记在内存里的，测试结束换回去
func capture(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zapcore.InfoLevel)
	old := sugar
	sugar = zap.New(core).Sugar()
	t.Cleanup(func() { sugar = old })
	return logs
}

// newDayFile 建一个写在临时目录里的日志文件，测试结束时关掉
func newDayFile(t *testing.T, keepDays int) *dayFile {
	t.Helper()
	f := &dayFile{logDir: t.TempDir(), keepDays: keepDays}
	t.Cleanup(func() {
		if f.out != nil {
			f.out.Close()
		}
	})
	return f
}

// write 往 f 里写一条 now 那个时刻的日志
func write(t *testing.T, f *dayFile, now time.Time, line string) {
	t.Helper()
	if _, err := f.write(now, []byte(line)); err != nil {
		t.Fatal(err)
	}
}

// content 读出 f 在 dir 那个日期目录里的文件
func content(f *dayFile, dir string) string {
	data, err := os.ReadFile(filepath.Join(f.logDir, dir, fileName))
	if err != nil {
		return "读不出来: " + err.Error()
	}
	return string(data)
}

// today 读出日志目录 logDir 下这个进程今天写的日志。测试正好跨零点时会是两个日期目录，按名字排好连起来
func today(t *testing.T, logDir string) string {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(logDir, "*", fileName))
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(names)
	var all []byte
	for _, name := range names {
		if _, err := time.ParseInLocation(dayLayout, filepath.Base(filepath.Dir(name)), time.Local); err != nil {
			t.Errorf("日期目录的名字不对: %s", name)
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, data...)
	}
	return string(all)
}

// 三种写法怎么把参数拼成一条消息，各个函数是什么级别
func TestMessageAndLevel(t *testing.T) {
	logs := capture(t)
	err := errors.New("boom")

	Println("write error:", err)                 // 带 ln：参数之间总有一个空格
	Errorln("HttpPost Error1:", "http://x", err) // 字符串挨着字符串也有空格
	Warnln("只有一个参数")
	Infoln("n =", 3)
	Print("a", "b", 1, 2, err) // 不带后缀：只有相邻两个都不是字符串时才加空格
	Error("err:", err)
	Info("x")
	Warn("y")
	Printf("%s=%d", "n", 3)
	Infof("%v", err)
	Warnf("%d%%", 50)
	Errorf("请求 %s 出错: %v", "/x", err)
	Debug("不输出")
	Debugf("不输出 %d", 1)
	Debugln("不输出")

	want := []struct {
		level zapcore.Level
		msg   string
	}{
		{zapcore.InfoLevel, "write error: boom"},
		{zapcore.ErrorLevel, "HttpPost Error1: http://x boom"},
		{zapcore.WarnLevel, "只有一个参数"},
		{zapcore.InfoLevel, "n = 3"},
		{zapcore.InfoLevel, "ab1 2 boom"},
		{zapcore.ErrorLevel, "err:boom"},
		{zapcore.InfoLevel, "x"},
		{zapcore.WarnLevel, "y"},
		{zapcore.InfoLevel, "n=3"},
		{zapcore.InfoLevel, "boom"},
		{zapcore.WarnLevel, "50%"},
		{zapcore.ErrorLevel, "请求 /x 出错: boom"},
	}
	got := logs.All()
	if len(got) != len(want) {
		t.Fatalf("记下了 %d 条，应当是 %d 条（Debug 一组不输出）", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Level != w.level || got[i].Message != w.msg {
			t.Errorf("第 %d 条是 [%s] %q，应当是 [%s] %q", i+1, got[i].Level, got[i].Message, w.level, w.msg)
		}
	}
}

// Panic 一组先写日志再 panic
func TestPanic(t *testing.T) {
	for _, c := range []struct {
		name, msg string
		fn        func()
	}{
		{"Panic", "重复注册1", func() { Panic("重复注册", 1) }},
		{"Panicf", "x 重复注册", func() { Panicf("%s 重复注册", "x") }},
		{"Panicln", "重复注册 x", func() { Panicln("重复注册", "x") }},
	} {
		logs := capture(t)
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s 没有 panic", c.name)
				}
			}()
			c.fn()
		}()
		if got := logs.All(); len(got) != 1 || got[0].Level != zapcore.PanicLevel || got[0].Message != c.msg {
			t.Errorf("%s 应当先写一条 PANIC 级别的日志 %q，实际是 %v", c.name, c.msg, got)
		}
	}
}

// 日志文件里一行一条 JSON，级别不带控制台的颜色码
func TestFileLine(t *testing.T) {
	f := newDayFile(t, defaultKeepDays)
	old := sugar
	sugar = newSugar(false, f)
	t.Cleanup(func() { sugar = old })

	Errorln("写操作日志失败:", errors.New("boom"))
	Println("第二条")

	data := today(t, f.logDir)
	lines := strings.Split(strings.TrimSuffix(data, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("文件里有 %d 行，应当是 2 行:\n%s", len(lines), data)
	}
	for i, w := range []string{`","msg":"写操作日志失败: boom"}`, `","msg":"第二条"}`} {
		level := []string{"ERROR", "INFO"}[i]
		if !strings.HasPrefix(lines[i], `{"level":"`+level+`","time":"20`) || !strings.HasSuffix(lines[i], w) {
			t.Errorf("第 %d 行不对: %s", i+1, lines[i])
		}
	}
	if strings.ContainsRune(data, 0x1b) || strings.Contains(data, `\u001b`) {
		t.Errorf("文件里有颜色控制码:\n%s", data)
	}
}

// 日志文件的名字是 <进程名>-<进程号>-mylog.log，进程名不带 .exe；文件是按 fileMode 建的，不是 lumberjack 默认的只有属主能读
func TestFileNameAndMode(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	proc := strings.TrimSuffix(filepath.Base(exe), ".exe") // 这个测试程序在 Windows 上叫 pnlogrus.test.exe，别的系统上叫 pnlogrus.test
	if want := fmt.Sprintf("%s-%d-mylog.log", proc, os.Getpid()); fileName != want {
		t.Errorf("文件名是 %q，应当是 %q", fileName, want)
	}
	if strings.Contains(strings.ToLower(fileName), ".exe") {
		t.Errorf("文件名 %q 里还带着 .exe", fileName)
	}

	f := newDayFile(t, defaultKeepDays)
	if _, err := f.Write([]byte("x\n")); err != nil {
		t.Fatal(err)
	}
	names, _ := filepath.Glob(filepath.Join(f.logDir, "*", fileName))
	if len(names) == 0 {
		t.Fatal("没有找到日志文件")
	}
	// 权限受进程的 umask 影响，所以和一个用同样的权限直接建出来的文件比（Windows 上没有这套权限，两边总是一样）
	ref := filepath.Join(t.TempDir(), "ref")
	file, err := os.OpenFile(ref, os.O_CREATE|os.O_WRONLY, fileMode)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	got, err := os.Stat(names[0])
	if err != nil {
		t.Fatal(err)
	}
	wantInfo, err := os.Stat(ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode().Perm() != wantInfo.Mode().Perm() {
		t.Errorf("日志文件的权限是 %o，应当是 %o", got.Mode().Perm(), wantInfo.Mode().Perm())
	}
}

// 终端才带颜色：普通文件、管道都不算终端
func TestIsTerminal(t *testing.T) {
	// 测试里拿不到真的终端。空设备（/dev/null、NUL）和终端一样是字符设备，拿它当正面的例子
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	if !isTerminal(null) {
		t.Error("字符设备应当算终端")
	}

	file, err := os.Create(filepath.Join(t.TempDir(), "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if isTerminal(file) || isTerminal(w) {
		t.Errorf("普通文件算终端=%v，管道算终端=%v，都应当不算", isTerminal(file), isTerminal(w))
	}
	file.Close()
	if isTerminal(file) {
		t.Error("已经关掉的文件不应该算终端")
	}
}

// SetDir：换到新目录接着写；空串是默认的目录；目录用不了时报错，日志仍然写在原来的地方
func TestSetDir(t *testing.T) {
	oldDir, err := filepath.Abs(t.TempDir()) // SetDir 拿来比较的是整理过的绝对路径，这里用同一种写法
	if err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	old := logFile
	logFile = &dayFile{logDir: oldDir, keepDays: defaultKeepDays}
	t.Cleanup(func() {
		if logFile.out != nil {
			logFile.out.Close()
		}
		logFile = old
	})
	line := func(s string) {
		t.Helper()
		if _, err := logFile.Write([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}

	line("1\n")
	first := logFile.out
	if err := SetDir(oldDir); err != nil || logFile.out != first {
		t.Errorf("目录没变时不应该重新开文件：err=%v", err)
	}

	t.Chdir(base) // 相对路径按当前目录算，多级目录自己建
	if err := SetDir(filepath.Join("a", "logs")); err != nil {
		t.Fatal(err)
	}
	newDir := filepath.Join(base, "a", "logs")
	if got, _ := filepath.EvalSymlinks(logFile.logDir); got != mustEval(t, newDir) {
		t.Errorf("日志目录是 %s，应当是 %s", logFile.logDir, newDir)
	}
	line("2\n")

	// 用不了的目录：路径中间有一截是文件
	bad := filepath.Join(base, "file")
	if err := os.WriteFile(bad, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SetDir(filepath.Join(bad, "logs")); err == nil || !strings.Contains(err.Error(), "用不了") {
		t.Errorf("目录用不了时应当报错，实际 err=%v", err)
	}
	line("3\n")

	if got := today(t, oldDir); got != "1\n" {
		t.Errorf("原来的目录里是 %q，应当是 %q", got, "1\n")
	}
	if got := today(t, newDir); got != "2\n3\n" {
		t.Errorf("新目录里是 %q，应当是 %q", got, "2\n3\n")
	}

	// 空串：用默认的目录（可执行文件所在目录下的 logs），同样要确认能用——今天的文件建出来了
	if err := SetDir(""); err != nil || logFile.logDir != defaultDir() {
		t.Fatalf("SetDir(\"\") 应当换到默认的目录 %s：err=%v logDir=%s", defaultDir(), err, logFile.logDir)
	}
	names, _ := filepath.Glob(filepath.Join(defaultDir(), "*", fileName))
	if len(names) == 0 {
		t.Error("默认目录里今天的文件应当已经建出来")
	}
	// 默认目录在测试程序旁边，不在 t.TempDir() 里：自己建的文件删掉，目录空了也删（不空就留着，里面是别人的东西）
	for _, name := range names {
		os.Remove(name)
		os.Remove(filepath.Dir(name))
	}
	os.Remove(defaultDir())
}

// SetFileOnly：打开之后控制台不再输出，日志文件照常；关掉又是两路都写
func TestSetFileOnly(t *testing.T) {
	console, err := os.Create(filepath.Join(t.TempDir(), "stdout.txt"))
	if err != nil {
		t.Fatal(err)
	}
	oldOut, oldSugar, oldFile := os.Stdout, sugar, logFile
	logFile = newDayFile(t, defaultKeepDays)
	os.Stdout = console // newSugar 这时候拿的标准输出就是这个文件
	sugar = newSugar(false, logFile)
	os.Stdout = oldOut
	t.Cleanup(func() {
		fileOnly.Store(false)
		sugar, logFile = oldSugar, oldFile
		console.Close()
	})

	Println("两路都有")
	SetFileOnly(true) // 它自己记的那一行提示两路都有
	Println("这一条只进文件")
	Errorln("错误也一样")
	SetFileOnly(false)
	Println("又是两路")

	// count 数一数 data 里有几行、其中几行带 marks 里的字样
	count := func(data string, marks ...string) (lines, hit int) {
		for _, l := range strings.Split(strings.TrimSuffix(data, "\n"), "\n") {
			lines++
			for _, m := range marks {
				if strings.Contains(l, m) {
					hit++
				}
			}
		}
		return lines, hit
	}
	out, err := os.ReadFile(console.Name())
	if err != nil {
		t.Fatal(err)
	}
	if lines, hit := count(string(out), "两路都有", "日志只写文件", "又是两路"); lines != 3 || hit != 3 {
		t.Errorf("控制台应当只有 3 行（打开之前的、提示、关掉之后的），实际:\n%s", out)
	}
	if !strings.Contains(string(out), logFile.logDir) {
		t.Errorf("提示里应当带着日志目录 %s:\n%s", logFile.logDir, out)
	}
	file := today(t, logFile.logDir)
	if lines, hit := count(file, "两路都有", "日志只写文件", "这一条只进文件", "错误也一样", "又是两路"); lines != 5 || hit != 5 {
		t.Errorf("日志文件应当 5 行都在，实际:\n%s", file)
	}
}

// mustEval 把路径里的符号链接解开（macOS、Windows 的临时目录路径里可能有）
func mustEval(t *testing.T, path string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

// 换了一天就换到新一天的目录，同时清掉过期的目录
func TestNewDay(t *testing.T) {
	logs := capture(t)
	f := newDayFile(t, 30)
	expired := filepath.Join(f.logDir, "[2099-11-30]") // 比 2100-01-01 早 32 天
	kept := filepath.Join(f.logDir, "[2099-12-02]")    // 正好早 30 天，还留着
	for _, dir := range []string{expired, kept} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	last := time.Date(2099, 12, 31, 23, 59, 58, 0, time.Local)
	write(t, f, last, "旧的一年\n")
	first := f.out
	write(t, f, last.Add(time.Second), "还是旧的一年\n")
	if f.out != first {
		t.Error("同一天里不应该换文件")
	}
	if _, err := os.Stat(expired); err != nil {
		t.Error("进程的第一条日志不应该触发清理")
	}
	write(t, f, last.Add(2*time.Second), "新的一年\n") // 过了零点，也过了年

	if got := content(f, "[2099-12-31]"); got != "旧的一年\n还是旧的一年\n" {
		t.Errorf("[2099-12-31] 里是 %q", got)
	}
	if got := content(f, "[2100-01-01]"); got != "新的一年\n" {
		t.Errorf("[2100-01-01] 里是 %q", got)
	}
	// 清理在另一个协程里做：删完目录才记日志，所以等到日志出现
	for range 200 {
		if logs.Len() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if logs.Len() != 1 || !strings.Contains(logs.All()[0].Message, "[2099-11-30] 超过 30 天，已删除") {
		t.Errorf("删目录应当记一条日志，实际 %d 条", logs.Len())
	}
	if _, err := os.Stat(expired); err == nil {
		t.Error("换了一天之后，过期的目录应当被清掉")
	}
	if _, err := os.Stat(kept); err != nil {
		t.Error("没过期的目录不应该删")
	}
}

// 夏令时在零点拨快的时区（那一天没有零点），和系统时钟往回调：都只看日期变没变
func TestOddClock(t *testing.T) {
	capture(t)
	loc, err := time.LoadLocation("America/Santiago") // 2026-09-06 的 0 点拨到 1 点
	if err != nil {
		t.Skip("本机没有时区数据：", err)
	}
	f := newDayFile(t, defaultKeepDays)

	late := time.Date(2026, 9, 5, 23, 10, 0, 0, loc)
	write(t, f, late, "a\n")
	first := f.out
	write(t, f, late.Add(30*time.Minute), "b\n")
	if f.out != first {
		t.Error("拨快前的那一个小时里不应该换文件")
	}
	write(t, f, late.Add(time.Hour), "c\n") // 第二天了
	if f.out == first {
		t.Error("到了第二天应当换文件")
	}
	write(t, f, late, "d\n") // 时钟调回前一天：写回前一天的文件
	write(t, f, late.Add(time.Hour), "e\n")

	if got := content(f, "[2026-09-05]"); got != "a\nb\nd\n" {
		t.Errorf("[2026-09-05] 里是 %q", got)
	}
	if got := content(f, "[2026-09-06]"); got != "c\ne\n" {
		t.Errorf("[2026-09-06] 里是 %q", got)
	}
}

// 只删名字是 [年-月-日]、而且过期了的目录
func TestRemoveExpired(t *testing.T) {
	capture(t)
	logDir := t.TempDir()
	now := time.Date(2026, 11, 16, 5, 0, 0, 0, time.Local)
	dirs := []string{
		"[2025-11-15]", "[2020-01-01]", // 过期：比 2026-11-16 早超过 365 天
		"[2025-11-16]", "[2026-11-15]", "[2026-11-16]", "[2026-11-17]", // 没过期（正好早 365 天的留着）
		"2020-01-01", "[2020-1-1]", "[2020-01-01]bak", "[2020-13-01]", "other", // 名字不是日期目录
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(logDir, d, "sub"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(logDir, d, "sub", "1-mylog.log"), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	want := append([]string{"[2019-01-01]"}, dirs[2:]...)
	// 名字像日期目录的文件不动
	if err := os.WriteFile(filepath.Join(logDir, "[2019-01-01]"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	// 名字像日期目录的符号链接也不动，它指向的目录里的东西更不能动（Windows 上没有权限建符号链接时跳过这一项）
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "keep.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(logDir, "[2019-06-01]")); err == nil {
		want = append(want, "[2019-06-01]")
	} else {
		t.Log("没有测符号链接：", err)
	}

	removeExpired(logDir, 365, now)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		left = append(left, e.Name())
	}
	slices.Sort(left)
	slices.Sort(want)
	if !slices.Equal(left, want) {
		t.Errorf("剩下的是 %v\n应当是 %v", left, want)
	}
	if _, err := os.Stat(filepath.Join(outside, "keep.txt")); err != nil {
		t.Error("符号链接指向的目录里的文件被动了")
	}

	removeExpired(filepath.Join(logDir, "没有这个目录"), 365, now) // 目录不存在时什么都不做
}

// SetKeepDays：不大于 0 按 365 天，太大的按上限；设完立刻清理一次。
// 这里用的是真实的日期，天数都留了很大的余量（正好在边界上的情况在 TestRemoveExpired 里），测试跨零点也不受影响
func TestSetKeepDays(t *testing.T) {
	capture(t)
	old := logFile
	logFile = &dayFile{logDir: t.TempDir(), keepDays: defaultKeepDays}
	t.Cleanup(func() { logFile = old })
	today := time.Now()
	day := func(ago int) string {
		return filepath.Join(logFile.logDir, today.AddDate(0, 0, -ago).Format(dayLayout))
	}
	for _, ago := range []int{0, 1, 100, 1000} {
		if err := os.Mkdir(day(ago), 0755); err != nil {
			t.Fatal(err)
		}
	}
	exists := func(ago int) bool { _, err := os.Stat(day(ago)); return err == nil }

	// 写成天文数字（想表示"永远不删"）：按上限算，什么都不该删。不设上限的话日期算溢出，连今天的目录都会删掉
	for _, days := range []int{1e15, 1 << 62, maxKeepDays + 1} {
		SetKeepDays(days)
		if logFile.keepDays != maxKeepDays {
			t.Errorf("SetKeepDays(%d) 之后是 %d 天，应当是上限 %d 天", days, logFile.keepDays, maxKeepDays)
		}
		if !exists(0) || !exists(1) || !exists(100) || !exists(1000) {
			t.Fatalf("SetKeepDays(%d) 不应该删任何目录：今天=%v 1 天前=%v 100 天前=%v 1000 天前=%v", days, exists(0), exists(1), exists(100), exists(1000))
		}
	}

	for _, days := range []int{0, -5} {
		SetKeepDays(days)
		if logFile.keepDays != defaultKeepDays {
			t.Errorf("SetKeepDays(%d) 之后是 %d 天，应当是 %d 天", days, logFile.keepDays, defaultKeepDays)
		}
	}
	if !exists(0) || !exists(1) || !exists(100) || exists(1000) {
		t.Errorf("按 365 天清理：今天=%v 1 天前=%v 100 天前=%v（都应当在），1000 天前=%v（应当不在）", exists(0), exists(1), exists(100), exists(1000))
	}

	SetKeepDays(30)
	if logFile.keepDays != 30 {
		t.Errorf("SetKeepDays(30) 之后是 %d 天", logFile.keepDays)
	}
	if !exists(0) || !exists(1) || exists(100) {
		t.Errorf("按 30 天清理：今天=%v 1 天前=%v（都应当在），100 天前=%v（应当不在）", exists(0), exists(1), exists(100))
	}
}
