package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewMySQL 创建 tk-user 的 MySQL 连接。
func NewMySQL(dsn string) (*gorm.DB, error) {
	// 论坛/用户服务统一使用 GORM + MySQL，日志级别保持 Warn 以兼顾可观测与性能。
	// DSN 来自 etc/user.yaml，便于本地与生产环境分离。
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		// 调用logger.Default.LogMode完成当前处理。
		Logger: logger.Default.LogMode(logger.Warn),
	})
}
