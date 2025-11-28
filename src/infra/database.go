package infra

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDatabase initializes the database connection using GORM
func InitDatabase(cfg DatabaseConfig) (*gorm.DB, error) {
	log := GetLogger()

	// Configure GORM logger - default to Warn level
	// GORM logging can be configured separately if needed
	gormLogLevel := logger.Warn

	// Create GORM config
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	}

	// Open database connection
	db, err := gorm.Open(postgres.Open(cfg.URL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool settings
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Database connection established", map[string]interface{}{
		"max_open_conns":     cfg.MaxOpenConns,
		"max_idle_conns":     cfg.MaxIdleConns,
		"conn_max_lifetime":  cfg.ConnMaxLifetime,
		"conn_max_idle_time": cfg.ConnMaxIdleTime,
	})

	return db, nil
}

// CloseDatabase closes the database connection
func CloseDatabase(db *gorm.DB) {
	if db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
			log := GetLogger()
			log.Info("Database connection closed", nil)
		}
	}
}
