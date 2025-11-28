package services

import (
	"fmt"
	"math"
	"sort"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
	"news-inshorts/src/types"
)

// NewsService defines the interface for news operations
type NewsService interface {
	ProcessNewsQuery(query string, location *models.Location) ([]models.EnrichedArticle, error)
	GetTrendingNews(lat, lon float64, limit int) ([]models.EnrichedArticle, error)
	FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error)
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
func (s *newsService) ProcessNewsQuery(query string, location *models.Location) ([]models.EnrichedArticle, error) {
	// Step 1: Analyze query using LLM to extract intents and entities
	analysis, err := s.llmService.ProcessQuery(query)
	if err != nil {
		s.logger.Error("Failed to analyze query with LLM", err, map[string]interface{}{
			"query": query,
		})
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}

	s.logger.Debug("Query analysis completed", map[string]interface{}{
		"entities_count": len(analysis.Entities),
		"intents_count":  len(analysis.Intents),
	})

	// Step 2: Retrieve initial article set (all articles)
	articles, err := s.articleRepo.FindAll()
	if err != nil {
		s.logger.Error("Failed to retrieve articles", err, nil)
		return nil, fmt.Errorf("failed to retrieve articles: %w", err)
	}

	s.logger.Debug("Retrieved initial articles", map[string]interface{}{
		"count": len(articles),
	})

	// Step 3: Execute filter chain with extracted intents
	filteredArticles, err := s.filterChain.Execute(articles, analysis.Intents, location)
	if err != nil {
		s.logger.Error("Failed to execute filter chain", err, nil)
		return nil, fmt.Errorf("failed to filter articles: %w", err)
	}

	s.logger.Debug("Filter chain execution completed", map[string]interface{}{
		"filtered_count": len(filteredArticles),
	})

	// Step 4: Limit results to top 5 articles
	if len(filteredArticles) > 5 {
		filteredArticles = filteredArticles[:5]
	}

	s.logger.Info("Query processing completed", map[string]interface{}{
		"query":        query,
		"result_count": len(filteredArticles),
	})

	// Step 5: Enrich articles with LLM summaries and distance information
	enrichedArticles, err := s.enrichArticles(filteredArticles, location, analysis)
	if err != nil {
		s.logger.Error("Failed to enrich articles", err, nil)
		return nil, fmt.Errorf("failed to enrich articles: %w", err)
	}

	return enrichedArticles, nil
}

// enrichArticles generates LLM summaries and adds distance information for articles
func (s *newsService) enrichArticles(articles []models.Article, location *models.Location, analysis *models.QueryAnalysis) ([]models.EnrichedArticle, error) {
	enrichedArticles := make([]models.EnrichedArticle, 0, len(articles))

	// Check if query has nearby intent to determine if we need distance calculation
	hasNearbyIntent := analysis.HasIntent("nearby")

	for _, article := range articles {
		enriched := models.EnrichedArticle{
			Article: article,
		}

		// Generate LLM summary for the article
		summary, err := s.llmService.GenerateSummary(article.Title, article.Description)
		if err != nil {
			// Handle summary generation errors gracefully by using empty string
			s.logger.Warn("Failed to generate summary for article", map[string]interface{}{
				"article_id": article.ID,
				"title":      article.Title,
				"error":      err.Error(),
			})
			enriched.LLMSummary = ""
		} else {
			enriched.LLMSummary = summary
		}

		// Add distance field for nearby queries
		if hasNearbyIntent && location != nil {
			distance := s.calculateDistance(
				article.Latitude,
				article.Longitude,
				location.Latitude,
				location.Longitude,
			)
			enriched.Distance = distance
		}

		enrichedArticles = append(enrichedArticles, enriched)
	}

	s.logger.Debug("Articles enriched", map[string]interface{}{
		"count": len(enrichedArticles),
	})

	return enrichedArticles, nil
}

// calculateDistance computes the distance between two geographic coordinates using Haversine formula
// Returns distance in kilometers
func (s *newsService) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180.0
	lon1Rad := lon1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0
	lon2Rad := lon2 * math.Pi / 180.0

	// Haversine formula
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// GetTrendingNews retrieves trending articles based on location
// Implements caching, trending score computation, and article enrichment
func (s *newsService) GetTrendingNews(lat, lon float64, limit int) ([]models.EnrichedArticle, error) {
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
		s.logger.Debug("Returning cached trending articles", map[string]interface{}{
			"count": len(cachedArticles),
		})

		// Enrich cached articles
		enrichedArticles := make([]models.EnrichedArticle, 0, len(cachedArticles))
		for _, article := range cachedArticles {
			// Generate summary for each article
			summary, err := s.llmService.GenerateSummary(article.Title, article.Description)
			if err != nil {
				s.logger.Warn("Failed to generate summary for cached article", map[string]interface{}{
					"article_id": article.ID,
					"error":      err.Error(),
				})
				summary = ""
			}

			enrichedArticles = append(enrichedArticles, models.EnrichedArticle{
				Article:    article,
				LLMSummary: summary,
			})
		}

		return enrichedArticles, nil
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

	// Step 7: Enrich articles with summaries
	enrichedArticles := make([]models.EnrichedArticle, 0, len(trendingArticles))
	for _, article := range trendingArticles {
		summary, err := s.llmService.GenerateSummary(article.Title, article.Description)
		if err != nil {
			s.logger.Warn("Failed to generate summary for trending article", map[string]interface{}{
				"article_id": article.ID,
				"error":      err.Error(),
			})
			summary = ""
		}

		enrichedArticles = append(enrichedArticles, models.EnrichedArticle{
			Article:    article,
			LLMSummary: summary,
		})
	}

	return enrichedArticles, nil
}

// FilterArticles dynamically filters articles based on provided parameters
// Supports filtering by category, source, and/or location (nearby)
// Multiple filters can be combined
func (s *newsService) FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error) {
	return s.articleRepo.FilterArticles(params)
}
