package services

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
	"news-inshorts/src/types"
)

// ArticleService defines the interface for news operations
type ArticleService interface {
	ProcessArticleQuery(query string, location *models.Location) ([]models.Article, error)
	GetTrendingNews(lat, lon float64, limit int) ([]models.Article, error)
	FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error)
	LoadFromJSON(filepath string) (*repositories.LoadStats, error)
	CreateArticle(article *models.Article) error
}

// articleService implements ArticleService
type articleService struct {
	llmService      LLMService
	filterChain     *FilterChain
	trendingService TrendingService
	articleRepo     repositories.ArticleRepository
	userEventRepo   repositories.UserEventRepository
	logger          infra.Logger
}

// NewArticleService creates a new instance of ArticleService
func NewArticleService(
	llmService LLMService,
	filterChain *FilterChain,
	trendingService TrendingService,
	articleRepo repositories.ArticleRepository,
	userEventRepo repositories.UserEventRepository,
) ArticleService {
	return &articleService{
		llmService:      llmService,
		filterChain:     filterChain,
		trendingService: trendingService,
		articleRepo:     articleRepo,
		userEventRepo:   userEventRepo,
		logger:          infra.GetLogger(),
	}
}

// ProcessArticleQuery orchestrates LLM query analysis and filter chain execution
// to retrieve and enrich relevant news articles
func (s *articleService) ProcessArticleQuery(query string, location *models.Location) ([]models.Article, error) {
	allowedSources, err := s.articleRepo.GetDistinctSourceNames()
	if err != nil {
		s.logger.Error("Failed to get allowed sources", err, nil)
		return nil, fmt.Errorf("failed to get allowed sources: %w", err)
	}

	allowedCategories, err := s.articleRepo.GetDistinctCategories()
	if err != nil {
		s.logger.Error("Failed to get allowed categories", err, nil)
		return nil, fmt.Errorf("failed to get allowed categories: %w", err)
	}

	analysis, err := s.llmService.ProcessQuery(query, allowedSources, allowedCategories)
	if err != nil {
		s.logger.Error("Failed to analyze query with LLM", err, map[string]interface{}{
			"query": query,
		})
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}

	filteredArticles, err := s.filterChain.Execute(analysis.Intents, analysis.Entities, location)
	if err != nil {
		s.logger.Error("Failed to execute filter chain", err, nil)
		return nil, fmt.Errorf("failed to filter articles: %w", err)
	}

	if len(filteredArticles) > 5 {
		filteredArticles = filteredArticles[:5]
	}

	return filteredArticles, nil
}

// GetTrendingNews retrieves trending articles based on location
func (s *articleService) GetTrendingNews(lat, lon float64, limit int) ([]models.Article, error) {
	s.logger.Info("Getting trending news", map[string]interface{}{
		"latitude":  lat,
		"longitude": lon,
		"limit":     limit,
	})

	location := models.Location{
		Latitude:  lat,
		Longitude: lon,
	}

	// cachedArticles, found := s.trendingService.GetCachedTrending(lat, lon, limit)
	// if found {
	// 	return cachedArticles, nil
	// }

	// Get distinct article IDs from user_events
	articleIDs, err := s.userEventRepo.GetArticlesFromUserEvents()
	if err != nil {
		s.logger.Error("Failed to get distinct article IDs from user events", err, nil)
		return nil, fmt.Errorf("failed to get distinct article IDs: %w", err)
	}

	if len(articleIDs) == 0 {
		s.logger.Info("No articles found in user events", nil)
		return []models.Article{}, nil
	}

	// Get articles by IDs
	articles, err := s.articleRepo.FindByIDs(articleIDs)
	if err != nil {
		s.logger.Error("Failed to retrieve articles for trending", err, nil)
		return nil, fmt.Errorf("failed to retrieve articles: %w", err)
	}

	type articleWithScore struct {
		article models.Article
		score   float64
	}

	articlesWithScores := make([]articleWithScore, 0, len(articles))

	for _, article := range articles {
		score, err := s.trendingService.ComputeTrendingScore(article, location)
		if err != nil {
			s.logger.Error("Failed to compute trending score for article", err, map[string]interface{}{
				"article_id": article.ID,
			})
			continue
		}

		articlesWithScores = append(articlesWithScores, articleWithScore{
			article: article,
			score:   score,
		})
	}

	sort.Slice(articlesWithScores, func(i, j int) bool {
		return articlesWithScores[i].score > articlesWithScores[j].score
	})

	s.logger.Debug("Sorted articles by trending score", map[string]interface{}{
		"total_scored": len(articlesWithScores),
	})

	if len(articlesWithScores) > limit {
		articlesWithScores = articlesWithScores[:limit]
	}

	trendingArticles := make([]models.Article, 0, len(articlesWithScores))
	for _, aws := range articlesWithScores {
		trendingArticles = append(trendingArticles, aws.article)
	}

	s.trendingService.CacheTrending(lat, lon, trendingArticles)

	s.logger.Info("Computed trending articles", map[string]interface{}{
		"count": len(trendingArticles),
	})

	return trendingArticles, nil
}

// FilterArticles filters articles based on provided parameters
func (s *articleService) FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error) {
	return s.articleRepo.FilterArticles(params)
}

// LoadFromJSON loads articles from a JSON file, enriches them with LLM summaries, and inserts them into the database
func (s *articleService) LoadFromJSON(filepath string) (*repositories.LoadStats, error) {
	s.logger.Info("Starting to load articles from JSON", map[string]interface{}{
		"filepath": filepath,
	})

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		s.logger.Error("JSON file not found", err, map[string]interface{}{
			"filepath": filepath,
		})
		return nil, fmt.Errorf("file not found: %s", filepath)
	}

	file, err := os.Open(filepath)
	if err != nil {
		s.logger.Error("Failed to open JSON file", err, map[string]interface{}{
			"filepath": filepath,
		})
		return nil, fmt.Errorf("failed to open JSON file: %w", err)
	}
	defer file.Close()

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

	s.logger.Info("Enriching articles with LLM summaries and embeddings", map[string]interface{}{
		"total": len(articles),
	})

	var wg sync.WaitGroup
	var mu sync.Mutex
	completedCount := 0

	for i := range articles {
		wg.Add(2)

		// Goroutine 1: Generate summary
		go func(idx int) {
			defer wg.Done()
			summary, err := s.llmService.GenerateSummary(articles[idx].Title, articles[idx].Description)
			if err != nil {
				s.logger.Warn("Failed to generate summary for article", map[string]interface{}{
					"index": idx,
					"title": articles[idx].Title,
					"error": err.Error(),
				})
				mu.Lock()
				articles[idx].Summary = ""
				mu.Unlock()
			} else {
				mu.Lock()
				articles[idx].Summary = summary
				mu.Unlock()
			}

			// Track progress
			mu.Lock()
			completedCount++
			currentCount := completedCount
			mu.Unlock()

			if currentCount%50 == 0 {
				s.logger.Info("Enrichment progress", map[string]interface{}{
					"completed": currentCount,
					"total":     len(articles) * 2, // 2 operations per article
				})
			}
		}(i)

		// Goroutine 2: Generate embedding
		go func(idx int) {
			defer wg.Done()
			embedding, err := s.llmService.GenerateEmbedding(articles[idx].Description)
			if err != nil {
				s.logger.Warn("Failed to generate embedding for article", map[string]interface{}{
					"index": idx,
					"title": articles[idx].Title,
					"error": err.Error(),
				})
				mu.Lock()
				articles[idx].DescriptionVector = nil
				mu.Unlock()
			} else {
				mu.Lock()
				articles[idx].DescriptionVector = embedding
				mu.Unlock()
			}

			// Track progress
			mu.Lock()
			completedCount++
			currentCount := completedCount
			mu.Unlock()

			if currentCount%50 == 0 {
				s.logger.Info("Enrichment progress", map[string]interface{}{
					"completed": currentCount,
					"total":     len(articles) * 2, // 2 operations per article
				})
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	s.logger.Info("Completed enriching articles with summaries and embeddings", map[string]interface{}{
		"total": len(articles),
	})

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
func (s *articleService) CreateArticle(article *models.Article) error {
	s.logger.Info("Creating article", map[string]interface{}{
		"title": article.Title,
	})

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Generate summary if not provided
	if article.Summary == "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			summary, err := s.llmService.GenerateSummary(article.Title, article.Description)
			if err != nil {
				s.logger.Warn("Failed to generate summary for article", map[string]interface{}{
					"title": article.Title,
					"error": err.Error(),
				})
				mu.Lock()
				article.Summary = ""
				mu.Unlock()
			} else {
				mu.Lock()
				article.Summary = summary
				mu.Unlock()
			}
		}()
	}

	// Generate embedding if not provided
	if len(article.DescriptionVector) == 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			embedding, err := s.llmService.GenerateEmbedding(article.Description)
			if err != nil {
				s.logger.Warn("Failed to generate embedding for article", map[string]interface{}{
					"title": article.Title,
					"error": err.Error(),
				})
				mu.Lock()
				article.DescriptionVector = nil
				mu.Unlock()
			} else {
				mu.Lock()
				article.DescriptionVector = embedding
				mu.Unlock()
			}
		}()
	}

	// Wait for both goroutines to complete
	wg.Wait()

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
