package controllers

import (
	"news-inshorts/src/infra"
	"news-inshorts/src/services"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Controllers holds all controller instances
type Controllers struct {
	Article         *ArticleController
	UserInteraction *UserInteractionController
	Services        *services.Services
}

// NewControllers creates and returns all controller instances
func NewControllers(
	cfg *infra.Config,
	db *gorm.DB,
	redisClient *redis.Client,
) *Controllers {
	svcs := services.NewServices(cfg, db, redisClient)

	return &Controllers{
		Article:         NewArticleController(svcs.Article, svcs.Repos.Article),
		UserInteraction: NewUserInteractionController(svcs.Repos.UserEvent),
		Services:        svcs,
	}
}
