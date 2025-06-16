package cache

import (
    "context"
    "testing"
    "time"

    "github.com/go-redis/redis/v8"
)

func setupRedisClient() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     "localhost:6380",
        Password: "tiktok123",
        DB:       1, 
    })
}

func TestRedisCache_SetAndGet(t *testing.T) {
    client := setupRedisClient()
    defer client.Close()

    // 测试连接
    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    cache := NewRedisCache(client)

    // 清理测试数据
    client.FlushDB(ctx)

    // 测试设置和获取
    err := cache.Set(ctx, "test_key", "test_value", time.Minute)
    if err != nil {
        t.Fatalf("Set failed: %v", err)
    }

    value, err := cache.Get(ctx, "test_key")
    if err != nil {
        t.Fatalf("Get failed: %v", err)
    }

    if value != "test_value" {
        t.Errorf("expected test_value, got %s", value)
    }
}

func TestRedisCache_GetNonExistent(t *testing.T) {
    client := setupRedisClient()
    defer client.Close()

    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    cache := NewRedisCache(client)
    client.FlushDB(ctx)

    // 测试获取不存在的key
    _, err := cache.Get(ctx, "nonexistent_key")
    if err != redis.Nil {
        t.Errorf("expected redis.Nil, got %v", err)
    }
}

func TestRedisCache_Delete(t *testing.T) {
    client := setupRedisClient()
    defer client.Close()

    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    cache := NewRedisCache(client)
    client.FlushDB(ctx)

    // 设置后删除
    cache.Set(ctx, "delete_key", "delete_value", time.Minute)
    
    err := cache.Del(ctx, "delete_key")
    if err != nil {
        t.Fatalf("Delete failed: %v", err)
    }

    // 验证已删除
    _, err = cache.Get(ctx, "delete_key")
    if err != redis.Nil {
        t.Error("key should be deleted")
    }
}

func TestRedisCache_Exists(t *testing.T) {
    client := setupRedisClient()
    defer client.Close()

    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    cache := NewRedisCache(client)
    client.FlushDB(ctx)

    // 测试不存在的key
    count, err := cache.Exists(ctx, "exists_key")
    if err != nil {
        t.Fatalf("Exists failed: %v", err)
    }
    if count != 0 {
        t.Error("key should not exist")
    }

    // 设置key后测试
    cache.Set(ctx, "exists_key", "value", time.Minute)
    count, err = cache.Exists(ctx, "exists_key")
    if err != nil {
        t.Fatalf("Exists failed: %v", err)
    }
    if count != 1 {
        t.Error("key should exist")
    }
}

func TestRedisCache_Expire(t *testing.T) {
    client := setupRedisClient()
    defer client.Close()

    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    cache := NewRedisCache(client)
    client.FlushDB(ctx)

    // 设置永久key
    cache.Set(ctx, "expire_key", "value", 0)
    
    // 设置过期时间
    err := cache.Expire(ctx, "expire_key", time.Millisecond*100)
    if err != nil {
        t.Fatalf("Expire failed: %v", err)
    }

    // 等待过期
    time.Sleep(time.Millisecond * 150)

    // 验证已过期
    _, err = cache.Get(ctx, "expire_key")
    if err != redis.Nil {
        t.Error("key should have expired")
    }
}

func TestRedisCache_JSON(t *testing.T) {
    client := setupRedisClient()
    defer client.Close()

    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    cache := NewRedisCache(client)
    client.FlushDB(ctx)

    // 测试JSON对象
    testData := map[string]interface{}{
        "name": "test",
        "age":  25,
        "tags": []string{"go", "redis"},
    }

    err := cache.SetJSON(ctx, "json_key", testData, time.Minute)
    if err != nil {
        t.Fatalf("SetJSON failed: %v", err)
    }

    var result map[string]interface{}
    err = cache.GetJSON(ctx, "json_key", &result)
    if err != nil {
        t.Fatalf("GetJSON failed: %v", err)
    }

    if result["name"] != "test" {
        t.Error("JSON data mismatch")
    }
}

func TestRedisCache_Hash(t *testing.T) {
    client := setupRedisClient()
    defer client.Close()

    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    cache := NewRedisCache(client)
    client.FlushDB(ctx)

    // 测试哈希操作
    err := cache.HSet(ctx, "hash_key", "field1", "value1", "field2", "value2")
    if err != nil {
        t.Fatalf("HSet failed: %v", err)
    }

    value, err := cache.HGet(ctx, "hash_key", "field1")
    if err != nil {
        t.Fatalf("HGet failed: %v", err)
    }

    if value != "value1" {
        t.Errorf("expected value1, got %s", value)
    }

    // 删除哈希字段
    err = cache.HDel(ctx, "hash_key", "field1")
    if err != nil {
        t.Fatalf("HDel failed: %v", err)
    }

    _, err = cache.HGet(ctx, "hash_key", "field1")
    if err != redis.Nil {
        t.Error("hash field should be deleted")
    }
}

func TestRedisCache_IncrDecr(t *testing.T) {
    client := setupRedisClient()
    defer client.Close()

    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    cache := NewRedisCache(client)
    client.FlushDB(ctx)

    // 测试自增
    val, err := cache.Incr(ctx, "counter")
    if err != nil {
        t.Fatalf("Incr failed: %v", err)
    }
    if val != 1 {
        t.Errorf("expected 1, got %d", val)
    }

    val, err = cache.Incr(ctx, "counter")
    if err != nil {
        t.Fatalf("Incr failed: %v", err)
    }
    if val != 2 {
        t.Errorf("expected 2, got %d", val)
    }

    // 测试自减
    val, err = cache.Decr(ctx, "counter")
    if err != nil {
        t.Fatalf("Decr failed: %v", err)
    }
    if val != 1 {
        t.Errorf("expected 1, got %d", val)
    }
}

func TestRedisCache_Set(t *testing.T) {
    client := setupRedisClient()
    defer client.Close()

    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    cache := NewRedisCache(client)
    client.FlushDB(ctx)

    // 测试集合操作
    err := cache.SAdd(ctx, "set_key", "member1", "member2", "member3")
    if err != nil {
        t.Fatalf("SAdd failed: %v", err)
    }

    exists, err := cache.SIsMember(ctx, "set_key", "member1")
    if err != nil {
        t.Fatalf("SIsMember failed: %v", err)
    }
    if !exists {
        t.Error("member1 should be in set")
    }

    // 移除成员
    err = cache.SRem(ctx, "set_key", "member1")
    if err != nil {
        t.Fatalf("SRem failed: %v", err)
    }

    exists, err = cache.SIsMember(ctx, "set_key", "member1")
    if err != nil {
        t.Fatalf("SIsMember failed: %v", err)
    }
    if exists {
        t.Error("member1 should not be in set")
    }
}

func TestRedisCache_Pipeline(t *testing.T) {
    client := setupRedisClient()
    defer client.Close()

    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        t.Skipf("Redis not available: %v", err)
    }

    cache := NewRedisCache(client)
    client.FlushDB(ctx)

    // 测试管道操作
    pipe := cache.Pipeline()
    pipe.Set(ctx, "pipe1", "value1", time.Minute)
    pipe.Set(ctx, "pipe2", "value2", time.Minute)
    pipe.Set(ctx, "pipe3", "value3", time.Minute)

    _, err := pipe.Exec(ctx)
    if err != nil {
        t.Fatalf("Pipeline exec failed: %v", err)
    }

    // 验证数据
    value, err := cache.Get(ctx, "pipe1")
    if err != nil {
        t.Fatalf("Get failed: %v", err)
    }
    if value != "value1" {
        t.Error("pipeline set failed")
    }
}