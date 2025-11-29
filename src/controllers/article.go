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
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "INVALID_QUERY_PARAMS",
			Error:     "Invalid query parameters",
		})
	}

	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "VALIDATION_ERROR",
			Error:     err.Error(),
		})
	}

	articles, err := ac.articleService.ProcessArticleQuery(req.Query, req.Location)
	if err != nil {
		ac.logger.Error("Failed to process article query", err, map[string]interface{}{
			"query":    req.Query,
			"location": req.Location,
		})
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			ErrorCode: "QUERY_PROCESSING_FAILED",
			Error:     "Failed to process query",
		})
	}

	response := types.QueryArticlesResponse{
		Articles: articles,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetTrending handles GET /api/v1/news/trending
func (ac *ArticleController) GetTrending(c *fiber.Ctx) error {
	var req types.GetTrendingRequest

	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "INVALID_QUERY_PARAMS",
			Error:     "Invalid query parameters",
		})
	}

	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "VALIDATION_ERROR",
			Error:     err.Error(),
		})
	}

	articles, err := ac.articleService.GetTrendingNews(req.Lat, req.Lon, req.Limit)
	if err != nil {
		ac.logger.Error("Failed to retrieve trending news", err, map[string]interface{}{
			"lat":   req.Lat,
			"lon":   req.Lon,
			"limit": req.Limit,
		})
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			ErrorCode: "TRENDING_NEWS_FAILED",
			Error:     "Failed to retrieve trending news",
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
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "INVALID_QUERY_PARAMS",
			Error:     "Invalid query parameters",
		})
	}

	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "VALIDATION_ERROR",
			Error:     err.Error(),
		})
	}

	articles, err := ac.articleService.FilterArticles(req)
	if err != nil {
		ac.logger.Error("Failed to filter articles", err, map[string]interface{}{
			"filters": req,
		})
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			ErrorCode: "FILTER_ARTICLES_FAILED",
			Error:     "Failed to filter articles",
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
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "INVALID_REQUEST_BODY",
			Error:     "Invalid request body",
		})
	}

	if req.Filepath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "FILEPATH_REQUIRED",
			Error:     "Filepath field is required",
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

		ac.logger.Error("Failed to load data from file", err, map[string]interface{}{
			"filepath": req.Filepath,
		})
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			ErrorCode: "DATA_LOAD_FAILED",
			Error:     "Failed to load data from file",
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
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "INVALID_REQUEST_BODY",
			Error:     "Invalid request body",
		})
	}

	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "VALIDATION_ERROR",
			Error:     err.Error(),
		})
	}

	publicationDate, err := time.Parse("2006-01-02T15:04:05", req.PublicationDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			ErrorCode: "INVALID_DATE_FORMAT",
			Error:     "Invalid publication_date format. Expected format: 2006-01-02T15:04:05",
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
		ac.logger.Error("Failed to create article", err, map[string]interface{}{
			"title":  req.Title,
			"source": req.SourceName,
			"url":    req.URL,
		})
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			ErrorCode: "ARTICLE_CREATION_FAILED",
			Error:     "Failed to create article",
		})
	}

	response := types.CreateArticleResponse{
		Success: true,
		Message: "Article created successfully",
		Article: *article,
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}
