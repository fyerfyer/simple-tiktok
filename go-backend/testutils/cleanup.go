package testutils

import (
	"context"
	"time"
)

// TestEnv 测试环境
type TestEnv struct {
	DB          *TestDB
	Redis       *TestRedis
	DataManager *TestDataManager
}

// NewTestEnv 创建测试环境
func NewTestEnv() (*TestEnv, error) {
	db, err := NewTestDB()
	if err != nil {
		return nil, err
	}

	redis, err := NewTestRedis()
	if err != nil {
		db.Close()
		return nil, err
	}

	dataManager := NewTestDataManager(db, redis)

	return &TestEnv{
		DB:          db,
		Redis:       redis,
		DataManager: dataManager,
	}, nil
}

// Setup 设置测试环境
func (te *TestEnv) Setup() error {
	// 清理数据
	if err := te.Cleanup(); err != nil {
		return err
	}

	// 不再初始化基础数据，让每个测试自己创建需要的数据
	return nil
}

// Cleanup 清理测试环境
func (te *TestEnv) Cleanup() error {
	// 清理数据库
	te.DB.TruncateAllTables()

	// 清理Redis
	te.Redis.FlushDB()

	return nil
}

// Close 关闭测试环境
func (te *TestEnv) Close() {
	if te.DB != nil {
		te.DB.Close()
	}
	if te.Redis != nil {
		te.Redis.Close()
	}
}

// CleanupFunc 清理函数类型
type CleanupFunc func()

// SetupTestWithCleanup 设置测试并返回清理函数
func SetupTestWithCleanup() (*TestEnv, CleanupFunc, error) {
	env, err := NewTestEnv()
	if err != nil {
		return nil, nil, err
	}

	if err := env.Setup(); err != nil {
		env.Close()
		return nil, nil, err
	}

	cleanup := func() {
		env.Cleanup()
		env.Close()
	}

	return env, cleanup, nil
}

// WithTestEnv 使用测试环境执行函数
func WithTestEnv(fn func(*TestEnv) error) error {
	env, cleanup, err := SetupTestWithCleanup()
	if err != nil {
		return err
	}
	defer cleanup()

	return fn(env)
}

// WithTestDB 仅使用数据库执行函数
func WithTestDB(fn func(*TestDB) error) error {
	db, err := NewTestDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// 清理数据
	db.TruncateAllTables()

	return fn(db)
}

// WithTestRedis 仅使用Redis执行函数
func WithTestRedis(fn func(*TestRedis) error) error {
	redis, err := NewTestRedis()
	if err != nil {
		return err
	}
	defer redis.Close()

	// 清理数据
	redis.FlushDB()

	return fn(redis)
}

// ContextWithTimeout 创建带超时的上下文
func ContextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}
