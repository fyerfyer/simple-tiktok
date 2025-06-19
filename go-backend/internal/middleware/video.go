package middleware

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"go-backend/api/common/v1"
	"go-backend/pkg/media"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// VideoMiddleware 视频中间件
type VideoMiddleware struct {
	processor *media.VideoProcessor
	log       *log.Helper
}

// NewVideoMiddleware 创建视频中间件
func NewVideoMiddleware(processor *media.VideoProcessor, logger log.Logger) *VideoMiddleware {
	return &VideoMiddleware{
		processor: processor,
		log:       log.NewHelper(logger),
	}
}

// FileUploadValidator 文件上传验证中间件
func (v *VideoMiddleware) FileUploadValidator() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			if ht, ok := tr.(http.Transporter); ok {
				httpReq := ht.Request()

				// 只对上传接口进行验证
				if httpReq.Method == "POST" && strings.Contains(httpReq.URL.Path, "/publish/action") {
					if err := v.validateUploadRequest(httpReq); err != nil {
						v.log.WithContext(ctx).Warnf("file upload validation failed: %v", err)
						return nil, err
					}
				}
			}

			return handler(ctx, req)
		}
	}
}

// VideoTitleValidator 视频标题验证中间件
func (v *VideoMiddleware) VideoTitleValidator() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			if ht, ok := tr.(http.Transporter); ok {
				httpReq := ht.Request()

				if httpReq.Method == "POST" && strings.Contains(httpReq.URL.Path, "/publish/action") {
					title := httpReq.FormValue("title")
					if err := v.processor.ValidateVideoTitle(title); err != nil {
						v.log.WithContext(ctx).Warnf("video title validation failed: %v", err)
						return nil, NewVideoError(v1.ErrorCode_PARAM_ERROR, err.Error())
					}
				}
			}

			return handler(ctx, req)
		}
	}
}

// FileSizeLimit 文件大小限制中间件
func (v *VideoMiddleware) FileSizeLimit() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			if ht, ok := tr.(http.Transporter); ok {
				httpReq := ht.Request()

				// 检查Content-Length
				if httpReq.ContentLength > v.processor.GetMaxFileSize() {
					v.log.WithContext(ctx).Warnf("file size %d exceeds limit %d",
						httpReq.ContentLength, v.processor.GetMaxFileSize())
					return nil, NewVideoError(v1.ErrorCode_VIDEO_SIZE_ERR, "file size exceeds limit")
				}
			}

			return handler(ctx, req)
		}
	}
}

// VideoFormatValidator 视频格式验证中间件
func (v *VideoMiddleware) VideoFormatValidator() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			if ht, ok := tr.(http.Transporter); ok {
				httpReq := ht.Request()

				if httpReq.Method == "POST" && strings.Contains(httpReq.URL.Path, "/publish/action") {
					if err := v.validateVideoFormat(httpReq); err != nil {
						v.log.WithContext(ctx).Warnf("video format validation failed: %v", err)
						return nil, err
					}
				}
			}

			return handler(ctx, req)
		}
	}
}

// validateUploadRequest 验证上传请求
func (v *VideoMiddleware) validateUploadRequest(req *http.Request) error {
	// 检查Content-Type
	contentType := req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		return NewVideoError(v1.ErrorCode_PARAM_ERROR, "invalid content type")
	}

	// 解析multipart form
	if err := req.ParseMultipartForm(32 << 20); err != nil { // 32MB
		return NewVideoError(v1.ErrorCode_PARAM_ERROR, "failed to parse multipart form")
	}

	// 检查必需字段
	if req.FormValue("token") == "" {
		return NewVideoError(v1.ErrorCode_TOKEN_INVALID, "token required")
	}

	if req.FormValue("title") == "" {
		return NewVideoError(v1.ErrorCode_PARAM_ERROR, "title required")
	}

	// 检查文件字段
	_, fileHeader, err := req.FormFile("data")
	if err != nil {
		return NewVideoError(v1.ErrorCode_PARAM_ERROR, "video file required")
	}

	if fileHeader.Size == 0 {
		return NewVideoError(v1.ErrorCode_PARAM_ERROR, "empty video file")
	}

	return nil
}

// validateVideoFormat 验证视频格式
func (v *VideoMiddleware) validateVideoFormat(req *http.Request) error {
	_, fileHeader, err := req.FormFile("data")
	if err != nil {
		return NewVideoError(v1.ErrorCode_PARAM_ERROR, "video file required")
	}

	// 验证文件扩展名
	filename := fileHeader.Filename
	ext := strings.ToLower(filepath.Ext(filename))

	supportedExts := []string{".mp4", ".avi", ".mov"}
	isSupported := false
	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return NewVideoError(v1.ErrorCode_VIDEO_FORMAT_ERR,
			fmt.Sprintf("unsupported video format: %s", ext))
	}

	// 验证文件大小
	if fileHeader.Size > v.processor.GetMaxFileSize() {
		return NewVideoError(v1.ErrorCode_VIDEO_SIZE_ERR, "video file too large")
	}

	return nil
}

// VideoContentValidator 视频内容验证中间件
func (v *VideoMiddleware) VideoContentValidator() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			if ht, ok := tr.(http.Transporter); ok {
				httpReq := ht.Request()

				if httpReq.Method == "POST" && strings.Contains(httpReq.URL.Path, "/publish/action") {
					if err := v.validateVideoContent(ctx, httpReq); err != nil {
						v.log.WithContext(ctx).Warnf("video content validation failed: %v", err)
						return nil, err
					}
				}
			}

			return handler(ctx, req)
		}
	}
}

// validateVideoContent 验证视频内容
func (v *VideoMiddleware) validateVideoContent(ctx context.Context, req *http.Request) error {
	file, fileHeader, err := req.FormFile("data")
	if err != nil {
		return NewVideoError(v1.ErrorCode_PARAM_ERROR, "video file required")
	}
	defer file.Close()

	// 验证视频文件内容
	isValid, err := v.processor.IsValidVideo(ctx, file, fileHeader.Filename, fileHeader.Size)
	if err != nil {
		return NewVideoError(v1.ErrorCode_VIDEO_FORMAT_ERR, "invalid video file")
	}

	if !isValid {
		return NewVideoError(v1.ErrorCode_VIDEO_FORMAT_ERR, "invalid video content")
	}

	return nil
}

// ExtractVideoFile 提取视频文件信息中间件
func (v *VideoMiddleware) ExtractVideoFile() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			if ht, ok := tr.(http.Transporter); ok {
				httpReq := ht.Request()

				if httpReq.Method == "POST" && strings.Contains(httpReq.URL.Path, "/publish/action") {
					_, fileHeader, err := httpReq.FormFile("data")
					if err == nil {
						ctx = WithVideoFileHeader(ctx, fileHeader)
					}
				}
			}

			return handler(ctx, req)
		}
	}
}

// WithVideoFileHeader 设置视频文件头到上下文
func WithVideoFileHeader(ctx context.Context, fileHeader *multipart.FileHeader) context.Context {
	return context.WithValue(ctx, "video_file_header", fileHeader)
}

// GetVideoFileHeaderFromContext 从上下文获取视频文件头
func GetVideoFileHeaderFromContext(ctx context.Context) (*multipart.FileHeader, bool) {
	fileHeader, ok := ctx.Value("video_file_header").(*multipart.FileHeader)
	return fileHeader, ok
}

// NewVideoError 创建视频错误
func NewVideoError(code v1.ErrorCode, message string) error {
	return fmt.Errorf("video error: %s", message)
}
