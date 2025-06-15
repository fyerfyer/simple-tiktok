package data

import (
	"go-backend/internal/conf"
	"go-backend/internal/data/cache"
	pkgcache "go-backend/pkg/cache"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewUserRepo,
	NewRelationRepo,
	NewRoleRepo,
	NewPermissionRepo,
	NewSessionRepo,
	NewUserCache,
	NewAuthCache,
	NewMultiLevelCache,
)

// Data .
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(logger)

	// 初始化MySQL
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}

	sqlDB.SetMaxIdleConns(int(c.Database.MaxIdleConns))
	sqlDB.SetMaxOpenConns(int(c.Database.MaxOpenConns))
	sqlDB.SetConnMaxLifetime(c.Database.ConnMaxLifetime.AsDuration())

	// 初始化Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           int(c.Redis.Db),
		DialTimeout:  c.Redis.DialTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		PoolSize:     int(c.Redis.PoolSize),
	})

	d := &Data{
		db:  db,
		rdb: rdb,
	}

	cleanup := func() {
		helper.Info("closing the data resources")
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		rdb.Close()
	}

	return d, cleanup, nil
}

// NewMultiLevelCache create multilevel cache
func NewMultiLevelCache(data *Data) *pkgcache.MultiLevelCache {
	config := &pkgcache.CacheConfig{
		LocalTTL: 5 * time.Minute,
		RedisTTL: 30 * time.Minute,
		EnableL1: true,
		EnableL2: true,
	}
	return pkgcache.NewMultiLevelCache(data.rdb, config)
}

// NewUserCache create user cache
func NewUserCache(multiCache *pkgcache.MultiLevelCache, logger log.Logger) *cache.UserCache {
	return cache.NewUserCache(multiCache, logger)
}

// NewAuthCache create auth cache
func NewAuthCache(multiCache *pkgcache.MultiLevelCache, logger log.Logger) *cache.AuthCache {
	return cache.NewAuthCache(multiCache, logger)
}
