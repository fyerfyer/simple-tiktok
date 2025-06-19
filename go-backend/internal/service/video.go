package service

import (
	"context"
	"io"
	"mime/multipart"

	commonv1 "go-backend/api/common/v1"
	v1 "go-backend/api/video/v1"
	"go-backend/internal/biz"
	"go-backend/internal/domain"
	"go-backend/internal/middleware"
	"go-backend/pkg/media"
	"go-backend/pkg/security"
	"go-backend/pkg/utils"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// VideoService 视频服务
type VideoService struct {
	v1.UnimplementedVideoServiceServer

	videoUc   *biz.VideoUsecase
	userUc    *biz.UserUsecase
	validator *security.Validator
	processor *media.VideoProcessor
	log       *log.Helper
}

// NewVideoService 创建视频服务
func NewVideoService(
	videoUc *biz.VideoUsecase,
	userUc *biz.UserUsecase,
	validator *security.Validator,
	processor *media.VideoProcessor,
	logger log.Logger,
) *VideoService {
	return &VideoService{
		videoUc:   videoUc,
		userUc:    userUc,
		validator: validator,
		processor: processor,
		log:       log.NewHelper(logger),
	}
}

// GetFeed 获取视频流
func (s *VideoService) GetFeed(ctx context.Context, req *v1.GetFeedRequest) (*v1.GetFeedResponse, error) {
	s.log.WithContext(ctx).Info("get feed request")

	// 获取当前用户ID（可选）
	var currentUserID int64
	if req.Token != "" {
		userID, _ := middleware.GetUserIDFromToken(ctx, req.Token)
		currentUserID = userID
	}

	// 获取视频流
	videos, nextTime, err := s.videoUc.GetFeed(ctx, req.LatestTime, 30)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get feed failed: %v", err)
		return &v1.GetFeedResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "get feed failed",
			},
		}, nil
	}

	// 转换为响应格式
	videoList := make([]*commonv1.Video, 0, len(videos))
	for _, video := range videos {
		videoItem, err := s.buildVideoResponse(ctx, video, currentUserID)
		if err != nil {
			s.log.WithContext(ctx).Warnf("build video response failed: %v", err)
			continue
		}
		videoList = append(videoList, videoItem)
	}

	return &v1.GetFeedResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.GetFeedData{
			NextTime:  nextTime,
			VideoList: videoList,
		},
	}, nil
}

// PublishVideo 发布视频
func (s *VideoService) PublishVideo(ctx context.Context, req *v1.PublishVideoRequest) (*v1.PublishVideoResponse, error) {
	s.log.WithContext(ctx).Info("publish video request")

	// 验证Token
	userID, ok := middleware.GetUserIDFromToken(ctx, req.Token)
	if !ok {
		return &v1.PublishVideoResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_TOKEN_INVALID),
				StatusMsg:  "invalid token",
			},
		}, nil
	}

	// 验证标题
	if err := s.validator.ValidateVideoTitle(req.Title); err != nil {
		return &v1.PublishVideoResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  "invalid title",
			},
		}, nil
	}

	var videoData []byte
	var filename string

	// 处理不同的数据源
	switch source := req.DataSource.(type) {
	case *v1.PublishVideoRequest_Data:
		if len(source.Data) == 0 {
			return &v1.PublishVideoResponse{
				Base: &commonv1.BaseResponse{
					StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
					StatusMsg:  "video data is empty",
				},
			}, nil
		}
		videoData = source.Data
		filename = utils.GenerateVideoFilename("video.mp4")

	case *v1.PublishVideoRequest_FileInfo:
		if source.FileInfo == nil {
			return &v1.PublishVideoResponse{
				Base: &commonv1.BaseResponse{
					StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
					StatusMsg:  "file info is required",
				},
			}, nil
		}
		// 对于FileInfo模式，这里应该从临时存储或缓存中获取文件数据
		// 简化实现：返回错误，提示使用UploadVideoFile接口
		return &v1.PublishVideoResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  "please use UploadVideoFile for file upload",
			},
		}, nil

	default:
		return &v1.PublishVideoResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  "data source is required",
			},
		}, nil
	}

	// 发布视频
	video, err := s.videoUc.PublishVideo(ctx, userID, req.Title, videoData, filename)
	if err != nil {
		s.log.WithContext(ctx).Errorf("publish video failed: %v", err)
		return &v1.PublishVideoResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(utils.GetErrorCode(err)),
				StatusMsg:  "publish video failed",
			},
		}, nil
	}

	return &v1.PublishVideoResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.PublishVideoData{
			VideoId: video.ID,
			Status:  v1.UploadStatus_UPLOAD_STATUS_COMPLETED,
		},
	}, nil
}

// UploadVideoFile 专门处理multipart文件上传
func (s *VideoService) UploadVideoFile(ctx context.Context, req *v1.UploadVideoFileRequest) (*v1.PublishVideoResponse, error) {
	s.log.WithContext(ctx).Info("upload video file request")

	// 验证Token
	userID, ok := middleware.GetUserIDFromToken(ctx, req.Token)
	if !ok {
		return &v1.PublishVideoResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_TOKEN_INVALID),
				StatusMsg:  "invalid token",
			},
		}, nil
	}

	// 验证标题
	if err := s.validator.ValidateVideoTitle(req.Title); err != nil {
		return &v1.PublishVideoResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  "invalid title",
			},
		}, nil
	}

	// 验证文件元数据
	if req.Metadata == nil {
		return &v1.PublishVideoResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  "file metadata is required",
			},
		}, nil
	}

	// 从中间件获取文件头
	fileHeader, ok := middleware.GetVideoFileHeaderFromContext(ctx)
	if !ok {
		return &v1.PublishVideoResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  "video file not found",
			},
		}, nil
	}

	// 处理文件上传
	video, err := s.handleVideoUpload(ctx, userID, req.Title, fileHeader)
	if err != nil {
		s.log.WithContext(ctx).Errorf("handle video upload failed: %v", err)
		return &v1.PublishVideoResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(utils.GetErrorCode(err)),
				StatusMsg:  "upload failed",
			},
		}, nil
	}

	return &v1.PublishVideoResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.PublishVideoData{
			VideoId: video.ID,
			Status:  v1.UploadStatus_UPLOAD_STATUS_COMPLETED,
		},
	}, nil
}

// GetPublishList 获取发布列表
func (s *VideoService) GetPublishList(ctx context.Context, req *v1.GetPublishListRequest) (*v1.GetPublishListResponse, error) {
	s.log.WithContext(ctx).Info("get publish list request")

	// 验证Token
	currentUserID, ok := middleware.GetUserIDFromToken(ctx, req.Token)
	if !ok {
		return &v1.GetPublishListResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_TOKEN_INVALID),
				StatusMsg:  "invalid token",
			},
		}, nil
	}

	// 验证用户ID
	if err := s.validator.ValidateUserID(req.UserId); err != nil {
		return &v1.GetPublishListResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  "invalid user id",
			},
		}, nil
	}

	// 获取用户发布列表
	videos, err := s.videoUc.GetPublishList(ctx, req.UserId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get publish list failed: %v", err)
		return &v1.GetPublishListResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "get publish list failed",
			},
		}, nil
	}

	// 转换为响应格式
	videoList := make([]*commonv1.Video, 0, len(videos))
	for _, video := range videos {
		videoItem, err := s.buildVideoResponse(ctx, video, currentUserID)
		if err != nil {
			s.log.WithContext(ctx).Warnf("build video response failed: %v", err)
			continue
		}
		videoList = append(videoList, videoItem)
	}

	return &v1.GetPublishListResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.GetPublishListData{
			VideoList: videoList,
		},
	}, nil
}

// GetUploadConfig 获取上传配置
func (s *VideoService) GetUploadConfig(ctx context.Context, req *v1.GetUploadConfigRequest) (*v1.GetUploadConfigResponse, error) {
	s.log.WithContext(ctx).Info("get upload config request")

	config, err := s.videoUc.GetUploadConfig(ctx)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get upload config failed: %v", err)
		return &v1.GetUploadConfigResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "get upload config failed",
			},
		}, nil
	}

	return &v1.GetUploadConfigResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.UploadConfig{
			MaxFileSize:          config.MaxFileSize,
			SupportedFormats:     config.SupportedFormats,
			ChunkSize:            config.ChunkSize,
			EnableResume:         config.EnableResume,
			MaxConcurrentUploads: 3,
		},
	}, nil
}

// GetUploadProgress 获取上传进度
func (s *VideoService) GetUploadProgress(ctx context.Context, req *v1.GetUploadProgressRequest) (*v1.GetUploadProgressResponse, error) {
	s.log.WithContext(ctx).Info("get upload progress request")

	// 验证Token
	_, ok := middleware.GetUserIDFromToken(ctx, req.Token)
	if !ok {
		return &v1.GetUploadProgressResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_TOKEN_INVALID),
				StatusMsg:  "invalid token",
			},
		}, nil
	}

	progress, err := s.videoUc.GetUploadProgress(ctx, req.UploadId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get upload progress failed: %v", err)
		return &v1.GetUploadProgressResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "get upload progress failed",
			},
		}, nil
	}

	return &v1.GetUploadProgressResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.UploadProgress{
			UploadId:      progress.UploadID,
			Progress:      progress.Progress,
			Status:        v1.UploadStatus(v1.UploadStatus_value[progress.Status]),
			TotalSize:     progress.TotalSize,
			UploadedSize:  progress.UploadedSize,
			ErrorMessage:  progress.ErrorMessage,
			EstimatedTime: progress.EstimatedTime,
		},
	}, nil
}

// GetVideoInfo gRPC内部调用 - 获取视频信息
func (s *VideoService) GetVideoInfo(ctx context.Context, req *v1.GetVideoInfoRequest) (*v1.GetVideoInfoResponse, error) {
	video, err := s.videoUc.GetVideo(ctx, req.VideoId)
	if err != nil {
		return nil, err
	}

	videoItem, err := s.buildVideoResponse(ctx, video, 0)
	if err != nil {
		return nil, err
	}

	return &v1.GetVideoInfoResponse{
		Video: videoItem,
	}, nil
}

// GetVideosInfo gRPC内部调用 - 批量获取视频信息
func (s *VideoService) GetVideosInfo(ctx context.Context, req *v1.GetVideosInfoRequest) (*v1.GetVideosInfoResponse, error) {
	videos, err := s.videoUc.GetVideos(ctx, req.VideoIds)
	if err != nil {
		return nil, err
	}

	videoList := make([]*commonv1.Video, 0, len(videos))
	for _, video := range videos {
		videoItem, err := s.buildVideoResponse(ctx, video, 0)
		if err != nil {
			s.log.WithContext(ctx).Warnf("build video response failed: %v", err)
			continue
		}
		videoList = append(videoList, videoItem)
	}

	return &v1.GetVideosInfoResponse{
		Videos: videoList,
	}, nil
}

// UpdateVideoStats gRPC内部调用 - 更新视频统计
func (s *VideoService) UpdateVideoStats(ctx context.Context, req *v1.UpdateVideoStatsRequest) (*emptypb.Empty, error) {
	var statsType string
	switch req.Type {
	case v1.UpdateVideoStatsType_UPDATE_VIDEO_STATS_FAVORITE_COUNT:
		statsType = "favorite"
	case v1.UpdateVideoStatsType_UPDATE_VIDEO_STATS_COMMENT_COUNT:
		statsType = "comment"
	case v1.UpdateVideoStatsType_UPDATE_VIDEO_STATS_PLAY_COUNT:
		statsType = "play"
	default:
		return nil, utils.ErrInvalidParam
	}

	err := s.videoUc.UpdateVideoStats(ctx, req.VideoId, statsType, req.Count)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// 以下是原video_handler.go中的方法

// handleVideoUpload 处理视频上传
func (s *VideoService) handleVideoUpload(ctx context.Context, userID int64, title string, fileHeader *multipart.FileHeader) (*domain.Video, error) {
	s.log.WithContext(ctx).Infof("handling video upload: user_id=%d, filename=%s, size=%d",
		userID, fileHeader.Filename, fileHeader.Size)

	// 验证文件大小
	if fileHeader.Size > s.processor.GetMaxFileSize() {
		return nil, utils.ErrVideoSizeErr
	}

	// 验证文件格式
	if err := s.processor.ValidateFormat(fileHeader.Filename, fileHeader.Size); err != nil {
		return nil, utils.ErrVideoFormatErr
	}

	// 打开文件
	file, err := fileHeader.Open()
	if err != nil {
		s.log.WithContext(ctx).Errorf("open file failed: %v", err)
		return nil, err
	}
	defer file.Close()

	// 验证视频内容
	if err := s.validateVideoContent(ctx, file, fileHeader); err != nil {
		return nil, err
	}

	// 重新定位文件指针到开始
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	// 读取文件数据
	data, err := io.ReadAll(file)
	if err != nil {
		s.log.WithContext(ctx).Errorf("read file failed: %v", err)
		return nil, err
	}

	// 生成唯一文件名
	filename := utils.GenerateVideoFilename(fileHeader.Filename)

	// 发布视频
	video, err := s.videoUc.PublishVideo(ctx, userID, title, data, filename)
	if err != nil {
		s.log.WithContext(ctx).Errorf("publish video failed: %v", err)
		return nil, err
	}

	s.log.WithContext(ctx).Infof("video upload completed: video_id=%d", video.ID)
	return video, nil
}

// validateVideoContent 验证视频内容
func (s *VideoService) validateVideoContent(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) error {
	// 验证视频文件格式
	if err := s.processor.ValidateVideoFile(ctx, file); err != nil {
		s.log.WithContext(ctx).Warnf("video content validation failed: %v", err)
		return utils.ErrVideoFormatErr
	}

	// 检查视频是否有效
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	isValid, err := s.processor.IsValidVideo(ctx, file, fileHeader.Filename, fileHeader.Size)
	if err != nil {
		s.log.WithContext(ctx).Warnf("video validation failed: %v", err)
		return utils.ErrVideoFormatErr
	}

	if !isValid {
		s.log.WithContext(ctx).Warn("invalid video file")
		return utils.ErrVideoFormatErr
	}

	return nil
}

// buildVideoResponse 构建视频响应
func (s *VideoService) buildVideoResponse(ctx context.Context, video *domain.Video, currentUserID int64) (*commonv1.Video, error) {
	// 获取作者信息
	author, err := s.userUc.GetUser(ctx, video.AuthorID)
	if err != nil {
		return nil, err
	}

	// 检查是否已点赞（简化实现）
	isFavorite := false
	if currentUserID > 0 {
		// TODO: 实现点赞状态检查
	}

	// 检查是否已关注（简化实现）
	isFollow := false
	if currentUserID > 0 && currentUserID != video.AuthorID {
		// TODO: 实现关注状态检查
	}

	return &commonv1.Video{
		Id: video.ID,
		Author: &commonv1.User{
			Id:              author.ID,
			Name:            author.Nickname,
			FollowCount:     int64(author.FollowCount),
			FollowerCount:   int64(author.FollowerCount),
			IsFollow:        isFollow,
			Avatar:          author.Avatar,
			BackgroundImage: author.BackgroundImage,
			Signature:       author.Signature,
			TotalFavorited:  author.TotalFavorited,
			WorkCount:       int64(author.WorkCount),
			FavoriteCount:   int64(author.FavoriteCount),
		},
		PlayUrl:       video.PlayURL,
		CoverUrl:      video.CoverURL,
		FavoriteCount: video.FavoriteCount,
		CommentCount:  video.CommentCount,
		IsFavorite:    isFavorite,
		Title:         video.Title,
		CreatedAt:     video.CreatedAt.Unix(),
	}, nil
}
