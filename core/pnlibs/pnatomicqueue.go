package pnlibs

import (
	"sync"
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

// pnAtomicNode 无锁队列节点
type pnAtomicNode[T any] struct {
	data T
	next atomic.Pointer[pnAtomicNode[T]]
}

// PnAtomicQueue 无锁 MPMC 原子队列
//
// 算法：Michael-Scott Queue（1996），工业级无锁队列经典实现
//
// 特性：
//   - Push / Pop 全程无锁，基于 CAS 实现
//   - PopWait 在队列为空时挂起当前 goroutine，有数据或关闭时唤醒
//   - maxSize > 0 时启用软上限，超出后 Push 返回 false（TOCTOU 允许短暂超限）
//   - head / tail / size 分布在不同缓存行，避免伪共享（false sharing）
//   - Go GC 保证无 ABA 问题，无需 hazard pointer
//
// 注意：不支持 PushHead（无锁双端队列需要更复杂的算法）
type PnAtomicQueue[T any] struct {
	// 缓存行填充：head 与 tail 独占各自的缓存行，防止伪共享
	_       cpu.CacheLinePad
	head    atomic.Pointer[pnAtomicNode[T]]
	_       cpu.CacheLinePad
	tail    atomic.Pointer[pnAtomicNode[T]]
	_       cpu.CacheLinePad
	size    atomic.Int64
	_       cpu.CacheLinePad
	maxSize int64 // 软上限，0 表示无限制

	// 阻塞通知：容量为 1 的信号槽
	// Push 成功后发送信号（非阻塞），PopWait 等待信号后重试 Pop
	// 容量 1 足够：消费者收到信号后会循环 Pop 直到队列为空
	sema chan struct{}

	// 关闭信号，Close() 时关闭此 channel，唤醒所有阻塞的 PopWait
	done      chan struct{}
	closeOnce sync.Once
	closed    atomic.Bool

	fnCheckElement FnCheckElement[T]
}

// NewAtomicQueue 创建无锁原子队列
//   - maxSize: 软上限，0 表示无限制
//   - fnCheck: 可选元素校验函数，传 nil 表示不校验
func NewAtomicQueue[T any](maxSize int64, fnCheck FnCheckElement[T]) *PnAtomicQueue[T] {
	q := &PnAtomicQueue[T]{
		maxSize:        maxSize,
		sema:           make(chan struct{}, 1),
		done:           make(chan struct{}),
		fnCheckElement: fnCheck,
	}
	sentinel := &pnAtomicNode[T]{}
	q.head.Store(sentinel)
	q.tail.Store(sentinel)
	return q
}

// Push 在队尾插入元素（无锁，线程安全）
// 队列已关闭或超出软上限时返回 false
func (q *PnAtomicQueue[T]) Push(val T) bool {
	if q.closed.Load() {
		return false
	}
	if q.maxSize > 0 && q.size.Load() >= q.maxSize {
		return false
	}
	if q.fnCheckElement != nil && !q.fnCheckElement(val) {
		return false
	}

	node := &pnAtomicNode[T]{data: val}
	for {
		tail := q.tail.Load()
		next := tail.next.Load()
		if tail != q.tail.Load() {
			continue
		}
		if next == nil {
			if tail.next.CompareAndSwap(nil, node) {
				q.tail.CompareAndSwap(tail, node)
				q.size.Add(1)
				// 通知阻塞的 PopWait（非阻塞，已有信号时跳过）
				select {
				case q.sema <- struct{}{}:
				default:
				}
				return true
			}
		} else {
			q.tail.CompareAndSwap(tail, next)
		}
	}
}

// Pop 取出队头元素（无锁，线程安全）
// 队列为空时立即返回零值和 false，不阻塞
func (q *PnAtomicQueue[T]) Pop() (T, bool) {
	for {
		head := q.head.Load()
		tail := q.tail.Load()
		next := head.next.Load()
		if head != q.head.Load() {
			continue
		}
		if head == tail {
			if next == nil {
				var zero T
				return zero, false
			}
			q.tail.CompareAndSwap(tail, next)
		} else {
			val := next.data
			if q.head.CompareAndSwap(head, next) {
				// next 成为新哨兵，清空 data 辅助 GC
				var zero T
				next.data = zero
				q.size.Add(-1)
				return val, true
			}
		}
	}
}

// PopWait 阻塞式取出队头元素
//   - 有数据时立即返回
//   - 队列为空时挂起，直到有新数据或队列关闭
//   - 队列关闭且清空后返回零值和 false
func (q *PnAtomicQueue[T]) PopWait() (T, bool) {
	for {
		// 优先尝试直接 Pop（避免不必要的 channel 操作）
		if val, ok := q.Pop(); ok {
			return val, true
		}
		// 队列为空，等待信号
		select {
		case <-q.sema:
			// 收到信号，重试 Pop（信号只代表"曾经有数据"，不保证此刻仍有）
		case <-q.done:
			// 队列已关闭，消费剩余数据后退出
			return q.Pop()
		}
	}
}

// Close 关闭队列，唤醒所有阻塞的 PopWait
// 关闭后 Push 立即返回 false，已入队数据可继续被 Pop/PopWait 消费
func (q *PnAtomicQueue[T]) Close() {
	q.closeOnce.Do(func() {
		q.closed.Store(true)
		close(q.done)
	})
}

// PopBatch 批量取出最多 maxItems 个元素（非阻塞）
func (q *PnAtomicQueue[T]) PopBatch(maxItems int) []T {
	if maxItems <= 0 {
		return nil
	}
	result := make([]T, 0, maxItems)
	for range maxItems {
		val, ok := q.Pop()
		if !ok {
			break
		}
		result = append(result, val)
	}
	return result
}

// Notif 返回数据到达通知 channel，可用于外部 select 同时监听多个队列
// 收到信号后应循环调用 Pop 直到返回 false（信号只保证"曾有数据"，不保证此刻仍有）
func (q *PnAtomicQueue[T]) Notif() <-chan struct{} {
	return q.sema
}

// Size 返回当前队列中的元素数量
func (q *PnAtomicQueue[T]) Size() int64 {
	return q.size.Load()
}

// Clear 清空队列（循环 Pop，并发安全）
func (q *PnAtomicQueue[T]) Clear() {
	for {
		if _, ok := q.Pop(); !ok {
			break
		}
	}
}
