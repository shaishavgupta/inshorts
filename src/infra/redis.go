package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// InitRedis initializes the Redis client connection
func InitRedis(cfg RedisConfig) (*redis.Client, error) {
	log := GetLogger()

	// Parse Redis URL or use individual components
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	}

	// Create Redis client
	client := redis.NewClient(opts)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	log.Info("Redis connection established", map[string]interface{}{
		"host":         cfg.Host,
		"port":         cfg.Port,
		"db":           cfg.DB,
		"pool_size":    cfg.PoolSize,
		"min_idle":     cfg.MinIdleConns,
	})

	return client, nil
}

// CloseRedis closes the Redis client connection
func CloseRedis(client *redis.Client) {
	if client != nil {
		client.Close()
		log := GetLogger()
		log.Info("Redis connection closed", nil)
	}
}

