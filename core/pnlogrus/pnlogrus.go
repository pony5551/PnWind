// Package pnlogrus 全局日志。调用方式沿用最初 logrus 的写法（log.Println、log.Errorf ……），底层是 zap。
//
// 两路输出：控制台（是终端时级别带颜色；SetFileOnly 可以关掉这一路）和日志文件（一行一条 JSON）。
// 日志文件是 logs/[日期]/<进程名>-<进程号>-mylog.log：一天一个目录，一个进程一个文件，
// 单个文件超过 100MB 由 lumberjack 切分，旧的压缩保存。
// logs 默认在可执行文件所在的目录下（SetDir 可以改）；日期目录保留 365 天（SetKeepDays 可以改），过期的整个删掉。
package pnlogrus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	defaultKeepDays = 365            // 日期目录默认保留的天数
	maxKeepDays     = 36500          // 保留天数的上限（100 年）：再大的数拿去算日期会溢出
	dayLayout       = "[2006-01-02]" // 日期目录的名字
	fileMode        = 0640           // 日志文件的权限：属主读写，同组可读（日志采集程序和服务在同一个组里就能读）
)

var (
	// fileName 日志文件的名字
	fileName = newFileName()
	// logFile 日志文件，全局一个
	logFile = &dayFile{logDir: defaultDir(), keepDays: defaultKeepDays}
	// fileOnly 为真时只写日志文件，不写控制台（SetFileOnly 设置）
	fileOnly atomic.Bool
	// sugar 全局日志。InitZapLog 之前只写控制台，之后加上日志文件
	sugar = newSugar(false, nil)
)

// InitZapLog 建立全局日志（控制台加日志文件）。只在启动时调用一次：core 包的 init 里。
// isdebug 为真时每条日志带调用位置，Error 以上带调用栈。
func InitZapLog(isdebug bool) {
	sugar = newSugar(isdebug, logFile)
}

// SetDir 定下日志文件放在哪个目录（下面还是一个个日期目录），并确认这个目录能用：
// 目录没有就建，建不出来、没有权限写时返回错误，日志仍然写在原来的地方。
// dir 是空串时用默认的目录：可执行文件所在目录下的 logs。相对路径按当前目录算。
func SetDir(dir string) error {
	if dir == "" {
		dir = defaultDir()
	}
	logDir, err := filepath.Abs(dir)
	if err == nil { // 目录在不等于能写，所以直接把今天的文件建出来
		err = touch(filepath.Join(logDir, time.Now().Format(dayLayout), fileName))
	}
	if err != nil {
		return fmt.Errorf("日志目录 %s 用不了: %w", dir, err)
	}
	logFile.mu.Lock()
	defer logFile.mu.Unlock()
	if logDir != logFile.logDir { // 换了目录：关掉原来的文件，下一条日志在新目录里重新开
		if logFile.out != nil {
			logFile.out.Close()
			logFile.out = nil
		}
		logFile.logDir, logFile.day = logDir, 0
	}
	return nil
}

// SetKeepDays 设置日期目录保留多少天，并立刻清理一次：今天和之前的 days 天留着，更早的目录整个删掉。
// 不调用是 365 天；days 不大于 0 按 365 天，大于 36500 按 36500 天。之后每换一天自动再清理一次。
func SetKeepDays(days int) {
	if days <= 0 {
		days = defaultKeepDays
	}
	days = min(days, maxKeepDays)
	logFile.mu.Lock()
	logFile.keepDays = days
	logDir := logFile.logDir
	logFile.mu.Unlock()
	Printf("日志保留 %d 天", days)
	removeExpired(logDir, days, time.Now())
}

// SetFileOnly on 为真时，之后的日志只写日志文件，不再写控制台；为假时两路都写（默认）。
// 标准输出交给 systemd、docker 收集的部署用它：两路都写的话，同一条日志在日志文件里和收集的地方各存一份。
func SetFileOnly(on bool) {
	if on {
		logFile.mu.Lock()
		logDir := logFile.logDir
		logFile.mu.Unlock()
		Printf("日志只写文件：控制台的输出到这一行为止，往后的看 %s", logDir)
	}
	fileOnly.Store(on)
}

// newSugar 建一个日志：写控制台；file 不是 nil 时同时写进它
func newSugar(isdebug bool, file zapcore.WriteSyncer) *zap.SugaredLogger {
	fileConf := zapcore.EncoderConfig{
		MessageKey:    "msg",
		LevelKey:      "level",
		TimeKey:       "time",
		CallerKey:     "caller",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeTime:    zapcore.TimeEncoderOfLayout(time.DateTime),
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}
	consoleConf := fileConf
	if isTerminal(os.Stdout) { // 颜色只给终端：不进日志文件，标准输出被重定向、被别的程序收集时也不带
		consoleConf.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// 控制台这一路能关：SetFileOnly 之后它不再输出，连这一路的编码都省了
	console := zap.LevelEnablerFunc(func(l zapcore.Level) bool { return l >= zapcore.InfoLevel && !fileOnly.Load() })
	cores := []zapcore.Core{zapcore.NewCore(zapcore.NewConsoleEncoder(consoleConf), os.Stdout, console)}
	if file != nil {
		cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(fileConf), file, zapcore.InfoLevel))
	}
	opts := []zap.Option{zap.AddCallerSkip(1)} // 调用位置要跳过本包这一层函数
	if isdebug {
		opts = append(opts, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	}
	return zap.New(zapcore.NewTee(cores...), opts...).Sugar()
}

// dayFile 日志文件：一天一个目录，换了一天自动换到新一天的目录，顺带清理过期的目录。
// 每换一天新建一个 lumberjack.Logger（它关闭之后后台协程还留着，一天多一个，不干活）；
// 不能复用同一个再改 Filename：那个后台协程读 Filename 时不加锁。
type dayFile struct {
	mu       sync.Mutex         // 管下面四个
	logDir   string             // 日志目录，下面是一个个日期目录
	keepDays int                // 日期目录保留的天数
	day      int                // out 是哪一天的文件：年月日拼成的数，例如 20261116
	out      *lumberjack.Logger // 当天的文件，超过 100MB 由它切分
}

func (f *dayFile) Write(p []byte) (int, error) { return f.write(time.Now(), p) }

// Sync 每条日志都是直接写进文件的，没有缓冲要刷
func (f *dayFile) Sync() error { return nil }

// write 把一条日志写进 now 那一天的文件。
// 换不换文件看日期变没变，不去算"下一个零点"：有的时区夏令时在零点拨快，那一天没有零点；
// 系统时钟往回调的时候，也不会一直写在后一天的目录里。
func (f *dayFile) write(now time.Time, p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	y, m, d := now.Date()
	if day := y*10000 + int(m)*100 + d; day != f.day {
		if f.out != nil { // 换了一天（不是进程的第一条日志）
			f.out.Close()
			// 另起协程：清理的时候要写日志，写日志又要等这把锁
			go removeExpired(f.logDir, f.keepDays, now)
		}
		name := filepath.Join(f.logDir, now.Format(dayLayout), fileName)
		_ = touch(name) // 建不出来不在这里管：下面 lumberjack 写的时候会再试，写不进去由它报
		f.day = day
		f.out = &lumberjack.Logger{
			Filename:   name,
			MaxSize:    100,  // 单个文件超过 100MB 就切分
			MaxBackups: 30,   // 同一天里切分出来的旧文件最多留 30 个
			Compress:   true, // 切分出来的旧文件压缩保存
			LocalTime:  true,
		}
	}
	return f.out.Write(p)
}

// removeExpired 删掉日志目录 logDir 下过期的日期目录：now 那一天和之前的 keepDays 天留着，更早的删掉。
// 只认名字是 [年-月-日] 的目录，别的文件和目录一概不动。keepDays 要在 1 到 maxKeepDays 之间。
func removeExpired(logDir string, keepDays int, now time.Time) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if !os.IsNotExist(err) { // 目录不存在是还没有写过日志
			Warnf("清理过期的日志目录出错: %v", err)
		}
		return
	}
	y, m, d := now.Date()
	oldest := time.Date(y, m, d-keepDays, 0, 0, 0, 0, now.Location()) // 还要保留的最早的一天
	for _, e := range entries {
		day, err := time.ParseInLocation(dayLayout, e.Name(), now.Location())
		if err != nil || !e.IsDir() || !day.Before(oldest) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(logDir, e.Name())); err != nil {
			Warnf("删除过期的日志目录 %s 出错: %v", e.Name(), err)
			continue
		}
		Printf("日志目录 %s 超过 %d 天，已删除", e.Name(), keepDays)
	}
}

// touch 把日志文件连同它的目录建出来。
// lumberjack 自己新建的文件只有属主能读（0600）；先建好，它接着写、切分、压缩都沿用这里给的权限。
func touch(name string) error {
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, fileMode)
	if err != nil {
		return err
	}
	return file.Close()
}

// newFileName 日志文件的名字：<进程名>-<进程号>-mylog.log。
// 进程名是可执行文件的名字（Windows 上去掉 .exe）：几个不同的程序共用一个日志目录时，看名字就知道是谁的。
// 带进程号：同一台机器、同一个容器里的几个进程各写各的文件。
// 几个容器把同一个目录挂进去当日志目录时分不开（各自的进程号常常都是 1）：这种部署要各用各的日志目录。
// 取不到可执行文件的路径时，名字是 <进程号>-mylog.log
func newFileName() string {
	name := fmt.Sprintf("%d-mylog.log", os.Getpid())
	if exe, err := os.Executable(); err == nil {
		proc := filepath.Base(exe)
		if ext := filepath.Ext(proc); len(proc) > len(ext) && strings.EqualFold(ext, ".exe") {
			proc = proc[:len(proc)-len(ext)]
		}
		name = proc + "-" + name
	}
	return name
}

// isTerminal 判断 f 是不是终端。终端是字符设备；重定向到文件、接到管道（systemd、docker 收集输出）都不是。
// /dev/null 也是字符设备，往那里多写几个颜色码没有影响。
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// defaultDir 默认的日志目录：可执行文件所在目录下的 logs（取不到可执行文件的路径时是当前目录下的 logs）
func defaultDir() string {
	dir := "."
	if exe, err := os.Executable(); err == nil {
		dir = filepath.Dir(exe)
	}
	return filepath.Join(dir, "logs")
}

// -------------------------------------------------------------------------
// 全局的日志函数
//
// 不带后缀的（Info）：几个参数直接拼在一起，只有相邻两个都不是字符串时才加空格（fmt.Sprint 的规则）。
// 带 ln 的（Infoln）：参数之间总是加一个空格（fmt.Sprintln 的规则），字符串末尾不用自己留空格。
// 带 f 的（Infof）：按格式串。
//
// Print 一组等同 Info；Debug 一组现在不输出（级别固定是 Info）。
// Panic 一组写完日志后 panic。
// Fatal 一组写完日志后直接结束进程：不执行 defer，不等后台任务，任何时候调用都绕过优雅退出。
// -------------------------------------------------------------------------

func Print(args ...any)                 { sugar.Info(args...) }
func Printf(format string, args ...any) { sugar.Infof(format, args...) }
func Println(args ...any)               { sugar.Infoln(args...) }

func Debug(args ...any)                 { sugar.Debug(args...) }
func Debugf(format string, args ...any) { sugar.Debugf(format, args...) }
func Debugln(args ...any)               { sugar.Debugln(args...) }

func Info(args ...any)                 { sugar.Info(args...) }
func Infof(format string, args ...any) { sugar.Infof(format, args...) }
func Infoln(args ...any)               { sugar.Infoln(args...) }

func Warn(args ...any)                 { sugar.Warn(args...) }
func Warnf(format string, args ...any) { sugar.Warnf(format, args...) }
func Warnln(args ...any)               { sugar.Warnln(args...) }

func Error(args ...any)                 { sugar.Error(args...) }
func Errorf(format string, args ...any) { sugar.Errorf(format, args...) }
func Errorln(args ...any)               { sugar.Errorln(args...) }

func Panic(args ...any)                 { sugar.Panic(args...) }
func Panicf(format string, args ...any) { sugar.Panicf(format, args...) }
func Panicln(args ...any)               { sugar.Panicln(args...) }

func Fatal(args ...any)                 { sugar.Fatal(args...) }
func Fatalf(format string, args ...any) { sugar.Fatalf(format, args...) }
func Fatalln(args ...any)               { sugar.Fatalln(args...) }
