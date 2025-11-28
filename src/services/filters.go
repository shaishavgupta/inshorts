package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
	"news-inshorts/src/types"
)

// FilterByCategory creates a filter that filters articles by category
func FilterByCategory(repo repositories.ArticleRepository, categories []string) Filter {
	return func(ctx context.Context, in []models.Article) ([]models.Article, error) {
		if len(categories) == 0 {
			return in, nil
		}

		dbResults, err := repo.FilterArticles(types.FilterArticlesRequest{
			Category: strings.Join(categories, ","),
		})
		if err != nil {
			return nil, fmt.Errorf("category filter failed: %w", err)
		}

		if len(in) > 0 {
			dbMap := make(map[string]models.Article)
			for _, article := range dbResults {
				dbMap[article.ID] = article
			}

			var filtered []models.Article
			for _, article := range in {
				if _, exists := dbMap[article.ID]; exists {
					filtered = append(filtered, article)
				}
			}

			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].PublicationDate.After(filtered[j].PublicationDate)
			})

			return filtered, nil
		}

		return dbResults, nil
	}
}

// FilterBySource creates a filter that filters articles by source name
func FilterBySource(repo repositories.ArticleRepository, source string) Filter {
	return func(ctx context.Context, in []models.Article) ([]models.Article, error) {
		if source == "" {
			return in, nil
		}

		dbResults, err := repo.FilterArticles(types.FilterArticlesRequest{
			Source: source,
		})
		if err != nil {
			return nil, fmt.Errorf("source filter failed: %w", err)
		}

		if len(in) > 0 {
			dbMap := make(map[string]models.Article)
			for _, article := range dbResults {
				dbMap[article.ID] = article
			}

			var filtered []models.Article
			for _, article := range in {
				if _, exists := dbMap[article.ID]; exists {
					filtered = append(filtered, article)
				}
			}

			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].PublicationDate.After(filtered[j].PublicationDate)
			})

			return filtered, nil
		}

		return dbResults, nil
	}
}

// FilterByScore creates a filter that filters articles by relevance score threshold
func FilterByScore(repo repositories.ArticleRepository, threshold float64) Filter {
	return func(ctx context.Context, in []models.Article) ([]models.Article, error) {
		dbResults, err := repo.FindByScoreThreshold(threshold)
		if err != nil {
			return nil, fmt.Errorf("score filter failed: %w", err)
		}

		if len(in) > 0 {
			dbMap := make(map[string]models.Article)
			for _, article := range dbResults {
				dbMap[article.ID] = article
			}

			var filtered []models.Article
			for _, article := range in {
				if _, exists := dbMap[article.ID]; exists {
					filtered = append(filtered, article)
				}
			}

			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].RelevanceScore > filtered[j].RelevanceScore
			})

			return filtered, nil
		}

		return dbResults, nil
	}
}

// FilterByTextSearch creates a filter that filters articles using text search
func FilterByTextSearch(repo repositories.ArticleRepository, query []string) Filter {
	return func(ctx context.Context, in []models.Article) ([]models.Article, error) {
		if len(query) == 0 {
			return in, nil
		}

		searchResults, err := repo.SearchByText(query)
		if err != nil {
			return nil, fmt.Errorf("search filter failed: %w", err)
		}

		if len(in) > 0 {
			searchMap := make(map[string]models.Article)
			for _, article := range searchResults {
				searchMap[article.ID] = article
			}

			var filtered []models.Article
			for _, article := range in {
				if _, exists := searchMap[article.ID]; exists {
					filtered = append(filtered, article)
				}
			}

			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].RelevanceScore > filtered[j].RelevanceScore
			})

			return filtered, nil
		}

		return searchResults, nil
	}
}

// FilterByRadius creates a filter that filters articles by geographic proximity
func FilterByRadius(repo repositories.ArticleRepository, lat, lon, radius float64) Filter {
	return func(ctx context.Context, in []models.Article) ([]models.Article, error) {
		if lat == 0 && lon == 0 {
			return in, nil
		}

		nearbyResults, err := repo.FilterArticles(types.FilterArticlesRequest{
			Lat:    lat,
			Lon:    lon,
			Radius: radius,
		})
		if err != nil {
			return nil, fmt.Errorf("nearby filter failed: %w", err)
		}

		if len(in) > 0 {
			nearbyMap := make(map[string]models.Article)
			for _, article := range nearbyResults {
				nearbyMap[article.ID] = article
			}

			var filtered []models.Article
			for _, article := range in {
				if _, exists := nearbyMap[article.ID]; exists {
					filtered = append(filtered, article)
				}
			}

			sort.Slice(filtered, func(i, j int) bool {
				distI := haversineDistance(lat, lon, filtered[i].Latitude, filtered[i].Longitude)
				distJ := haversineDistance(lat, lon, filtered[j].Latitude, filtered[j].Longitude)
				return distI < distJ
			})

			return filtered, nil
		}

		return nearbyResults, nil
	}
}

// haversineDistance calculates the distance between two geographic coordinates in kilometers
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}
