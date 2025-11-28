package services

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
	"news-inshorts/src/types"
)

// NewsService defines the interface for news operations
type NewsService interface {
	ProcessNewsQuery(query string, location *models.Location) ([]models.Article, error)
	GetTrendingNews(lat, lon float64, limit int) ([]models.Article, error)
	FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error)
	LoadFromJSON(filepath string) (*repositories.LoadStats, error)
	CreateArticle(article *models.Article) error
}

// newsService implements NewsService
type newsService struct {
	llmService      LLMService
	filterChain     *FilterChain
	trendingService TrendingService
	articleRepo     repositories.ArticleRepository
	logger          infra.Logger
}

// NewNewsService creates a new instance of NewsService
func NewNewsService(
	llmService LLMService,
	filterChain *FilterChain,
	trendingService TrendingService,
	articleRepo repositories.ArticleRepository,
) NewsService {
	return &newsService{
		llmService:      llmService,
		filterChain:     filterChain,
		trendingService: trendingService,
		articleRepo:     articleRepo,
		logger:          infra.GetLogger(),
	}
}

// ProcessNewsQuery orchestrates LLM query analysis and filter chain execution
// to retrieve and enrich relevant news articles
func (s *newsService) ProcessNewsQuery(query string, location *models.Location) ([]models.Article, error) {
	// Step 1: Analyze query using LLM to extract intents and entities
	analysis, err := s.llmService.ProcessQuery(query)
	if err != nil {
		s.logger.Error("Failed to analyze query with LLM", err, map[string]interface{}{
			"query": query,
		})
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}

	// Step 2: Execute filter chain with extracted intents
	filteredArticles, err := s.filterChain.Execute(analysis.Intents, analysis.Entities, location)
	if err != nil {
		s.logger.Error("Failed to execute filter chain", err, nil)
		return nil, fmt.Errorf("failed to filter articles: %w", err)
	}

	// Step 3: Limit results to top 5 articles
	if len(filteredArticles) > 5 {
		filteredArticles = filteredArticles[:5]
	}

	return filteredArticles, nil
}

// GetTrendingNews retrieves trending articles based on location
// Implements caching, trending score computation, and article enrichment
func (s *newsService) GetTrendingNews(lat, lon float64, limit int) ([]models.Article, error) {
	s.logger.Info("Getting trending news", map[string]interface{}{
		"latitude":  lat,
		"longitude": lon,
		"limit":     limit,
	})

	location := models.Location{
		Latitude:  lat,
		Longitude: lon,
	}

	// Step 1: Check cache first
	cachedArticles, found := s.trendingService.GetCachedTrending(lat, lon, limit)
	if found {
		return cachedArticles, nil
	}

	// Step 2: Cache miss - retrieve all articles
	articles, err := s.articleRepo.FindAll()
	if err != nil {
		s.logger.Error("Failed to retrieve articles for trending", err, nil)
		return nil, fmt.Errorf("failed to retrieve articles: %w", err)
	}

	s.logger.Debug("Retrieved articles for trending computation", map[string]interface{}{
		"count": len(articles),
	})

	// Step 3: Compute trending scores for each article
	type articleWithScore struct {
		article models.Article
		score   float64
	}

	articlesWithScores := make([]articleWithScore, 0, len(articles))

	for _, article := range articles {
		score, err := s.trendingService.ComputeTrendingScore(article, location)
		if err != nil {
			s.logger.Warn("Failed to compute trending score for article", map[string]interface{}{
				"article_id": article.ID,
				"error":      err.Error(),
			})
			// Skip articles with score computation errors
			continue
		}

		articlesWithScores = append(articlesWithScores, articleWithScore{
			article: article,
			score:   score,
		})
	}

	// Step 4: Sort by trending score (highest first)
	sort.Slice(articlesWithScores, func(i, j int) bool {
		return articlesWithScores[i].score > articlesWithScores[j].score
	})

	s.logger.Debug("Sorted articles by trending score", map[string]interface{}{
		"total_scored": len(articlesWithScores),
	})

	// Step 5: Apply limit
	if len(articlesWithScores) > limit {
		articlesWithScores = articlesWithScores[:limit]
	}

	// Extract articles for caching
	trendingArticles := make([]models.Article, 0, len(articlesWithScores))
	for _, aws := range articlesWithScores {
		trendingArticles = append(trendingArticles, aws.article)
	}

	// Step 6: Cache results before returning
	s.trendingService.CacheTrending(lat, lon, trendingArticles)

	s.logger.Info("Computed and cached trending articles", map[string]interface{}{
		"count": len(trendingArticles),
	})

	return trendingArticles, nil
}

// FilterArticles dynamically filters articles based on provided parameters
// Supports filtering by category, source, and/or location (nearby)
// Multiple filters can be combined
func (s *newsService) FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error) {
	return s.articleRepo.FilterArticles(params)
}

// LoadFromJSON loads articles from a JSON file, enriches them with LLM summaries, and inserts them into the database
func (s *newsService) LoadFromJSON(filepath string) (*repositories.LoadStats, error) {
	s.logger.Info("Starting to load articles from JSON", map[string]interface{}{
		"filepath": filepath,
	})

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		s.logger.Error("JSON file not found", err, map[string]interface{}{
			"filepath": filepath,
		})
		return nil, fmt.Errorf("file not found: %s", filepath)
	}

	// Read the JSON file
	file, err := os.Open(filepath)
	if err != nil {
		s.logger.Error("Failed to open JSON file", err, map[string]interface{}{
			"filepath": filepath,
		})
		return nil, fmt.Errorf("failed to open JSON file: %w", err)
	}
	defer file.Close()

	// Parse JSON
	var articles []models.Article
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&articles); err != nil {
		s.logger.Error("Failed to decode JSON", err, map[string]interface{}{
			"filepath": filepath,
		})
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	if len(articles) == 0 {
		s.logger.Warn("No articles found in JSON file", map[string]interface{}{
			"filepath": filepath,
		})
		return &repositories.LoadStats{
			TotalArticles: 0,
		}, nil
	}

	s.logger.Info("Parsed articles from JSON", map[string]interface{}{
		"total": len(articles),
	})

	// Enrich each article with summary via LLM service
	s.logger.Info("Enriching articles with LLM summaries", map[string]interface{}{
		"total": len(articles),
	})

	for i := range articles {
		summary, err := s.llmService.GenerateSummary(articles[i].Title, articles[i].Description)
		if err != nil {
			s.logger.Warn("Failed to generate summary for article", map[string]interface{}{
				"index": i,
				"title": articles[i].Title,
				"error": err.Error(),
			})
			// Continue with empty summary if LLM fails
			articles[i].Summary = ""
		} else {
			articles[i].Summary = summary
		}

		// Log progress every 50 articles
		if (i+1)%50 == 0 {
			s.logger.Info("Enrichment progress", map[string]interface{}{
				"enriched": i + 1,
				"total":    len(articles),
			})
		}
	}

	s.logger.Info("Completed enriching articles with summaries", map[string]interface{}{
		"total": len(articles),
	})

	// Bulk insert articles into the database
	stats, err := s.articleRepo.BulkInsert(articles)
	if err != nil {
		s.logger.Error("Failed to bulk insert articles", err, map[string]interface{}{
			"filepath": filepath,
		})
		return stats, fmt.Errorf("failed to bulk insert articles: %w", err)
	}

	s.logger.Info("Completed loading articles from JSON", map[string]interface{}{
		"filepath":      filepath,
		"total":         stats.TotalArticles,
		"success_count": stats.SuccessCount,
		"error_count":   stats.ErrorCount,
	})

	return stats, nil
}

// CreateArticle creates a single article in the database
// Optionally enriches it with an LLM-generated summary if summary is empty
func (s *newsService) CreateArticle(article *models.Article) error {
	s.logger.Info("Creating article", map[string]interface{}{
		"title": article.Title,
	})

	// If summary is empty, generate one using LLM
	if article.Summary == "" {
		summary, err := s.llmService.GenerateSummary(article.Title, article.Description)
		if err != nil {
			s.logger.Warn("Failed to generate summary for article", map[string]interface{}{
				"title": article.Title,
				"error": err.Error(),
			})
			// Continue with empty summary if LLM fails
			article.Summary = ""
		} else {
			article.Summary = summary
		}
	}

	// Insert article into database
	if err := s.articleRepo.Insert(article); err != nil {
		s.logger.Error("Failed to create article", err, map[string]interface{}{
			"title": article.Title,
		})
		return fmt.Errorf("failed to create article: %w", err)
	}

	s.logger.Info("Successfully created article", map[string]interface{}{
		"id":    article.ID,
		"title": article.Title,
	})

	return nil
}
