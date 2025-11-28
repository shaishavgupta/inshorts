package controllers

import (
	"news-inshorts/src/infra"
	"news-inshorts/src/repositories"
	"news-inshorts/src/services"
	"news-inshorts/src/types"

	"github.com/gofiber/fiber/v2"
)

// NewsController handles news-related HTTP requests
type NewsController struct {
	newsService services.NewsService
	articleRepo repositories.ArticleRepository
	logger      infra.Logger
}

// NewNewsController creates a new instance of NewsController
func NewNewsController(newsService services.NewsService, articleRepo repositories.ArticleRepository) *NewsController {
	return &NewsController{
		newsService: newsService,
		articleRepo: articleRepo,
		logger:      infra.GetLogger(),
	}
}

// QueryNews handles POST /api/v1/news/query
// Processes natural language queries and returns relevant news articles
func (nc *NewsController) QueryNews(c *fiber.Ctx) error {
	var req types.QueryNewsRequest

	// Parse and validate request body
	if err := c.BodyParser(&req); err != nil {
		nc.logger.Error("Failed to parse request body", err, map[string]interface{}{
			"path": c.Path(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query field is required",
		})
	}

	// Validate location if provided
	if req.Location != nil {
		if req.Location.Latitude < -90 || req.Location.Latitude > 90 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Latitude must be between -90 and 90",
			})
		}
		if req.Location.Longitude < -180 || req.Location.Longitude > 180 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Longitude must be between -180 and 180",
			})
		}
	}

	nc.logger.Info("Processing news query request", map[string]interface{}{
		"query":        req.Query,
		"has_location": req.Location != nil,
	})

	// Call NewsService to process the query
	articles, err := nc.newsService.ProcessNewsQuery(req.Query, req.Location)
	if err != nil {
		nc.logger.Error("Failed to process news query", err, map[string]interface{}{
			"query": req.Query,
		})

		// Return appropriate error status
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process query",
		})
	}

	// Format and return JSON response
	response := types.QueryNewsResponse{
		Articles: articles,
	}

	nc.logger.Info("News query processed successfully", map[string]interface{}{
		"query":         req.Query,
		"article_count": len(articles),
	})

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetTrending handles GET /api/v1/news/trending
// Returns trending news articles based on location
func (nc *NewsController) GetTrending(c *fiber.Ctx) error {
	// Parse and validate query parameters
	lat := c.QueryFloat("lat", 0)
	lon := c.QueryFloat("lon", 0)
	limit := c.QueryInt("limit", 10)

	// Validate latitude
	if lat < -90 || lat > 90 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Latitude must be between -90 and 90",
		})
	}

	// Validate longitude
	if lon < -180 || lon > 180 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Longitude must be between -180 and 180",
		})
	}

	// Validate limit
	if limit <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Limit must be greater than 0",
		})
	}

	if limit > 100 {
		limit = 100 // Cap at 100 articles
	}

	nc.logger.Info("Processing trending news request", map[string]interface{}{
		"latitude":  lat,
		"longitude": lon,
		"limit":     limit,
	})

	// Call NewsService to get trending news
	articles, err := nc.newsService.GetTrendingNews(lat, lon, limit)
	if err != nil {
		nc.logger.Error("Failed to get trending news", err, map[string]interface{}{
			"latitude":  lat,
			"longitude": lon,
			"limit":     limit,
		})

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve trending news",
		})
	}

	// Format and return JSON response
	response := types.QueryNewsResponse{
		Articles: articles,
	}

	nc.logger.Info("Trending news retrieved successfully", map[string]interface{}{
		"latitude":      lat,
		"longitude":     lon,
		"limit":         limit,
		"article_count": len(articles),
	})

	return c.Status(fiber.StatusOK).JSON(response)
}

// FilterArticles handles GET /api/v1/news/filter
// Generic endpoint that filters articles by category, source, and/or location (nearby)
// Supports multiple query parameters that can be combined
func (nc *NewsController) FilterArticles(c *fiber.Ctx) error {
	var req types.FilterArticlesRequest

	// Parse query parameters into struct
	if err := c.QueryParser(&req); err != nil {
		nc.logger.Error("Failed to parse query parameters", err, map[string]interface{}{
			"path": c.Path(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	// Validate request using struct validation
	if err := req.Validate(); err != nil {
		nc.logger.Error("Validation failed", err, map[string]interface{}{
			"path": c.Path(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Call NewsService to filter articles
	articles, err := nc.newsService.FilterArticles(req)
	if err != nil {
		nc.logger.Error("Failed to filter articles", err, map[string]interface{}{
			"query": req,
		})

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to filter articles",
		})
	}

	return c.Status(fiber.StatusOK).JSON(articles)
}

// LoadData handles POST /api/v1/news/load
// Loads news articles from a JSON file into the database with validation
func (nc *NewsController) LoadData(c *fiber.Ctx) error {
	var req types.LoadDataRequest

	// Parse and validate request body
	if err := c.BodyParser(&req); err != nil {
		nc.logger.Error("Failed to parse request body", err, map[string]interface{}{
			"path": c.Path(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Filepath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Filepath field is required",
		})
	}

	// Call ArticleRepository to load data
	stats, err := nc.articleRepo.LoadFromJSON(req.Filepath)
	if err != nil {
		nc.logger.Error("Failed to load data from JSON", err, map[string]interface{}{
			"filepath": req.Filepath,
		})

		// If we have stats with validation errors, return them
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

	nc.logger.Info("Data loaded successfully via API", map[string]interface{}{
		"filepath":      req.Filepath,
		"total":         stats.TotalArticles,
		"success_count": stats.SuccessCount,
		"error_count":   stats.ErrorCount,
	})

	// Return success response with statistics
	response := types.LoadDataResponse{
		Success:       true,
		Message:       "Data loaded successfully",
		TotalArticles: stats.TotalArticles,
		SuccessCount:  stats.SuccessCount,
		ErrorCount:    stats.ErrorCount,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
