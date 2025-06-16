package testutils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
)

// TestUser 测试用户数据
type TestUser struct {
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
	Status          int8
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TestRole 测试角色数据
type TestRole struct {
	ID          int64
	Name        string
	Description string
	Status      int8
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TestPermission 测试权限数据
type TestPermission struct {
	ID          int64
	Name        string
	Resource    string
	Action      string
	Description string
	Status      int8
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TestDataManager 测试数据管理器
type TestDataManager struct {
	db    *TestDB
	redis *TestRedis
}

// NewTestDataManager 创建测试数据管理器
func NewTestDataManager(db *TestDB, redis *TestRedis) *TestDataManager {
	return &TestDataManager{
		db:    db,
		redis: redis,
	}
}

// CreateTestUsers 创建测试用户
func (tdm *TestDataManager) CreateTestUsers(count int) ([]*TestUser, error) {
	users := make([]*TestUser, 0, count)

	for i := 0; i < count; i++ {
		hash, salt, err := hashPassword(fmt.Sprintf("password%d", i+1))
		if err != nil {
			return nil, err
		}

		user := &TestUser{
			Username:        fmt.Sprintf("testuser%d", i+1),
			PasswordHash:    hash,
			Salt:            salt,
			Nickname:        fmt.Sprintf("Test User %d", i+1),
			Avatar:          "https://example.com/avatar.jpg",
			BackgroundImage: "https://example.com/bg.jpg",
			Signature:       fmt.Sprintf("I am test user %d", i+1),
			FollowCount:     0,
			FollowerCount:   0,
			TotalFavorited:  0,
			WorkCount:       0,
			FavoriteCount:   0,
			Status:          1,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err = tdm.db.DB.Create(user).Error
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

// CreateTestRoles 创建测试角色
func (tdm *TestDataManager) CreateTestRoles() ([]*TestRole, error) {
	roles := []*TestRole{
		{
			Name:        "user",
			Description: "Regular user",
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "admin",
			Description: "Administrator",
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "moderator",
			Description: "Content moderator",
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, role := range roles {
		err := tdm.db.DB.Create(role).Error
		if err != nil {
			return nil, err
		}
	}

	return roles, nil
}

// CreateTestPermissions 创建测试权限
func (tdm *TestDataManager) CreateTestPermissions() ([]*TestPermission, error) {
	permissions := []*TestPermission{
		{
			Name:        "user:read",
			Resource:    "/user",
			Action:      "GET",
			Description: "Read user information",
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "user:update",
			Resource:    "/user",
			Action:      "PUT",
			Description: "Update user information",
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "video:create",
			Resource:    "/video",
			Action:      "POST",
			Description: "Create video",
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "admin:all",
			Resource:    "/*",
			Action:      "*",
			Description: "Administrator full access",
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, perm := range permissions {
		err := tdm.db.DB.Create(perm).Error
		if err != nil {
			return nil, err
		}
	}

	return permissions, nil
}

// AssignRoleToUser 给用户分配角色
func (tdm *TestDataManager) AssignRoleToUser(userID, roleID int64) error {
	userRole := map[string]interface{}{
		"user_id":    userID,
		"role_id":    roleID,
		"created_at": time.Now(),
	}

	return tdm.db.DB.Table("user_roles").Create(userRole).Error
}

// CreateFollowRelation 创建关注关系
func (tdm *TestDataManager) CreateFollowRelation(userID, followUserID int64) error {
	follow := map[string]interface{}{
		"user_id":        userID,
		"follow_user_id": followUserID,
		"created_at":     time.Now(),
	}

	return tdm.db.DB.Table("user_follows").Create(follow).Error
}

// SetTestCache 设置测试缓存
func (tdm *TestDataManager) SetTestCache(key string, value interface{}, expiration time.Duration) error {
	return tdm.redis.Set(key, value, expiration)
}

// hashPassword 加密密码
func hashPassword(password string) (string, string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	saltStr := base64.RawStdEncoding.EncodeToString(salt)
	hashStr := base64.RawStdEncoding.EncodeToString(hash)

	return hashStr, saltStr, nil
}
