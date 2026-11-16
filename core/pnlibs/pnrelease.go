package pnlibs

import (
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

// 程序开始退出
var processQuitReady int32 = 0

func SetProcessQuitReady(value bool) {
	var intVal int32
	if value {
		intVal = 1
	}
	atomic.StoreInt32(&processQuitReady, intVal)
}

func GetProcessQuitReady() bool {
	return atomic.LoadInt32(&processQuitReady) != 0
}

// 监听信号：从调用这一刻起登记退出信号（SIGINT、SIGTERM、SIGQUIT），收到之后按顺序执行关闭函数，每一个都等到做完。
// 调用之前（服务还在启动）收到退出信号，按系统默认处理：进程立刻结束。
func ExitWatch(ExitName string, fns ...func() error) {
	// SIGHUP 不算退出信号，忽略它：终端断开（ssh 会话结束、关掉终端窗口）、有人 kill -HUP 时服务照常运行。
	// 只是不登记不行：没登记的 SIGHUP 按系统默认处理，进程被直接结束，谁也不等
	signal.Ignore(syscall.SIGHUP)
	// 程序无法捕获信号 SIGKILL 和 SIGSTOP （终止和暂停进程），因此 os/signal 包对这两个信号无效。
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// 阻塞
	s := <-quitChannel
	SetProcessQuitReady(true)
	log.Printf("%s:接收到信号[%s]执行关闭函数", ExitName, s.String())

	// 等待期间再来的信号只记日志，不打断等待。
	// 这里原来是 close(quitChannel)：通道关了却还在接收信号，第二个信号一到就 panic，关闭函数跑不完。
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		for {
			select {
			case s := <-quitChannel:
				log.Warnf("%s:已在退出中，正在等待关闭函数执行完，忽略重复的退出信号[%s]", ExitName, s.String())
			case <-finished:
				return
			}
		}
	}()

	for i := range fns {
		if fns[i] == nil {
			continue
		}
		if err := fns[i](); err != nil {
			log.Println("Watch:", err)
		}
	}
	log.Printf("%s:关闭函数执行完成", ExitName)
}
