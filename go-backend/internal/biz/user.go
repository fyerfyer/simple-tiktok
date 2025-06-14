package biz

import (
	"context"
	"time"

	v1 "go-backend/api/common/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrUserNotFound is user not found.
	ErrUserNotFound  = errors.NotFound(v1.ErrorCode_USER_NOT_EXIST.String(), "user not found")
	ErrUserExist     = errors.BadRequest(v1.ErrorCode_USER_EXIST.String(), "user already exists")
	ErrPasswordError = errors.BadRequest(v1.ErrorCode_PASSWORD_ERROR.String(), "password error")
)

// User is a User model.
type User struct {
	ID              int64
	Username        string
	PasswordHash    string
	Salt            string
	Nickname        string
	Avatar          string
	BackgroundImage string
	Signature       string
	FollowCount     int
	FollowerCount   int
	TotalFavorited  int64
	WorkCount       int
	FavoriteCount   int
	IsFollow        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// UserStats 用户统计更新
type UserStats struct {
	FollowCountDelta    int
	FollowerCountDelta  int
	WorkCountDelta      int
	FavoriteCountDelta  int
	TotalFavoritedDelta int64
}

// UserRepo is a User repo.
type UserRepo interface {
	CreateUser(context.Context, *User) (*User, error)
	GetUser(context.Context, int64) (*User, error)
	GetUserByUsername(context.Context, string) (*User, error)
	GetUsers(context.Context, []int64) ([]*User, error)
	UpdateUser(context.Context, *User) error
	UpdateUserStats(context.Context, int64, *UserStats) error
}

// UserUsecase is a User usecase.
type UserUsecase struct {
	repo UserRepo
	log  *log.Helper
}

// NewUserUsecase new a User usecase.
func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{repo: repo, log: log.NewHelper(logger)}
}

// Register creates a User, and returns the new User.
func (uc *UserUsecase) Register(ctx context.Context, username, password string) (*User, error) {
	uc.log.WithContext(ctx).Infof("Register user: %s", username)

	// 检查用户是否已存在
	if _, err := uc.repo.GetUserByUsername(ctx, username); err == nil {
		return nil, ErrUserExist
	}

	// 创建用户（这里简化处理，实际需要密码加密）
	user := &User{
		Username:        username,
		Nickname:        username,
		Avatar:          "https://example.com/default-avatar.jpg",
		BackgroundImage: "https://example.com/default-bg.jpg",
		Signature:       "",
	}

	return uc.repo.CreateUser(ctx, user)
}

// Login authenticates a user.
func (uc *UserUsecase) Login(ctx context.Context, username, password string) (*User, error) {
	uc.log.WithContext(ctx).Infof("Login user: %s", username)

	user, err := uc.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 这里简化处理，实际需要验证密码
	// if !verifyPassword(password, user.PasswordHash, user.Salt) {
	//     return nil, ErrPasswordError
	// }

	return user, nil
}

// GetUser gets a user by ID.
func (uc *UserUsecase) GetUser(ctx context.Context, userID int64) (*User, error) {
	return uc.repo.GetUser(ctx, userID)
}

// GetUsers gets users by IDs.
func (uc *UserUsecase) GetUsers(ctx context.Context, userIDs []int64) ([]*User, error) {
	return uc.repo.GetUsers(ctx, userIDs)
}

// UpdateUser updates user info.
func (uc *UserUsecase) UpdateUser(ctx context.Context, user *User) error {
	uc.log.WithContext(ctx).Infof("Update user: %d", user.ID)
	return uc.repo.UpdateUser(ctx, user)
}

// GetUserByUsername gets a user by username.
func (uc *UserUsecase) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return uc.repo.GetUserByUsername(ctx, username)
}

// UpdateUserStats updates user statistics.
func (uc *UserUsecase) UpdateUserStats(ctx context.Context, userID int64, stats *UserStats) error {
	uc.log.WithContext(ctx).Infof("Update user stats: %d", userID)
	return uc.repo.UpdateUserStats(ctx, userID, stats)
}
