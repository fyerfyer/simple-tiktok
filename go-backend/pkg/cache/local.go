package cache

import (
	"sync"
	"time"
)

// LocalCacheItem 本地缓存项
type LocalCacheItem struct {
	Value      interface{}
	Expiration int64
}

// LocalCache 本地缓存实现
type LocalCache struct {
	items   map[string]*LocalCacheItem
	mutex   sync.RWMutex
	janitor *time.Ticker
	stop    chan bool
}

// NewLocalCache 创建本地缓存
func NewLocalCache(cleanupInterval time.Duration) *LocalCache {
	cache := &LocalCache{
		items: make(map[string]*LocalCacheItem),
		stop:  make(chan bool),
	}

	if cleanupInterval > 0 {
		cache.janitor = time.NewTicker(cleanupInterval)
		go cache.cleanup()
	}

	return cache
}

// Get 获取缓存
func (c *LocalCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
		return nil, false
	}

	return item.Value, true
}

// Set 设置缓存
func (c *LocalCache) Set(key string, value interface{}, duration time.Duration) {
	var expiration int64
	if duration > 0 {
		expiration = time.Now().Add(duration).UnixNano()
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = &LocalCacheItem{
		Value:      value,
		Expiration: expiration,
	}
}

// Delete 删除缓存
func (c *LocalCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.items, key)
}

// Clear 清空缓存
func (c *LocalCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.items = make(map[string]*LocalCacheItem)
}

// Size 获取缓存大小
func (c *LocalCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.items)
}

// Close 关闭缓存
func (c *LocalCache) Close() {
	if c.janitor != nil {
		c.janitor.Stop()
	}
	close(c.stop)
}

// cleanup 清理过期缓存
func (c *LocalCache) cleanup() {
	for {
		select {
		case <-c.janitor.C:
			c.deleteExpired()
		case <-c.stop:
			return
		}
	}
}

func (c *LocalCache) deleteExpired() {
	now := time.Now().UnixNano()
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for key, item := range c.items {
		if item.Expiration > 0 && now > item.Expiration {
			delete(c.items, key)
		}
	}
}
