package mutexmap

import (
	"sync"
)

type MutexMap[K comparable] struct {
	m        sync.RWMutex
	mutexMap map[K]*sync.Mutex
}

func New[K comparable](cap int) *MutexMap[K] {
	return &MutexMap[K]{
		mutexMap: make(map[K]*sync.Mutex, cap),
	}
}

func (mm *MutexMap[K]) getOrSet(key K) *sync.Mutex {
	mm.m.RLock()
	m, ok := mm.mutexMap[key]
	mm.m.RUnlock()

	if ok {
		return m
	}

	mm.m.Lock()
	m, ok = mm.mutexMap[key]
	if ok {
		mm.m.Unlock()
		return m
	}
	m = &sync.Mutex{}
	mm.mutexMap[key] = m
	mm.m.Unlock()
	return m
}

func (mm *MutexMap[K]) Lock(key K) {
	mm.getOrSet(key).Lock()
}

func (mm *MutexMap[K]) get(key K) (*sync.Mutex, bool) {
	mm.m.RLock()
	m, ok := mm.mutexMap[key]
	mm.m.RUnlock()

	return m, ok
}

func (mm *MutexMap[K]) Unlock(key K) {
	m, ok := mm.get(key)

	if ok {
		m.Unlock()
	}
}

type RWMutexMap[K comparable] struct {
	m        sync.RWMutex
	mutexMap map[K]*sync.RWMutex
}

func NewRW[K comparable](cap int) *RWMutexMap[K] {
	return &RWMutexMap[K]{
		mutexMap: make(map[K]*sync.RWMutex, cap),
	}
}

func (mm *RWMutexMap[K]) getOrSet(key K) *sync.RWMutex {
	mm.m.RLock()
	m, ok := mm.mutexMap[key]
	mm.m.RUnlock()

	if ok {
		return m
	}

	mm.m.Lock()
	m, ok = mm.mutexMap[key]
	if ok {
		mm.m.Unlock()
		return m
	}
	m = &sync.RWMutex{}
	mm.mutexMap[key] = m
	mm.m.Unlock()
	return m
}

func (mm *RWMutexMap[K]) Lock(key K) {
	mm.getOrSet(key).Lock()
}

func (mm *RWMutexMap[K]) RLock(key K) {
	mm.getOrSet(key).RLock()
}

func (mm *RWMutexMap[K]) get(key K) (*sync.RWMutex, bool) {
	mm.m.RLock()
	m, ok := mm.mutexMap[key]
	mm.m.RUnlock()

	return m, ok
}

func (mm *RWMutexMap[K]) Unlock(key K) {
	m, ok := mm.get(key)

	if ok {
		m.Unlock()
	}
}

func (mm *RWMutexMap[K]) RUnlock(key K) {
	m, ok := mm.get(key)

	if ok {
		m.RUnlock()
	}
}
