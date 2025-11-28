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

// QueryNews handles GET /api/v1/news/query
// Processes natural language queries and returns relevant news articles
func (nc *NewsController) QueryNews(c *fiber.Ctx) error {
	var req types.QueryNewsRequest

	// Parse and validate query parameters
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	// Validate request using struct validation
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Call NewsService to process the query
	articles, err := nc.newsService.ProcessNewsQuery(req.Query, req.Location)
	if err != nil {
		// Return appropriate error status
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process query",
		})
	}

	// Format and return JSON response
	response := types.QueryNewsResponse{
		Articles: articles,
	}

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

	// Call NewsService to get trending news
	articles, err := nc.newsService.GetTrendingNews(lat, lon, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve trending news",
		})
	}

	// Format and return JSON response
	response := types.QueryNewsResponse{
		Articles: articles,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// FilterArticles handles GET /api/v1/news/filter
// Generic endpoint that filters articles by category, source, and/or location (nearby)
// Supports multiple query parameters that can be combined
func (nc *NewsController) FilterArticles(c *fiber.Ctx) error {
	var req types.FilterArticlesRequest

	// Parse query parameters into struct
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	// Validate request using struct validation
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Call NewsService to filter articles
	articles, err := nc.newsService.FilterArticles(req)
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
// Loads news articles from a JSON file into the database with validation
func (nc *NewsController) LoadData(c *fiber.Ctx) error {
	var req types.LoadDataRequest

	// Parse and validate request body
	if err := c.BodyParser(&req); err != nil {
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

	// Call NewsService to load data
	stats, err := nc.newsService.LoadFromJSON(req.Filepath)
	if err != nil {
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

// CreateArticle handles POST /api/v1/news
// Creates a new article from the request payload
func (nc *NewsController) CreateArticle(c *fiber.Ctx) error {
	var req types.CreateArticleRequest

	// Parse and validate request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request using struct validation
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Parse publication date
	publicationDate, err := time.Parse("2006-01-02T15:04:05", req.PublicationDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid publication_date format. Expected format: 2006-01-02T15:04:05",
		})
	}

	// Create article model from request
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

	// Call NewsService to create the article
	if err := nc.newsService.CreateArticle(article); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create article",
		})
	}

	// Return success response with created article
	response := types.CreateArticleResponse{
		Success: true,
		Message: "Article created successfully",
		Article: *article,
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}
