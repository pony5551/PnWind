package pnlibs

import (
	"sync"

	log "github.com/pony5551/PnWind/core/pnlogrus"
)

type PnObjectsEx[K comparable, V any] struct {
	m sync.Map
}

func NewPnObjectsEx[K comparable, V any]() *PnObjectsEx[K, V] {
	return &PnObjectsEx[K, V]{}
}

func (sm *PnObjectsEx[K, V]) Store(key K, value V) {
	sm.m.Store(key, value)
}

func (sm *PnObjectsEx[K, V]) Load(key K) (V, bool) {
	value, ok := sm.m.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	return value.(V), true
}

func (sm *PnObjectsEx[K, V]) Delete(key K) {
	sm.m.Delete(key)
}

func (sm *PnObjectsEx[K, V]) Range(f func(key K, value V) bool) {
	sm.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func (sm *PnObjectsEx[K, V]) Clear(fn func(value V) error) {
	sm.m.Range(func(key, value any) bool {
		if fn != nil {
			err := fn(value.(V))
			if err != nil {
				log.Errorln("SafeMap.Clear:", err)
			}
		}
		sm.m.Delete(key)
		return true
	})
}

func init() {
	//sm := PnObjectsEx[string, int]{}
	//sm.Store("a", 1)
	//sm.Store("b", 2)
	//sm.Range(func(key string, value int) bool {
	//	println(key, value)
	//	return true
	//})
}
