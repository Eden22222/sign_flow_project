package db

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sign_flow_project/internal/model"
	"strings"
	"sync"
	"time"
)

var (
	dbOnce sync.Once
	gormDB *gorm.DB
	dbErr  error
)

func PostgresSetup() (*gorm.DB, error) {
	dbOnce.Do(func() {

		//Todo： 这里先写死，后面再改成配置文件
		host := "localhost"
		port := "5432"
		user := "postgres"
		password := "123456"
		dbName := "signflow"
		sslmode := "disable"
		timeZone := "Asia/Shanghai"
		clientEncoding := "UTF8"
		loc, err := time.LoadLocation(timeZone)
		if err != nil {
			dbErr = fmt.Errorf("load location failed: %w", err)
			return
		}

		//TODO 数据库时间跟当地时间并不一致
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=%s client_encoding=%s",
			host, port, user, password, dbName, sslmode, timeZone, clientEncoding)

		gormConfig := &gorm.Config{
			Logger: logger.Default.LogMode(logger.Error),
			NowFunc: func() time.Time {
				// Ensure GORM auto timestamps use explicit business timezone.
				return time.Now().In(loc)
			},
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

		// 显式设置会话编码，避免客户端连接编码被环境覆盖。
		if err := gormDB.Exec("SET client_encoding TO 'UTF8'").Error; err != nil {
			dbErr = fmt.Errorf("set client encoding failed: %w", err)
			return
		}
		// Force session timezone to keep DB defaults and app writes consistent.
		if err := gormDB.Exec("SET TIME ZONE '" + timeZone + "'").Error; err != nil {
			dbErr = fmt.Errorf("set session timezone failed: %w", err)
			return
		}

		if err := ensureUTF8Database(gormDB); err != nil {
			dbErr = err
			return
		}

		if err := gormDB.AutoMigrate(
			&model.DocumentModel{},
			&model.DocumentFieldModel{},
			&model.TaskModel{},
			&model.WorkflowModel{},
			&model.WorkflowSignerModel{},
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

func ensureUTF8Database(db *gorm.DB) error {
	var encoding string
	if err := db.Raw(
		"SELECT pg_encoding_to_char(encoding) FROM pg_database WHERE datname = current_database()",
	).Scan(&encoding).Error; err != nil {
		return fmt.Errorf("check database encoding failed: %w", err)
	}

	if strings.ToUpper(encoding) != "UTF8" {
		return fmt.Errorf(
			"database encoding must be UTF8, current is %s; recreate database with UTF8, e.g. CREATE DATABASE signflow WITH ENCODING 'UTF8' TEMPLATE template0",
			encoding,
		)
	}

	return nil
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
