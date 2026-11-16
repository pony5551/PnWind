package pnlibs

import (
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"runtime"
	"strings"
	"sync"
)

var (

	// qualified package name, cached at first use
	logrusPackage string

	// Positions in the call stack when tracing to report the calling method
	minimumCallerDepth int

	// Used for caller information initialisation
	callerInitOnce sync.Once
)

const (
	maximumCallerDepth int = 25
	knownLogrusFrames  int = 4
)

func init() {
	// start at the bottom of the stack before the package-name cache is primed
	minimumCallerDepth = 1
}

// getPackageName reduces a fully qualified function name to the package name
// There really ought to be to be a better way...
func getPackageName(f string) string {
	for {
		lastPeriod := strings.LastIndex(f, ".")
		lastSlash := strings.LastIndex(f, "/")
		if lastPeriod > lastSlash {
			f = f[:lastPeriod]
		} else {
			break
		}
	}

	return f
}

// getCaller retrieves the name of the first non-logrus calling function
func PnGetCaller() *runtime.Frame {
	// cache this package's fully-qualified name
	callerInitOnce.Do(func() {
		pcs := make([]uintptr, maximumCallerDepth)
		_ = runtime.Callers(0, pcs)

		// dynamic get the package name and the minimum caller depth
		for i := 0; i < maximumCallerDepth; i++ {
			funcName := runtime.FuncForPC(pcs[i]).Name()
			log.Println("funcName:", funcName)
			if strings.Contains(funcName, "00controllers.init") {
				logrusPackage = getPackageName(funcName)
				break
			}
		}

		minimumCallerDepth = knownLogrusFrames
	})

	// Restrict the lookback frames to avoid runaway lookups
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)

		// If the caller isn't part of this package, we're done
		if pkg != logrusPackage {
			return &f //nolint:scopelint
		}
	}

	// if we got here, we failed to find the caller's context
	return nil
}

func Test2() {

	// 上2层函数信息
	log.Println("上2层函数信息")
	pc, file, line, ok := runtime.Caller(2)
	// 是否获取成功
	log.Println(ok)
	// 函数指针
	log.Println(pc)
	// 所属文件
	log.Println(file)
	// 所属行
	log.Println(line)
	// 获取函数信息
	f := runtime.FuncForPC(pc)
	// 函数名
	log.Println(f.Name())

	// 当前函数信息
	log.Println("当前函数信息")
	pc, file, line, ok = runtime.Caller(0)
	log.Println(pc)
	log.Println(file)
	log.Println(line)
	log.Println(ok)
	f = runtime.FuncForPC(pc)
	log.Println(f.Name())

	// 上一层函数信息
	log.Println("上一层函数信息")
	pc, file, line, ok = runtime.Caller(1)
	log.Println(pc)
	log.Println(file)
	log.Println(line)
	log.Println(ok)
	f = runtime.FuncForPC(pc)
	log.Println(f.Name())
}
