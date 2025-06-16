package cache

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func setupCacheStrategy() (*CacheStrategy, func()) {
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

	multiCache := NewMultiLevelCache(client, config)
	strategy := NewCacheStrategy(multiCache)

	cleanup := func() {
		client.FlushDB(context.Background())
		multiCache.Close()
		client.Close()
	}

	return strategy, cleanup
}

func TestCacheStrategy_UserCache(t *testing.T) {
	strategy, cleanup := setupCacheStrategy()
	defer cleanup()

	ctx := context.Background()

	// 测试Redis连接
	if err := strategy.cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	userID := int64(123)
	userData := map[string]interface{}{
		"id":       userID,
		"username": "testuser",
		"nickname": "Test User",
	}

	// 缓存用户数据
	err := strategy.CacheUser(ctx, userID, userData)
	if err != nil {
		t.Fatalf("CacheUser failed: %v", err)
	}

	// 获取用户缓存
	value, exists := strategy.GetUser(ctx, userID)
	if !exists {
		t.Error("user cache should exist")
	}

	data, ok := value.(map[string]interface{})
	if !ok {
		t.Error("user data type mismatch")
	}

	if data["username"] != "testuser" {
		t.Error("user data content mismatch")
	}

	// 删除用户缓存
	err = strategy.DeleteUser(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// 验证已删除
	_, exists = strategy.GetUser(ctx, userID)
	if exists {
		t.Error("user cache should be deleted")
	}
}

func TestCacheStrategy_FollowRelation(t *testing.T) {
	strategy, cleanup := setupCacheStrategy()
	defer cleanup()

	ctx := context.Background()

	if err := strategy.cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	userID := int64(123)
	followUserID := int64(456)

	// 缓存关注关系
	err := strategy.CacheFollow(ctx, userID, followUserID, true)
	if err != nil {
		t.Fatalf("CacheFollow failed: %v", err)
	}

	// 获取关注关系
	isFollow, exists := strategy.GetFollow(ctx, userID, followUserID)
	if !exists {
		t.Error("follow relation should exist")
	}
	if !isFollow {
		t.Error("should be following")
	}

	// 更新为未关注
	err = strategy.CacheFollow(ctx, userID, followUserID, false)
	if err != nil {
		t.Fatalf("CacheFollow update failed: %v", err)
	}

	isFollow, exists = strategy.GetFollow(ctx, userID, followUserID)
	if !exists {
		t.Error("follow relation should still exist")
	}
	if isFollow {
		t.Error("should not be following")
	}
}

func TestCacheStrategy_FeedCache(t *testing.T) {
	strategy, cleanup := setupCacheStrategy()
	defer cleanup()

	ctx := context.Background()

	if err := strategy.cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	userID := int64(123)
	lastTime := time.Now().Unix()
	feedData := []map[string]interface{}{
		{"id": 1, "title": "Video 1"},
		{"id": 2, "title": "Video 2"},
	}

	// 缓存Feed流
	err := strategy.CacheFeed(ctx, userID, lastTime, feedData)
	if err != nil {
		t.Fatalf("CacheFeed failed: %v", err)
	}

	// 获取Feed缓存
	value, exists := strategy.GetFeed(ctx, userID, lastTime)
	if !exists {
		t.Error("feed cache should exist")
	}

	feeds, ok := value.([]map[string]interface{})
	if !ok {
		t.Error("feed data type mismatch")
	}

	if len(feeds) != 2 {
		t.Errorf("expected 2 feeds, got %d", len(feeds))
	}
}

func TestCacheStrategy_HotFeed(t *testing.T) {
	strategy, cleanup := setupCacheStrategy()
	defer cleanup()

	ctx := context.Background()

	if err := strategy.cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	hotFeedData := []map[string]interface{}{
		{"id": 1, "title": "Hot Video 1", "likes": 1000},
		{"id": 2, "title": "Hot Video 2", "likes": 2000},
	}

	// 缓存热门Feed
	err := strategy.CacheHotFeed(ctx, hotFeedData)
	if err != nil {
		t.Fatalf("CacheHotFeed failed: %v", err)
	}

	// 获取热门Feed
	value, exists := strategy.GetHotFeed(ctx)
	if !exists {
		t.Error("hot feed cache should exist")
	}

	feeds, ok := value.([]map[string]interface{})
	if !ok {
		t.Error("hot feed data type mismatch")
	}

	if len(feeds) != 2 {
		t.Errorf("expected 2 hot feeds, got %d", len(feeds))
	}
}

func TestCacheStrategy_TokenBlacklist(t *testing.T) {
	strategy, cleanup := setupCacheStrategy()
	defer cleanup()

	ctx := context.Background()

	if err := strategy.cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	tokenID := "test_token_123"

	// Token应该不在黑名单
	if strategy.IsTokenBlacklisted(ctx, tokenID) {
		t.Error("token should not be blacklisted initially")
	}

	// 添加到黑名单
	err := strategy.AddTokenToBlacklist(ctx, tokenID, time.Minute)
	if err != nil {
		t.Fatalf("AddTokenToBlacklist failed: %v", err)
	}

	// 检查是否在黑名单
	if !strategy.IsTokenBlacklisted(ctx, tokenID) {
		t.Error("token should be blacklisted")
	}
}

func TestCacheStrategy_KeyGeneration(t *testing.T) {
	strategy, cleanup := setupCacheStrategy()
	defer cleanup()

	// 测试各种key生成方法
	userKey := strategy.UserKey(123)
	expectedUserKey := "user:123"
	if userKey != expectedUserKey {
		t.Errorf("expected %s, got %s", expectedUserKey, userKey)
	}

	statsKey := strategy.UserStatsKey(123)
	expectedStatsKey := "user:stats:123"
	if statsKey != expectedStatsKey {
		t.Errorf("expected %s, got %s", expectedStatsKey, statsKey)
	}

	followKey := strategy.FollowKey(123, 456)
	expectedFollowKey := "follow:123:456"
	if followKey != expectedFollowKey {
		t.Errorf("expected %s, got %s", expectedFollowKey, followKey)
	}

	feedKey := strategy.FeedKey(123, 1640995200)
	expectedFeedKey := "feed:123:1640995200"
	if feedKey != expectedFeedKey {
		t.Errorf("expected %s, got %s", expectedFeedKey, feedKey)
	}

	hotFeedKey := strategy.HotFeedKey()
	expectedHotFeedKey := "feed:hot"
	if hotFeedKey != expectedHotFeedKey {
		t.Errorf("expected %s, got %s", expectedHotFeedKey, hotFeedKey)
	}

	blacklistKey := strategy.TokenBlacklistKey("token123")
	expectedBlacklistKey := "token:blacklist:token123"
	if blacklistKey != expectedBlacklistKey {
		t.Errorf("expected %s, got %s", expectedBlacklistKey, blacklistKey)
	}
}

func TestCacheStrategy_DeleteFollowCache(t *testing.T) {
	strategy, cleanup := setupCacheStrategy()
	defer cleanup()

	ctx := context.Background()

	if err := strategy.cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	userID := int64(123)

	// 设置一些关注关系缓存
	strategy.CacheFollow(ctx, userID, 456, true)
	strategy.CacheFollow(ctx, userID, 789, true)

	// 删除用户的关注缓存
	err := strategy.DeleteFollowCache(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteFollowCache failed: %v", err)
	}

	// 验证缓存已被清除
	_, exists := strategy.GetFollow(ctx, userID, 456)
	if exists {
		t.Error("follow cache should be deleted")
	}
}

func TestCacheStrategy_CacheExpiration(t *testing.T) {
	strategy, cleanup := setupCacheStrategy()
	defer cleanup()

	ctx := context.Background()

	if err := strategy.cache.redis.client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// 使用短TTL测试过期
	userID := int64(123)
	userData := map[string]interface{}{
		"id":       userID,
		"username": "expiring_user",
	}

	// 直接使用MultiLevelCache设置短TTL
	err := strategy.cache.Set(ctx, strategy.UserKey(userID), userData, time.Millisecond*100)
	if err != nil {
		t.Fatalf("Set with short TTL failed: %v", err)
	}

	// 立即检查应该存在
	_, exists := strategy.GetUser(ctx, userID)
	if !exists {
		t.Error("user should exist immediately")
	}

	// 等待过期
	time.Sleep(time.Millisecond * 150)

	// 检查是否过期
	_, exists = strategy.GetUser(ctx, userID)
	if exists {
		t.Error("user cache should have expired")
	}
}
