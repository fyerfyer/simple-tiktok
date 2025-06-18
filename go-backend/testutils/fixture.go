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

// TableName 指定表名
func (TestRole) TableName() string {
	return "roles"
}

func (TestUser) TableName() string {
	return "users"
}

func (TestPermission) TableName() string {
	return "permissions"
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
	// 检查现有用户数量
	var existingCount int64
	err := tdm.db.DB.Model(&TestUser{}).Count(&existingCount).Error
	if err != nil {
		return nil, err
	}

	users := make([]*TestUser, 0, count)

	for i := 0; i < count; i++ {
		hash, salt, err := hashPassword(fmt.Sprintf("password%d", i+1))
		if err != nil {
			return nil, err
		}

		username := fmt.Sprintf("testuser%d", i+1)

		user := &TestUser{
			Username:        username,
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
	// 检查现有角色数量
	var existingCount int64
	err := tdm.db.DB.Model(&TestRole{}).Count(&existingCount).Error
	if err != nil {
		return nil, err
	}

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
	// 检查现有权限数量
	var existingCount int64
	err := tdm.db.DB.Model(&TestPermission{}).Count(&existingCount).Error
	if err != nil {
		return nil, err
	}

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
			Name:        "video:read",
			Resource:    "/video",
			Action:      "GET",
			Description: "Read video",
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

	err := tdm.db.DB.Table("user_roles").Create(userRole).Error
	if err != nil {
		return err
	}

	return nil
}

// CreateFollowRelation 创建关注关系
func (tdm *TestDataManager) CreateFollowRelation(userID, followUserID int64) error {
	follow := map[string]interface{}{
		"user_id":        userID,
		"follow_user_id": followUserID,
		"created_at":     time.Now(),
	}

	err := tdm.db.DB.Table("user_follows").Create(follow).Error
	if err != nil {
		return err
	}

	// 清除相关缓存，确保缓存一致性
	if tdm.redis != nil {
		keys := []string{
			fmt.Sprintf("follow:%d:%d", userID, followUserID),
			fmt.Sprintf("follow:%d:*", userID),
			fmt.Sprintf("follow:*:%d", followUserID),
		}
		// 忽略缓存清理错误，因为这是测试环境
		tdm.redis.Del(keys...)
	}

	// 验证插入是否成功
	var count int64
	err = tdm.db.DB.Table("user_follows").
		Where("user_id = ? AND follow_user_id = ?", userID, followUserID).
		Count(&count).Error
	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("follow relation was not created successfully")
	}

	return nil
}

// SetTestCache 设置测试缓存
func (tdm *TestDataManager) SetTestCache(key string, value interface{}, expiration time.Duration) error {
	err := tdm.redis.Set(key, value, expiration)
	if err != nil {
		return err
	}

	return nil
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
