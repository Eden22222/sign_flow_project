package db

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sign_flow_project/interval/model"
	"sync"
	"time"
)

var (
	dbOnce sync.Once
	gormDB *gorm.DB
	dbErr  error
)

func InitPostgres() (*gorm.DB, error) {
	dbOnce.Do(func() {

		//Todo： 这里先写死，后面再改成配置文件
		host := "localhost"
		port := "5432"
		user := "postgres"
		password := "123456"
		dbName := "signflow"
		sslmode := "disable"
		timeZone := "Asia/Shanghai"

		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
			host, port, user, password, dbName, sslmode, timeZone)

		gormConfig := &gorm.Config{
			Logger: logger.Default.LogMode(logger.Error),
		}

		gormDB, dbErr = gorm.Open(postgres.Open(dsn), gormConfig)
		if dbErr != nil {
			return
		}

		sqlDB, err := gormDB.DB()
		if err != nil {
			dbErr = fmt.Errorf("get underlying sql.DB failed: %w", err)
			return
		}

		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)

		if err := sqlDB.Ping(); err != nil {
			dbErr = fmt.Errorf("ping underlying DB failed: %w", err)
			return
		}

		if err := gormDB.AutoMigrate(
			&model.DocumentModel{},
			&model.TaskModel{},
			&model.WorkflowModel{},
		); err != nil {
			dbErr = fmt.Errorf("auto migrate models failed: %w", err)
			return
		}
	})
	return gormDB, dbErr
}

func GetPostgres() *gorm.DB {
	return gormDB
}

// CheckDatabaseHealth 检查数据库健康状态
func CheckDatabaseHealth() error {
	if gormDB == nil {
		return fmt.Errorf("database is not initialized")
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB failed: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// GetDatabaseStats 获取连接池状态
func GetDatabaseStats() (map[string]interface{}, error) {
	if gormDB == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB failed: %w", err)
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"OpenConnections":   stats.OpenConnections,
		"InUse":             stats.InUse,
		"Idle":              stats.Idle,
		"WaitCount":         stats.WaitCount,
		"WaitDuration":      stats.WaitDuration.String(),
		"MaxIdleClosed":     stats.MaxIdleClosed,
		"MaxLifetimeClosed": stats.MaxLifetimeClosed,
	}, nil
}
