package auth

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// TokenBlacklist Token黑名单接口
type TokenBlacklist interface {
	Add(tokenID string, expiry time.Duration) error
	IsBlacklisted(tokenID string) bool
	Remove(tokenID string) error
	Clear() error
}

// MemoryTokenBlacklist 内存Token黑名单实现
type MemoryTokenBlacklist struct {
	tokens map[string]time.Time
	mutex  sync.RWMutex
}

// NewMemoryTokenBlacklist 创建内存Token黑名单
func NewMemoryTokenBlacklist() *MemoryTokenBlacklist {
	blacklist := &MemoryTokenBlacklist{
		tokens: make(map[string]time.Time),
	}

	// 启动清理goroutine
	go blacklist.cleanup()

	return blacklist
}

// Add 添加Token到黑名单
func (b *MemoryTokenBlacklist) Add(tokenID string, expiry time.Duration) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	expiryTime := time.Now().Add(expiry)
	b.tokens[tokenID] = expiryTime

	return nil
}

// IsBlacklisted 检查Token是否在黑名单中
func (b *MemoryTokenBlacklist) IsBlacklisted(tokenID string) bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	expiryTime, exists := b.tokens[tokenID]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Now().After(expiryTime) {
		// 过期了，删除并返回false
		b.mutex.RUnlock()
		b.mutex.Lock()
		delete(b.tokens, tokenID)
		b.mutex.Unlock()
		b.mutex.RLock()
		return false
	}

	return true
}

// Remove 从黑名单中移除Token
func (b *MemoryTokenBlacklist) Remove(tokenID string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	delete(b.tokens, tokenID)
	return nil
}

// Clear 清空黑名单
func (b *MemoryTokenBlacklist) Clear() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.tokens = make(map[string]time.Time)
	return nil
}

// cleanup 定期清理过期的Token
func (b *MemoryTokenBlacklist) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		b.cleanupExpired()
	}
}

// cleanupExpired 清理过期的Token
func (b *MemoryTokenBlacklist) cleanupExpired() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	now := time.Now()
	for tokenID, expiryTime := range b.tokens {
		if now.After(expiryTime) {
			delete(b.tokens, tokenID)
		}
	}
}

// Size 获取黑名单大小
func (b *MemoryTokenBlacklist) Size() int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return len(b.tokens)
}

// RedisTokenBlacklist Redis Token黑名单实现
type RedisTokenBlacklist struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisTokenBlacklist 创建Redis Token黑名单
func NewRedisTokenBlacklist(client *redis.Client) *RedisTokenBlacklist {
	return &RedisTokenBlacklist{
		client: client,
		ctx:    context.Background(),
	}
}

// Add 添加Token到Redis黑名单
func (r *RedisTokenBlacklist) Add(tokenID string, expiry time.Duration) error {
	key := r.getTokenKey(tokenID)
	return r.client.SetEX(r.ctx, key, "1", expiry).Err()
}

// IsBlacklisted 检查Token是否在Redis黑名单中
func (r *RedisTokenBlacklist) IsBlacklisted(tokenID string) bool {
	key := r.getTokenKey(tokenID)
	result, err := r.client.Exists(r.ctx, key).Result()
	return err == nil && result > 0
}

// Remove 从Redis黑名单中移除Token
func (r *RedisTokenBlacklist) Remove(tokenID string) error {
	key := r.getTokenKey(tokenID)
	return r.client.Del(r.ctx, key).Err()
}

// Clear 清空Redis黑名单
func (r *RedisTokenBlacklist) Clear() error {
	pattern := r.getTokenKey("*")
	keys, err := r.client.Keys(r.ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.client.Del(r.ctx, keys...).Err()
	}

	return nil
}

// BatchAdd 批量添加Token到黑名单
func (r *RedisTokenBlacklist) BatchAdd(tokens map[string]time.Duration) error {
	pipe := r.client.Pipeline()

	for tokenID, expiry := range tokens {
		key := r.getTokenKey(tokenID)
		pipe.SetEX(r.ctx, key, "1", expiry)
	}

	_, err := pipe.Exec(r.ctx)
	return err
}

// BatchRemove 批量移除Token
func (r *RedisTokenBlacklist) BatchRemove(tokenIDs []string) error {
	if len(tokenIDs) == 0 {
		return nil
	}

	keys := make([]string, len(tokenIDs))
	for i, tokenID := range tokenIDs {
		keys[i] = r.getTokenKey(tokenID)
	}

	return r.client.Del(r.ctx, keys...).Err()
}

// GetBlacklistedCount 获取黑名单数量
func (r *RedisTokenBlacklist) GetBlacklistedCount() (int64, error) {
	pattern := r.getTokenKey("*")
	keys, err := r.client.Keys(r.ctx, pattern).Result()
	if err != nil {
		return 0, err
	}
	return int64(len(keys)), nil
}

// CleanExpiredTokens 清理过期Token (Redis会自动过期，这里主要用于统计)
func (r *RedisTokenBlacklist) CleanExpiredTokens() error {
	// Redis的SETEX会自动过期，不需要手动清理
	// 这个方法保留用于接口兼容和可能的监控需求
	return nil
}

// getTokenKey 生成Redis key
func (r *RedisTokenBlacklist) getTokenKey(tokenID string) string {
	return "blacklist:token:" + tokenID
}

// HybridTokenBlacklist 混合Token黑名单 (本地缓存+Redis)
type HybridTokenBlacklist struct {
	memory *MemoryTokenBlacklist
	redis  *RedisTokenBlacklist
	mutex  sync.RWMutex
}

// NewHybridTokenBlacklist 创建混合Token黑名单
func NewHybridTokenBlacklist(redisClient *redis.Client) *HybridTokenBlacklist {
	return &HybridTokenBlacklist{
		memory: NewMemoryTokenBlacklist(),
		redis:  NewRedisTokenBlacklist(redisClient),
	}
}

// Add 添加Token到混合黑名单
func (h *HybridTokenBlacklist) Add(tokenID string, expiry time.Duration) error {
	// 同时添加到内存和Redis
	h.memory.Add(tokenID, expiry)
	return h.redis.Add(tokenID, expiry)
}

// IsBlacklisted 检查Token是否在混合黑名单中
func (h *HybridTokenBlacklist) IsBlacklisted(tokenID string) bool {
	// 先检查本地缓存
	if h.memory.IsBlacklisted(tokenID) {
		return true
	}

	// 再检查Redis
	if h.redis.IsBlacklisted(tokenID) {
		// 回写到本地缓存
		h.memory.Add(tokenID, 5*time.Minute) // 本地缓存5分钟
		return true
	}

	return false
}

// Remove 从混合黑名单中移除Token
func (h *HybridTokenBlacklist) Remove(tokenID string) error {
	h.memory.Remove(tokenID)
	return h.redis.Remove(tokenID)
}

// Clear 清空混合黑名单
func (h *HybridTokenBlacklist) Clear() error {
	h.memory.Clear()
	return h.redis.Clear()
}

// AddWithUserContext 添加用户相关的Token到黑名单
func (h *HybridTokenBlacklist) AddWithUserContext(userID int64, tokens map[string]time.Duration) error {
	// 批量添加用户的所有Token
	err := h.redis.BatchAdd(tokens)
	if err != nil {
		return err
	}

	// 添加到本地缓存
	for tokenID, expiry := range tokens {
		h.memory.Add(tokenID, expiry)
	}

	return nil
}

// RevokeUserTokens 撤销用户的所有Token
func (h *HybridTokenBlacklist) RevokeUserTokens(userID int64, tokenIDs []string, expiry time.Duration) error {
	tokens := make(map[string]time.Duration)
	for _, tokenID := range tokenIDs {
		tokens[tokenID] = expiry
	}

	return h.AddWithUserContext(userID, tokens)
}
