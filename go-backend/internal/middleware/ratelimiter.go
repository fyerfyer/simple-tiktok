package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-backend/api/common/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	limiters map[string]*rate.Limiter
	mutex    sync.RWMutex
	log      *log.Helper
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(logger log.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiters: make(map[string]*rate.Limiter),
		log:      log.NewHelper(logger),
	}
}

// Limit 全局限流
func (m *RateLimitMiddleware) Limit() middleware.Middleware {
	return m.LimitWithConfig(100, 10) // 每秒100次，突发10次
}

// LimitWithConfig 自定义限流配置
func (m *RateLimitMiddleware) LimitWithConfig(rps, burst int) middleware.Middleware {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if !limiter.Allow() {
				m.log.WithContext(ctx).Warn("rate limit exceeded")
				return nil, NewAuthError(v1.ErrorCode_RATE_LIMIT, "rate limit exceeded")
			}
			return handler(ctx, req)
		}
	}
}

// LimitByIP IP限流
func (m *RateLimitMiddleware) LimitByIP(rps, burst int) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			ip := m.getClientIP(ctx)
			if ip == "" {
				return handler(ctx, req)
			}

			limiter := m.getLimiter(ip, rps, burst)
			if !limiter.Allow() {
				m.log.WithContext(ctx).Warnf("rate limit exceeded for IP: %s", ip)
				return nil, NewAuthError(v1.ErrorCode_RATE_LIMIT, "rate limit exceeded")
			}

			return handler(ctx, req)
		}
	}
}

// LimitByUser 用户限流
func (m *RateLimitMiddleware) LimitByUser(rps, burst int) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			userID, ok := GetUserIDFromContext(ctx)
			if !ok {
				// 未认证用户使用IP限流
				return m.LimitByIP(rps/2, burst/2)(handler)(ctx, req)
			}

			key := fmt.Sprintf("user:%d", userID)
			limiter := m.getLimiter(key, rps, burst)
			if !limiter.Allow() {
				m.log.WithContext(ctx).Warnf("rate limit exceeded for user: %d", userID)
				return nil, NewAuthError(v1.ErrorCode_RATE_LIMIT, "rate limit exceeded")
			}

			return handler(ctx, req)
		}
	}
}

// LimitLogin 登录限流
func (m *RateLimitMiddleware) LimitLogin() middleware.Middleware {
	return m.LimitByIP(5, 2) // 每秒5次，突发2次
}

// LimitRegister 注册限流
func (m *RateLimitMiddleware) LimitRegister() middleware.Middleware {
	return m.LimitByIP(2, 1) // 每秒2次，突发1次
}

// LimitUpload 上传限流
func (m *RateLimitMiddleware) LimitUpload() middleware.Middleware {
	return m.LimitByUser(10, 5) // 每秒10次，突发5次
}

// LimitComment 评论限流
func (m *RateLimitMiddleware) LimitComment() middleware.Middleware {
	return m.LimitByUser(20, 10) // 每秒20次，突发10次
}

// getLimiter 获取或创建限流器
func (m *RateLimitMiddleware) getLimiter(key string, rps, burst int) *rate.Limiter {
	m.mutex.RLock()
	limiter, exists := m.limiters[key]
	m.mutex.RUnlock()

	if !exists {
		m.mutex.Lock()
		// 双重检查
		if limiter, exists = m.limiters[key]; !exists {
			limiter = rate.NewLimiter(rate.Limit(rps), burst)
			m.limiters[key] = limiter
		}
		m.mutex.Unlock()
	}

	return limiter
}

// getClientIP 获取客户端IP
func (m *RateLimitMiddleware) getClientIP(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}

	if ht, ok := tr.(http.Transporter); ok {
		req := ht.Request()

		// 检查X-Forwarded-For头
		if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
			return xff
		}

		// 检查X-Real-IP头
		if xri := req.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}

		// 使用RemoteAddr
		return req.RemoteAddr
	}

	return ""
}

// cleanup 定期清理过期的限流器
func (m *RateLimitMiddleware) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.mutex.Lock()
		// 简单清理策略：定期清空所有限流器
		// 实际项目中可以实现更精细的清理逻辑
		if len(m.limiters) > 10000 {
			m.limiters = make(map[string]*rate.Limiter)
		}
		m.mutex.Unlock()
	}
}

// StartCleanup 启动清理goroutine
func (m *RateLimitMiddleware) StartCleanup() {
	go m.cleanup()
}

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	window   time.Duration
	limit    int
	requests map[string][]time.Time
	mutex    sync.RWMutex
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(window time.Duration, limit int) *SlidingWindowLimiter {
	limiter := &SlidingWindowLimiter{
		window:   window,
		limit:    limit,
		requests: make(map[string][]time.Time),
	}
	go limiter.cleanup()
	return limiter
}

// Allow 检查是否允许请求
func (l *SlidingWindowLimiter) Allow(key string) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()
	requests := l.requests[key]

	// 清理过期请求
	cutoff := now.Add(-l.window)
	validRequests := make([]time.Time, 0, len(requests))
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// 检查是否超限
	if len(validRequests) >= l.limit {
		l.requests[key] = validRequests
		return false
	}

	// 添加当前请求
	validRequests = append(validRequests, now)
	l.requests[key] = validRequests
	return true
}

// cleanup 清理过期数据
func (l *SlidingWindowLimiter) cleanup() {
	ticker := time.NewTicker(l.window)
	defer ticker.Stop()

	for range ticker.C {
		l.mutex.Lock()
		now := time.Now()
		cutoff := now.Add(-l.window)

		for key, requests := range l.requests {
			validRequests := make([]time.Time, 0, len(requests))
			for _, reqTime := range requests {
				if reqTime.After(cutoff) {
					validRequests = append(validRequests, reqTime)
				}
			}

			if len(validRequests) == 0 {
				delete(l.requests, key)
			} else {
				l.requests[key] = validRequests
			}
		}
		l.mutex.Unlock()
	}
}
