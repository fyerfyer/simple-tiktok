package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-backend/internal/biz"
	"go-backend/pkg/cache"

	"github.com/go-kratos/kratos/v2/log"
)

// UserCache 用户缓存
type UserCache struct {
	cache    *cache.MultiLevelCache
	strategy *cache.CacheStrategy
	log      *log.Helper
}

// NewUserCache 创建用户缓存
func NewUserCache(multiCache *cache.MultiLevelCache, logger log.Logger) *UserCache {
	strategy := cache.NewCacheStrategy(multiCache)
	return &UserCache{
		cache:    multiCache,
		strategy: strategy,
		log:      log.NewHelper(logger),
	}
}

// GetUser 获取用户缓存
func (c *UserCache) GetUser(ctx context.Context, userID int64) (*biz.User, error) {
	key := c.strategy.UserKey(userID)

	data, exists := c.cache.Get(ctx, key)
	if !exists {
		return nil, nil
	}

	// JSON反序列化
	jsonData, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("invalid cache data type")
	}

	var user biz.User
	if err := json.Unmarshal([]byte(jsonData), &user); err != nil {
		c.log.WithContext(ctx).Errorf("unmarshal user cache failed: %v", err)
		return nil, err
	}

	return &user, nil
}

// SetUser 设置用户缓存
func (c *UserCache) SetUser(ctx context.Context, user *biz.User) error {
	key := c.strategy.UserKey(user.ID)

	// JSON序列化
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal user failed: %w", err)
	}

	return c.cache.SetString(ctx, key, string(data), 30*time.Minute)
}

// DeleteUser 删除用户缓存
func (c *UserCache) DeleteUser(ctx context.Context, userID int64) error {
	keys := []string{
		c.strategy.UserKey(userID),
		c.strategy.UserStatsKey(userID),
	}

	for _, key := range keys {
		if err := c.cache.Delete(ctx, key); err != nil {
			c.log.WithContext(ctx).Errorf("delete user cache failed: %v", err)
		}
	}

	return nil
}

// GetUserStats 获取用户统计缓存
func (c *UserCache) GetUserStats(ctx context.Context, userID int64) (map[string]int64, error) {
	key := c.strategy.UserStatsKey(userID)

	data, exists := c.cache.Get(ctx, key)
	if !exists {
		return nil, nil
	}

	jsonData, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("invalid stats cache data type")
	}

	var stats map[string]int64
	if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// SetUserStats 设置用户统计缓存
func (c *UserCache) SetUserStats(ctx context.Context, userID int64, stats map[string]int64) error {
	key := c.strategy.UserStatsKey(userID)

	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshal user stats failed: %w", err)
	}

	return c.cache.SetString(ctx, key, string(data), 10*time.Minute)
}

// GetFollowRelation 获取关注关系缓存
func (c *UserCache) GetFollowRelation(ctx context.Context, userID, followUserID int64) (bool, bool) {
	return c.strategy.GetFollow(ctx, userID, followUserID)
}

// SetFollowRelation 设置关注关系缓存
func (c *UserCache) SetFollowRelation(ctx context.Context, userID, followUserID int64, isFollow bool) error {
	return c.strategy.CacheFollow(ctx, userID, followUserID, isFollow)
}

// DeleteFollowRelation 删除关注关系缓存
func (c *UserCache) DeleteFollowRelation(ctx context.Context, userID int64) error {
	return c.strategy.DeleteFollowCache(ctx, userID)
}

// GetFollowList 获取关注列表缓存
func (c *UserCache) GetFollowList(ctx context.Context, userID int64, page int32) ([]*biz.User, error) {
	key := c.strategy.FollowListKey(userID, page)

	data, exists := c.cache.Get(ctx, key)
	if !exists {
		return nil, nil
	}

	jsonData, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("invalid follow list cache data type")
	}

	var users []*biz.User
	if err := json.Unmarshal([]byte(jsonData), &users); err != nil {
		return nil, err
	}

	return users, nil
}

// SetFollowList 设置关注列表缓存
func (c *UserCache) SetFollowList(ctx context.Context, userID int64, page int32, users []*biz.User) error {
	key := c.strategy.FollowListKey(userID, page)

	data, err := json.Marshal(users)
	if err != nil {
		return fmt.Errorf("marshal follow list failed: %w", err)
	}

	return c.cache.SetString(ctx, key, string(data), 5*time.Minute)
}

// GetFollowerList 获取粉丝列表缓存
func (c *UserCache) GetFollowerList(ctx context.Context, userID int64, page int32) ([]*biz.User, error) {
	key := c.strategy.FollowerListKey(userID, page)

	data, exists := c.cache.Get(ctx, key)
	if !exists {
		return nil, nil
	}

	jsonData, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("invalid follower list cache data type")
	}

	var users []*biz.User
	if err := json.Unmarshal([]byte(jsonData), &users); err != nil {
		return nil, err
	}

	return users, nil
}

// SetFollowerList 设置粉丝列表缓存
func (c *UserCache) SetFollowerList(ctx context.Context, userID int64, page int32, users []*biz.User) error {
	key := c.strategy.FollowerListKey(userID, page)

	data, err := json.Marshal(users)
	if err != nil {
		return fmt.Errorf("marshal follower list failed: %w", err)
	}

	return c.cache.SetString(ctx, key, string(data), 5*time.Minute)
}

// BatchGetUsers 批量获取用户缓存
func (c *UserCache) BatchGetUsers(ctx context.Context, userIDs []int64) (map[int64]*biz.User, []int64) {
	users := make(map[int64]*biz.User)
	missedIDs := make([]int64, 0)

	for _, userID := range userIDs {
		user, err := c.GetUser(ctx, userID)
		if err != nil || user == nil {
			missedIDs = append(missedIDs, userID)
		} else {
			users[userID] = user
		}
	}

	return users, missedIDs
}

// BatchSetUsers 批量设置用户缓存
func (c *UserCache) BatchSetUsers(ctx context.Context, users []*biz.User) error {
	for _, user := range users {
		if err := c.SetUser(ctx, user); err != nil {
			c.log.WithContext(ctx).Errorf("batch set user cache failed: %v", err)
		}
	}
	return nil
}

// InvalidateUserCache 失效用户相关缓存
func (c *UserCache) InvalidateUserCache(ctx context.Context, userID int64) error {
	patterns := []string{
		fmt.Sprintf("user:%d*", userID),
		fmt.Sprintf("follow:%d:*", userID),
		fmt.Sprintf("follow:*:%d", userID),
	}

	for _, pattern := range patterns {
		if err := c.cache.Invalidate(ctx, pattern); err != nil {
			c.log.WithContext(ctx).Errorf("invalidate user cache failed: %v", err)
		}
	}

	return nil
}
