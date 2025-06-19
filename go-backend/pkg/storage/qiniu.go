package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"go-backend/pkg/utils"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
)

// QiniuConfig 七牛云配置
type QiniuConfig struct {
	AccessKey  string
	SecretKey  string
	BucketName string
	Domain     string
	Region     string
	UseHTTPS   bool
	RecordDir  string // 断点续传记录目录
}

// QiniuStorage 七牛云存储实现
type QiniuStorage struct {
	creds         *credentials.Credentials
	bucketName    string
	domain        string
	useHTTPS      bool
	uploadManager *uploader.UploadManager
}

// NewQiniuStorage 创建七牛云存储客户端
func NewQiniuStorage(config *QiniuConfig) (*QiniuStorage, error) {
	creds := credentials.NewCredentials(config.AccessKey, config.SecretKey)

	// 设置上传管理器选项
	options := &uploader.UploadManagerOptions{
		Options: http_client.Options{
			Credentials: creds,
		},
		PartSize:    4 * 1024 * 1024, // 4MB分片
		Concurrency: 3,               // 并发数
	}

	// 如果提供了记录目录，启用断点续传
	if config.RecordDir != "" {
		options.ResumableRecorder = resumablerecorder.NewJsonFileSystemResumableRecorder(config.RecordDir)
	}

	uploadManager := uploader.NewUploadManager(options)

	return &QiniuStorage{
		creds:         creds,
		bucketName:    config.BucketName,
		domain:        config.Domain,
		useHTTPS:      config.UseHTTPS,
		uploadManager: uploadManager,
	}, nil
}

// Upload 上传文件
func (q *QiniuStorage) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, opts *UploadOptions) (*FileInfo, error) {
	objectOptions := &uploader.ObjectOptions{
		BucketName: q.bucketName,
		ObjectName: &objectName,
	}

	if opts != nil {
		if opts.Metadata != nil {
			objectOptions.CustomVars = opts.Metadata
		}
	}

	err := q.uploadManager.UploadReader(ctx, reader, objectOptions, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to qiniu: %w", err)
	}

	fileInfo := &FileInfo{
		Name:       objectName,
		Size:       size,
		URL:        q.buildURL(objectName),
		UploadedAt: time.Now(),
	}

	if opts != nil && opts.ContentType != "" {
		fileInfo.ContentType = opts.ContentType
	}

	return fileInfo, nil
}

// Download 下载文件
func (q *QiniuStorage) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("download not implemented for qiniu storage")
}

// Delete 删除文件
func (q *QiniuStorage) Delete(ctx context.Context, objectName string) error {
	return fmt.Errorf("delete not implemented for qiniu storage")
}

// GetPresignedURL 获取预签名URL
func (q *QiniuStorage) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (string, error) {
	return q.buildURL(objectName), nil
}

// Exists 检查文件是否存在
func (q *QiniuStorage) Exists(ctx context.Context, objectName string) (bool, error) {
	return false, fmt.Errorf("exists not implemented for qiniu storage")
}

// GetFileInfo 获取文件信息
func (q *QiniuStorage) GetFileInfo(ctx context.Context, objectName string) (*FileInfo, error) {
	return nil, fmt.Errorf("get file info not implemented for qiniu storage")
}

// UploadVideo 上传视频文件
func (q *QiniuStorage) UploadVideo(ctx context.Context, filename string, reader io.Reader, size int64) (string, error) {
	videoID := utils.MustGenerateID()
	ext := filepath.Ext(filename)
	objectName := fmt.Sprintf("videos/%d%s", videoID, ext)

	opts := &UploadOptions{
		ContentType: q.getVideoContentType(ext),
		Metadata: map[string]string{
			"original-filename": filename,
			"video-id":          fmt.Sprintf("%d", videoID),
		},
	}

	_, err := q.Upload(ctx, objectName, reader, size, opts)
	if err != nil {
		return "", err
	}

	return objectName, nil
}

// UploadCover 上传封面文件
func (q *QiniuStorage) UploadCover(ctx context.Context, filename string, reader io.Reader, size int64) (string, error) {
	coverID := utils.MustGenerateID()
	objectName := fmt.Sprintf("covers/%d.jpg", coverID)

	opts := &UploadOptions{
		ContentType: "image/jpeg",
		Metadata: map[string]string{
			"original-filename": filename,
			"cover-id":          fmt.Sprintf("%d", coverID),
		},
	}

	_, err := q.Upload(ctx, objectName, reader, size, opts)
	if err != nil {
		return "", err
	}

	return objectName, nil
}

// GenerateVideoURL 生成视频访问URL
func (q *QiniuStorage) GenerateVideoURL(ctx context.Context, objectName string) (string, error) {
	return q.buildURL(objectName), nil
}

// GenerateCoverURL 生成封面访问URL
func (q *QiniuStorage) GenerateCoverURL(ctx context.Context, objectName string) (string, error) {
	return q.buildURL(objectName), nil
}

// buildURL 构建访问URL
func (q *QiniuStorage) buildURL(objectName string) string {
	protocol := "http"
	if q.useHTTPS {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/%s", protocol, q.domain, objectName)
}

// getVideoContentType 获取视频内容类型
func (q *QiniuStorage) getVideoContentType(ext string) string {
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
