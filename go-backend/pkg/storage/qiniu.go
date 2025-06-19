package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"go-backend/pkg/utils"

	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

// QiniuConfig 七牛云配置
type QiniuConfig struct {
	AccessKey  string
	SecretKey  string
	BucketName string
	Domain     string
	Region     string
	UseHTTPS   bool
}

// QiniuStorage 七牛云存储实现
type QiniuStorage struct {
	mac        *qbox.Mac
	bucketName string
	domain     string
	cfg        *storage.Config
	useHTTPS   bool
}

// NewQiniuStorage 创建七牛云存储客户端
func NewQiniuStorage(config *QiniuConfig) (*QiniuStorage, error) {
	mac := qbox.NewMac(config.AccessKey, config.SecretKey)

	cfg := &storage.Config{
		UseHTTPS:      config.UseHTTPS,
		UseCdnDomains: false,
	}

	// 设置区域
	switch config.Region {
	case "z0":
		cfg.Region = &storage.ZoneHuadong
	case "z1":
		cfg.Region = &storage.ZoneHuabei
	case "z2":
		cfg.Region = &storage.ZoneHuanan
	case "na0":
		cfg.Region = &storage.ZoneBeimei
	case "as0":
		cfg.Region = &storage.ZoneXinjiapo
	default:
		cfg.Region = &storage.ZoneHuadong
	}

	return &QiniuStorage{
		mac:        mac,
		bucketName: config.BucketName,
		domain:     config.Domain,
		cfg:        cfg,
		useHTTPS:   config.UseHTTPS,
	}, nil
}

// Upload 上传文件
func (q *QiniuStorage) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, opts *UploadOptions) (*FileInfo, error) {
	putPolicy := storage.PutPolicy{
		Scope: fmt.Sprintf("%s:%s", q.bucketName, objectName),
	}

	if opts != nil && opts.Expires > 0 {
		putPolicy.Expires = uint64(time.Now().Add(opts.Expires).Unix())
	}

	upToken := putPolicy.UploadToken(q.mac)
	formUploader := storage.NewFormUploader(q.cfg)

	putExtra := storage.PutExtra{}
	if opts != nil && opts.Metadata != nil {
		putExtra.Params = opts.Metadata
	}

	ret := storage.PutRet{}
	err := formUploader.Put(ctx, &ret, upToken, objectName, reader, size, &putExtra)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to qiniu: %w", err)
	}

	fileInfo := &FileInfo{
		Name:       objectName,
		Size:       size,
		URL:        q.buildURL(objectName),
		ETag:       ret.Hash,
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
	bucketManager := storage.NewBucketManager(q.mac, q.cfg)
	return bucketManager.Delete(q.bucketName, objectName)
}

// GetPresignedURL 获取预签名URL
func (q *QiniuStorage) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (string, error) {
	deadline := time.Now().Add(expires).Unix()
	privateAccessURL := storage.MakePrivateURL(q.mac, q.domain, objectName, deadline)
	return privateAccessURL, nil
}

// Exists 检查文件是否存在
func (q *QiniuStorage) Exists(ctx context.Context, objectName string) (bool, error) {
	bucketManager := storage.NewBucketManager(q.mac, q.cfg)
	_, err := bucketManager.Stat(q.bucketName, objectName)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetFileInfo 获取文件信息
func (q *QiniuStorage) GetFileInfo(ctx context.Context, objectName string) (*FileInfo, error) {
	bucketManager := storage.NewBucketManager(q.mac, q.cfg)
	fileInfo, err := bucketManager.Stat(q.bucketName, objectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &FileInfo{
		Name:        objectName,
		Size:        fileInfo.Fsize,
		ContentType: fileInfo.MimeType,
		URL:         q.buildURL(objectName),
		ETag:        fileInfo.Hash,
		UploadedAt:  time.Unix(fileInfo.PutTime/10000000, 0),
	}, nil
}

// UploadVideo 上传视频文件
func (q *QiniuStorage) UploadVideo(ctx context.Context, filename string, reader io.Reader, size int64) (string, error) {
	videoID := utils.MustGenerateID()
	ext := filepath.Ext(filename)
	objectName := fmt.Sprintf("videos/%d%s", videoID, ext)

	opts := &UploadOptions{
		ContentType: q.getVideoContentType(ext),
		Metadata: map[string]string{
			"x:original-filename": filename,
			"x:video-id":          fmt.Sprintf("%d", videoID),
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
			"x:original-filename": filename,
			"x:cover-id":          fmt.Sprintf("%d", coverID),
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
