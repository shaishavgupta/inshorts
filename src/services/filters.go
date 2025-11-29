package services

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
	"news-inshorts/src/types"
	"news-inshorts/src/utils"
)

// FilterByCategory creates a filter that filters articles by category
func FilterByCategory(repo repositories.ArticleRepository, categories []string) Filter {
	return func(ctx context.Context, in *[]models.Article) (*[]models.Article, error) {
		if len(categories) == 0 {
			return in, nil
		}

		articles := *in
		filteredArticles := []models.Article{}

		if len(articles) > 0 {
			for _, article := range articles {
				// Check if article has any matching category
				for _, articleCategory := range article.Category {
					if slices.Contains(categories, articleCategory) {
						filteredArticles = append(filteredArticles, article)
						break
					}
				}
			}
		} else {
			dbResults, err := repo.FilterArticles(types.FilterArticlesRequest{
				Category: strings.Join(categories, ","),
			})
			if err != nil {
				return nil, fmt.Errorf("category filter failed: %w", err)
			}
			filteredArticles = dbResults
		}

		return &filteredArticles, nil
	}
}

// FilterBySource creates a filter that filters articles by source name
func FilterBySource(repo repositories.ArticleRepository, sources []string) Filter {
	return func(ctx context.Context, in *[]models.Article) (*[]models.Article, error) {
		if len(sources) == 0 {
			return in, nil
		}

		articles := *in
		filteredArticles := []models.Article{}

		if len(articles) > 0 {
			for _, article := range articles {
				if slices.Contains(sources, article.SourceName) {
					filteredArticles = append(filteredArticles, article)
				}
			}
		} else {
			dbResults, err := repo.FilterArticles(types.FilterArticlesRequest{
				Source: utils.FormatStringsForLikeQuery(sources),
			})
			if err != nil {
				return nil, fmt.Errorf("source filter failed: %w", err)
			}
			filteredArticles = dbResults
		}

		return &filteredArticles, nil
	}
}

// FilterByScore creates a filter that filters articles by relevance score threshold
func FilterByScore(repo repositories.ArticleRepository, threshold float64) Filter {
	return func(ctx context.Context, in *[]models.Article) (*[]models.Article, error) {
		articles := *in
		filteredArticles := []models.Article{}

		if len(articles) > 0 {
			for _, article := range articles {
				if article.RelevanceScore >= threshold {
					filteredArticles = append(filteredArticles, article)
				}
			}
			sort.Slice(filteredArticles, func(i, j int) bool {
				return articles[i].RelevanceScore > articles[j].RelevanceScore
			})
		} else {
			dbResults, err := repo.FilterArticles(types.FilterArticlesRequest{
				ScoreThreshold: threshold,
			})
			if err != nil {
				return nil, fmt.Errorf("score filter failed: %w", err)
			}
			filteredArticles = dbResults
		}

		return &filteredArticles, nil
	}
}

// FilterByTextSearch creates a filter that filters articles using cosine similarity search
func FilterByTextSearch(repo repositories.ArticleRepository, llmService LLMService, query []string) Filter {
	return func(ctx context.Context, in *[]models.Article) (*[]models.Article, error) {
		if len(query) == 0 {
			return in, nil
		}

		articles := *in
		if len(articles) == 0 {
			return in, nil
		}

		// Join query strings into a single query string
		queryString := strings.Join(query, " ")
		if queryString == "" {
			return in, nil
		}

		// Generate embedding for the query
		queryVector, err := llmService.GenerateEmbedding(queryString)
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}

		// Calculate cosine similarity for each article with DescriptionVector
		type articleWithSimilarity struct {
			article    models.Article
			similarity float64
		}

		articlesWithSimilarity := make([]articleWithSimilarity, 0, len(articles))

		for _, article := range articles {
			// Skip articles without DescriptionVector
			if len(article.DescriptionVector) == 0 {
				continue
			}

			// Calculate cosine similarity
			similarity := cosineSimilarity(queryVector, article.DescriptionVector)

			articlesWithSimilarity = append(articlesWithSimilarity, articleWithSimilarity{
				article:    article,
				similarity: similarity,
			})
		}

		// Sort by similarity score (descending)
		sort.Slice(articlesWithSimilarity, func(i, j int) bool {
			return articlesWithSimilarity[i].similarity > articlesWithSimilarity[j].similarity
		})

		// Extract articles in sorted order
		filteredArticles := make([]models.Article, 0, len(articlesWithSimilarity))
		for _, aws := range articlesWithSimilarity {
			filteredArticles = append(filteredArticles, aws.article)
		}

		return &filteredArticles, nil
	}
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64
	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	norm1 = math.Sqrt(norm1)
	norm2 = math.Sqrt(norm2)

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (norm1 * norm2)
}

// FilterByRadius creates a filter that filters articles by geographic proximity
func FilterByRadius(repo repositories.ArticleRepository, lat, lon, radius float64) Filter {
	return func(ctx context.Context, in *[]models.Article) (*[]models.Article, error) {
		if lat == 0 && lon == 0 {
			return in, nil
		}

		articles := *in
		filteredArticles := []models.Article{}

		if len(articles) > 0 {
			// Filter in-memory by calculating distance
			for _, article := range articles {
				distance := haversineDistance(lat, lon, article.Latitude, article.Longitude)
				if distance <= radius {
					filteredArticles = append(filteredArticles, article)
				}
			}
			sort.Slice(filteredArticles, func(i, j int) bool {
				distI := haversineDistance(lat, lon, articles[i].Latitude, articles[i].Longitude)
				distJ := haversineDistance(lat, lon, articles[j].Latitude, articles[j].Longitude)
				return distI < distJ
			})
		} else {
			nearbyResults, err := repo.FilterArticles(types.FilterArticlesRequest{
				Lat:    lat,
				Lon:    lon,
				Radius: radius,
			})
			if err != nil {
				return nil, fmt.Errorf("nearby filter failed: %w", err)
			}
			filteredArticles = nearbyResults
		}

		return &filteredArticles, nil
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
