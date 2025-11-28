package infra

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Infrastructure holds all infrastructure components (DB, Redis, Logger)
type Infrastructure struct {
	DB     *gorm.DB
	Redis  *redis.Client
	Logger Logger
}

// NewInfrastructure initializes and returns all infrastructure components
// This includes database, Redis, and logger
func NewInfrastructure(cfg *Config) (*Infrastructure, error) {
	logger := GetLogger()
	logger.Info("Initializing infrastructure components", nil)

	// Initialize database connection
	db, err := InitDatabase(cfg.Database)
	if err != nil {
		return nil, err
	}

	// Initialize Redis connection
	redisClient, err := InitRedis(cfg.Redis)
	if err != nil {
		return nil, err
	}

	logger.Info("Infrastructure initialized successfully", map[string]interface{}{
		"database_initialized": true,
		"redis_initialized":    true,
	})

	infra := &Infrastructure{
		DB:     db,
		Redis:  redisClient,
		Logger: GetLogger(),
	}

	return infra, nil
}

// Close gracefully closes all infrastructure connections
func (infra *Infrastructure) Close() {
	if infra.Redis != nil {
		CloseRedis(infra.Redis)
	}
	if infra.DB != nil {
		CloseDatabase(infra.DB)
	}
	infra.Logger.Info("Infrastructure connections closed", nil)
}
