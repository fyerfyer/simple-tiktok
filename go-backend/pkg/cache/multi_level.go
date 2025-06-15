package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

// MultiLevelCache 多级缓存
type MultiLevelCache struct {
	local  *LocalCache
	redis  *RedisCache
	config *CacheConfig
}

// CacheConfig 缓存配置
type CacheConfig struct {
	LocalTTL time.Duration // 本地缓存TTL
	RedisTTL time.Duration // Redis缓存TTL
	EnableL1 bool          // 启用一级缓存(本地)
	EnableL2 bool          // 启用二级缓存(Redis)
}

// NewMultiLevelCache 创建多级缓存
func NewMultiLevelCache(redisClient *redis.Client, config *CacheConfig) *MultiLevelCache {
	if config == nil {
		config = &CacheConfig{
			LocalTTL: 5 * time.Minute,
			RedisTTL: 30 * time.Minute,
			EnableL1: true,
			EnableL2: true,
		}
	}

	cache := &MultiLevelCache{
		redis:  NewRedisCache(redisClient),
		config: config,
	}

	if config.EnableL1 {
		cache.local = NewLocalCache(time.Minute)
	}

	return cache
}

// Get 获取缓存
func (c *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, bool) {
	// 先从本地缓存获取
	if c.config.EnableL1 && c.local != nil {
		if value, exists := c.local.Get(key); exists {
			return value, true
		}
	}

	// 再从Redis获取
	if c.config.EnableL2 {
		val, err := c.redis.Get(ctx, key)
		if err == nil {
			var result interface{}
			if err := json.Unmarshal([]byte(val), &result); err == nil {
				// 回写到本地缓存
				if c.config.EnableL1 && c.local != nil {
					c.local.Set(key, result, c.config.LocalTTL)
				}
				return result, true
			}
		}
	}

	return nil, false
}

// Set 设置缓存
func (c *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, duration time.Duration) error {
	// 设置本地缓存
	if c.config.EnableL1 && c.local != nil {
		localTTL := c.config.LocalTTL
		if duration > 0 && duration < localTTL {
			localTTL = duration
		}
		c.local.Set(key, value, localTTL)
	}

	// 设置Redis缓存
	if c.config.EnableL2 {
		redisTTL := c.config.RedisTTL
		if duration > 0 {
			redisTTL = duration
		}
		return c.redis.SetJSON(ctx, key, value, redisTTL)
	}

	return nil
}

// Delete 删除缓存
func (c *MultiLevelCache) Delete(ctx context.Context, key string) error {
	// 删除本地缓存
	if c.config.EnableL1 && c.local != nil {
		c.local.Delete(key)
	}

	// 删除Redis缓存
	if c.config.EnableL2 {
		return c.redis.Del(ctx, key)
	}

	return nil
}

// GetString 获取字符串
func (c *MultiLevelCache) GetString(ctx context.Context, key string) (string, error) {
	// 先从本地缓存获取
	if c.config.EnableL1 && c.local != nil {
		if value, exists := c.local.Get(key); exists {
			if str, ok := value.(string); ok {
				return str, nil
			}
		}
	}

	// 从Redis获取
	if c.config.EnableL2 {
		val, err := c.redis.Get(ctx, key)
		if err == nil {
			// 回写到本地缓存
			if c.config.EnableL1 && c.local != nil {
				c.local.Set(key, val, c.config.LocalTTL)
			}
			return val, nil
		}
		return "", err
	}

	return "", redis.Nil
}

// SetString 设置字符串
func (c *MultiLevelCache) SetString(ctx context.Context, key, value string, duration time.Duration) error {
	// 设置本地缓存
	if c.config.EnableL1 && c.local != nil {
		localTTL := c.config.LocalTTL
		if duration > 0 && duration < localTTL {
			localTTL = duration
		}
		c.local.Set(key, value, localTTL)
	}

	// 设置Redis缓存
	if c.config.EnableL2 {
		redisTTL := c.config.RedisTTL
		if duration > 0 {
			redisTTL = duration
		}
		return c.redis.Set(ctx, key, value, redisTTL)
	}

	return nil
}

// Invalidate 失效缓存
func (c *MultiLevelCache) Invalidate(ctx context.Context, pattern string) error {
	// 清空本地缓存(简单处理)
	if c.config.EnableL1 && c.local != nil {
		c.local.Clear()
	}

	// Redis模糊删除
	if c.config.EnableL2 {
		return c.invalidateRedis(ctx, pattern)
	}

	return nil
}

func (c *MultiLevelCache) invalidateRedis(ctx context.Context, pattern string) error {
	iter := c.redis.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.redis.Del(ctx, keys...)
	}

	return nil
}

// Close 关闭缓存
func (c *MultiLevelCache) Close() {
	if c.local != nil {
		c.local.Close()
	}
}
