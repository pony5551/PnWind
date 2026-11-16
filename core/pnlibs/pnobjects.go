package pnlibs

import (
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
)

const defaultShardCount = 32

// shard 是一个分片，持有独立的锁和 map
type shard[K comparable, V any] struct {
	mu   sync.RWMutex
	objs map[K]V
	_    [56]byte // 填充，避免 false sharing（与 cache line 对齐）
}

// PnObjects 分片并发安全的泛型 map
type PnObjects[K comparable, V any] struct {
	shards [defaultShardCount]shard[K, V]
	size   atomic.Int64
	hashFn func(K) uint64
}

// NewPnObjects 使用自定义哈希函数创建 PnObjects
func NewPnObjects[K comparable, V any](hashFn func(K) uint64) *PnObjects[K, V] {
	p := &PnObjects[K, V]{hashFn: hashFn}
	for i := range p.shards {
		p.shards[i].objs = make(map[K]V)
	}
	return p
}

func (p *PnObjects[K, V]) getShard(key K) *shard[K, V] {
	idx := p.hashFn(key) % defaultShardCount
	return &p.shards[idx]
}

// Store 存入或更新键值对。
// loaded=true 表示 key 已存在（本次为更新），loaded=false 表示本次为新增。
func (p *PnObjects[K, V]) Store(key K, value V) (loaded bool) {
	s := p.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.objs[key]
	s.objs[key] = value
	if !exists {
		p.size.Add(1)
	}
	return exists
}

// Load 读取键对应的值
func (p *PnObjects[K, V]) Load(key K) (V, bool) {
	s := p.getShard(key)
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.objs[key]
	return v, ok
}

// Delete 删除键，返回是否存在并被删除
func (p *PnObjects[K, V]) Delete(key K) bool {
	s := p.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.objs[key]; !ok {
		return false
	}
	delete(s.objs, key)
	p.size.Add(-1)
	return true
}

// LoadOrStore 若 key 不存在则存入；若已存在则返回旧值。
// actual 为实际存储的值，loaded=true 表示返回了已有值。
func (p *PnObjects[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	s := p.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.objs[key]; ok {
		return existing, true
	}
	s.objs[key] = value
	p.size.Add(1)
	return value, false
}

// Range 遍历所有键值对，f 返回 false 时停止遍历。
// 注意：遍历期间对其他分片的并发读写不受影响。
func (p *PnObjects[K, V]) Range(f func(key K, value V) bool) {
	for i := range p.shards {
		s := &p.shards[i]
		s.mu.RLock()
		for k, v := range s.objs {
			if !f(k, v) {
				s.mu.RUnlock()
				return
			}
		}
		s.mu.RUnlock()
	}
}

// Clear 遍历所有分片，逐个锁定清空，避免持有所有锁时执行回调。
// fn 在持有分片锁期间调用，应尽量轻量。
func (p *PnObjects[K, V]) Clear(fn func(V) error) {
	for i := range p.shards {
		s := &p.shards[i]
		s.mu.Lock()
		for k, v := range s.objs {
			if fn != nil {
				if err := fn(v); err != nil {
					fmt.Println("PnObjects.Clear error:", err)
				}
			}
			delete(s.objs, k)
		}
		s.mu.Unlock()
	}
	p.size.Store(0)
}

// Size 返回当前元素总数
func (p *PnObjects[K, V]) Size() int64 {
	return p.size.Load()
}

func NewStringHash() func(string) uint64 {
	return func(k string) uint64 {
		h := fnv.New64a()
		h.Write([]byte(k))
		return h.Sum64()
	}
}
