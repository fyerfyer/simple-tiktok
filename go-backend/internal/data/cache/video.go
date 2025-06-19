package cache

import (
	"context"
	"fmt"
	"time"

	"go-backend/internal/biz"
	"go-backend/internal/domain"
	pkgcache "go-backend/pkg/cache"

	"github.com/go-kratos/kratos/v2/log"
)

// VideoCache 视频缓存实现
type VideoCache struct {
	cache *pkgcache.MultiLevelCache
	log   *log.Helper
}

// NewVideoCache 创建视频缓存
func NewVideoCache(cache *pkgcache.MultiLevelCache, logger log.Logger) biz.VideoCacheRepo {
	return &VideoCache{
		cache: cache,
		log:   log.NewHelper(logger),
	}
}

// GetVideo 获取视频缓存
func (c *VideoCache) GetVideo(ctx context.Context, videoID int64) (*domain.Video, bool) {
	key := c.videoKey(videoID)

	data, exists := c.cache.Get(ctx, key)
	if !exists {
		return nil, false
	}

	video, ok := data.(*domain.Video)
	if !ok {
		c.log.WithContext(ctx).Warnf("invalid video cache data type for key: %s", key)
		c.cache.Delete(ctx, key)
		return nil, false
	}

	return video, true
}

// SetVideo 设置视频缓存
func (c *VideoCache) SetVideo(ctx context.Context, video *domain.Video) {
	key := c.videoKey(video.ID)
	// 热门视频缓存1小时，普通视频缓存30分钟
	expiry := 30 * time.Minute
	if video.PlayCount > 10000 || video.FavoriteCount > 1000 {
		expiry = time.Hour
	}

	if err := c.cache.Set(ctx, key, video, expiry); err != nil {
		c.log.WithContext(ctx).Errorf("set video cache failed: %v", err)
	}
}

// DeleteVideo 删除视频缓存
func (c *VideoCache) DeleteVideo(ctx context.Context, videoID int64) {
	key := c.videoKey(videoID)
	if err := c.cache.Delete(ctx, key); err != nil {
		c.log.WithContext(ctx).Errorf("delete video cache failed: %v", err)
	}
}

// GetUserVideos 获取用户视频列表缓存
func (c *VideoCache) GetUserVideos(ctx context.Context, userID int64) ([]*domain.Video, bool) {
	key := c.userVideosKey(userID)

	data, exists := c.cache.Get(ctx, key)
	if !exists {
		return nil, false
	}

	videos, ok := data.([]*domain.Video)
	if !ok {
		c.log.WithContext(ctx).Warnf("invalid user videos cache data type for key: %s", key)
		c.cache.Delete(ctx, key)
		return nil, false
	}

	return videos, true
}

// SetUserVideos 设置用户视频列表缓存
func (c *VideoCache) SetUserVideos(ctx context.Context, userID int64, videos []*domain.Video) {
	key := c.userVideosKey(userID)
	// 活跃用户缓存时间更长
	expiry := 15 * time.Minute
	if len(videos) > 50 {
		expiry = 30 * time.Minute
	}

	if err := c.cache.Set(ctx, key, videos, expiry); err != nil {
		c.log.WithContext(ctx).Errorf("set user videos cache failed: %v", err)
	}
}

// DeleteUserVideos 删除用户视频列表缓存
func (c *VideoCache) DeleteUserVideos(ctx context.Context, userID int64) {
	key := c.userVideosKey(userID)
	if err := c.cache.Delete(ctx, key); err != nil {
		c.log.WithContext(ctx).Errorf("delete user videos cache failed: %v", err)
	}
}

// GetFeedVideos 获取Feed视频缓存
func (c *VideoCache) GetFeedVideos(ctx context.Context, lastTime int64) ([]*domain.Video, bool) {
	key := c.feedKey(lastTime)

	data, exists := c.cache.Get(ctx, key)
	if !exists {
		return nil, false
	}

	videos, ok := data.([]*domain.Video)
	if !ok {
		c.log.WithContext(ctx).Warnf("invalid feed cache data type for key: %s", key)
		c.cache.Delete(ctx, key)
		return nil, false
	}

	return videos, true
}

// SetFeedVideos 设置Feed视频缓存
func (c *VideoCache) SetFeedVideos(ctx context.Context, lastTime int64, videos []*domain.Video) {
	key := c.feedKey(lastTime)
	// Feed流缓存时间较短，保证时效性
	if err := c.cache.Set(ctx, key, videos, 5*time.Minute); err != nil {
		c.log.WithContext(ctx).Errorf("set feed cache failed: %v", err)
	}
}

// DeleteFeedCache 删除Feed缓存
func (c *VideoCache) DeleteFeedCache(ctx context.Context) {
	pattern := "feed:*"
	if err := c.cache.Invalidate(ctx, pattern); err != nil {
		c.log.WithContext(ctx).Errorf("delete feed cache failed: %v", err)
	}
}

// GetVideoStats 获取视频统计缓存
func (c *VideoCache) GetVideoStats(ctx context.Context, videoID int64) (map[string]int64, bool) {
	key := c.videoStatsKey(videoID)

	data, exists := c.cache.Get(ctx, key)
	if !exists {
		return nil, false
	}

	stats, ok := data.(map[string]int64)
	if !ok {
		c.log.WithContext(ctx).Warnf("invalid video stats cache data type for key: %s", key)
		c.cache.Delete(ctx, key)
		return nil, false
	}

	return stats, true
}

// SetVideoStats 设置视频统计缓存
func (c *VideoCache) SetVideoStats(ctx context.Context, videoID int64, stats map[string]int64) {
	key := c.videoStatsKey(videoID)
	// 统计数据缓存10分钟，允许一定延迟
	if err := c.cache.Set(ctx, key, stats, 10*time.Minute); err != nil {
		c.log.WithContext(ctx).Errorf("set video stats cache failed: %v", err)
	}
}

// IncrVideoStats 增加视频统计计数
func (c *VideoCache) IncrVideoStats(ctx context.Context, videoID int64, field string, delta int64) {
	// 先尝试从缓存获取
	stats, exists := c.GetVideoStats(ctx, videoID)
	if !exists {
		// 如果缓存不存在，创建新的统计数据
		stats = make(map[string]int64)
	}

	// 更新计数
	stats[field] += delta
	if stats[field] < 0 {
		stats[field] = 0
	}

	// 重新设置缓存
	c.SetVideoStats(ctx, videoID, stats)
}

// GetHotVideos 获取热门视频缓存
func (c *VideoCache) GetHotVideos(ctx context.Context, timeRange string) ([]*domain.Video, bool) {
	key := c.hotVideosKey(timeRange)

	data, exists := c.cache.Get(ctx, key)
	if !exists {
		return nil, false
	}

	videos, ok := data.([]*domain.Video)
	if !ok {
		c.log.WithContext(ctx).Warnf("invalid hot videos cache data type for key: %s", key)
		c.cache.Delete(ctx, key)
		return nil, false
	}

	return videos, true
}

// SetHotVideos 设置热门视频缓存
func (c *VideoCache) SetHotVideos(ctx context.Context, timeRange string, videos []*domain.Video) {
	key := c.hotVideosKey(timeRange)
	// 热门视频缓存时间较长
	expiry := time.Hour
	if timeRange == "daily" {
		expiry = 30 * time.Minute
	}

	if err := c.cache.Set(ctx, key, videos, expiry); err != nil {
		c.log.WithContext(ctx).Errorf("set hot videos cache failed: %v", err)
	}
}

// WarmupCache 预热缓存
func (c *VideoCache) WarmupCache(ctx context.Context, videos []*domain.Video) {
	for _, video := range videos {
		c.SetVideo(ctx, video)

		// 预热统计数据
		stats := map[string]int64{
			"play_count":     video.PlayCount,
			"favorite_count": video.FavoriteCount,
			"comment_count":  video.CommentCount,
		}
		c.SetVideoStats(ctx, video.ID, stats)
	}
}

// 缓存key生成方法
func (c *VideoCache) videoKey(videoID int64) string {
	return fmt.Sprintf("video:%d", videoID)
}

func (c *VideoCache) userVideosKey(userID int64) string {
	return fmt.Sprintf("user:videos:%d", userID)
}

func (c *VideoCache) feedKey(lastTime int64) string {
	return fmt.Sprintf("feed:%d", lastTime)
}

func (c *VideoCache) videoStatsKey(videoID int64) string {
	return fmt.Sprintf("video:stats:%d", videoID)
}

func (c *VideoCache) hotVideosKey(timeRange string) string {
	return fmt.Sprintf("hot:videos:%s", timeRange)
}
