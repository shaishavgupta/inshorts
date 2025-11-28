package repositories

import (
	"fmt"
	"strings"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/types"

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
	BulkInsert(articles []models.Article) (*LoadStats, error)
	Insert(article *models.Article) error
	FindAll() ([]models.Article, error)
	FindByScoreThreshold(threshold float64) ([]models.Article, error)
	SearchByText(query []string) ([]models.Article, error)
	FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error)
	FindByIDs(ids []string) ([]models.Article, error)
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

// FilterArticles filters articles based on category, source, and/or location
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
	`

	var conditions []string

	if params.Category != "" {
		cats := strings.Split(params.Category, ",")
		quoted := make([]string, len(cats))
		for i, c := range cats {
			quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(c, "'", "''"))
		}
		conditions = append(conditions, fmt.Sprintf(`category @> ARRAY[%s]`, strings.Join(quoted, ",")))
	}

	if params.Source != "" {
		// Escape single quotes in source name
		escapedSource := strings.ReplaceAll(params.Source, "'", "''")
		conditions = append(conditions, fmt.Sprintf(`source_name = '%s'`, escapedSource))
	}

	if params.Lat != 0 && params.Lon != 0 {
		if params.Radius > 0 {
			// Use PostGIS for radius-based filtering
			conditions = append(conditions, fmt.Sprintf(`ST_DWithin(
				ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography,
				ST_SetSRID(ST_MakePoint(%f, %f), 4326)::geography,
				%f * 1000
			)`, params.Lon, params.Lat, params.Radius))
		} else {
			// Exact location match
			conditions = append(conditions, fmt.Sprintf(`latitude = %f AND longitude = %f`, params.Lat, params.Lon))
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var orderBy string
	if params.Lat != 0 && params.Lon != 0 && params.Radius > 0 {
		// Order by distance when using radius
		orderBy = fmt.Sprintf(`ST_Distance(
			ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography,
			ST_SetSRID(ST_MakePoint(%f, %f), 4326)::geography
		) ASC`, params.Lon, params.Lat)
	} else {
		orderBy = "publication_date DESC"
	}

	var articles []models.Article
	if err := r.db.Raw(query).Order(orderBy).Scan(&articles).Error; err != nil {
		r.log.Error("Failed to query articles", err, map[string]interface{}{
			"query": query,
		})
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}

	return articles, nil
}

// FindByIDs retrieves articles by their IDs
func (r *articleRepository) FindByIDs(ids []string) ([]models.Article, error) {
	if len(ids) == 0 {
		return []models.Article{}, nil
	}

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
		WHERE id = ANY(?)
		ORDER BY publication_date DESC
	`

	var articles []models.Article
	if err := r.db.Raw(query, pq.Array(ids)).Scan(&articles).Error; err != nil {
		r.log.Error("Failed to query articles by IDs", err, map[string]interface{}{
			"ids_count": len(ids),
		})
		return nil, fmt.Errorf("failed to query articles by IDs: %w", err)
	}

	r.log.Info("Retrieved articles by IDs", map[string]interface{}{
		"requested_count": len(ids),
		"found_count":     len(articles),
	})

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
func (r *articleRepository) SearchByText(query []string) ([]models.Article, error) {
	// Uses simple text search with ILIKE for pattern matching
	if len(query) == 0 {
		return []models.Article{}, nil
	}

	// Build WHERE clause with OR conditions for each query term
	var conditions []string
	var args []interface{}

	for _, term := range query {
		conditions = append(conditions, "(title ILIKE '%' || ? || '%' OR description ILIKE '%' || ? || '%')")
		args = append(args, term, term)
	}

	whereClause := strings.Join(conditions, " OR ")

	sqlQuery := fmt.Sprintf(`
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
		WHERE %s
		ORDER BY
			relevance_score DESC,
			publication_date DESC
	`, whereClause)

	var articles []models.Article
	if err := r.db.Raw(sqlQuery, args...).Scan(&articles).Error; err != nil {
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

// BulkInsert inserts multiple articles into the database in a single transaction
func (r *articleRepository) BulkInsert(articles []models.Article) (*LoadStats, error) {
	stats := &LoadStats{
		TotalArticles:    len(articles),
		ValidationErrors: []string{},
	}

	if len(articles) == 0 {
		r.log.Warn("No articles provided for bulk insert", nil)
		return stats, nil
	}

	r.log.Info("Starting bulk insert of articles", map[string]interface{}{
		"total": len(articles),
	})

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

	// Prepare insert statement with summary field
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
			longitude,
			summary
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
			article.Summary,
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
			r.log.Info("Bulk insert progress", map[string]interface{}{
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

	r.log.Info("Completed bulk insert of articles", map[string]interface{}{
		"total":         len(articles),
		"success_count": successCount,
		"error_count":   errorCount,
	})

	return stats, nil
}

// Insert inserts a single article into the database
func (r *articleRepository) Insert(article *models.Article) error {
	// Validate article
	validationErrors := r.validateArticle(article, 0)
	if len(validationErrors) > 0 {
		r.log.Error("Validation failed for article", nil, map[string]interface{}{
			"errors": validationErrors,
		})
		return fmt.Errorf("validation failed: %v", validationErrors)
	}

	// Prepare insert statement with summary field
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
			longitude,
			summary
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
			?,
			?
		) RETURNING id;
	`

	var insertedID string
	if err := r.db.Raw(insertQuery,
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
		article.Summary,
	).Scan(&insertedID).Error; err != nil {
		r.log.Error("Failed to insert article", err, map[string]interface{}{
			"title": article.Title,
		})
		return fmt.Errorf("failed to insert article: %w", err)
	}

	// Update article ID if it was generated
	if article.ID == "" {
		article.ID = insertedID
	}

	r.log.Info("Successfully inserted article", map[string]interface{}{
		"id":    article.ID,
		"title": article.Title,
	})

	return nil
}
