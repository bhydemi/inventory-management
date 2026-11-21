// Package cache provides a tiny in-memory, TTL-based cache for GET
// responses. It's intentionally simple: a map guarded by a mutex, not a
// general-purpose cache library, since the only thing this project needs
// cached is a handful of read endpoints.
package cache

import (
	"sync"
	"time"
)

type entry struct {
	value     []byte
	expiresAt time.Time
}

type Cache struct {
	mu      sync.RWMutex
	entries map[string]entry
	ttl     time.Duration
}

// New creates a cache whose entries expire after ttl.
func New(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]entry),
		ttl:     ttl,
	}
}

// Get returns the cached bytes for key, if present and not expired.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

// Set stores value under key for the cache's configured TTL.
func (c *Cache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = entry{value: value, expiresAt: time.Now().Add(c.ttl)}
}

// Flush clears every cached entry. Called after any write (create, update,
// delete) so a client never reads stale data back from the cache.
func (c *Cache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]entry)
}
