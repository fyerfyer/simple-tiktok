package data

import (
	"bytes"
	"context"
	"strings"
	"time"

	"go-backend/internal/biz"
	"go-backend/internal/domain"
	"go-backend/pkg/storage"
	"go-backend/pkg/utils"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// VideoModel 视频数据模型
type VideoModel struct {
    ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
    AuthorID      int64     `gorm:"not null;index:idx_author_created" json:"author_id"`
    Title         string    `gorm:"size:255;not null" json:"title"`
    PlayURL       string    `gorm:"size:500;not null" json:"play_url"`
    CoverURL      string    `gorm:"size:500" json:"cover_url"`
    FavoriteCount int64     `gorm:"default:0" json:"favorite_count"`
    CommentCount  int64     `gorm:"default:0" json:"comment_count"`
    PlayCount     int64     `gorm:"default:0" json:"play_count"`
    Status        int32     `gorm:"default:1" json:"status"`
    CreatedAt     time.Time `gorm:"autoCreateTime;index:idx_created_at,sort:desc;index:idx_author_created,sort:desc" json:"created_at"`
    UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (VideoModel) TableName() string {
    return "videos"
}

// videoRepo 视频仓储实现
type videoRepo struct {
    data       *Data
    storage    storage.VideoStorage
    log        *log.Helper
    videoCache biz.VideoCacheRepo
    producer   domain.VideoEventPublisher
}

// NewVideoRepo 创建视频仓储
func NewVideoRepo(data *Data, storage storage.VideoStorage, videoCache biz.VideoCacheRepo, producer domain.VideoEventPublisher, logger log.Logger) biz.VideoRepo {
    return &videoRepo{
        data:       data,
        storage:    storage,
        videoCache: videoCache,
        producer:   producer,
        log:        log.NewHelper(logger),
    }
}

// CreateVideo 创建视频
func (r *videoRepo) CreateVideo(ctx context.Context, video *domain.Video) error {
    model := &VideoModel{
        ID:            video.ID,
        AuthorID:      video.AuthorID,
        Title:         video.Title,
        PlayURL:       video.PlayURL,
        CoverURL:      video.CoverURL,
        FavoriteCount: video.FavoriteCount,
        CommentCount:  video.CommentCount,
        PlayCount:     video.PlayCount,
        Status:        video.Status,
    }

    err := r.data.db.Transaction(func(tx *gorm.DB) error {
        if err := tx.WithContext(ctx).Create(model).Error; err != nil {
            return err
        }

        video.ID = model.ID
        video.CreatedAt = model.CreatedAt
        video.UpdatedAt = model.UpdatedAt

        // 发布视频上传事件
        event := &domain.VideoUploadedEvent{
            VideoID:    video.ID,
            AuthorID:   video.AuthorID,
            Title:      video.Title,
            PlayURL:    video.PlayURL,
            CoverURL:   video.CoverURL,
            Size:       0, // TODO: 从视频元数据获取
            Format:     "mp4",
            UploadedAt: video.CreatedAt,
            EventID:    utils.GenerateEventID(),
            EventTime:  time.Now(),
        }

        if err := r.producer.PublishVideoUploadedEvent(ctx, event); err != nil {
            r.log.WithContext(ctx).Warnf("publish video uploaded event failed: %v", err)
        }

        return nil
    })

    if err != nil {
        r.log.WithContext(ctx).Errorf("create video failed: %v", err)
        return err
    }

    // 清除相关缓存
    r.videoCache.DeleteUserVideos(ctx, video.AuthorID)
    r.videoCache.DeleteFeedCache(ctx)

    return nil
}

// GetVideo 获取视频信息
func (r *videoRepo) GetVideo(ctx context.Context, videoID int64) (*domain.Video, error) {
    // 先从缓存获取
    if video, ok := r.videoCache.GetVideo(ctx, videoID); ok {
        return video, nil
    }

    var model VideoModel
    if err := r.data.db.WithContext(ctx).Where("id = ? AND status != ?", videoID, domain.VideoStatusDeleted).First(&model).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, utils.ErrVideoNotFound
        }
        r.log.WithContext(ctx).Errorf("get video failed: %v", err)
        return nil, err
    }

    video := r.modelToDomain(&model)

    // 缓存结果
    r.videoCache.SetVideo(ctx, video)

    return video, nil
}

// GetVideos 批量获取视频信息
func (r *videoRepo) GetVideos(ctx context.Context, videoIDs []int64) ([]*domain.Video, error) {
    if len(videoIDs) == 0 {
        return []*domain.Video{}, nil
    }

    var models []VideoModel
    if err := r.data.db.WithContext(ctx).Where("id IN ? AND status != ?", videoIDs, domain.VideoStatusDeleted).Find(&models).Error; err != nil {
        r.log.WithContext(ctx).Errorf("get videos failed: %v", err)
        return nil, err
    }

    videos := make([]*domain.Video, len(models))
    for i, model := range models {
        videos[i] = r.modelToDomain(&model)
    }

    return videos, nil
}

// GetUserVideos 获取用户视频列表
func (r *videoRepo) GetUserVideos(ctx context.Context, userID int64, limit int) ([]*domain.Video, error) {
    // 先从缓存获取
    if videos, ok := r.videoCache.GetUserVideos(ctx, userID); ok {
        if len(videos) > limit {
            return videos[:limit], nil
        }
        return videos, nil
    }

    var models []VideoModel
    if err := r.data.db.WithContext(ctx).
        Where("author_id = ? AND status = ?", userID, domain.VideoStatusPublished).
        Order("created_at DESC").
        Limit(limit).
        Find(&models).Error; err != nil {
        r.log.WithContext(ctx).Errorf("get user videos failed: %v", err)
        return nil, err
    }

    videos := make([]*domain.Video, len(models))
    for i, model := range models {
        videos[i] = r.modelToDomain(&model)
    }

    // 缓存结果
    r.videoCache.SetUserVideos(ctx, userID, videos)

    return videos, nil
}

// GetFeedVideos 获取视频流
func (r *videoRepo) GetFeedVideos(ctx context.Context, latestTime time.Time, limit int) ([]*domain.Video, error) {
    var models []VideoModel
    query := r.data.db.WithContext(ctx).Where("status = ?", domain.VideoStatusPublished)

    if !latestTime.IsZero() {
        query = query.Where("created_at < ?", latestTime)
    }

    if err := query.Order("created_at DESC").Limit(limit).Find(&models).Error; err != nil {
        r.log.WithContext(ctx).Errorf("get feed videos failed: %v", err)
        return nil, err
    }

    videos := make([]*domain.Video, len(models))
    for i, model := range models {
        videos[i] = r.modelToDomain(&model)
    }

    return videos, nil
}

// UpdateVideoStats 更新视频统计
func (r *videoRepo) UpdateVideoStats(ctx context.Context, videoID int64, field string, delta int64) error {
    var oldValue int64
    
    err := r.data.db.Transaction(func(tx *gorm.DB) error {
        // 获取旧值
        var video VideoModel
        if err := tx.WithContext(ctx).Select(field).Where("id = ?", videoID).First(&video).Error; err != nil {
            return err
        }

        switch field {
        case "favorite_count":
            oldValue = video.FavoriteCount
        case "comment_count":
            oldValue = video.CommentCount
        case "play_count":
            oldValue = video.PlayCount
        }

        // 更新统计
        if err := tx.WithContext(ctx).Model(&VideoModel{}).
            Where("id = ?", videoID).
            UpdateColumn(field, gorm.Expr(field+" + ?", delta)).Error; err != nil {
            return err
        }

        // 发布统计更新事件
        event := &domain.VideoStatsUpdatedEvent{
            VideoID:   videoID,
            StatsType: field,
            OldValue:  oldValue,
            NewValue:  oldValue + delta,
            Delta:     delta,
            UpdatedAt: time.Now(),
            EventID:   utils.GenerateEventID(),
            EventTime: time.Now(),
        }

        if err := r.producer.PublishVideoStatsUpdatedEvent(ctx, event); err != nil {
            r.log.WithContext(ctx).Warnf("publish video stats updated event failed: %v", err)
        }

        return nil
    })

    if err != nil {
        r.log.WithContext(ctx).Errorf("update video stats failed: %v", err)
        return err
    }

    // 清除缓存
    r.videoCache.DeleteVideo(ctx, videoID)

    return nil
}

// UpdateVideo 更新视频信息
func (r *videoRepo) UpdateVideo(ctx context.Context, video *domain.Video) error {
    model := &VideoModel{
        ID:            video.ID,
        AuthorID:      video.AuthorID,
        Title:         video.Title,
        PlayURL:       video.PlayURL,
        CoverURL:      video.CoverURL,
        FavoriteCount: video.FavoriteCount,
        CommentCount:  video.CommentCount,
        PlayCount:     video.PlayCount,
        Status:        video.Status,
    }

    if err := r.data.db.WithContext(ctx).Model(model).Where("id = ?", video.ID).Updates(model).Error; err != nil {
        r.log.WithContext(ctx).Errorf("update video failed: %v", err)
        return err
    }

    // 清除缓存
    r.videoCache.DeleteVideo(ctx, video.ID)
    r.videoCache.DeleteUserVideos(ctx, video.AuthorID)

    return nil
}

// UploadVideo 上传视频文件
func (r *videoRepo) UploadVideo(ctx context.Context, file *domain.VideoFile) (string, error) {
    reader := bytes.NewReader(file.Data)
    objectName, err := r.storage.UploadVideo(ctx, file.Filename, reader, file.Size)
    if err != nil {
        r.log.WithContext(ctx).Errorf("upload video to storage failed: %v", err)
        return "", err
    }

    // 生成访问URL
    url, err := r.storage.GenerateVideoURL(ctx, objectName)
    if err != nil {
        r.log.WithContext(ctx).Errorf("generate video URL failed: %v", err)
        return "", err
    }

    return url, nil
}

// UploadCover 上传封面文件
func (r *videoRepo) UploadCover(ctx context.Context, file *domain.VideoFile) (string, error) {
    reader := bytes.NewReader(file.Data)
    objectName, err := r.storage.UploadCover(ctx, file.Filename, reader, file.Size)
    if err != nil {
        r.log.WithContext(ctx).Errorf("upload cover to storage failed: %v", err)
        return "", err
    }

    // 生成访问URL
    url, err := r.storage.GenerateCoverURL(ctx, objectName)
    if err != nil {
        r.log.WithContext(ctx).Errorf("generate cover URL failed: %v", err)
        return "", err
    }

    return url, nil
}

// DeleteFile 删除文件
func (r *videoRepo) DeleteFile(ctx context.Context, url string) error {
    objectName := r.extractObjectName(url)
    return r.storage.Delete(ctx, objectName)
}

// GetPreviewURL 生成预览URL
func (r *videoRepo) GetPreviewURL(ctx context.Context, url string) (string, error) {
    objectName := r.extractObjectName(url)
    return r.storage.GetPresignedURL(ctx, objectName, time.Hour)
}

// modelToDomain 模型转领域对象
func (r *videoRepo) modelToDomain(model *VideoModel) *domain.Video {
    return &domain.Video{
        ID:            model.ID,
        AuthorID:      model.AuthorID,
        Title:         model.Title,
        PlayURL:       model.PlayURL,
        CoverURL:      model.CoverURL,
        FavoriteCount: model.FavoriteCount,
        CommentCount:  model.CommentCount,
        PlayCount:     model.PlayCount,
        Status:        model.Status,
        CreatedAt:     model.CreatedAt,
        UpdatedAt:     model.UpdatedAt,
    }
}

// extractObjectName 从URL提取对象名称
func (r *videoRepo) extractObjectName(url string) string {
    // TODO: 简化实现：从URL路径中提取对象名称
    // 实际实现应该根据存储服务的URL格式进行解析
    parts := strings.Split(url, "/")
    if len(parts) >= 2 {
        return strings.Join(parts[len(parts)-2:], "/")
    }
    return url
}