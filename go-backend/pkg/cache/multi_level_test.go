package cache

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func setupMultiLevelCache() (*MultiLevelCache, func()) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6380",
		Password: "tiktok123",
		DB:       1,
	})

	config := &CacheConfig{
		LocalTTL: 5 * time.Minute,
		RedisTTL: 30 * time.Minute,
		EnableL1: true,
		EnableL2: true,
	}

	cache := NewMultiLevelCache(client, config)

	cleanup := func() {
		client.FlushDB(context.Background())
		cache.Close()
		client.Close()
	}

	return cache, cleanup
}

func TestMultiLevelCache_BasicSetGet(t *testing.T) {
	cache, cleanup := setupMultiLevelCache()
	defer cleanup()

	ctx := context.Background()

	// 测试Redis连接
	if err := cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// 设置缓存
	testData := map[string]interface{}{
		"id":   123,
		"name": "test",
	}

	err := cache.Set(ctx, "test_key", testData, time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 获取缓存
	value, exists := cache.Get(ctx, "test_key")
	if !exists {
		t.Error("value should exist")
	}

	// 验证数据
	data, ok := value.(map[string]interface{})
	if !ok {
		t.Error("value type mismatch")
	}

	if data["name"] != "test" {
		t.Error("data content mismatch")
	}
}

func TestMultiLevelCache_L1CacheHit(t *testing.T) {
	cache, cleanup := setupMultiLevelCache()
	defer cleanup()

	ctx := context.Background()

	if err := cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// 首次设置，会存入L1和L2
	cache.Set(ctx, "l1_test", "value1", time.Minute)

	// 先获取一次，确保L1有缓存
	cache.Get(ctx, "l1_test")

	// 直接从L1缓存获取
	if cache.local != nil {
		value, exists := cache.local.Get("l1_test")
		if !exists {
			t.Error("L1 cache should have the value")
		}
		if value != "value1" {
			t.Error("L1 cache value mismatch")
		}
	}
}

func TestMultiLevelCache_L2Fallback(t *testing.T) {
	cache, cleanup := setupMultiLevelCache()
	defer cleanup()

	ctx := context.Background()

	if err := cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// 设置缓存
	cache.Set(ctx, "l2_test", "value2", time.Minute)

	// 清空L1缓存
	if cache.local != nil {
		cache.local.Clear()
	}

	// 从L2获取应该成功
	value, exists := cache.Get(ctx, "l2_test")
	if !exists {
		t.Error("L2 fallback should work")
	}

	if value != "value2" {
		t.Error("L2 fallback value mismatch")
	}

	// 验证值已回写到L1
	if cache.local != nil {
		l1Value, l1Exists := cache.local.Get("l2_test")
		if !l1Exists {
			t.Error("value should be written back to L1")
		}
		if l1Value != "value2" {
			t.Error("L1 write-back value mismatch")
		}
	}
}

func TestMultiLevelCache_StringOperations(t *testing.T) {
	cache, cleanup := setupMultiLevelCache()
	defer cleanup()

	ctx := context.Background()

	if err := cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// 设置字符串
	err := cache.SetString(ctx, "string_key", "string_value", time.Minute)
	if err != nil {
		t.Fatalf("SetString failed: %v", err)
	}

	// 获取字符串
	value, err := cache.GetString(ctx, "string_key")
	if err != nil {
		t.Fatalf("GetString failed: %v", err)
	}

	if value != "string_value" {
		t.Errorf("expected string_value, got %s", value)
	}
}

func TestMultiLevelCache_Delete(t *testing.T) {
	cache, cleanup := setupMultiLevelCache()
	defer cleanup()

	ctx := context.Background()

	if err := cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// 设置缓存
	cache.Set(ctx, "delete_test", "delete_value", time.Minute)

	// 验证存在
	_, exists := cache.Get(ctx, "delete_test")
	if !exists {
		t.Error("value should exist before delete")
	}

	// 删除缓存
	err := cache.Delete(ctx, "delete_test")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证已删除
	_, exists = cache.Get(ctx, "delete_test")
	if exists {
		t.Error("value should not exist after delete")
	}

	// 验证L1也已删除
	if cache.local != nil {
		_, l1Exists := cache.local.Get("delete_test")
		if l1Exists {
			t.Error("L1 cache should also be deleted")
		}
	}
}

func TestMultiLevelCache_OnlyL1Enabled(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6380",
		Password: "tiktok123",
		DB:       1,
	})
	defer client.Close()

	config := &CacheConfig{
		LocalTTL: 5 * time.Minute,
		RedisTTL: 30 * time.Minute,
		EnableL1: true,
		EnableL2: false, // 只启用L1
	}

	cache := NewMultiLevelCache(client, config)
	defer cache.Close()

	ctx := context.Background()

	// 设置缓存
	cache.Set(ctx, "l1_only", "l1_value", time.Minute)

	// 获取缓存
	value, exists := cache.Get(ctx, "l1_only")
	if !exists {
		t.Error("L1 only cache should work")
	}

	if value != "l1_value" {
		t.Error("L1 only value mismatch")
	}
}

func TestMultiLevelCache_OnlyL2Enabled(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6380",
		Password: "tiktok123",
		DB:       1,
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	client.FlushDB(ctx)

	config := &CacheConfig{
		LocalTTL: 5 * time.Minute,
		RedisTTL: 30 * time.Minute,
		EnableL1: false, // 只启用L2
		EnableL2: true,
	}

	cache := NewMultiLevelCache(client, config)
	defer cache.Close()

	// 设置缓存
	cache.Set(ctx, "l2_only", "l2_value", time.Minute)

	// 获取缓存
	value, exists := cache.Get(ctx, "l2_only")
	if !exists {
		t.Error("L2 only cache should work")
	}

	if value != "l2_value" {
		t.Error("L2 only value mismatch")
	}
}

func TestMultiLevelCache_InvalidatePattern(t *testing.T) {
	cache, cleanup := setupMultiLevelCache()
	defer cleanup()

	ctx := context.Background()

	if err := cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// 设置多个相关的key
	cache.Set(ctx, "user:1:profile", "profile1", time.Minute)
	cache.Set(ctx, "user:1:settings", "settings1", time.Minute)
	cache.Set(ctx, "user:2:profile", "profile2", time.Minute)

	// 失效user:1相关的缓存
	err := cache.Invalidate(ctx, "user:1:*")
	if err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	// user:1相关的应该被清除
	_, exists := cache.Get(ctx, "user:1:profile")
	if exists {
		t.Error("user:1:profile should be invalidated")
	}

	// user:2的应该还存在（但由于L1被清空了，只能从L2获取）
	value, exists := cache.Get(ctx, "user:2:profile")
	if !exists {
		t.Error("user:2:profile should still exist")
	}
	if value != "profile2" {
		t.Error("user:2:profile value mismatch")
	}
}

func TestMultiLevelCache_TTLRespect(t *testing.T) {
	cache, cleanup := setupMultiLevelCache()
	defer cleanup()

	ctx := context.Background()

	if err := cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// 设置短TTL的缓存
	cache.Set(ctx, "short_ttl", "short_value", time.Millisecond*100)

	// 立即获取应该存在
	value, exists := cache.Get(ctx, "short_ttl")
	if !exists {
		t.Error("value should exist immediately")
	}
	if value != "short_value" {
		t.Error("value mismatch")
	}

	// 等待过期
	time.Sleep(time.Millisecond * 150)

	// 应该已过期
	_, exists = cache.Get(ctx, "short_ttl")
	if exists {
		t.Error("value should have expired")
	}
}
