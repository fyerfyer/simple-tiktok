package cache

import (
	"context"
	"fmt"
	"time"
)

// CacheStrategy 缓存策略
type CacheStrategy struct {
	cache  *MultiLevelCache
	config *StrategyConfig
}

// StrategyConfig 策略配置
type StrategyConfig struct {
	UserCacheTTL     time.Duration
	RelationCacheTTL time.Duration
	VideoFeedTTL     time.Duration
	HotDataTTL       time.Duration
}

// NewCacheStrategy 创建缓存策略
func NewCacheStrategy(cache *MultiLevelCache) *CacheStrategy {
	return &CacheStrategy{
		cache: cache,
		config: &StrategyConfig{
			UserCacheTTL:     30 * time.Minute,
			RelationCacheTTL: 10 * time.Minute,
			VideoFeedTTL:     5 * time.Minute,
			HotDataTTL:       60 * time.Minute,
		},
	}
}

// 用户相关缓存键
func (s *CacheStrategy) UserKey(userID int64) string {
	return fmt.Sprintf("user:%d", userID)
}

func (s *CacheStrategy) UserStatsKey(userID int64) string {
	return fmt.Sprintf("user:stats:%d", userID)
}

// 关系相关缓存键
func (s *CacheStrategy) FollowKey(userID, followUserID int64) string {
	return fmt.Sprintf("follow:%d:%d", userID, followUserID)
}

func (s *CacheStrategy) FollowListKey(userID int64, page int32) string {
	return fmt.Sprintf("follow:list:%d:%d", userID, page)
}

func (s *CacheStrategy) FollowerListKey(userID int64, page int32) string {
	return fmt.Sprintf("follower:list:%d:%d", userID, page)
}

// Feed相关缓存键
func (s *CacheStrategy) FeedKey(userID int64, lastTime int64) string {
	return fmt.Sprintf("feed:%d:%d", userID, lastTime)
}

func (s *CacheStrategy) HotFeedKey() string {
	return "feed:hot"
}

// Token相关缓存键
func (s *CacheStrategy) TokenBlacklistKey(tokenID string) string {
	return fmt.Sprintf("token:blacklist:%s", tokenID)
}

// 缓存用户信息
func (s *CacheStrategy) CacheUser(ctx context.Context, userID int64, user interface{}) error {
	key := s.UserKey(userID)
	return s.cache.Set(ctx, key, user, s.config.UserCacheTTL)
}

// 获取用户缓存
func (s *CacheStrategy) GetUser(ctx context.Context, userID int64) (interface{}, bool) {
	key := s.UserKey(userID)
	return s.cache.Get(ctx, key)
}

// 删除用户缓存
func (s *CacheStrategy) DeleteUser(ctx context.Context, userID int64) error {
	keys := []string{
		s.UserKey(userID),
		s.UserStatsKey(userID),
	}

	for _, key := range keys {
		if err := s.cache.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// 缓存关注关系
func (s *CacheStrategy) CacheFollow(ctx context.Context, userID, followUserID int64, isFollow bool) error {
	key := s.FollowKey(userID, followUserID)
	value := "0"
	if isFollow {
		value = "1"
	}
	return s.cache.SetString(ctx, key, value, s.config.RelationCacheTTL)
}

// 获取关注关系缓存
func (s *CacheStrategy) GetFollow(ctx context.Context, userID, followUserID int64) (bool, bool) {
	key := s.FollowKey(userID, followUserID)
	val, err := s.cache.GetString(ctx, key)
	if err != nil {
		return false, false
	}
	return val == "1", true
}

// 删除关注关系缓存
func (s *CacheStrategy) DeleteFollowCache(ctx context.Context, userID int64) error {
	pattern := fmt.Sprintf("follow:%d:*", userID)
	return s.cache.Invalidate(ctx, pattern)
}

// 缓存Feed流
func (s *CacheStrategy) CacheFeed(ctx context.Context, userID int64, lastTime int64, videos interface{}) error {
	key := s.FeedKey(userID, lastTime)
	return s.cache.Set(ctx, key, videos, s.config.VideoFeedTTL)
}

// 获取Feed缓存
func (s *CacheStrategy) GetFeed(ctx context.Context, userID int64, lastTime int64) (interface{}, bool) {
	key := s.FeedKey(userID, lastTime)
	return s.cache.Get(ctx, key)
}

// 缓存热门Feed
func (s *CacheStrategy) CacheHotFeed(ctx context.Context, videos interface{}) error {
	key := s.HotFeedKey()
	return s.cache.Set(ctx, key, videos, s.config.HotDataTTL)
}

// 获取热门Feed缓存
func (s *CacheStrategy) GetHotFeed(ctx context.Context) (interface{}, bool) {
	key := s.HotFeedKey()
	return s.cache.Get(ctx, key)
}

// Token黑名单操作
func (s *CacheStrategy) AddTokenToBlacklist(ctx context.Context, tokenID string, expireTime time.Duration) error {
	key := s.TokenBlacklistKey(tokenID)
	return s.cache.SetString(ctx, key, "1", expireTime)
}

// 检查Token是否在黑名单
func (s *CacheStrategy) IsTokenBlacklisted(ctx context.Context, tokenID string) bool {
	key := s.TokenBlacklistKey(tokenID)
	_, err := s.cache.GetString(ctx, key)
	return err == nil
}

// 预热缓存
func (s *CacheStrategy) WarmupCache(ctx context.Context) error {
	// 这里可以实现缓存预热逻辑
	// 比如预热热门用户、热门视频等
	return nil
}

// 清理过期缓存
func (s *CacheStrategy) CleanupExpiredCache(ctx context.Context) error {
	// 清理Token黑名单过期项
	pattern := "token:blacklist:*"
	return s.cache.Invalidate(ctx, pattern)
}
