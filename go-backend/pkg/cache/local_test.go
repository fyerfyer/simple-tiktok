package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestLocalCache_SetAndGet(t *testing.T) {
	cache := NewLocalCache(time.Minute)
	defer cache.Close()

	// 测试设置和获取
	cache.Set("key1", "value1", time.Minute)
	value, exists := cache.Get("key1")
	if !exists {
		t.Error("key should exist")
	}
	if value != "value1" {
		t.Errorf("expected value1, got %v", value)
	}

	// 测试不存在的key
	_, exists = cache.Get("nonexistent")
	if exists {
		t.Error("key should not exist")
	}
}

func TestLocalCache_Expiration(t *testing.T) {
	cache := NewLocalCache(time.Millisecond * 100)
	defer cache.Close()

	// 设置短期过期的缓存
	cache.Set("expiring_key", "value", time.Millisecond*50)

	// 立即检查，应该存在
	value, exists := cache.Get("expiring_key")
	if !exists {
		t.Error("key should exist immediately")
	}
	if value != "value" {
		t.Errorf("expected value, got %v", value)
	}

	// 等待过期
	time.Sleep(time.Millisecond * 100)

	// 检查是否过期
	_, exists = cache.Get("expiring_key")
	if exists {
		t.Error("key should have expired")
	}
}

func TestLocalCache_PermanentStorage(t *testing.T) {
	cache := NewLocalCache(0) // 无清理
	defer cache.Close()

	// 设置永久缓存（duration为0）
	cache.Set("permanent_key", "permanent_value", 0)

	// 等待一段时间
	time.Sleep(time.Millisecond * 50)

	// 应该仍然存在
	value, exists := cache.Get("permanent_key")
	if !exists {
		t.Error("permanent key should exist")
	}
	if value != "permanent_value" {
		t.Errorf("expected permanent_value, got %v", value)
	}
}

func TestLocalCache_Delete(t *testing.T) {
	cache := NewLocalCache(time.Minute)
	defer cache.Close()

	// 设置并删除
	cache.Set("delete_key", "delete_value", time.Minute)
	cache.Delete("delete_key")

	_, exists := cache.Get("delete_key")
	if exists {
		t.Error("key should be deleted")
	}
}

func TestLocalCache_Clear(t *testing.T) {
	cache := NewLocalCache(time.Minute)
	defer cache.Close()

	// 设置多个key
	cache.Set("key1", "value1", time.Minute)
	cache.Set("key2", "value2", time.Minute)
	cache.Set("key3", "value3", time.Minute)

	// 验证存在
	if cache.Size() != 3 {
		t.Errorf("expected size 3, got %d", cache.Size())
	}

	// 清空缓存
	cache.Clear()

	// 验证已清空
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", cache.Size())
	}

	_, exists := cache.Get("key1")
	if exists {
		t.Error("key1 should not exist after clear")
	}
}

func TestLocalCache_Size(t *testing.T) {
	cache := NewLocalCache(time.Minute)
	defer cache.Close()

	// 初始大小应为0
	if cache.Size() != 0 {
		t.Errorf("initial size should be 0, got %d", cache.Size())
	}

	// 添加项目
	cache.Set("key1", "value1", time.Minute)
	if cache.Size() != 1 {
		t.Errorf("size should be 1, got %d", cache.Size())
	}

	cache.Set("key2", "value2", time.Minute)
	if cache.Size() != 2 {
		t.Errorf("size should be 2, got %d", cache.Size())
	}

	// 删除项目
	cache.Delete("key1")
	if cache.Size() != 1 {
		t.Errorf("size should be 1 after delete, got %d", cache.Size())
	}
}

func TestLocalCache_OverwriteValue(t *testing.T) {
	cache := NewLocalCache(time.Minute)
	defer cache.Close()

	// 设置初始值
	cache.Set("key", "value1", time.Minute)
	value, _ := cache.Get("key")
	if value != "value1" {
		t.Errorf("expected value1, got %v", value)
	}

	// 覆盖值
	cache.Set("key", "value2", time.Minute)
	value, _ = cache.Get("key")
	if value != "value2" {
		t.Errorf("expected value2, got %v", value)
	}

	// 大小应该保持为1
	if cache.Size() != 1 {
		t.Errorf("size should remain 1, got %d", cache.Size())
	}
}

func TestLocalCache_TypeSafety(t *testing.T) {
	cache := NewLocalCache(time.Minute)
	defer cache.Close()

	// 测试不同类型的值
	cache.Set("string", "string_value", time.Minute)
	cache.Set("int", 42, time.Minute)
	cache.Set("bool", true, time.Minute)
	cache.Set("slice", []string{"a", "b", "c"}, time.Minute)

	// 验证类型
	if val, _ := cache.Get("string"); val != "string_value" {
		t.Error("string value mismatch")
	}

	if val, _ := cache.Get("int"); val != 42 {
		t.Error("int value mismatch")
	}

	if val, _ := cache.Get("bool"); val != true {
		t.Error("bool value mismatch")
	}

	if val, _ := cache.Get("slice"); len(val.([]string)) != 3 {
		t.Error("slice value mismatch")
	}
}

func TestLocalCache_ConcurrentAccess(t *testing.T) {
	cache := NewLocalCache(time.Minute)
	defer cache.Close()

	done := make(chan bool)

	// 启动多个goroutine同时访问缓存
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := fmt.Sprintf("key%d", id)
			value := fmt.Sprintf("value%d", id)

			// 设置值
			cache.Set(key, value, time.Minute)

			// 获取值
			if val, exists := cache.Get(key); !exists || val != value {
				t.Errorf("concurrent access failed for %s", key)
			}

			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证最终状态
	if cache.Size() != 10 {
		t.Errorf("expected 10 items, got %d", cache.Size())
	}
}

func TestLocalCache_AutoCleanup(t *testing.T) {
	cache := NewLocalCache(time.Millisecond * 50)
	defer cache.Close()

	// 设置会过期的key
	cache.Set("cleanup_key", "cleanup_value", time.Millisecond*30)

	// 验证存在
	_, exists := cache.Get("cleanup_key")
	if !exists {
		t.Error("key should exist before cleanup")
	}

	// 等待自动清理
	time.Sleep(time.Millisecond * 100)

	// 验证已被清理
	if cache.Size() != 0 {
		t.Error("expired items should be cleaned up automatically")
	}
}
