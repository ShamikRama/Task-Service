package cache

import (
	"context"
	"sync"
)

type MapCache[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func NewMapCache[K comparable, V any]() *MapCache[K, V] {
	return &MapCache[K, V]{m: make(map[K]V)}
}

func (c *MapCache[K, V]) Get(ctx context.Context, key K) (v V, ok bool) {
	_ = ctx
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok = c.m[key]
	return v, ok
}

func (c *MapCache[K, V]) Set(ctx context.Context, key K, v V) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = v
}

func (c *MapCache[K, V]) Delete(ctx context.Context, key K) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
}
