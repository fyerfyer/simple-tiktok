package data

import (
	"context"
	"fmt"
	"time"

	"go-backend/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username        string    `gorm:"uniqueIndex;size:32;not null" json:"username"`
	PasswordHash    string    `gorm:"size:128;not null" json:"-"`
	Salt            string    `gorm:"size:32;not null" json:"-"`
	Nickname        string    `gorm:"size:50" json:"nickname"`
	Avatar          string    `gorm:"size:255" json:"avatar"`
	BackgroundImage string    `gorm:"size:255" json:"background_image"`
	Signature       string    `gorm:"size:200" json:"signature"`
	FollowCount     int       `gorm:"default:0" json:"follow_count"`
	FollowerCount   int       `gorm:"default:0" json:"follower_count"`
	TotalFavorited  int64     `gorm:"default:0" json:"total_favorited"`
	WorkCount       int       `gorm:"default:0" json:"work_count"`
	FavoriteCount   int       `gorm:"default:0" json:"favorite_count"`
	Status          int8      `gorm:"default:1" json:"status"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

type userRepo struct {
	data *Data
	log  *log.Helper
}

// NewUserRepo .
func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *userRepo) CreateUser(ctx context.Context, user *biz.User) (*biz.User, error) {
	u := &User{
		Username:        user.Username,
		PasswordHash:    user.PasswordHash,
		Salt:            user.Salt,
		Nickname:        user.Nickname,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Signature:       user.Signature,
	}

	if err := r.data.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, err
	}

	// 清除缓存
	r.clearUserCache(ctx, u.ID, u.Username)

	return r.convertToUser(u), nil
}

func (r *userRepo) GetUser(ctx context.Context, userID int64) (*biz.User, error) {
	// 先从缓存获取
	if user := r.getUserFromCache(ctx, userID); user != nil {
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
	r.setUserCache(ctx, user)

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

	return r.convertToUser(&u), nil
}

func (r *userRepo) GetUsers(ctx context.Context, userIDs []int64) ([]*biz.User, error) {
	var users []User
	if err := r.data.db.WithContext(ctx).Where("id IN ? AND status = 1", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.User, 0, len(users))
	for _, u := range users {
		result = append(result, r.convertToUser(&u))
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

	if err := r.data.db.WithContext(ctx).Model(&User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		return err
	}

	// 清除缓存
	r.clearUserCache(ctx, user.ID, user.Username)

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

	// 清除缓存
	r.clearUserCache(ctx, userID, "")

	return nil
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
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
}

// 缓存相关方法
func (r *userRepo) getUserCache(ctx context.Context, userID int64) string {
	return fmt.Sprintf("user:%d", userID)
}

func (r *userRepo) getUserFromCache(ctx context.Context, userID int64) *biz.User {
	// 这里简化处理，实际项目中需要序列化/反序列化
	return nil
}

func (r *userRepo) setUserCache(ctx context.Context, user *biz.User) {
	key := r.getUserCache(ctx, user.ID)
	// 这里简化处理，实际项目中需要序列化
	r.data.rdb.Set(ctx, key, "cached", 30*time.Minute)
}

func (r *userRepo) clearUserCache(ctx context.Context, userID int64, username string) {
	keys := []string{r.getUserCache(ctx, userID)}
	if username != "" {
		keys = append(keys, fmt.Sprintf("user:username:%s", username))
	}
	r.data.rdb.Del(ctx, keys...)
}
