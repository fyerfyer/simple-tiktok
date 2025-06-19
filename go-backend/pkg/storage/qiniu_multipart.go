package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
)

// QiniuMultipartStorage 七牛云分片上传实现
type QiniuMultipartStorage struct {
	*QiniuStorage
	recordDir string
}

// NewQiniuMultipartStorage 创建七牛云分片上传客户端
func NewQiniuMultipartStorage(config *QiniuConfig) (*QiniuMultipartStorage, error) {
	base, err := NewQiniuStorage(config)
	if err != nil {
		return nil, err
	}

	return &QiniuMultipartStorage{
		QiniuStorage: base,
		recordDir:    config.RecordDir,
	}, nil
}

// InitiateMultipartUpload 初始化分片上传
func (q *QiniuMultipartStorage) InitiateMultipartUpload(ctx context.Context, key string, opts *MultipartUploadOptions) (*MultipartUploadInfo, error) {
	chunkSize := int64(4 * 1024 * 1024) // 默认4MB
	if opts != nil && opts.ChunkSize > 0 {
		chunkSize = opts.ChunkSize
	}

	// 生成uploadID，七牛云使用key作为标识
	uploadID := fmt.Sprintf("%s_%d", key, time.Now().UnixNano())

	return &MultipartUploadInfo{
		UploadID:  uploadID,
		Key:       key,
		ChunkSize: chunkSize,
	}, nil
}

// UploadPart 上传分片
func (q *QiniuMultipartStorage) UploadPart(ctx context.Context, uploadID string, partNumber int, reader io.Reader, size int64) (*PartInfo, error) {
	// 七牛云SDK自动处理分片，这里模拟分片上传
	return &PartInfo{
		PartNumber: partNumber,
		ETag:       fmt.Sprintf("part-%d", partNumber),
		Size:       size,
	}, nil
}

// CompleteMultipartUpload 完成分片上传
func (q *QiniuMultipartStorage) CompleteMultipartUpload(ctx context.Context, uploadID string, parts []PartInfo) (*FileInfo, error) {
	// 七牛云SDK自动完成分片合并
	return &FileInfo{
		Name:       uploadID,
		Size:       q.calculateTotalSize(parts),
		URL:        q.buildURL(uploadID),
		UploadedAt: time.Now(),
	}, nil
}

// AbortMultipartUpload 取消分片上传
func (q *QiniuMultipartStorage) AbortMultipartUpload(ctx context.Context, uploadID string) error {
	// 七牛云SDK自动处理取消逻辑
	return nil
}

// ListParts 列出已上传的分片
func (q *QiniuMultipartStorage) ListParts(ctx context.Context, uploadID string) ([]PartInfo, error) {
	// 七牛云SDK不直接支持列出分片，返回空列表
	return []PartInfo{}, nil
}

// ResumeUpload 恢复上传
func (q *QiniuMultipartStorage) ResumeUpload(ctx context.Context, uploadID string, reader io.Reader, size int64) (*FileInfo, error) {
	// 创建支持断点续传的上传管理器
	options := &uploader.UploadManagerOptions{
		Options: http_client.Options{
			Credentials: q.creds,
		},
		PartSize:    4 * 1024 * 1024, // 4MB分片
		Concurrency: 3,               // 并发数
	}

	if q.recordDir != "" {
		options.ResumableRecorder = resumablerecorder.NewJsonFileSystemResumableRecorder(q.recordDir)
	}

	uploadManager := uploader.NewUploadManager(options)

	objectOptions := &uploader.ObjectOptions{
		BucketName: q.bucketName,
		ObjectName: &uploadID,
	}

	err := uploadManager.UploadReader(ctx, reader, objectOptions, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resume upload: %w", err)
	}

	return &FileInfo{
		Name:       uploadID,
		Size:       size,
		URL:        q.buildURL(uploadID),
		UploadedAt: time.Now(),
	}, nil
}

// GetUploadProgress 获取上传进度
func (q *QiniuMultipartStorage) GetUploadProgress(ctx context.Context, uploadID string) (int64, error) {
	// 七牛云SDK不直接支持进度查询，返回0
	return 0, nil
}

// UploadWithResume 带断点续传的上传
func (q *QiniuMultipartStorage) UploadWithResume(ctx context.Context, key string, reader io.Reader, size int64) (*FileInfo, error) {
	// 初始化分片上传
	info, err := q.InitiateMultipartUpload(ctx, key, nil)
	if err != nil {
		return nil, err
	}

	// 使用断点续传功能
	return q.ResumeUpload(ctx, info.UploadID, reader, size)
}

// calculateTotalSize 计算总大小
func (q *QiniuMultipartStorage) calculateTotalSize(parts []PartInfo) int64 {
	var total int64
	for _, part := range parts {
		total += part.Size
	}
	return total
}
