package util

import (
	"sync"
)

type Map[K comparable, V any] struct {
	sync.RWMutex
	m map[K]V
}

func (m *Map[K, V]) init() {
	if m.m == nil {
		m.m = make(map[K]V)
	}
}

func (m *Map[K, V]) UnsafeGet(key K) V {
	if m.m == nil {
		var zero V
		return zero
	} else {
		return m.m[key]
	}
}

func (m *Map[K, V]) Get(key K) V {
	m.RLock()
	defer m.RUnlock()
	return m.UnsafeGet(key)
}

func (m *Map[K, V]) UnsafeSet(key K, value V) {
	m.init()
	m.m[key] = value
}

func (m *Map[K, V]) Set(key K, value V) {
	m.Lock()
	defer m.Unlock()
	m.UnsafeSet(key, value)
}

func (m *Map[K, V]) TestAndSet(key K, value V) V {
	m.Lock()
	defer m.Unlock()

	m.init()

	if v, ok := m.m[key]; ok {
		return v
	} else {
		m.m[key] = value
		var zero V
		return zero
	}
}

func (m *Map[K, V]) UnsafeDel(key K) {
	m.init()
	delete(m.m, key)
}

func (m *Map[K, V]) Del(key K) {
	m.Lock()
	defer m.Unlock()
	m.UnsafeDel(key)
}

func (m *Map[K, V]) UnsafeLen() int {
	if m.m == nil {
		return 0
	} else {
		return len(m.m)
	}
}

func (m *Map[K, V]) Len() int {
	m.RLock()
	defer m.RUnlock()
	return m.UnsafeLen()
}

func (m *Map[K, V]) UnsafeRange(f func(K, V)) {
	if m.m == nil {
		return
	}
	for k, v := range m.m {
		f(k, v)
	}
}

func (m *Map[K, V]) RLockRange(f func(K, V)) {
	m.RLock()
	defer m.RUnlock()
	m.UnsafeRange(f)
}

func (m *Map[K, V]) LockRange(f func(K, V)) {
	m.Lock()
	defer m.Unlock()
	m.UnsafeRange(f)
}
