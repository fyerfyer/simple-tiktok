package storage

import (
    "context"
    "io"
)

// MultipartUploadOptions 分片上传选项
type MultipartUploadOptions struct {
    ContentType string
    Metadata    map[string]string
    ChunkSize   int64
}

// MultipartUploadInfo 分片上传信息
type MultipartUploadInfo struct {
    UploadID string
    Key      string
    ChunkSize int64
}

// PartInfo 分片信息
type PartInfo struct {
    PartNumber int
    ETag       string
    Size       int64
}

// MultipartUpload 分片上传接口
type MultipartUpload interface {
    // InitiateMultipartUpload 初始化分片上传
    InitiateMultipartUpload(ctx context.Context, key string, opts *MultipartUploadOptions) (*MultipartUploadInfo, error)
    
    // UploadPart 上传分片
    UploadPart(ctx context.Context, uploadID string, partNumber int, reader io.Reader, size int64) (*PartInfo, error)
    
    // CompleteMultipartUpload 完成分片上传
    CompleteMultipartUpload(ctx context.Context, uploadID string, parts []PartInfo) (*FileInfo, error)
    
    // AbortMultipartUpload 取消分片上传
    AbortMultipartUpload(ctx context.Context, uploadID string) error
    
    // ListParts 列出已上传的分片
    ListParts(ctx context.Context, uploadID string) ([]PartInfo, error)
}

// ResumableUpload 断点续传接口
type ResumableUpload interface {
    MultipartUpload
    
    // ResumeUpload 恢复上传
    ResumeUpload(ctx context.Context, uploadID string, reader io.Reader, size int64) (*FileInfo, error)
    
    // GetUploadProgress 获取上传进度
    GetUploadProgress(ctx context.Context, uploadID string) (int64, error)
}