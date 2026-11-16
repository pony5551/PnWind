package pntask

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const everySecond = "* * * * * *"

// waitFor 等条件成立，最多等 5 秒。
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("等不到：%s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// info 取一个任务的状态。
func info(t *testing.T, m *Manager, name string) TaskInfo {
	t.Helper()
	for _, i := range m.GetAllTasks() {
		if i.Name == name {
			return i
		}
	}
	t.Fatalf("没有任务 %s", name)
	return TaskInfo{}
}

func TestFiresOnSchedule(t *testing.T) {
	t.Parallel()
	m := NewManager()
	var n atomic.Int64
	if err := m.AddTask(NewTask("每秒", everySecond, func(context.Context) error { n.Add(1); return nil })); err != nil {
		t.Fatal(err)
	}
	if i := info(t, m, "每秒"); !i.NextTime.IsZero() || i.Runs != 0 {
		t.Errorf("调度没启动时不应当有下一次时间：%+v", i)
	}

	begin := time.Now()
	m.StartTask()
	m.StartTask() // 重复调用没有效果
	if i := info(t, m, "每秒"); i.NextTime.Sub(begin) > time.Second || !i.NextTime.After(begin.Add(-time.Second)) {
		t.Errorf("下一次时间应当在一秒之内：%+v", i)
	}
	waitFor(t, "触发 3 次", func() bool { return n.Load() >= 3 })
	// 只卡下限：不能比时间表跑得快。上限由 waitFor 的 5 秒兜着，机器忙的时候不至于误报。
	if cost := time.Since(begin); cost < 2*time.Second {
		t.Errorf("每秒一次，触发 3 次只用了 %s", cost)
	}
	m.Stop()

	i := info(t, m, "每秒")
	if i.Runs != uint64(n.Load()) || i.Fails != 0 || i.Skips != 0 || i.PrevTime.IsZero() || i.Running || i.LastError != "" {
		t.Errorf("状态不对：%+v", i)
	}
	if !i.NextTime.IsZero() {
		t.Errorf("停止之后不应当有下一次时间：%+v", i)
	}
	after := n.Load()
	time.Sleep(1200 * time.Millisecond)
	if n.Load() != after {
		t.Error("停止之后不应当再触发")
	}
}

// 上一轮没跑完：这一轮跳过并计数，不会同时跑两个。
func TestNoOverlap(t *testing.T) {
	t.Parallel()
	m := NewManager()
	var running, peak, n atomic.Int64
	release := make(chan struct{})
	_ = m.AddTask(NewTask("慢", everySecond, func(context.Context) error {
		if c := running.Add(1); c > peak.Load() {
			peak.Store(c)
		}
		n.Add(1)
		<-release
		running.Add(-1)
		return nil
	}))
	m.StartTask()
	waitFor(t, "跳过两轮", func() bool { return info(t, m, "慢").Skips >= 2 })
	if i := info(t, m, "慢"); !i.Running || n.Load() != 1 {
		t.Errorf("上一轮没跑完时不应当再启动：执行了 %d 次，%+v", n.Load(), i)
	}
	// 手动触发同样不重叠
	if err := m.RunTask("慢"); !errors.Is(err, ErrTaskRunning) {
		t.Errorf("正在执行时手动触发应当返回 ErrTaskRunning，得到 %v", err)
	}
	close(release)
	m.Stop()
	if peak.Load() != 1 {
		t.Errorf("同时在跑的峰值是 %d，期望 1", peak.Load())
	}
}

func TestPauseResume(t *testing.T) {
	t.Parallel()
	m := NewManager()
	var n atomic.Int64
	_ = m.AddTask(NewTask("每秒", everySecond, func(context.Context) error { n.Add(1); return nil }))
	if err := m.PauseTask("每秒"); err != nil {
		t.Fatal(err)
	}
	m.StartTask()
	time.Sleep(1300 * time.Millisecond)
	if i := info(t, m, "每秒"); n.Load() != 0 || !i.Paused || !i.NextTime.IsZero() {
		t.Fatalf("暂停的任务不应当触发：执行了 %d 次，%+v", n.Load(), i)
	}

	// 暂停时手动触发仍然可以
	if err := m.RunTask("每秒"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "手动触发执行完", func() bool { return n.Load() == 1 })

	if err := m.ResumeTask("每秒"); err != nil {
		t.Fatal(err)
	}
	if i := info(t, m, "每秒"); i.Paused || i.NextTime.IsZero() {
		t.Errorf("恢复之后应当有下一次时间：%+v", i)
	}
	waitFor(t, "恢复后按时触发", func() bool { return n.Load() >= 2 })
	m.Stop()

	if err := m.PauseTask("没有"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("暂停不存在的任务应当返回 ErrTaskNotFound，得到 %v", err)
	}
	if err := m.ResumeTask("没有"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("恢复不存在的任务应当返回 ErrTaskNotFound，得到 %v", err)
	}
}

// 调度启动之后再添加、再删除，都立即生效。
func TestAddAndDeleteWhileRunning(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.StartTask()
	var n atomic.Int64
	if err := m.AddTask(NewTask("后加的", everySecond, func(context.Context) error { n.Add(1); return nil })); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "后加的任务按时触发", func() bool { return n.Load() >= 1 })

	if !m.DeleteTask("后加的") || m.DeleteTask("后加的") {
		t.Error("DeleteTask 应当第一次返回 true，第二次返回 false")
	}
	if len(m.GetAllTasks()) != 0 {
		t.Error("删除之后不应当还能列出")
	}
	if err := m.RunTask("后加的"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("删除之后手动触发应当返回 ErrTaskNotFound，得到 %v", err)
	}
	after := n.Load()
	time.Sleep(1200 * time.Millisecond)
	if n.Load() != after {
		t.Error("删除之后不应当再触发")
	}
	m.Stop()
}

// 删除一个正在执行的任务之后马上添加同名的任务：旧的那一轮跑完之前新任务不启动，两轮不会同时执行。
func TestReAddWhileOldRoundRunning(t *testing.T) {
	t.Parallel()
	m := NewManager()
	started, release := make(chan struct{}), make(chan struct{})
	var oldRunning atomic.Bool
	_ = m.AddTask(NewTask("同名", "0 0 3 * * *", func(context.Context) error {
		oldRunning.Store(true)
		close(started)
		<-release
		oldRunning.Store(false)
		return nil
	}))
	if err := m.RunTask("同名"); err != nil {
		t.Fatal(err)
	}
	<-started
	if !m.DeleteTask("同名") {
		t.Fatal("DeleteTask 应当返回 true")
	}

	var n, overlap atomic.Int64
	if err := m.AddTask(NewTask("同名", everySecond, func(context.Context) error {
		if oldRunning.Load() {
			overlap.Add(1)
		}
		n.Add(1)
		return nil
	})); err != nil {
		t.Fatalf("旧的一轮还在跑时添加同名的任务应当成功，得到 %v", err)
	}
	if i := info(t, m, "同名"); !i.Running || i.Runs != 0 {
		t.Errorf("旧的一轮还在跑时，同名的新任务应当显示正在执行、还没有执行过：%+v", i)
	}
	if err := m.RunTask("同名"); !errors.Is(err, ErrTaskRunning) {
		t.Errorf("旧的一轮还在跑时手动触发应当返回 ErrTaskRunning，得到 %v", err)
	}
	m.StartTask()
	waitFor(t, "到点的一轮被跳过", func() bool { return info(t, m, "同名").Skips >= 1 })
	if n.Load() != 0 {
		t.Errorf("旧的一轮还在跑时新任务执行了 %d 次", n.Load())
	}

	close(release)
	waitFor(t, "旧的一轮跑完之后新任务开始执行", func() bool { return n.Load() >= 1 })
	m.Stop()
	if i := info(t, m, "同名"); i.Running || i.Runs != uint64(n.Load()) || overlap.Load() != 0 {
		t.Errorf("状态不对：%+v，和旧的一轮重叠了 %d 次", i, overlap.Load())
	}
}

// 手动触发不需要启动调度。错误、panic、Goexit 都记成失败，下一次成功后清掉错误。
func TestRunTaskRecordsResult(t *testing.T) {
	m := NewManager()
	var result atomic.Value // 这一次任务要做什么
	_ = m.AddTask(NewTask("手动", "0 0 3 * * *", func(context.Context) error {
		switch result.Load() {
		case "error":
			return errors.New("坏了")
		case "panic":
			panic("炸了")
		case "goexit":
			runtime.Goexit()
		}
		return nil
	}))
	run := func(what string) TaskInfo {
		result.Store(what)
		before := info(t, m, "手动").Runs
		if err := m.RunTask("手动"); err != nil {
			t.Fatal(err)
		}
		waitFor(t, what+" 执行完", func() bool { i := info(t, m, "手动"); return i.Runs == before+1 && !i.Running })
		return info(t, m, "手动")
	}

	if i := run("error"); i.Fails != 1 || i.LastError != "坏了" {
		t.Errorf("返回错误：%+v", i)
	}
	if i := run("panic"); i.Fails != 2 || i.LastError != "panic: 炸了" {
		t.Errorf("panic：%+v", i)
	}
	if i := run("goexit"); i.Fails != 3 || !strings.Contains(i.LastError, "Goexit") {
		t.Errorf("Goexit：%+v", i)
	}
	if i := run("ok"); i.Fails != 3 || i.Runs != 4 || i.LastError != "" || i.PrevTime.IsZero() {
		t.Errorf("成功之后应当清掉上一次的错误：%+v", i)
	}
	m.Stop() // 任务被 panic、Goexit 带走过，也不能卡住
}

func TestStopWaitsForRunningTask(t *testing.T) {
	m := NewManager()
	started, release := make(chan struct{}), make(chan struct{})
	var done atomic.Bool
	_ = m.AddTask(NewTask("慢", "0 0 3 * * *", func(context.Context) error {
		close(started)
		<-release
		time.Sleep(50 * time.Millisecond)
		done.Store(true)
		return nil
	}))
	_ = m.RunTask("慢")
	<-started

	stopped := make(chan struct{})
	go func() { m.Stop(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("任务还在执行，Stop 就返回了")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	<-stopped
	if !done.Load() {
		t.Error("Stop 返回时任务应当已经跑完")
	}
}

func TestGoAndQuit(t *testing.T) {
	m := NewManager()
	var cleaned atomic.Bool
	err := m.Go("消费循环", func(context.Context) error {
		<-m.Quit() // 收到退出通知之后还要收尾
		time.Sleep(50 * time.Millisecond)
		cleaned.Store(true)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = m.Go("很快结束", func(context.Context) error { return errors.New("连不上") })
	_ = m.Go("会 panic", func(context.Context) error { panic("炸了") })
	waitFor(t, "结束的协程从列表里消失", func() bool { return len(m.GetAllGos()) == 1 })
	if gos := m.GetAllGos(); gos[0].Name != "消费循环" || gos[0].StartTime.IsZero() {
		t.Errorf("GetAllGos = %+v", gos)
	}
	select {
	case <-m.Quit():
		t.Fatal("没有调用 Stop，Quit 通道不应当关闭")
	default:
	}

	m.Stop()
	if !cleaned.Load() || len(m.GetAllGos()) != 0 {
		t.Errorf("Stop 应当等常驻协程收尾完才返回：cleaned=%v gos=%v", cleaned.Load(), m.GetAllGos())
	}
	select {
	case <-m.Quit():
	default:
		t.Error("Stop 之后 Quit 通道应当已经关闭")
	}
}

// 没启动过调度也能停；可以重复停、并发停；停了之后不再接受任何新的活。
func TestStop(t *testing.T) {
	m := NewManager()
	_ = m.AddTask(NewTask("任务", everySecond, func(context.Context) error { return nil }))

	done := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		for range 4 {
			wg.Go(m.Stop)
		}
		wg.Wait()
		m.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("没启动过调度，Stop 却没有返回")
	}

	fn := func(context.Context) error { return nil }
	if err := m.AddTask(NewTask("新任务", everySecond, fn)); !errors.Is(err, ErrStopped) {
		t.Errorf("AddTask 得到 %v", err)
	}
	if err := m.RunTask("任务"); !errors.Is(err, ErrStopped) {
		t.Errorf("RunTask 得到 %v", err)
	}
	if err := m.Go("协程", fn); !errors.Is(err, ErrStopped) {
		t.Errorf("Go 得到 %v", err)
	}
	m.StartTask() // 停止之后启动没有效果
	if i := info(t, m, "任务"); !i.NextTime.IsZero() {
		t.Errorf("停止之后不应当再调度：%+v", i)
	}
}

func TestAddTaskErrors(t *testing.T) {
	m := NewManager()
	fn := func(context.Context) error { return nil }
	if err := m.AddTask(NewTask("任务", everySecond, fn)); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		task *Task
		want string
	}{
		{"nil", nil, "名字不能为空"},
		{"名字为空", NewTask("", everySecond, fn), "名字不能为空"},
		{"没有函数", NewTask("x", everySecond, nil), "没有任务函数"},
		{"表达式段数不对", NewTask("x", "* * * * *", fn), "应当有 6 段"},
		{"表达式越界", NewTask("x", "0 0 25 * * *", fn), "第 3 段（时）"},
		{"永远不会触发", NewTask("x", "0 0 0 30 2 *", fn), "永远不会触发"},
		{"重名", NewTask("任务", everySecond, fn), "已经存在"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.AddTask(tt.task)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("应当返回包含 %q 的错误，得到 %v", tt.want, err)
			}
		})
	}
	if err := m.Go("协程", nil); err == nil {
		t.Error("Go 的函数为 nil 应当返回错误")
	}
	if len(m.GetAllTasks()) != 1 {
		t.Errorf("出错的任务不应当被登记：%+v", m.GetAllTasks())
	}
	m.Stop()
}

// 各个方法并发调用：配合 -race 检查。
func TestConcurrentUse(t *testing.T) {
	m := NewManager()
	m.StartTask()
	fn := func(context.Context) error { return nil }
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 200 {
				name := fmt.Sprintf("任务%d-%d", g, i%5)
				_ = m.AddTask(NewTask(name, everySecond, fn))
				_ = m.GetAllTasks()
				_ = m.PauseTask(name)
				_ = m.RunTask(name)
				_ = m.ResumeTask(name)
				_ = m.GetAllGos()
				if i%3 == 0 {
					m.DeleteTask(name)
				}
			}
		})
	}
	wg.Wait()
	m.Stop()
	for _, i := range m.GetAllTasks() {
		if i.Running {
			t.Errorf("停止之后不应当还有任务在执行：%+v", i)
		}
	}
}
