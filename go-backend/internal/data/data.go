package data

import (
	"go-backend/internal/biz"
	"go-backend/internal/conf"
	"go-backend/internal/data/cache"
	pkgcache "go-backend/pkg/cache"
	"go-backend/pkg/storage"
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
	NewVideoRepo,
	NewMinIOStorage,
	NewUserCache,
	NewAuthCache,
	NewVideoCache,
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

// NewMinIOStorage create MinIO storage
func NewMinIOStorage(c *conf.Data, logger log.Logger) (storage.VideoStorage, error) {
	config := &storage.MinIOConfig{
		Endpoint:   c.Minio.Endpoint,
		AccessKey:  c.Minio.AccessKey,
		SecretKey:  c.Minio.SecretKey,
		BucketName: c.Minio.BucketName,
		Region:     c.Minio.Region,
		UseSSL:     c.Minio.UseSsl,
		BaseURL:    c.Minio.BaseUrl,
	}

	return storage.NewMinIOStorage(config)
}

// NewVideoCache create video cache
func NewVideoCache(multiCache *pkgcache.MultiLevelCache, logger log.Logger) biz.VideoCacheRepo {
	return cache.NewVideoCache(multiCache, logger)
}
