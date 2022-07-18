package cache

import (
	"sync"
)

type Cache[K comparable, V any] struct {
	m        sync.RWMutex
	valueMap map[K]V
}

func New[K comparable, V any](cap int) *Cache[K, V] {
	return &Cache[K, V]{
		valueMap: make(map[K]V, cap),
	}
}

func (c *Cache[K, V]) Set(k K, v V) {
	c.m.Lock()
	c.valueMap[k] = v
	c.m.Unlock()
}

func (c *Cache[K, V]) Update(k K, fn func(current V) V) {
	c.m.Lock()
	current, ok := c.valueMap[k]
	if !ok {
		c.m.Unlock()
		return
	}
	c.valueMap[k] = fn(current)
	c.m.Unlock()
}

func (c *Cache[K, V]) UpdateOrSet(k K, fn func(current V, exist bool) V) {
	c.m.Lock()
	current, ok := c.valueMap[k]
	c.valueMap[k] = fn(current, ok)
	c.m.Unlock()
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	c.m.RLock()
	v, ok := c.valueMap[k]
	c.m.RUnlock()
	return v, ok
}

func (c *Cache[K, V]) GetOrSet(k K, fn func() (V, error)) (res V, err error) {
	v, ok := c.Get(k)
	if ok {
		return v, nil
	}
	v, err = fn()
	if err != nil {
		return res, err
	}
	c.Set(k, v)
	return v, nil
}

// fn実行中はcをロックする
func (c *Cache[K, V]) GetOrSetLock(k K, fn func() (V, error)) (res V, err error) {
	c.m.RLock()
	v, ok := c.valueMap[k]
	c.m.RUnlock()
	if ok {
		return v, nil
	}
	c.m.Lock()
	v, ok = c.valueMap[k]
	if ok {
		c.m.Unlock()
		return v, nil
	}
	v, err = fn()
	if err != nil {
		c.m.Unlock()
		return res, err
	}
	c.valueMap[k] = v
	c.m.Unlock()
	return v, nil
}

func (c *Cache[K, V]) Delete(k K) (v V) {
	c.m.Lock()
	v = c.valueMap[k]
	delete(c.valueMap, k)
	c.m.Unlock()
	return v
}
