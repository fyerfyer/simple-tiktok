package middleware

import "context"

// 上下文key类型
type contextKey string

const (
	userIDKey       contextKey = "user_id"
	usernameKey     contextKey = "username"
	tokenIDKey      contextKey = "token_id"
	refreshTokenKey contextKey = "refresh_token"
)

// WithUserID 设置用户ID到上下文
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserIDFromContext 从上下文获取用户ID
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}

// WithUsername 设置用户名到上下文
func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey, username)
}

// GetUsernameFromContext 从上下文获取用户名
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameKey).(string)
	return username, ok
}

// WithTokenID 设置TokenID到上下文
func WithTokenID(ctx context.Context, tokenID string) context.Context {
	return context.WithValue(ctx, tokenIDKey, tokenID)
}

// GetTokenIDFromContext 从上下文获取TokenID
func GetTokenIDFromContext(ctx context.Context) (string, bool) {
	tokenID, ok := ctx.Value(tokenIDKey).(string)
	return tokenID, ok
}

// WithRefreshToken 设置刷新Token到上下文
func WithRefreshToken(ctx context.Context, refreshToken string) context.Context {
	return context.WithValue(ctx, refreshTokenKey, refreshToken)
}

// GetRefreshTokenFromContext 从上下文获取刷新Token
func GetRefreshTokenFromContext(ctx context.Context) (string, bool) {
	refreshToken, ok := ctx.Value(refreshTokenKey).(string)
	return refreshToken, ok
}

// MustGetUserID 从上下文获取用户ID（必须存在）
func MustGetUserID(ctx context.Context) int64 {
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		panic("user ID not found in context")
	}
	return userID
}

// MustGetUsername 从上下文获取用户名（必须存在）
func MustGetUsername(ctx context.Context) string {
	username, ok := GetUsernameFromContext(ctx)
	if !ok {
		panic("username not found in context")
	}
	return username
}

// IsAuthenticated 检查是否已认证
func IsAuthenticated(ctx context.Context) bool {
	_, ok := GetUserIDFromContext(ctx)
	return ok
}

// GetCurrentUser 获取当前用户信息
func GetCurrentUser(ctx context.Context) (userID int64, username string, ok bool) {
	userID, userOk := GetUserIDFromContext(ctx)
	username, usernameOk := GetUsernameFromContext(ctx)
	ok = userOk && usernameOk
	return
}
