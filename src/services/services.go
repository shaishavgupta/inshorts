package services

import (
	"news-inshorts/src/infra"
	"news-inshorts/src/repositories"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Services holds all service instances
type Services struct {
	LLM         LLMService
	Trending    TrendingService
	Article     ArticleService
	FilterChain *FilterChain
	Repos       *repositories.Repositories
}

// NewServices creates and returns all service instances
func NewServices(
	cfg *infra.Config,
	db *gorm.DB,
	redisClient *redis.Client,
) *Services {
	// Initialize repositories
	repos := repositories.NewRepositories(db)
	infra.GetLogger().Info("Repositories initialized", nil)

	// Initialize LLM service
	llmService := NewLLMService(&cfg.LLM)

	// Initialize filter chain with all filters
	filterChain := NewFilterChain(repos.Article)

	// Initialize trending service
	trendingService := NewTrendingService(repos.UserEvent, redisClient, cfg.Cache.TTL)

	// Initialize news service
	newsService := NewArticleService(llmService, filterChain, trendingService, repos.Article)

	return &Services{
		LLM:         llmService,
		Trending:    trendingService,
		Article:     newsService,
		FilterChain: filterChain,
		Repos:       repos,
	}
}
