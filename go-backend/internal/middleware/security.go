package middleware

import (
	"context"
	"regexp"
	"strings"

	"go-backend/api/common/v1"
	"go-backend/pkg/security"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	transportHttp "github.com/go-kratos/kratos/v2/transport/http"
)

// SecurityMiddleware 安全中间件
type SecurityMiddleware struct {
	validator *security.Validator
	log       *log.Helper
}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware(validator *security.Validator, logger log.Logger) *SecurityMiddleware {
	return &SecurityMiddleware{
		validator: validator,
		log:       log.NewHelper(logger),
	}
}

// GlobalSecurityHandler 全局安全处理
func (m *SecurityMiddleware) GlobalSecurityHandler() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 设置安全头
			if err := m.setSecurityHeaders(ctx); err != nil {
				m.log.WithContext(ctx).Errorf("set security headers failed: %v", err)
			}

			// SQL注入检测
			if err := m.checkSQLInjection(ctx); err != nil {
				m.log.WithContext(ctx).Warnf("SQL injection detected: %v", err)
				return nil, NewAuthError(v1.ErrorCode_PARAM_ERROR, "invalid request")
			}

			// XSS检测
			if err := m.checkXSS(ctx); err != nil {
				m.log.WithContext(ctx).Warnf("XSS attack detected: %v", err)
				return nil, NewAuthError(v1.ErrorCode_PARAM_ERROR, "invalid request")
			}

			return handler(ctx, req)
		}
	}
}

// CSRFProtection CSRF保护
func (m *SecurityMiddleware) CSRFProtection() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			if ht, ok := tr.(transportHttp.Transporter); ok {
				httpReq := ht.Request()

				// 对于非安全方法检查CSRF
				if httpReq.Method != "GET" && httpReq.Method != "HEAD" && httpReq.Method != "OPTIONS" {
					// 检查Referer头
					referer := httpReq.Header.Get("Referer")
					if referer == "" {
						m.log.WithContext(ctx).Warn("missing referer header")
						return nil, NewAuthError(v1.ErrorCode_PARAM_ERROR, "missing referer")
					}

					// 检查Origin头
					origin := httpReq.Header.Get("Origin")
					if origin != "" {
						// 简化检查：确保Origin与Host一致
						host := httpReq.Header.Get("Host")
						if !strings.Contains(origin, host) {
							m.log.WithContext(ctx).Warnf("origin mismatch: origin=%s, host=%s", origin, host)
							return nil, NewAuthError(v1.ErrorCode_PARAM_ERROR, "origin mismatch")
						}
					}
				}
			}

			return handler(ctx, req)
		}
	}
}

// ContentSecurityPolicy 内容安全策略
func (m *SecurityMiddleware) ContentSecurityPolicy() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if err := m.setCSPHeaders(ctx); err != nil {
				m.log.WithContext(ctx).Errorf("set CSP headers failed: %v", err)
			}
			return handler(ctx, req)
		}
	}
}

// InputValidation 输入验证
func (m *SecurityMiddleware) InputValidation() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 检查请求大小
			if err := m.checkRequestSize(ctx); err != nil {
				return nil, err
			}

			// 检查文件类型
			if err := m.checkFileType(ctx); err != nil {
				return nil, err
			}

			return handler(ctx, req)
		}
	}
}

// setSecurityHeaders 设置安全头
func (m *SecurityMiddleware) setSecurityHeaders(ctx context.Context) error {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil
	}

	if ht, ok := tr.(transportHttp.Transporter); ok {
		header := ht.ReplyHeader()

		// 设置各种安全头
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("X-XSS-Protection", "1; mode=block")
		header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		header.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
	}

	return nil
}

// setCSPHeaders 设置CSP头
func (m *SecurityMiddleware) setCSPHeaders(ctx context.Context) error {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil
	}

	if ht, ok := tr.(transportHttp.Transporter); ok {
		header := ht.ReplyHeader()

		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"media-src 'self'; " +
			"object-src 'none'; " +
			"child-src 'none'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'"

		header.Set("Content-Security-Policy", csp)
	}

	return nil
}

// checkSQLInjection SQL注入检测
func (m *SecurityMiddleware) checkSQLInjection(ctx context.Context) error {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil
	}

	if ht, ok := tr.(transportHttp.Transporter); ok {
		req := ht.Request()

		// SQL注入关键词
		sqlPatterns := []string{
			`(?i)\bunion\b.*\bselect\b`,
			`(?i)\bselect\b.*\bfrom\b`,
			`(?i)\binsert\b.*\binto\b`,
			`(?i)\bupdate\b.*\bset\b`,
			`(?i)\bdelete\b.*\bfrom\b`,
			`(?i)\bdrop\b.*\btable\b`,
			`(?i);\s*(drop|alter|create|truncate)`,
			`(?i)\bor\b.*=.*\bor\b`,
			`(?i)'\s*(or|and)\s*'?1'?\s*=\s*'?1`,
			`--\s*$`,
			`/\*.*\*/`,
		}

		// 检查查询参数
		for _, values := range req.URL.Query() {
			for _, value := range values {
				for _, pattern := range sqlPatterns {
					if matched, _ := regexp.MatchString(pattern, value); matched {
						return NewAuthError(v1.ErrorCode_PARAM_ERROR, "potential SQL injection")
					}
				}
			}
		}

		// 检查请求体（如果是POST请求）
		if req.Method == "POST" {
			req.ParseForm()
			for _, values := range req.PostForm {
				for _, value := range values {
					for _, pattern := range sqlPatterns {
						if matched, _ := regexp.MatchString(pattern, value); matched {
							return NewAuthError(v1.ErrorCode_PARAM_ERROR, "potential SQL injection")
						}
					}
				}
			}
		}
	}

	return nil
}

// checkXSS XSS检测
func (m *SecurityMiddleware) checkXSS(ctx context.Context) error {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil
	}

	if ht, ok := tr.(transportHttp.Transporter); ok {
		req := ht.Request()

		// XSS模式
		xssPatterns := []string{
			`(?i)<script.*?>.*?</script>`,
			`(?i)<iframe.*?>.*?</iframe>`,
			`(?i)javascript:`,
			`(?i)on\w+\s*=`,
			`(?i)<.*?\bstyle\s*=.*?\bexpression\s*\(`,
			`(?i)<.*?\bsrc\s*=.*?\bjavascript:`,
		}

		// 检查所有参数
		allParams := make([]string, 0)

		// URL参数
		for _, values := range req.URL.Query() {
			allParams = append(allParams, values...)
		}

		// POST参数
		if req.Method == "POST" {
			req.ParseForm()
			for _, values := range req.PostForm {
				allParams = append(allParams, values...)
			}
		}

		// 检查XSS
		for _, param := range allParams {
			for _, pattern := range xssPatterns {
				if matched, _ := regexp.MatchString(pattern, param); matched {
					return NewAuthError(v1.ErrorCode_PARAM_ERROR, "potential XSS attack")
				}
			}
		}
	}

	return nil
}

// checkRequestSize 检查请求大小
func (m *SecurityMiddleware) checkRequestSize(ctx context.Context) error {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil
	}

	if ht, ok := tr.(transportHttp.Transporter); ok {
		req := ht.Request()

		// 检查Content-Length
		if req.ContentLength > 10*1024*1024 { // 10MB
			return NewAuthError(v1.ErrorCode_PARAM_ERROR, "request too large")
		}
	}

	return nil
}

// checkFileType 检查文件类型
func (m *SecurityMiddleware) checkFileType(ctx context.Context) error {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil
	}

	if ht, ok := tr.(transportHttp.Transporter); ok {
		req := ht.Request()

		contentType := req.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "multipart/form-data") {
			// 文件上传请求，检查文件类型
			if err := req.ParseMultipartForm(32 << 20); err != nil { // 32MB
				return NewAuthError(v1.ErrorCode_PARAM_ERROR, "invalid multipart form")
			}

			if req.MultipartForm != nil {
				for _, fileHeaders := range req.MultipartForm.File {
					for _, fileHeader := range fileHeaders {
						// 检查文件扩展名
						if !m.isAllowedFileType(fileHeader.Filename) {
							return NewAuthError(v1.ErrorCode_PARAM_ERROR, "file type not allowed")
						}
					}
				}
			}
		}
	}

	return nil
}

// isAllowedFileType 检查是否为允许的文件类型
func (m *SecurityMiddleware) isAllowedFileType(filename string) bool {
	allowedExtensions := []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp", // 图片
		".mp4", ".avi", ".mov", ".wmv", ".flv", // 视频
		".mp3", ".wav", ".aac", ".ogg", // 音频
		".txt", ".pdf", ".doc", ".docx", // 文档
	}

	filename = strings.ToLower(filename)
	for _, ext := range allowedExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

// LogSecurityEvent 记录安全事件
func (m *SecurityMiddleware) LogSecurityEvent(ctx context.Context, eventType, message string) {
	userID, _ := GetUserIDFromContext(ctx)
	ip := ""

	if tr, ok := transport.FromServerContext(ctx); ok {
		if ht, ok := tr.(transportHttp.Transporter); ok {
			req := ht.Request()
			ip = req.RemoteAddr
		}
	}

	m.log.WithContext(ctx).Warnf("security event: type=%s, user=%d, ip=%s, message=%s",
		eventType, userID, ip, message)
}
