package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"go-backend/pkg/utils"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOConfig MinIO配置
type MinIOConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
	UseSSL     bool
	BaseURL    string
}

// MinIOStorage MinIO存储实现
type MinIOStorage struct {
	client     *minio.Client
	bucketName string
	baseURL    string
}

// NewMinIOStorage 创建MinIO存储客户端
func NewMinIOStorage(config *MinIOConfig) (*MinIOStorage, error) {
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
		Region: config.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	storage := &MinIOStorage{
		client:     client,
		bucketName: config.BucketName,
		baseURL:    config.BaseURL,
	}

	if err := storage.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket: %w", err)
	}

	return storage, nil
}

// ensureBucket 确保存储桶存在
func (s *MinIOStorage) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return err
	}

	if !exists {
		return s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{
			Region: "us-east-1",
		})
	}

	return nil
}

// Upload 上传文件
func (s *MinIOStorage) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, opts *UploadOptions) (*FileInfo, error) {
	putOpts := minio.PutObjectOptions{}

	if opts != nil {
		if opts.ContentType != "" {
			putOpts.ContentType = opts.ContentType
		}
		if opts.Metadata != nil {
			putOpts.UserMetadata = opts.Metadata
		}
	}

	info, err := s.client.PutObject(ctx, s.bucketName, objectName, reader, size, putOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}

	fileInfo := &FileInfo{
		Name:       objectName,
		Size:       info.Size,
		ETag:       info.ETag,
		URL:        s.buildObjectURL(objectName),
		UploadedAt: time.Now(),
	}

	if opts != nil && opts.ContentType != "" {
		fileInfo.ContentType = opts.ContentType
	}

	return fileInfo, nil
}

// Download 下载文件
func (s *MinIOStorage) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return object, nil
}

// Delete 删除文件
func (s *MinIOStorage) Delete(ctx context.Context, objectName string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object: %w", err)
	}
	return nil
}

// GetPresignedURL 获取预签名URL
func (s *MinIOStorage) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (string, error) {
	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucketName, objectName, expires, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presignedURL.String(), nil
}

// Exists 检查文件是否存在
func (s *MinIOStorage) Exists(ctx context.Context, objectName string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat object: %w", err)
	}
	return true, nil
}

// GetFileInfo 获取文件信息
func (s *MinIOStorage) GetFileInfo(ctx context.Context, objectName string) (*FileInfo, error) {
	stat, err := s.client.StatObject(ctx, s.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	return &FileInfo{
		Name:        objectName,
		Size:        stat.Size,
		ContentType: stat.ContentType,
		ETag:        stat.ETag,
		URL:         s.buildObjectURL(objectName),
		UploadedAt:  stat.LastModified,
	}, nil
}

// UploadVideo 上传视频文件
func (s *MinIOStorage) UploadVideo(ctx context.Context, filename string, reader io.Reader, size int64) (string, error) {
	videoID := utils.MustGenerateID()
	ext := filepath.Ext(filename)
	objectName := fmt.Sprintf("videos/%d%s", videoID, ext)

	opts := &UploadOptions{
		ContentType: s.getVideoContentType(ext),
		Metadata: map[string]string{
			"original-filename": filename,
			"video-id":          fmt.Sprintf("%d", videoID),
		},
	}

	_, err := s.Upload(ctx, objectName, reader, size, opts)
	if err != nil {
		return "", err
	}

	return objectName, nil
}

// UploadCover 上传封面文件
func (s *MinIOStorage) UploadCover(ctx context.Context, filename string, reader io.Reader, size int64) (string, error) {
	coverID := utils.MustGenerateID()
	objectName := fmt.Sprintf("covers/%d.jpg", coverID)

	opts := &UploadOptions{
		ContentType: "image/jpeg",
		Metadata: map[string]string{
			"original-filename": filename,
			"cover-id":          fmt.Sprintf("%d", coverID),
		},
	}

	_, err := s.Upload(ctx, objectName, reader, size, opts)
	if err != nil {
		return "", err
	}

	return objectName, nil
}

// GenerateVideoURL 生成视频访问URL
func (s *MinIOStorage) GenerateVideoURL(ctx context.Context, objectName string) (string, error) {
	return s.buildObjectURL(objectName), nil
}

// GenerateCoverURL 生成封面访问URL
func (s *MinIOStorage) GenerateCoverURL(ctx context.Context, objectName string) (string, error) {
	return s.buildObjectURL(objectName), nil
}

// buildObjectURL 构建对象URL
func (s *MinIOStorage) buildObjectURL(objectName string) string {
	if s.baseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.baseURL, "/"), objectName)
	}
	return fmt.Sprintf("http://%s/%s/%s", s.client.EndpointURL().Host, s.bucketName, objectName)
}

// getVideoContentType 获取视频内容类型
func (s *MinIOStorage) getVideoContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp4":
		return "video/mp4"
	case ".avi":
		return "video/avi"
	case ".mov":
		return "video/quicktime"
	default:
		return "video/mp4"
	}
}
