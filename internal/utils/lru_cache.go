package utils

import (
	"container/list"
	"sync"
)

// LRUCache is a thread-safe LRU cache implementation.
type LRUCache[K comparable, V any] struct {
	mu       sync.RWMutex
	capacity int
	items    map[K]*list.Element
	order    *list.List
}

// cacheEntry represents an entry in the cache.
type cacheEntry[K comparable, V any] struct {
	key   K
	value V
}

// NewLRUCache creates a new LRU cache with the specified capacity.
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		order:    list.New(),
	}
}

// Get retrieves a value from the cache.
// Returns the value and true if found, zero value and false otherwise.
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		return elem.Value.(*cacheEntry[K, V]).value, true
	}

	var zero V
	return zero, false
}

// Peek retrieves a value without updating recency.
// Returns the value and true if found, zero value and false otherwise.
func (c *LRUCache[K, V]) Peek(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if elem, ok := c.items[key]; ok {
		return elem.Value.(*cacheEntry[K, V]).value, true
	}

	var zero V
	return zero, false
}

// Set adds or updates a value in the cache.
func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		elem.Value.(*cacheEntry[K, V]).value = value
		return
	}

	// Add new entry
	elem := c.order.PushFront(&cacheEntry[K, V]{key: key, value: value})
	c.items[key] = elem

	// Evict oldest if over capacity
	if c.order.Len() > c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			entry := oldest.Value.(*cacheEntry[K, V])
			delete(c.items, entry.key)
		}
	}
}

// Delete removes a key from the cache.
// Returns true if the key was found and removed.
func (c *LRUCache[K, V]) Delete(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.Remove(elem)
		delete(c.items, key)
		return true
	}
	return false
}

// Has checks if a key exists in the cache.
func (c *LRUCache[K, V]) Has(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.items[key]
	return ok
}

// Clear removes all entries from the cache.
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*list.Element)
	c.order = list.New()
}

// Size returns the current number of entries in the cache.
func (c *LRUCache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}

// MemoizeWithLRU creates a memoized function with LRU eviction.
// The key function generates a cache key from the function arguments.
func MemoizeWithLRU[K comparable, V any](
	fn func(K) V,
	maxCacheSize int,
) func(K) V {
	cache := NewLRUCache[K, V](maxCacheSize)

	return func(key K) V {
		if val, ok := cache.Get(key); ok {
			return val
		}

		result := fn(key)
		cache.Set(key, result)
		return result
	}
}

// MemoizeAsyncWithLRU creates a memoized async function with LRU eviction.
// The cache stores promises (channels) to deduplicate concurrent calls.
func MemoizeAsyncWithLRU[K comparable, V any](
	fn func(K) (V, error),
	maxCacheSize int,
) func(K) (V, error) {
	type promise struct {
		result V
		err    error
		ready  chan struct{}
	}

	cache := NewLRUCache[K, *promise](maxCacheSize)
	var mu sync.Mutex

	return func(key K) (V, error) {
		mu.Lock()

		// Check if we have a cached promise
		if p, ok := cache.Peek(key); ok {
			mu.Unlock()
			<-p.ready
			return p.result, p.err
		}

		// Create new promise
		p := &promise{ready: make(chan struct{})}
		cache.Set(key, p)
		mu.Unlock()

		// Execute the function
		result, err := fn(key)

		p.result = result
		p.err = err
		close(p.ready)

		return result, err
	}
}

// CacheInterface provides an interface for cache operations.
type CacheInterface[K comparable, V any] interface {
	Get(key K) (V, bool)
	Set(key K, value V)
	Delete(key K) bool
	Has(key K) bool
	Clear()
	Size() int
}
