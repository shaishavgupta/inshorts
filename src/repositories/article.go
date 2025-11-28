package repositories

import (
	"encoding/json"
	"fmt"
	"os"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/types"
	"news-inshorts/src/utils"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// LoadStats represents statistics from loading articles
type LoadStats struct {
	TotalArticles    int      `json:"total_articles"`
	SuccessCount     int      `json:"success_count"`
	ErrorCount       int      `json:"error_count"`
	ValidationErrors []string `json:"validation_errors,omitempty"`
}

// ArticleRepository defines the interface for article data access
type ArticleRepository interface {
	LoadFromJSON(filepath string) (*LoadStats, error)
	FindAll() ([]models.Article, error)
	FindByScoreThreshold(threshold float64) ([]models.Article, error)
	SearchByText(query string) ([]models.Article, error)
	FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error)
}

// articleRepository implements ArticleRepository
type articleRepository struct {
	db  *gorm.DB
	log infra.Logger
}

// NewArticleRepository creates a new instance of ArticleRepository
func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &articleRepository{
		db:  db,
		log: infra.GetLogger(),
	}
}

// FindAll retrieves all articles from the database
func (r *articleRepository) FindAll() ([]models.Article, error) {
	query := `
		SELECT
			id,
			title,
			description,
			url,
			publication_date,
			source_name,
			category,
			relevance_score,
			latitude,
			longitude
		FROM articles
		ORDER BY publication_date DESC
	`

	var articles []models.Article
	if err := r.db.Raw(query).Scan(&articles).Error; err != nil {
		r.log.Error("Failed to query all articles", err, nil)
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}

	r.log.Info("Retrieved all articles", map[string]interface{}{
		"count": len(articles),
	})

	return articles, nil
}

func (r *articleRepository) FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error) {
	query := `
		SELECT
			id,
			title,
			description,
			url,
			publication_date,
			source_name,
			category,
			relevance_score,
			latitude,
			longitude,
			summary
		FROM articles
		WHERE
	`

	if params.Category != "" {
		query += fmt.Sprintf(`  '%s' = ANY(category) AND`, params.Category)
	}

	if params.Source != "" {
		query += fmt.Sprintf(` source_name = '%s' AND`, params.Source)
	}

	if params.Lat != 0 && params.Lon != 0 {
		query += fmt.Sprintf(` latitude = %f AND longitude = %f`, params.Lat, params.Lon)
	}

	query = utils.RemoveTrailingAnd(query)

	var articles []models.Article
	if err := r.db.Raw(query).Order("publication_date DESC").Scan(&articles).Error; err != nil {
		r.log.Error("Failed to query articles", err, map[string]interface{}{
			"query": query,
		})
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}

	return articles, nil
}

// FindByScoreThreshold retrieves articles with relevance score above the threshold
func (r *articleRepository) FindByScoreThreshold(threshold float64) ([]models.Article, error) {
	query := `
		SELECT
			id,
			title,
			description,
			url,
			publication_date,
			source_name,
			category,
			relevance_score,
			latitude,
			longitude
		FROM articles
		WHERE relevance_score >= ?
		ORDER BY relevance_score DESC
	`

	var articles []models.Article
	if err := r.db.Raw(query, threshold).Scan(&articles).Error; err != nil {
		r.log.Error("Failed to query articles by score threshold", err, map[string]interface{}{
			"threshold": threshold,
		})
		return nil, fmt.Errorf("failed to query articles by score threshold: %w", err)
	}

	r.log.Info("Retrieved articles by score threshold", map[string]interface{}{
		"threshold": threshold,
		"count":     len(articles),
	})

	return articles, nil
}

// SearchByText performs text search on article titles and descriptions
func (r *articleRepository) SearchByText(query string) ([]models.Article, error) {
	// Uses simple text search with ILIKE for pattern matching

	sqlQuery := `
		SELECT
			id,
			title,
			description,
			url,
			publication_date,
			source_name,
			category,
			relevance_score,
			latitude,
			longitude
		FROM articles
		WHERE
			title ILIKE '%' || ? || '%'
			OR description ILIKE '%' || ? || '%'
		ORDER BY
			relevance_score DESC,
			publication_date DESC
	`

	var articles []models.Article
	if err := r.db.Raw(sqlQuery, query, query).Scan(&articles).Error; err != nil {
		r.log.Error("Failed to search articles by text", err, map[string]interface{}{
			"query": query,
		})
		return nil, fmt.Errorf("failed to search articles by text: %w", err)
	}

	r.log.Info("Retrieved articles by text search", map[string]interface{}{
		"query": query,
		"count": len(articles),
	})

	return articles, nil
}

// validateArticle validates an article structure
func (r *articleRepository) validateArticle(article *models.Article, index int) []string {
	var errors []string

	// Validate required fields
	if article.Title == "" {
		errors = append(errors, fmt.Sprintf("Article %d: title is required", index))
	}

	if article.URL == "" {
		errors = append(errors, fmt.Sprintf("Article %d: url is required", index))
	}

	if article.SourceName == "" {
		errors = append(errors, fmt.Sprintf("Article %d: source_name is required", index))
	}

	if len(article.Category) == 0 {
		errors = append(errors, fmt.Sprintf("Article %d: at least one category is required", index))
	}

	// Validate relevance score
	if article.RelevanceScore < 0 || article.RelevanceScore > 1 {
		errors = append(errors, fmt.Sprintf("Article %d: relevance_score must be between 0 and 1", index))
	}

	// Validate latitude
	if article.Latitude < -90 || article.Latitude > 90 {
		errors = append(errors, fmt.Sprintf("Article %d: latitude must be between -90 and 90", index))
	}

	// Validate longitude
	if article.Longitude < -180 || article.Longitude > 180 {
		errors = append(errors, fmt.Sprintf("Article %d: longitude must be between -180 and 180", index))
	}

	// Validate publication date
	if article.PublicationDate.IsZero() {
		errors = append(errors, fmt.Sprintf("Article %d: publication_date is required", index))
	}

	return errors
}

// LoadFromJSON loads articles from a JSON file and inserts them into the database
func (r *articleRepository) LoadFromJSON(filepath string) (*LoadStats, error) {
	stats := &LoadStats{
		ValidationErrors: []string{},
	}

	r.log.Info("Starting to load articles from JSON", map[string]interface{}{
		"filepath": filepath,
	})

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		r.log.Error("JSON file not found", err, map[string]interface{}{
			"filepath": filepath,
		})
		return nil, fmt.Errorf("file not found: %s", filepath)
	}

	// Read the JSON file
	file, err := os.Open(filepath)
	if err != nil {
		r.log.Error("Failed to open JSON file", err, map[string]interface{}{
			"filepath": filepath,
		})
		return nil, fmt.Errorf("failed to open JSON file: %w", err)
	}
	defer file.Close()

	// Parse JSON
	var articles []models.Article
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&articles); err != nil {
		r.log.Error("Failed to decode JSON", err, map[string]interface{}{
			"filepath": filepath,
		})
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	stats.TotalArticles = len(articles)

	if len(articles) == 0 {
		r.log.Warn("No articles found in JSON file", map[string]interface{}{
			"filepath": filepath,
		})
		return stats, nil
	}

	// Validate all articles first
	r.log.Info("Validating article structures", map[string]interface{}{
		"total": len(articles),
	})

	for i, article := range articles {
		validationErrors := r.validateArticle(&article, i)
		if len(validationErrors) > 0 {
			stats.ValidationErrors = append(stats.ValidationErrors, validationErrors...)
		}
	}

	// If there are validation errors, return them without inserting
	if len(stats.ValidationErrors) > 0 {
		r.log.Error("Validation failed for articles", nil, map[string]interface{}{
			"error_count": len(stats.ValidationErrors),
		})
		return stats, fmt.Errorf("validation failed: %d errors found", len(stats.ValidationErrors))
	}

	r.log.Info("All articles validated successfully", nil)

	// Begin transaction for batch insert
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Prepare insert statement
	insertQuery := `
		INSERT INTO articles (
			id,
			title,
			description,
			url,
			publication_date,
			source_name,
			category,
			relevance_score,
			latitude,
			longitude
		) VALUES (
			COALESCE(?::uuid, uuid_generate_v4()),
			?,
			?,
			?,
			?,
			?,
			?,
			?,
			?,
			?
		) ON CONFLICT (id) DO NOTHING;
	`

	successCount := 0
	errorCount := 0

	// Insert articles in batches
	for i, article := range articles {

		if err := tx.Exec(insertQuery,
			article.ID,
			article.Title,
			article.Description,
			article.URL,
			article.PublicationDate,
			article.SourceName,
			pq.Array(article.Category),
			article.RelevanceScore,
			article.Latitude,
			article.Longitude,
		).Error; err != nil {
			errorCount++
			r.log.Error("Failed to insert article", err, map[string]interface{}{
				"index": i,
				"title": article.Title,
			})
			continue
		}

		successCount++

		// Log progress every 100 articles
		if (i+1)%100 == 0 {
			r.log.Info("Loading progress", map[string]interface{}{
				"loaded": i + 1,
				"total":  len(articles),
			})
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		r.log.Error("Failed to commit transaction", err, nil)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	stats.SuccessCount = successCount
	stats.ErrorCount = errorCount

	r.log.Info("Completed loading articles from JSON", map[string]interface{}{
		"filepath":      filepath,
		"total":         len(articles),
		"success_count": successCount,
		"error_count":   errorCount,
	})

	return stats, nil
}
