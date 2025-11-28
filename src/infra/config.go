package infra

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	LLM      LLMConfig
	Cache    CacheConfig
	Redis    RedisConfig
	Log      LogConfig
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// ServerConfig holds server settings
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// LLMConfig holds LLM API settings
type LLMConfig struct {
	APIKey string
	APIURL string
}

// CacheConfig holds cache settings
type CacheConfig struct {
	TTL time.Duration
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
}

// LogConfig holds logging settings
type LogConfig struct {
	Level string
}

// Load loads configuration from .env file and environment variables
// Environment variables take precedence over .env file values
func Load() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	// This allows the app to work with just environment variables
	_ = godotenv.Load()

	cfg := &Config{
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		},
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		},
		LLM: LLMConfig{
			APIKey: getEnv("LLM_API_KEY", ""),
			APIURL: getEnv("LLM_API_URL", "https://api.openai.com/v1"),
		},
		Cache: CacheConfig{
			TTL: getEnvAsDuration("CACHE_TTL", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnvAsInt("REDIS_PORT", 6379),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvAsInt("REDIS_DB", 0),
			DialTimeout:  getEnvAsDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getEnvAsDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getEnvAsDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsDuration retrieves an environment variable as a duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate required fields
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.LLM.APIKey == "" {
		return fmt.Errorf("LLM_API_KEY is required")
	}

	if c.LLM.APIURL == "" {
		return fmt.Errorf("LLM_API_URL is required")
	}

	// Validate database connection pool settings
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be greater than 0")
	}

	if c.Database.MaxIdleConns <= 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must be greater than 0")
	}

	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS cannot be greater than DB_MAX_OPEN_CONNS")
	}

	// Validate server settings
	if c.Server.Port == "" {
		return fmt.Errorf("PORT is required")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Log.Level] {
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error")
	}

	// Validate cache TTL
	if c.Cache.TTL <= 0 {
		return fmt.Errorf("CACHE_TTL must be greater than 0")
	}

	return nil
}
