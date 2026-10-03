package hn

import (
	"sync"
	"time"
)

type cacheEntry[V any] struct {
	v   V
	exp time.Time
}

type cache[K comparable, V any] struct {
	mu    sync.RWMutex
	ttl   time.Duration
	m     map[K]cacheEntry[V]
	swept time.Time
}

func newCache[K comparable, V any](ttl time.Duration) *cache[K, V] {
	return &cache[K, V]{ttl: ttl, m: map[K]cacheEntry[V]{}, swept: time.Now()}
}

func (c *cache[K, V]) get(k K) (V, bool) {
	c.mu.RLock()
	e, ok := c.m[k]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.exp) {
		var zero V
		return zero, false
	}
	return e.v, true
}

func (c *cache[K, V]) set(k K, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.m[k] = cacheEntry[V]{v, now.Add(c.ttl)}
	if now.Sub(c.swept) > 2*c.ttl {
		for k, e := range c.m {
			if now.After(e.exp) {
				delete(c.m, k)
			}
		}
		c.swept = now
	}
}
