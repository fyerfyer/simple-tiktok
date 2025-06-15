package data

import (
	"context"
	"fmt"
	"time"

	"go-backend/internal/biz"
	"go-backend/internal/data/cache"
	"go-backend/pkg/auth"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username        string     `gorm:"uniqueIndex;size:32;not null" json:"username"`
	PasswordHash    string     `gorm:"size:128;not null" json:"-"`
	Salt            string     `gorm:"size:32;not null" json:"-"`
	Nickname        string     `gorm:"size:50" json:"nickname"`
	Avatar          string     `gorm:"size:255" json:"avatar"`
	BackgroundImage string     `gorm:"size:255" json:"background_image"`
	Signature       string     `gorm:"size:200" json:"signature"`
	FollowCount     int        `gorm:"default:0" json:"follow_count"`
	FollowerCount   int        `gorm:"default:0" json:"follower_count"`
	TotalFavorited  int64      `gorm:"default:0" json:"total_favorited"`
	WorkCount       int        `gorm:"default:0" json:"work_count"`
	FavoriteCount   int        `gorm:"default:0" json:"favorite_count"`
	Status          int8       `gorm:"default:1" json:"status"`
	LastLoginAt     *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

type userRepo struct {
	data        *Data
	log         *log.Helper
	userCache   *cache.UserCache
	passwordMgr *auth.PasswordManager
}

// NewUserRepo .
func NewUserRepo(data *Data, userCache *cache.UserCache, passwordMgr *auth.PasswordManager, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data:        data,
		log:         log.NewHelper(logger),
		userCache:   userCache,
		passwordMgr: passwordMgr,
	}
}

func (r *userRepo) CreateUser(ctx context.Context, user *biz.User) (*biz.User, error) {
	// 加密密码
	hash, salt, err := r.passwordMgr.HashPassword(user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("hash password failed: %w", err)
	}

	u := &User{
		Username:        user.Username,
		PasswordHash:    hash,
		Salt:            salt,
		Nickname:        user.Nickname,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Signature:       user.Signature,
		Status:          1,
	}

	if err := r.data.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, err
	}

	result := r.convertToUser(u)

	// 设置缓存
	r.userCache.SetUser(ctx, result)

	return result, nil
}

func (r *userRepo) GetUser(ctx context.Context, userID int64) (*biz.User, error) {
	// 先从缓存获取
	if user, err := r.userCache.GetUser(ctx, userID); err == nil && user != nil {
		return user, nil
	}

	var u User
	if err := r.data.db.WithContext(ctx).Where("id = ? AND status = 1", userID).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}

	user := r.convertToUser(&u)

	// 设置缓存
	r.userCache.SetUser(ctx, user)

	return user, nil
}

func (r *userRepo) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
	var u User
	if err := r.data.db.WithContext(ctx).Where("username = ? AND status = 1", username).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}

	user := r.convertToUser(&u)

	// 设置缓存
	r.userCache.SetUser(ctx, user)

	return user, nil
}

func (r *userRepo) GetUsers(ctx context.Context, userIDs []int64) ([]*biz.User, error) {
	// 批量从缓存获取
	cachedUsers, missedIDs := r.userCache.BatchGetUsers(ctx, userIDs)

	var result []*biz.User
	var dbUsers []User

	// 查询缓存未命中的用户
	if len(missedIDs) > 0 {
		if err := r.data.db.WithContext(ctx).Where("id IN ? AND status = 1", missedIDs).Find(&dbUsers).Error; err != nil {
			return nil, err
		}

		// 转换并缓存
		var cacheBatch []*biz.User
		for _, u := range dbUsers {
			user := r.convertToUser(&u)
			result = append(result, user)
			cacheBatch = append(cacheBatch, user)
		}

		// 批量设置缓存
		r.userCache.BatchSetUsers(ctx, cacheBatch)
	}

	// 合并缓存和数据库结果
	for _, userID := range userIDs {
		if cachedUser, exists := cachedUsers[userID]; exists {
			result = append(result, cachedUser)
		}
	}

	return result, nil
}

func (r *userRepo) UpdateUser(ctx context.Context, user *biz.User) error {
	updates := map[string]interface{}{
		"nickname":         user.Nickname,
		"avatar":           user.Avatar,
		"background_image": user.BackgroundImage,
		"signature":        user.Signature,
		"updated_at":       time.Now(),
	}

	if user.LastLoginAt != nil {
		updates["last_login_at"] = user.LastLoginAt
	}

	if err := r.data.db.WithContext(ctx).Model(&User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		return err
	}

	// 删除缓存
	r.userCache.DeleteUser(ctx, user.ID)

	return nil
}

func (r *userRepo) UpdateUserStats(ctx context.Context, userID int64, stats *biz.UserStats) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if stats.FollowCountDelta != 0 {
		updates["follow_count"] = gorm.Expr("follow_count + ?", stats.FollowCountDelta)
	}
	if stats.FollowerCountDelta != 0 {
		updates["follower_count"] = gorm.Expr("follower_count + ?", stats.FollowerCountDelta)
	}
	if stats.WorkCountDelta != 0 {
		updates["work_count"] = gorm.Expr("work_count + ?", stats.WorkCountDelta)
	}
	if stats.FavoriteCountDelta != 0 {
		updates["favorite_count"] = gorm.Expr("favorite_count + ?", stats.FavoriteCountDelta)
	}
	if stats.TotalFavoritedDelta != 0 {
		updates["total_favorited"] = gorm.Expr("total_favorited + ?", stats.TotalFavoritedDelta)
	}

	if err := r.data.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return err
	}

	// 删除缓存
	r.userCache.DeleteUser(ctx, userID)

	return nil
}

func (r *userRepo) VerifyPassword(ctx context.Context, username, password string) (*biz.User, error) {
	var u User
	if err := r.data.db.WithContext(ctx).Where("username = ? AND status = 1", username).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}

	// 验证密码
	isValid, err := r.passwordMgr.VerifyPassword(password, u.PasswordHash, u.Salt)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, biz.ErrPasswordError
	}

	return r.convertToUser(&u), nil
}

// convertToUser 转换为业务模型
func (r *userRepo) convertToUser(u *User) *biz.User {
	return &biz.User{
		ID:              u.ID,
		Username:        u.Username,
		PasswordHash:    u.PasswordHash,
		Salt:            u.Salt,
		Nickname:        u.Nickname,
		Avatar:          u.Avatar,
		BackgroundImage: u.BackgroundImage,
		Signature:       u.Signature,
		FollowCount:     u.FollowCount,
		FollowerCount:   u.FollowerCount,
		TotalFavorited:  u.TotalFavorited,
		WorkCount:       u.WorkCount,
		FavoriteCount:   u.FavoriteCount,
		LastLoginAt:     u.LastLoginAt,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
}
