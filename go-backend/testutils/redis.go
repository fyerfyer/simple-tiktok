package testutils

import (
    "context"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
)

// TestRedis 测试Redis配置
type TestRedis struct {
    Client *redis.Client
    config *RedisConfig
}

// RedisConfig Redis配置
type RedisConfig struct {
    Host     string
    Port     int
    Password string
    DB       int
}

// NewTestRedis 创建测试Redis连接
func NewTestRedis() (*TestRedis, error) {
    config := &RedisConfig{
        Host:     "localhost",
        Port:     6380,
        Password: "tiktok123",
        DB:       1, // 使用DB1避免与开发环境冲突
    }

    client := redis.NewClient(&redis.Options{
        Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
        Password:     config.Password,
        DB:           config.DB,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
        PoolSize:     10,
    })

    // 测试连接
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect redis: %w", err)
    }

    return &TestRedis{
        Client: client,
        config: config,
    }, nil
}

// Close 关闭Redis连接
func (tr *TestRedis) Close() error {
    return tr.Client.Close()
}

// FlushDB 清空当前数据库
func (tr *TestRedis) FlushDB() error {
    ctx := context.Background()
    return tr.Client.FlushDB(ctx).Err()
}

// FlushAll 清空所有数据库
func (tr *TestRedis) FlushAll() error {
    ctx := context.Background()
    return tr.Client.FlushAll(ctx).Err()
}

// Set 设置键值
func (tr *TestRedis) Set(key string, value interface{}, expiration time.Duration) error {
    ctx := context.Background()
    return tr.Client.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func (tr *TestRedis) Get(key string) (string, error) {
    ctx := context.Background()
    return tr.Client.Get(ctx, key).Result()
}

// Del 删除键
func (tr *TestRedis) Del(keys ...string) error {
    ctx := context.Background()
    return tr.Client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (tr *TestRedis) Exists(keys ...string) (int64, error) {
    ctx := context.Background()
    return tr.Client.Exists(ctx, keys...).Result()
}

// Keys 获取匹配模式的所有键
func (tr *TestRedis) Keys(pattern string) ([]string, error) {
    ctx := context.Background()
    return tr.Client.Keys(ctx, pattern).Result()
}

// DeletePattern 删除匹配模式的所有键
func (tr *TestRedis) DeletePattern(pattern string) error {
    keys, err := tr.Keys(pattern)
    if err != nil {
        return err
    }
    
    if len(keys) > 0 {
        return tr.Del(keys...)
    }
    
    return nil
}

// GetDBSize 获取数据库大小
func (tr *TestRedis) GetDBSize() (int64, error) {
    ctx := context.Background()
    return tr.Client.DBSize(ctx).Result()
}