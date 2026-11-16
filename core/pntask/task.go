// Package pntask 任务管理：按 cron 表达式定时执行的任务，和常驻的后台协程。
//
// 借鉴 beego 的 task 模块：任务有名字，登记在管理器里，管理页面可以列出、手动触发、暂停和恢复。
// 和它不同的是：并发安全；上一轮没跑完就跳过这一轮，不会重叠；任务 panic 有兜底；
// 停止时无条件等正在执行的任务结束，管理器没启动过也能正常停。
package pntask

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	log "github.com/pony5551/PnWind/core/pnlogrus"
)

// TaskFunc 任务函数。返回的错误和 panic 都记成这一轮失败，并写日志。
// ctx 不会因为进程退出而被取消：退出时任务管理器等任务自己跑完。
type TaskFunc = func(ctx context.Context) error

var (
	// ErrStopped 任务管理器已经停止。
	ErrStopped = errors.New("任务管理器已停止")
	// ErrTaskNotFound 没有这个名字的任务。
	ErrTaskNotFound = errors.New("任务不存在")
	// ErrTaskRunning 任务正在执行，这一次没有再启动它。
	ErrTaskRunning = errors.New("任务正在执行")
)

// Task 一个定时任务。用 NewTask 创建，交给 Manager.AddTask。
type Task struct {
	name  string
	spec  string
	fn    TaskFunc
	sched *schedule

	// 以下是运行状态，由 Manager 的锁保护
	paused   bool
	prev     time.Time
	next     time.Time
	lastCost time.Duration
	lastErr  string
	runs     uint64
	fails    uint64
	skips    uint64
}

// NewTask 定义一个定时任务：名字、六段 cron 表达式（秒 分 时 日 月 周）、任务函数。
// 表达式是否合法留到 AddTask 时检查，所以这里不返回错误。表达式的写法见 cron.go。
//
//	t := pntask.NewTask("清理过期令牌", "0 0 3 * * *", func(ctx context.Context) error {
//		return svc.DeleteExpiredTokens(ctx)
//	})
//	err := core.GConfig.Tasks.AddTask(t)
func NewTask(name, spec string, fn TaskFunc) *Task {
	return &Task{name: name, spec: spec, fn: fn}
}

// TaskInfo 一个任务某一时刻的状态，供管理页面使用。
type TaskInfo struct {
	Name      string        `json:"name"`
	Spec      string        `json:"spec"`
	Paused    bool          `json:"paused"`
	Running   bool          `json:"running"`   // 这个名字有一轮正在执行（删除之前的同名任务留下的最后一轮也算）
	PrevTime  time.Time     `json:"prevTime"`  // 上一次开始执行的时间；没执行过是零值
	NextTime  time.Time     `json:"nextTime"`  // 下一次计划执行的时间；暂停、调度没启动或已停止时是零值
	LastCost  time.Duration `json:"lastCost"`  // 上一次执行的耗时
	LastError string        `json:"lastError"` // 上一次执行的错误；成功为空
	Runs      uint64        `json:"runs"`      // 累计执行次数
	Fails     uint64        `json:"fails"`     // 累计失败次数：返回了错误，或者 panic
	Skips     uint64        `json:"skips"`     // 因为上一轮还没跑完而跳过的次数
}

// GoInfo 一个常驻后台协程的状态。
type GoInfo struct {
	Name      string    `json:"name"`
	StartTime time.Time `json:"startTime"`
}

// PanicError 任务 panic 时转成的错误。
type PanicError struct {
	Value any    // panic 的值
	Stack []byte // 发生 panic 时的堆栈
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Value)
}

// errGoexit 任务函数调用了 runtime.Goexit（例如测试里的 t.Fatal），没有走到返回。
var errGoexit = errors.New("任务没有正常结束（调用了 runtime.Goexit）")

// call 调用任务函数，把 panic 转成带堆栈的错误。
func call(fn TaskFunc) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &PanicError{Value: r, Stack: debug.Stack()}
		}
	}()
	return fn(context.Background())
}

// logFailure 记录一次失败。panic 连同堆栈一起记。kind 是"定时任务"或"常驻协程"。
func logFailure(kind, name string, err error) {
	var pe *PanicError
	if errors.As(err, &pe) {
		log.Errorf("%s %s panic: %v\n%s", kind, name, pe.Value, pe.Stack)
		return
	}
	log.Errorf("%s %s 执行失败: %v", kind, name, err)
}
