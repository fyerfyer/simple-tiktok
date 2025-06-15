package auth

import (
	"errors"
	"time"

	"go-backend/pkg/security"

	"github.com/golang-jwt/jwt/v4"
)

// Claims JWT Claims结构
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	TokenID  string `json:"token_id"`
	jwt.RegisteredClaims
}

// RefreshClaims Refresh Token Claims
type RefreshClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	TokenID  string `json:"token_id"`
	jwt.RegisteredClaims
}

// TokenPair Token对
type TokenPair struct {
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	AccessExpiry  time.Time `json:"access_expiry"`
	RefreshExpiry time.Time `json:"refresh_expiry"`
}

// JWTManager JWT管理器
type JWTManager struct {
	accessSecret   string
	refreshSecret  string
	accessExpiry   time.Duration
	refreshExpiry  time.Duration
	tokenBlacklist TokenBlacklist
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(accessSecret string, accessExpiry time.Duration) *JWTManager {
	return &JWTManager{
		accessSecret:   accessSecret,
		refreshSecret:  accessSecret + "_refresh", // 简化处理，实际应该用不同密钥
		accessExpiry:   accessExpiry,
		refreshExpiry:  7 * 24 * time.Hour, // 7天
		tokenBlacklist: NewMemoryTokenBlacklist(),
	}
}

// SetTokenBlacklist 设置Token黑名单
func (j *JWTManager) SetTokenBlacklist(blacklist TokenBlacklist) {
	j.tokenBlacklist = blacklist
}

// GenerateToken 生成单个Access Token (兼容现有代码)
func (j *JWTManager) GenerateToken(userID int64, username string) (string, error) {
	tokenID, err := security.GenerateTokenID()
	if err != nil {
		return "", err
	}

	claims := &Claims{
		UserID:   userID,
		Username: username,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tiktok-service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.accessSecret))
}

// GenerateTokenPair 生成Token对
func (j *JWTManager) GenerateTokenPair(userID int64, username string) (*TokenPair, error) {
	accessTokenID, err := security.GenerateTokenID()
	if err != nil {
		return nil, err
	}

	refreshTokenID, err := security.GenerateTokenID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	accessExpiry := now.Add(j.accessExpiry)
	refreshExpiry := now.Add(j.refreshExpiry)

	// 生成Access Token
	accessClaims := &Claims{
		UserID:   userID,
		Username: username,
		TokenID:  accessTokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "tiktok-service",
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(j.accessSecret))
	if err != nil {
		return nil, err
	}

	// 生成Refresh Token
	refreshClaims := &RefreshClaims{
		UserID:   userID,
		Username: username,
		TokenID:  refreshTokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "tiktok-service",
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(j.refreshSecret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:   accessTokenString,
		RefreshToken:  refreshTokenString,
		AccessExpiry:  accessExpiry,
		RefreshExpiry: refreshExpiry,
	}, nil
}

// VerifyToken 验证Access Token (兼容现有代码)
func (j *JWTManager) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(j.accessSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// 检查Token是否在黑名单中
		if j.tokenBlacklist.IsBlacklisted(claims.TokenID) {
			return nil, errors.New("token is blacklisted")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// VerifyRefreshToken 验证Refresh Token
func (j *JWTManager) VerifyRefreshToken(tokenString string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(j.refreshSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*RefreshClaims); ok && token.Valid {
		// 检查Token是否在黑名单中
		if j.tokenBlacklist.IsBlacklisted(claims.TokenID) {
			return nil, errors.New("refresh token is blacklisted")
		}
		return claims, nil
	}

	return nil, errors.New("invalid refresh token")
}

// RefreshToken 刷新Token (兼容现有代码)
func (j *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := j.VerifyToken(tokenString)
	if err != nil {
		return "", err
	}

	return j.GenerateToken(claims.UserID, claims.Username)
}

// RefreshTokenPair 通过Refresh Token刷新Token对
func (j *JWTManager) RefreshTokenPair(refreshTokenString string) (*TokenPair, error) {
	refreshClaims, err := j.VerifyRefreshToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	// 将旧的refresh token加入黑名单
	j.tokenBlacklist.Add(refreshClaims.TokenID, time.Until(time.Unix(refreshClaims.ExpiresAt.Unix(), 0)))

	// 生成新的Token对
	return j.GenerateTokenPair(refreshClaims.UserID, refreshClaims.Username)
}

// RevokeToken 撤销Token
func (j *JWTManager) RevokeToken(tokenString string) error {
	claims, err := j.VerifyToken(tokenString)
	if err != nil {
		return err
	}

	// 将Token加入黑名单
	expiry := time.Until(time.Unix(claims.ExpiresAt.Unix(), 0))
	j.tokenBlacklist.Add(claims.TokenID, expiry)

	return nil
}

// RevokeRefreshToken 撤销Refresh Token
func (j *JWTManager) RevokeRefreshToken(refreshTokenString string) error {
	claims, err := j.VerifyRefreshToken(refreshTokenString)
	if err != nil {
		return err
	}

	// 将Refresh Token加入黑名单
	expiry := time.Until(time.Unix(claims.ExpiresAt.Unix(), 0))
	j.tokenBlacklist.Add(claims.TokenID, expiry)

	return nil
}

// GetTokenID 从Token中获取TokenID
func (j *JWTManager) GetTokenID(tokenString string) (string, error) {
	claims, err := j.VerifyToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.TokenID, nil
}
