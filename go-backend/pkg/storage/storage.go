package storage

import (
	"context"
	"io"
	"time"
)

// FileInfo 文件信息
type FileInfo struct {
	Name        string
	Size        int64
	ContentType string
	URL         string
	ETag        string
	UploadedAt  time.Time
}

// UploadOptions 上传选项
type UploadOptions struct {
	ContentType string
	Metadata    map[string]string
	Expires     time.Duration
}

// Storage 存储接口
type Storage interface {
	// Upload 上传文件
	Upload(ctx context.Context, objectName string, reader io.Reader, size int64, opts *UploadOptions) (*FileInfo, error)

	// Download 下载文件
	Download(ctx context.Context, objectName string) (io.ReadCloser, error)

	// Delete 删除文件
	Delete(ctx context.Context, objectName string) error

	// GetPresignedURL 获取预签名URL
	GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (string, error)

	// Exists 检查文件是否存在
	Exists(ctx context.Context, objectName string) (bool, error)

	// GetFileInfo 获取文件信息
	GetFileInfo(ctx context.Context, objectName string) (*FileInfo, error)
}

// VideoStorage 视频存储接口
type VideoStorage interface {
	Storage

	// UploadVideo 上传视频文件
	UploadVideo(ctx context.Context, filename string, reader io.Reader, size int64) (string, error)

	// UploadCover 上传封面文件
	UploadCover(ctx context.Context, filename string, reader io.Reader, size int64) (string, error)

	// GenerateVideoURL 生成视频访问URL
	GenerateVideoURL(ctx context.Context, objectName string) (string, error)

	// GenerateCoverURL 生成封面访问URL
	GenerateCoverURL(ctx context.Context, objectName string) (string, error)
}

// MultipartStorage 分片上传存储接口
type MultipartStorage interface {
	Storage
	MultipartUpload
}

// ResumableStorage 断点续传存储接口
type ResumableStorage interface {
	MultipartStorage
	ResumableUpload
}

// Provider 存储提供商类型
type Provider string

const (
	ProviderMinIO Provider = "minio"
	ProviderQiniu Provider = "qiniu"
)
