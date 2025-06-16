package testutils

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDB 测试数据库配置
type TestDB struct {
	DB     *gorm.DB
	SqlDB  *sql.DB
	config *DBConfig
}

// DBConfig 数据库配置
type DBConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

// NewTestDB 创建测试数据库连接
func NewTestDB() (*TestDB, error) {
	config := &DBConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "tiktok",
		Password: "tiktok123",
		Database: "tiktok",
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &TestDB{
		DB:     db,
		SqlDB:  sqlDB,
		config: config,
	}, nil
}

// Close 关闭数据库连接
func (tdb *TestDB) Close() error {
	return tdb.SqlDB.Close()
}

// TruncateTable 清空表数据
func (tdb *TestDB) TruncateTable(tableName string) error {
	return tdb.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s", tableName)).Error
}

// TruncateAllTables 清空所有测试相关表
func (tdb *TestDB) TruncateAllTables() error {
	tables := []string{
		"user_sessions",
		"token_blacklist",
		"user_roles",
		"role_permissions",
		"user_follows",
		"users",
		"roles",
		"permissions",
	}

	// 禁用外键检查
	tdb.DB.Exec("SET FOREIGN_KEY_CHECKS = 0")

	for _, table := range tables {
		if err := tdb.TruncateTable(table); err != nil {
			// 继续清理其他表，不因单个表失败而停止
			continue
		}
	}

	// 重新启用外键检查
	tdb.DB.Exec("SET FOREIGN_KEY_CHECKS = 1")

	return nil
}

// ExecSQL 执行原生SQL
func (tdb *TestDB) ExecSQL(sql string, args ...interface{}) error {
	return tdb.DB.Exec(sql, args...).Error
}

// QueryRow 查询单行
func (tdb *TestDB) QueryRow(sql string, args ...interface{}) *sql.Row {
	return tdb.SqlDB.QueryRow(sql, args...)
}

// IsTableExists 检查表是否存在
func (tdb *TestDB) IsTableExists(tableName string) bool {
	var count int64
	err := tdb.DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?",
		tdb.config.Database, tableName).Scan(&count).Error
	return err == nil && count > 0
}

// GetTableRowCount 获取表行数
func (tdb *TestDB) GetTableRowCount(tableName string) (int64, error) {
	var count int64
	err := tdb.DB.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count).Error
	return count, err
}
