package pntask

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	log "github.com/pony5551/PnWind/core/pnlogrus"
)

// Manager 任务管理器。全局只有一个，在 core.GConfig.Tasks。
type Manager struct {
	mu      sync.Mutex
	tasks   map[string]*Task     // 定时任务，key 是任务名
	running map[string]struct{}  // 有一轮正在执行的任务名。按名字记：任务删除之后，它的最后一轮跑完之前同名的新任务不启动
	gos     map[*GoInfo]struct{} // 还在运行的常驻协程
	started bool                 // 调度协程已经启动
	stopped bool                 // 已经停止，不再接受任何新的活
	wake    chan struct{}        // 任务有增删、暂停、恢复时，叫醒调度协程重新算下一次时间
	quit    chan struct{}        // Stop 时关闭
	wg      sync.WaitGroup       // 调度协程、正在执行的任务、常驻协程
}

// NewManager 创建任务管理器。
func NewManager() *Manager {
	return &Manager{
		tasks:   make(map[string]*Task),
		running: make(map[string]struct{}),
		gos:     make(map[*GoInfo]struct{}),
		wake:    make(chan struct{}, 1),
		quit:    make(chan struct{}),
	}
}

// AddTask 登记一个定时任务。调度启动之后也可以添加，立即生效。
// 表达式不合法或者永远不会触发、名字为空或重复、任务函数为 nil、管理器已停止，都返回错误。
func (m *Manager) AddTask(t *Task) error {
	if t == nil || t.name == "" {
		return errors.New("定时任务的名字不能为空")
	}
	if t.fn == nil {
		return fmt.Errorf("定时任务 %s 没有任务函数", t.name)
	}
	sched, err := parse(t.spec)
	if err != nil {
		return fmt.Errorf("定时任务 %s 的 %w", t.name, err)
	}
	now := time.Now()
	if sched.next(now).IsZero() {
		return fmt.Errorf("定时任务 %s 的 cron 表达式 %q 永远不会触发", t.name, t.spec)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return ErrStopped
	}
	if _, exists := m.tasks[t.name]; exists {
		return fmt.Errorf("定时任务 %s 已经存在", t.name)
	}
	t.sched = sched
	if m.started && !t.paused {
		t.next = sched.next(now)
	}
	m.tasks[t.name] = t
	m.wakeUp()
	return nil
}

// DeleteTask 删除一个定时任务，返回它是否存在。正在执行的那一轮照常跑完；
// 它跑完之前又添加了同名的任务，新任务先不启动（到点的那一轮跳过），两轮不会同时执行。
func (m *Manager) DeleteTask(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.tasks[name]
	delete(m.tasks, name)
	m.wakeUp()
	return exists
}

// GetAllTasks 返回全部定时任务的状态，按名字排序。
func (m *Manager) GetAllTasks() []TaskInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	infos := make([]TaskInfo, 0, len(m.tasks))
	for _, t := range m.tasks {
		info := TaskInfo{
			Name: t.name, Spec: t.spec, Paused: t.paused, Running: m.busy(t.name),
			PrevTime: t.prev, LastCost: t.lastCost, LastError: t.lastErr,
			Runs: t.runs, Fails: t.fails, Skips: t.skips,
		}
		if m.started && !m.stopped && !t.paused {
			info.NextTime = t.next
		}
		infos = append(infos, info)
	}
	slices.SortFunc(infos, func(a, b TaskInfo) int { return strings.Compare(a.Name, b.Name) })
	return infos
}

// StartTask 开始按时间表调度。由 core.Boot 在服务启动之后调用。重复调用、停止之后调用都没有效果。
func (m *Manager) StartTask() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started || m.stopped {
		return
	}
	m.started = true
	now := time.Now()
	for _, t := range m.tasks {
		if !t.paused {
			t.next = t.sched.next(now)
		}
	}
	m.wg.Add(1)
	go m.schedule()
}

// RunTask 手动触发：立刻在新协程里执行一次，不改变下一次的计划时间；它还没跑完时到点的那一轮会被跳过。
// 调度没启动、任务被暂停时也可以用。
// 任务不存在返回 ErrTaskNotFound，这个名字有一轮正在执行（包括同名的旧任务最后一轮还没跑完）返回 ErrTaskRunning，
// 管理器已停止返回 ErrStopped。
func (m *Manager) RunTask(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return ErrStopped
	}
	t, exists := m.tasks[name]
	if !exists {
		return ErrTaskNotFound
	}
	if m.busy(name) {
		return ErrTaskRunning
	}
	m.start(t, time.Now())
	return nil
}

// busy 这个名字是不是有一轮正在执行。按名字看，不按任务对象看：任务删除之后它的最后一轮照常跑完，
// 这期间添加的同名任务不能启动，否则两轮会同时执行。调用方必须持有 mu。
func (m *Manager) busy(name string) bool {
	_, isOk := m.running[name]
	return isOk
}

// PauseTask 暂停一个任务：不再按时间表触发，手动触发仍然可以。任务不存在返回 ErrTaskNotFound。
func (m *Manager) PauseTask(name string) error {
	return m.setPaused(name, true)
}

// ResumeTask 恢复一个暂停的任务，从现在起重新按时间表触发。任务不存在返回 ErrTaskNotFound。
func (m *Manager) ResumeTask(name string) error {
	return m.setPaused(name, false)
}

func (m *Manager) setPaused(name string, paused bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, exists := m.tasks[name]
	if !exists {
		return ErrTaskNotFound
	}
	if t.paused != paused {
		t.paused = paused
		t.next = time.Time{}
		if !paused && m.started {
			t.next = t.sched.next(time.Now())
		}
		m.wakeUp()
	}
	return nil
}

// Go 启动一个常驻的后台协程，例如消息队列的消费循环。退出时任务管理器一直等到它返回，
// 所以 fn 要留意 Quit() 通道：通道关闭表示进程开始退出，做完手上的事就该返回了。
//
//	tasks := core.GConfig.Tasks
//	tasks.Go("消费订单消息", func(ctx context.Context) error {
//		for {
//			select {
//			case <-tasks.Quit():
//				return nil
//			case msg := <-deliveries:
//				handle(ctx, msg)
//			}
//		}
//	})
//
// fn 返回的错误和 panic 会写日志。管理器已停止时返回 ErrStopped。
func (m *Manager) Go(name string, fn TaskFunc) error {
	if fn == nil {
		return fmt.Errorf("常驻协程 %s 没有函数", name)
	}
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return ErrStopped
	}
	info := &GoInfo{Name: name, StartTime: time.Now()}
	m.gos[info] = struct{}{}
	m.wg.Add(1)
	m.mu.Unlock()

	go func() {
		defer m.wg.Done()
		err := errGoexit
		defer func() {
			m.mu.Lock()
			delete(m.gos, info)
			m.mu.Unlock()
			if err != nil {
				logFailure("常驻协程", name, err)
			}
		}()
		err = call(fn)
	}()
	return nil
}

// GetAllGos 返回还在运行的常驻协程，按名字排序。
func (m *Manager) GetAllGos() []GoInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	infos := make([]GoInfo, 0, len(m.gos))
	for info := range m.gos {
		infos = append(infos, *info)
	}
	slices.SortFunc(infos, func(a, b GoInfo) int {
		if c := strings.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return a.StartTime.Compare(b.StartTime)
	})
	return infos
}

// Quit 返回一个通道，进程开始退出（Stop 被调用）时关闭。常驻协程据此收尾返回。
func (m *Manager) Quit() <-chan struct{} {
	return m.quit
}

// Stop 停止任务管理器：不再调度，关闭 Quit 通道，然后等正在执行的任务和常驻协程全部返回。
//
// 不设超时（优雅退出必须等后台任务完成）。可以重复调用、并发调用；没调过 StartTask 也能正常返回。
// 不要在任务函数、常驻协程里调 Stop：它要等自己返回，永远等不到。
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.stopped {
		m.stopped = true
		close(m.quit)
	}
	m.mu.Unlock()
	m.wg.Wait()
}

// wakeUp 叫醒调度协程。已经被叫过了就什么也不做；调度协程还没启动时留下一个信号，启动后多算一遍，没有影响。
func (m *Manager) wakeUp() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

// schedule 是调度协程：把到点的任务启动起来，然后睡到最近的下一个时刻；被叫醒时重新算。
func (m *Manager) schedule() {
	defer m.wg.Done()
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for {
		wait, skipped, stopped := m.fireDue()
		if stopped {
			return
		}
		for _, name := range skipped {
			log.Warnf("定时任务 %s 上一轮还没跑完，跳过这一轮", name)
		}
		timer.Reset(wait)
		select {
		case <-timer.C:
		case <-m.wake:
		case <-m.quit:
			return
		}
	}
}

// fireDue 启动所有到点的任务，返回离最近的下一个时刻还有多久，以及因为上一轮没跑完而被跳过的任务。
//
// 下一次时间从"现在"算起，不从上一次的计划时间算起：调度协程醒晚了，错过的那几轮不补，也不会连着触发。
func (m *Manager) fireDue() (wait time.Duration, skipped []string, stopped bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return 0, nil, true
	}
	now := time.Now()
	wait = time.Hour // 没有任务时睡着，等 wakeUp 来叫
	for _, t := range m.tasks {
		if t.paused || t.next.IsZero() { // 暂停的不看；next 是零值表示五年之内都不会再触发
			continue
		}
		if !t.next.After(now) {
			if m.busy(t.name) {
				t.skips++
				skipped = append(skipped, t.name)
			} else {
				m.start(t, now)
			}
			t.next = t.sched.next(now)
		}
		if d := t.next.Sub(now); d < wait {
			wait = d
		}
	}
	return wait, skipped, false
}

// start 在新协程里执行一次任务。调用方必须持有 mu，并且已经确认这个名字没有一轮正在执行（busy）、管理器没有停止。
func (m *Manager) start(t *Task, now time.Time) {
	m.running[t.name] = struct{}{}
	t.prev = now
	m.wg.Add(1)
	go m.run(t)
}

// run 执行一次任务并记账。
func (m *Manager) run(t *Task) {
	defer m.wg.Done()
	begin := time.Now()
	err := errGoexit
	defer func() {
		m.mu.Lock()
		delete(m.running, t.name)
		t.runs++
		t.lastCost = time.Since(begin)
		t.lastErr = ""
		if err != nil {
			t.fails++
			t.lastErr = err.Error()
		}
		m.mu.Unlock()
		if err != nil {
			logFailure("定时任务", t.name, err)
		}
	}()
	err = call(t.fn)
}
