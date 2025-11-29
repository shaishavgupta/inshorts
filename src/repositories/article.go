package repositories

import (
	"fmt"
	"strconv"
	"strings"

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
	BulkInsert(articles []models.Article) (*LoadStats, error)
	Insert(article *models.Article) error
	FindAll() ([]models.Article, error)
	SearchByText(query []string) ([]models.Article, error)
	FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error)
	FindByIDs(ids []string) ([]models.Article, error)
	GetDistinctSourceNames() ([]string, error)
	GetDistinctCategories() ([]string, error)
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
		quoted := utils.QuoteAndEscapeStrings(params.Category)
		conditions = append(conditions, fmt.Sprintf(`category @> ARRAY[%s]`, strings.Join(quoted, ",")))
	}

	if params.Source != "" {
		conditions = append(conditions, fmt.Sprintf(`source_name ILIKE ANY (ARRAY[%s])`, params.Source))
	}

	if params.Lat != 0 && params.Lon != 0 {
		if params.Radius > 0 {
			conditions = append(conditions, fmt.Sprintf(`ST_DWithin(
				ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography,
				ST_SetSRID(ST_MakePoint(%f, %f), 4326)::geography,
				%f * 1000
			)`, params.Lon, params.Lat, params.Radius))
		} else {
			conditions = append(conditions, fmt.Sprintf(`latitude = %f AND longitude = %f`, params.Lat, params.Lon))
		}
	}

	if params.ScoreThreshold > 0 {
		conditions = append(conditions, fmt.Sprintf(`relevance_score >= %f`, params.ScoreThreshold))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var orderBy string
	if params.Lat != 0 && params.Lon != 0 && params.Radius > 0 {
		orderBy = fmt.Sprintf(`ST_Distance(
			ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography,
			ST_SetSRID(ST_MakePoint(%f, %f), 4326)::geography
		) ASC`, params.Lon, params.Lat)
	} else if params.ScoreThreshold > 0 {
		orderBy = "relevance_score DESC"
	} else {
		orderBy = "publication_date DESC"
	}

	var articles []models.Article
	fmt.Println(r.db.Raw(query).Order(orderBy).Statement.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Order(orderBy)
	}))

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

// SearchByText performs text search on article titles and descriptions
func (r *articleRepository) SearchByText(query []string) ([]models.Article, error) {
	if len(query) == 0 {
		return []models.Article{}, nil
	}

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

	if article.RelevanceScore < 0 || article.RelevanceScore > 1 {
		errors = append(errors, fmt.Sprintf("Article %d: relevance_score must be between 0 and 1", index))
	}

	if article.Latitude < -90 || article.Latitude > 90 {
		errors = append(errors, fmt.Sprintf("Article %d: latitude must be between -90 and 90", index))
	}

	if article.Longitude < -180 || article.Longitude > 180 {
		errors = append(errors, fmt.Sprintf("Article %d: longitude must be between -180 and 180", index))
	}

	if article.PublicationDate.IsZero() {
		errors = append(errors, fmt.Sprintf("Article %d: publication_date is required", index))
	}

	return errors
}

// formatVector formats a float64 slice as a pgvector string format: "[0.1,0.2,0.3]"
func formatVector(vector []float64) string {
	if len(vector) == 0 {
		return "[]"
	}
	var parts []string
	for _, v := range vector {
		parts = append(parts, strconv.FormatFloat(v, 'f', -1, 64))
	}
	return "[" + strings.Join(parts, ",") + "]"
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

	r.log.Info("Validating article structures", map[string]interface{}{
		"total": len(articles),
	})

	for i, article := range articles {
		validationErrors := r.validateArticle(&article, i)
		if len(validationErrors) > 0 {
			stats.ValidationErrors = append(stats.ValidationErrors, validationErrors...)
		}
	}

	if len(stats.ValidationErrors) > 0 {
		r.log.Error("Validation failed for articles", nil, map[string]interface{}{
			"error_count": len(stats.ValidationErrors),
		})
		return stats, fmt.Errorf("validation failed: %d errors found", len(stats.ValidationErrors))
	}

	r.log.Info("All articles validated successfully", nil)

	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

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
			summary,
			description_vector
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
			?,
			?::vector
		) ON CONFLICT (id) DO NOTHING;
	`

	successCount := 0
	errorCount := 0

	for i, article := range articles {
		// Format vector as string for pgvector
		var vectorStr interface{}
		if len(article.DescriptionVector) > 0 {
			vectorStr = formatVector(article.DescriptionVector)
		} else {
			vectorStr = nil
		}

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
			vectorStr,
		).Error; err != nil {
			errorCount++
			r.log.Error("Failed to insert article", err, map[string]interface{}{
				"index": i,
				"title": article.Title,
			})
			continue
		}

		successCount++

		if (i+1)%100 == 0 {
			r.log.Info("Bulk insert progress", map[string]interface{}{
				"loaded": i + 1,
				"total":  len(articles),
			})
		}
	}

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
	validationErrors := r.validateArticle(article, 0)
	if len(validationErrors) > 0 {
		r.log.Error("Validation failed for article", nil, map[string]interface{}{
			"errors": validationErrors,
		})
		return fmt.Errorf("validation failed: %v", validationErrors)
	}

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
			summary,
			description_vector
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
			?,
			?::vector
		) RETURNING id;
	`

	// Format vector as string for pgvector
	var vectorStr interface{}
	if len(article.DescriptionVector) > 0 {
		vectorStr = formatVector(article.DescriptionVector)
	} else {
		vectorStr = nil
	}

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
		vectorStr,
	).Scan(&insertedID).Error; err != nil {
		r.log.Error("Failed to insert article", err, map[string]interface{}{
			"title": article.Title,
		})
		return fmt.Errorf("failed to insert article: %w", err)
	}

	if article.ID == "" {
		article.ID = insertedID
	}

	r.log.Info("Successfully inserted article", map[string]interface{}{
		"id":    article.ID,
		"title": article.Title,
	})

	return nil
}

// GetDistinctSourceNames retrieves all distinct source names from the articles table
func (r *articleRepository) GetDistinctSourceNames() ([]string, error) {
	query := `
		SELECT DISTINCT source_name
		FROM articles
		WHERE source_name IS NOT NULL AND source_name != ''
		ORDER BY source_name ASC
	`

	var sourceNames []string
	if err := r.db.Raw(query).Scan(&sourceNames).Error; err != nil {
		r.log.Error("Failed to query distinct source names", err, nil)
		return nil, fmt.Errorf("failed to query distinct source names: %w", err)
	}

	r.log.Info("Retrieved distinct source names", map[string]interface{}{
		"count": len(sourceNames),
	})

	return sourceNames, nil
}

// GetDistinctCategories retrieves all distinct categories from the articles table
func (r *articleRepository) GetDistinctCategories() ([]string, error) {
	query := `
		SELECT DISTINCT unnest(category) AS category
		FROM articles
		WHERE category IS NOT NULL AND array_length(category, 1) > 0
		ORDER BY category ASC
	`

	var categories []string
	if err := r.db.Raw(query).Scan(&categories).Error; err != nil {
		r.log.Error("Failed to query distinct categories", err, nil)
		return nil, fmt.Errorf("failed to query distinct categories: %w", err)
	}

	r.log.Info("Retrieved distinct categories", map[string]interface{}{
		"count": len(categories),
	})

	return categories, nil
}
