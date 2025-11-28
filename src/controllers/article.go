package controllers

import (
	"time"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
	"news-inshorts/src/services"
	"news-inshorts/src/types"

	"github.com/gofiber/fiber/v2"
)

// ArticleController handles news-related HTTP requests
type ArticleController struct {
	articleService services.ArticleService
	articleRepo    repositories.ArticleRepository
	logger         infra.Logger
}

// NewArticleController creates a new instance of ArticleController
func NewArticleController(articleService services.ArticleService, articleRepo repositories.ArticleRepository) *ArticleController {
	return &ArticleController{
		articleService: articleService,
		articleRepo:    articleRepo,
		logger:         infra.GetLogger(),
	}
}

// QueryArticles handles GET /api/v1/news/query
func (ac *ArticleController) QueryArticles(c *fiber.Ctx) error {
	var req types.QueryArticlesRequest

	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	articles, err := ac.articleService.ProcessArticleQuery(req.Query, req.Location)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process query",
		})
	}

	response := types.QueryArticlesResponse{
		Articles: articles,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetTrending handles GET /api/v1/news/trending
func (ac *ArticleController) GetTrending(c *fiber.Ctx) error {
	lat := c.QueryFloat("lat", 0)
	lon := c.QueryFloat("lon", 0)
	limit := c.QueryInt("limit", 10)

	if lat < -90 || lat > 90 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Latitude must be between -90 and 90",
		})
	}

	if lon < -180 || lon > 180 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Longitude must be between -180 and 180",
		})
	}

	if limit <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Limit must be greater than 0",
		})
	}

	if limit > 100 {
		limit = 100
	}

	articles, err := ac.articleService.GetTrendingNews(lat, lon, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve trending news",
		})
	}

	response := types.QueryArticlesResponse{
		Articles: articles,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// FilterArticles handles GET /api/v1/news/filter
func (ac *ArticleController) FilterArticles(c *fiber.Ctx) error {
	var req types.FilterArticlesRequest

	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	articles, err := ac.articleService.FilterArticles(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to filter articles",
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.FilterArticlesResponse{
		Articles: articles,
	})
}

// LoadData handles POST /api/v1/news/load
func (ac *ArticleController) LoadData(c *fiber.Ctx) error {
	var req types.LoadDataRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Filepath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Filepath field is required",
		})
	}

	stats, err := ac.articleService.LoadFromJSON(req.Filepath)
	if err != nil {
		if stats != nil && len(stats.ValidationErrors) > 0 {
			response := types.LoadDataResponse{
				Success:          false,
				Message:          "Validation failed",
				TotalArticles:    stats.TotalArticles,
				SuccessCount:     stats.SuccessCount,
				ErrorCount:       stats.ErrorCount,
				ValidationErrors: stats.ValidationErrors,
			}
			return c.Status(fiber.StatusBadRequest).JSON(response)
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := types.LoadDataResponse{
		Success:       true,
		Message:       "Data loaded successfully",
		TotalArticles: stats.TotalArticles,
		SuccessCount:  stats.SuccessCount,
		ErrorCount:    stats.ErrorCount,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// CreateArticle handles POST /api/v1/news
func (ac *ArticleController) CreateArticle(c *fiber.Ctx) error {
	var req types.CreateArticleRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	publicationDate, err := time.Parse("2006-01-02T15:04:05", req.PublicationDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid publication_date format. Expected format: 2006-01-02T15:04:05",
		})
	}

	article := &models.Article{
		Title:           req.Title,
		Description:     req.Description,
		URL:             req.URL,
		PublicationDate: publicationDate,
		SourceName:      req.SourceName,
		Category:        req.Category,
		RelevanceScore:  req.RelevanceScore,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		Summary:         req.Summary,
	}

	if err := ac.articleService.CreateArticle(article); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create article",
		})
	}

	response := types.CreateArticleResponse{
		Success: true,
		Message: "Article created successfully",
		Article: *article,
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}
