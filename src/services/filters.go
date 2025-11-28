package services

import (
	"context"
	"fmt"
	"math"
	"sort"

	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
	"news-inshorts/src/types"
)

// FilterByCategory creates a filter that filters articles by category using DB queries
func FilterByCategory(repo repositories.ArticleRepository, category string) Filter {
	return func(ctx context.Context, in []models.Article) ([]models.Article, error) {
		if category == "" {
			return in, nil
		}

		// Query database for articles with this category
		dbResults, err := repo.FilterArticles(types.FilterArticlesRequest{
			Category: category,
		})
		if err != nil {
			return nil, fmt.Errorf("category filter failed: %w", err)
		}

		// If we have input articles, intersect with DB results
		if len(in) > 0 {
			// Create a map of article IDs from DB results for quick lookup
			dbMap := make(map[string]models.Article)
			for _, article := range dbResults {
				dbMap[article.ID] = article
			}

			// Filter input articles to only include those in DB results
			var filtered []models.Article
			for _, article := range in {
				if _, exists := dbMap[article.ID]; exists {
					filtered = append(filtered, article)
				}
			}

			// Sort by publication date (most recent first)
			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].PublicationDate.After(filtered[j].PublicationDate)
			})

			return filtered, nil
		}

		// If no input articles, return DB results directly (already sorted by publication_date DESC)
		return dbResults, nil
	}
}

// FilterBySource creates a filter that filters articles by source name using DB queries
func FilterBySource(repo repositories.ArticleRepository, source string) Filter {
	return func(ctx context.Context, in []models.Article) ([]models.Article, error) {
		if source == "" {
			return in, nil
		}

		// Query database for articles from this source
		dbResults, err := repo.FilterArticles(types.FilterArticlesRequest{
			Source: source,
		})
		if err != nil {
			return nil, fmt.Errorf("source filter failed: %w", err)
		}

		// If we have input articles, intersect with DB results
		if len(in) > 0 {
			// Create a map of article IDs from DB results for quick lookup
			dbMap := make(map[string]models.Article)
			for _, article := range dbResults {
				dbMap[article.ID] = article
			}

			// Filter input articles to only include those in DB results
			var filtered []models.Article
			for _, article := range in {
				if _, exists := dbMap[article.ID]; exists {
					filtered = append(filtered, article)
				}
			}

			// Sort by publication date (most recent first)
			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].PublicationDate.After(filtered[j].PublicationDate)
			})

			return filtered, nil
		}

		// If no input articles, return DB results directly (already sorted by publication_date DESC)
		return dbResults, nil
	}
}

// FilterByScore creates a filter that filters articles by relevance score threshold using DB queries
func FilterByScore(repo repositories.ArticleRepository, threshold float64) Filter {
	return func(ctx context.Context, in []models.Article) ([]models.Article, error) {
		// Query database for articles above the threshold
		dbResults, err := repo.FindByScoreThreshold(threshold)
		if err != nil {
			return nil, fmt.Errorf("score filter failed: %w", err)
		}

		// If we have input articles, intersect with DB results
		if len(in) > 0 {
			// Create a map of article IDs from DB results for quick lookup
			dbMap := make(map[string]models.Article)
			for _, article := range dbResults {
				dbMap[article.ID] = article
			}

			// Filter input articles to only include those in DB results
			var filtered []models.Article
			for _, article := range in {
				if _, exists := dbMap[article.ID]; exists {
					filtered = append(filtered, article)
				}
			}

			// Sort by relevance score (highest first)
			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].RelevanceScore > filtered[j].RelevanceScore
			})

			return filtered, nil
		}

		// If no input articles, return DB results directly (already sorted by relevance_score DESC)
		return dbResults, nil
	}
}

// FilterByTextSearch creates a filter that filters articles using text search via repository
func FilterByTextSearch(repo repositories.ArticleRepository, query []string) Filter {
	return func(ctx context.Context, in []models.Article) ([]models.Article, error) {
		if len(query) == 0 {
			return in, nil
		}

		// Get search results from repository
		searchResults, err := repo.SearchByText(query)
		if err != nil {
			return nil, fmt.Errorf("search filter failed: %w", err)
		}

		// If we have input articles, intersect with search results
		if len(in) > 0 {
			// Create a map of article IDs from search results for quick lookup
			searchMap := make(map[string]models.Article)
			for _, article := range searchResults {
				searchMap[article.ID] = article
			}

			// Filter input articles to only include those in search results
			var filtered []models.Article
			for _, article := range in {
				if _, exists := searchMap[article.ID]; exists {
					filtered = append(filtered, article)
				}
			}

			// Sort by relevance score (already sorted by DB, but maintain order for consistency)
			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].RelevanceScore > filtered[j].RelevanceScore
			})

			return filtered, nil
		}

		// If no input articles, return search results directly (already sorted by relevance_score DESC from DB)
		return searchResults, nil
	}
}

// FilterByRadius creates a filter that filters articles by geographic proximity using DB queries
func FilterByRadius(repo repositories.ArticleRepository, lat, lon, radius float64) Filter {
	return func(ctx context.Context, in []models.Article) ([]models.Article, error) {
		if lat == 0 && lon == 0 {
			return in, nil
		}

		// Get nearby results from repository using PostGIS (already sorted by distance)
		nearbyResults, err := repo.FilterArticles(types.FilterArticlesRequest{
			Lat:    lat,
			Lon:    lon,
			Radius: radius,
		})
		if err != nil {
			return nil, fmt.Errorf("nearby filter failed: %w", err)
		}

		// If we have input articles, intersect with nearby results
		if len(in) > 0 {
			// Create a map of article IDs from nearby results for quick lookup
			nearbyMap := make(map[string]models.Article)
			for _, article := range nearbyResults {
				nearbyMap[article.ID] = article
			}

			// Filter input articles to only include those in nearby results
			var filtered []models.Article
			for _, article := range in {
				if _, exists := nearbyMap[article.ID]; exists {
					filtered = append(filtered, article)
				}
			}

			// Sort by distance (closest first) using Haversine formula
			// Note: DB already sorted by distance, but we recalculate for consistency with filtered set
			sort.Slice(filtered, func(i, j int) bool {
				distI := haversineDistance(lat, lon, filtered[i].Latitude, filtered[i].Longitude)
				distJ := haversineDistance(lat, lon, filtered[j].Latitude, filtered[j].Longitude)
				return distI < distJ
			})

			return filtered, nil
		}

		// If no input articles, return nearby results directly (already sorted by distance from DB)
		return nearbyResults, nil
	}
}

// haversineDistance calculates the distance between two geographic coordinates in kilometers
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}
