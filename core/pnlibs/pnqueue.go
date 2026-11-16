package pnlibs

import (
	"sync"
	"sync/atomic"
)

// 代表每一个节点
type pnnode[T any] struct {
	data T
	next *pnnode[T]
}

// 为每种类型创建独立的对象池
type nodePool[T any] struct {
	pool sync.Pool
}

// 创建新的节点池
func newNodePool[T any]() *nodePool[T] {
	return &nodePool[T]{
		pool: sync.Pool{
			New: func() any {
				return new(pnnode[T])
			},
		},
	}
}

// 获取节点
func (np *nodePool[T]) getNode() *pnnode[T] {
	return np.pool.Get().(*pnnode[T])
}

// 释放节点
func (np *nodePool[T]) releaseNode(node *pnnode[T]) {
	// 我们不能将泛型T设为nil，但可以清除next引用
	var zero T
	node.data = zero
	node.next = nil
	np.pool.Put(node)
}

type FnCheckElement[T any] func(T) bool

type PnQueue[T any] struct {
	// 头节点
	head *pnnode[T]
	// 队尾节点
	tail *pnnode[T]
	// 队列大小
	size atomic.Int64
	np   *nodePool[T]
	// 队列锁
	mu sync.Mutex
	// 元素检测函数
	fnCheckElement FnCheckElement[T]
}

func NewQueue[T any](fnCheck FnCheckElement[T]) *PnQueue[T] {
	return &PnQueue[T]{
		np:             newNodePool[T](),
		fnCheckElement: fnCheck,
	}
}

// Push 尾插法
func (q *PnQueue[T]) Push(element T) bool {
	// 移除与nil比较的检查，因为泛型T可能是值类型
	//if element == nil {
	//	return false
	//}
	if q.fnCheckElement != nil {
		if !q.fnCheckElement(element) {
			return false
		}
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	//n := new(pnnode)
	// 使用对象池获取节点
	n := q.np.getNode()

	n.data = element

	if q.tail == nil {
		q.head = n
		q.tail = n
	} else {
		q.tail.next = n
		q.tail = n
	}
	q.size.Add(1)
	return true
}

// PushHead 头插法，在队列头部插入一个元素
func (q *PnQueue[T]) PushHead(element T) bool {
	// 移除与nil比较的检查，因为泛型T可能是值类型
	//if element == nil {
	//	return false
	//}
	if q.fnCheckElement != nil {
		if !q.fnCheckElement(element) {
			return false
		}
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	//n := new(pnnode)
	// 使用对象池获取节点
	n := q.np.getNode()

	n.data = element
	if q.head == nil {
		q.head = n
		q.tail = n
	} else {
		n.next = q.head
		q.head = n
	}
	q.size.Add(1)
	return true
}

// Pop 获取并删除队列头部的元素
func (q *PnQueue[T]) Pop() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.head == nil {
		var zero T
		return zero, false
	}

	n := q.head
	result := n.data

	// 代表队列中仅一个元素
	if n.next == nil {
		q.head = nil
		q.tail = nil

	} else {
		q.head = n.next
	}
	q.size.Add(-1)

	// 释放节点回对象池
	q.np.releaseNode(n)

	return result, true
}

// Pops 取出并删除队列头部最多 num 个元素
func (q *PnQueue[T]) Pops(num int) []T {
	if num <= 0 {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.head == nil {
		return nil
	}

	result := make([]T, 0, num)
	count := 0

	for q.head != nil && count < num {
		n := q.head
		result = append(result, n.data)
		q.head = n.next
		q.np.releaseNode(n)
		count++
	}

	if q.head == nil {
		q.tail = nil
	}

	q.size.Add(-int64(count))
	return result
}

// 添加批量处理方法提高效率
func (q *PnQueue[T]) PopBatch(maxItems int) []T {
	if maxItems <= 0 {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.head == nil {
		return nil
	}

	result := make([]T, 0, maxItems)
	count := 0

	for q.head != nil && count < maxItems {
		n := q.head
		result = append(result, n.data)
		q.head = n.next

		// 释放节点回对象池
		q.np.releaseNode(n)

		count++
	}

	if q.head == nil {
		q.tail = nil
	}

	q.size.Add(-int64(count))
	return result
}

func (q *PnQueue[T]) Size() int64 {
	return q.size.Load()
}

// 清空队列并释放所有节点到对象池
func (q *PnQueue[T]) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 释放所有节点到对象池
	current := q.head
	for current != nil {
		next := current.next
		q.np.releaseNode(current)
		current = next
	}

	q.head = nil
	q.tail = nil
	q.size.Store(0)
}

// RemoveIf 遍历队列，移除所有满足 predicate 的节点，返回移除数量
func (q *PnQueue[T]) RemoveIf(predicate func(T) bool) int {
	q.mu.Lock()
	defer q.mu.Unlock()

	removed := 0
	var prev *pnnode[T]
	current := q.head

	for current != nil {
		next := current.next // 必须在 releaseNode 前保存
		if predicate(current.data) {
			if prev == nil {
				q.head = next
			} else {
				prev.next = next
			}
			if current == q.tail {
				q.tail = prev
			}
			q.np.releaseNode(current)
			removed++
			// prev 不变
		} else {
			prev = current
		}
		current = next
	}

	if removed > 0 {
		q.size.Add(-int64(removed))
	}
	return removed
}
