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

// CategoryFilter filters articles by category
type CategoryFilter struct {
	logger infra.Logger
}

// NewCategoryFilter creates a new CategoryFilter instance
func NewCategoryFilter() *CategoryFilter {
	return &CategoryFilter{
		logger: infra.GetLogger(),
	}
}

// CanHandle checks if this filter can handle the given intent
func (f *CategoryFilter) CanHandle(intent models.Intent) bool {
	return intent.Type == "category"
}

// Filter applies category-based filtering and sorts by publication date (most recent first)
func (f *CategoryFilter) Filter(articles []models.Article, params map[string]interface{}) ([]models.Article, error) {
	category, ok := params["category"].(string)
	if !ok || category == "" {
		return articles, nil
	}

	var filtered []models.Article
	for _, article := range articles {
		for _, cat := range article.Category {
			if cat == category {
				filtered = append(filtered, article)
				break
			}
		}
	}

	// Sort by publication date (most recent first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].PublicationDate.After(filtered[j].PublicationDate)
	})

	f.logger.Debug("Category filter applied", map[string]interface{}{
		"category":       category,
		"input_count":    len(articles),
		"filtered_count": len(filtered),
	})

	return filtered, nil
}

// SourceFilter filters articles by source name
type SourceFilter struct {
	logger infra.Logger
}

// NewSourceFilter creates a new SourceFilter instance
func NewSourceFilter() *SourceFilter {
	return &SourceFilter{
		logger: infra.GetLogger(),
	}
}

// CanHandle checks if this filter can handle the given intent
func (f *SourceFilter) CanHandle(intent models.Intent) bool {
	return intent.Type == "source"
}

// Filter applies source-based filtering and sorts by publication date (most recent first)
func (f *SourceFilter) Filter(articles []models.Article, params map[string]interface{}) ([]models.Article, error) {
	source, ok := params["source"].(string)
	if !ok || source == "" {
		return articles, nil
	}

	var filtered []models.Article
	for _, article := range articles {
		if article.SourceName == source {
			filtered = append(filtered, article)
		}
	}

	// Sort by publication date (most recent first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].PublicationDate.After(filtered[j].PublicationDate)
	})

	f.logger.Debug("Source filter applied", map[string]interface{}{
		"source":         source,
		"input_count":    len(articles),
		"filtered_count": len(filtered),
	})

	return filtered, nil
}

// ScoreFilter filters articles by relevance score threshold
type ScoreFilter struct {
	logger infra.Logger
}

// NewScoreFilter creates a new ScoreFilter instance
func NewScoreFilter() *ScoreFilter {
	return &ScoreFilter{
		logger: infra.GetLogger(),
	}
}

// CanHandle checks if this filter can handle the given intent
func (f *ScoreFilter) CanHandle(intent models.Intent) bool {
	return intent.Type == "score"
}

// Filter applies score-based filtering and sorts by relevance score (highest first)
func (f *ScoreFilter) Filter(articles []models.Article, params map[string]interface{}) ([]models.Article, error) {
	// Default threshold is 0.7 as per requirements
	threshold := 0.7

	if thresholdParam, ok := params["threshold"]; ok {
		switch v := thresholdParam.(type) {
		case float64:
			threshold = v
		case int:
			threshold = float64(v)
		}
	}

	var filtered []models.Article
	for _, article := range articles {
		if article.RelevanceScore >= threshold {
			filtered = append(filtered, article)
		}
	}

	// Sort by relevance score (highest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].RelevanceScore > filtered[j].RelevanceScore
	})

	f.logger.Debug("Score filter applied", map[string]interface{}{
		"threshold":      threshold,
		"input_count":    len(articles),
		"filtered_count": len(filtered),
	})

	return filtered, nil
}

// SearchFilter filters articles using text search via repository
type SearchFilter struct {
	repo   repositories.ArticleRepository
	logger infra.Logger
}

// NewSearchFilter creates a new SearchFilter instance
func NewSearchFilter(repo repositories.ArticleRepository) *SearchFilter {
	return &SearchFilter{
		repo:   repo,
		logger: infra.GetLogger(),
	}
}

// CanHandle checks if this filter can handle the given intent
func (f *SearchFilter) CanHandle(intent models.Intent) bool {
	return intent.Type == "search"
}

// Filter applies text search filtering with combined scoring (relevance + similarity)
func (f *SearchFilter) Filter(articles []models.Article, params map[string]interface{}) ([]models.Article, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return articles, nil
	}

	// Get search results from repository
	searchResults, err := f.repo.SearchByText(query)
	if err != nil {
		f.logger.Error("Search filter failed", err, map[string]interface{}{
			"query": query,
		})
		return nil, fmt.Errorf("search filter failed: %w", err)
	}

	// If we have input articles, intersect with search results
	if len(articles) > 0 {
		// Create a map of article IDs from search results for quick lookup
		searchMap := make(map[string]models.Article)
		for _, article := range searchResults {
			searchMap[article.ID] = article
		}

		// Filter input articles to only include those in search results
		var filtered []models.Article
		for _, article := range articles {
			if _, exists := searchMap[article.ID]; exists {
				filtered = append(filtered, article)
			}
		}

		// Sort by combined score: relevance_score * 0.3 + similarity * 0.7
		// Since we don't have actual vector similarity scores, we use relevance_score as proxy
		sort.Slice(filtered, func(i, j int) bool {
			scoreI := filtered[i].RelevanceScore
			scoreJ := filtered[j].RelevanceScore
			return scoreI > scoreJ
		})

		f.logger.Debug("Search filter applied with intersection", map[string]interface{}{
			"query":          query,
			"input_count":    len(articles),
			"search_count":   len(searchResults),
			"filtered_count": len(filtered),
		})

		return filtered, nil
	}

	// If no input articles, return search results directly
	// Sort by combined score (using relevance_score as proxy for now)
	sort.Slice(searchResults, func(i, j int) bool {
		return searchResults[i].RelevanceScore > searchResults[j].RelevanceScore
	})

	f.logger.Debug("Search filter applied", map[string]interface{}{
		"query":          query,
		"filtered_count": len(searchResults),
	})

	return searchResults, nil
}

// NearbyFilter filters articles by geographic proximity
type NearbyFilter struct {
	repo   repositories.ArticleRepository
	logger infra.Logger
}

// NewNearbyFilter creates a new NearbyFilter instance
func NewNearbyFilter(repo repositories.ArticleRepository) *NearbyFilter {
	return &NearbyFilter{
		repo:   repo,
		logger: infra.GetLogger(),
	}
}

// CanHandle checks if this filter can handle the given intent
func (f *NearbyFilter) CanHandle(intent models.Intent) bool {
	return intent.Type == "nearby"
}

// Filter applies geographic proximity filtering and sorts by distance (closest first)
func (f *NearbyFilter) Filter(articles []models.Article, params map[string]interface{}) ([]models.Article, error) {
	// Extract latitude and longitude
	lat, latOk := params["latitude"].(float64)
	lon, lonOk := params["longitude"].(float64)

	if !latOk || !lonOk {
		f.logger.Warn("Nearby filter missing location parameters", map[string]interface{}{
			"params": params,
		})
		return articles, nil
	}

	// Default radius is 50km
	radius := 50.0
	if radiusParam, ok := params["radius"]; ok {
		switch v := radiusParam.(type) {
		case float64:
			radius = v
		case int:
			radius = float64(v)
		}
	}

	// Get nearby results from repository (already sorted by distance)
	nearbyResults, err := f.repo.FilterArticles(types.FilterArticlesRequest{
		Lat:    lat,
		Lon:    lon,
		Radius: radius,
	})
	if err != nil {
		f.logger.Error("Nearby filter failed", err, map[string]interface{}{
			"latitude":  lat,
			"longitude": lon,
			"radius":    radius,
		})
		return nil, fmt.Errorf("nearby filter failed: %w", err)
	}

	// If we have input articles, intersect with nearby results
	if len(articles) > 0 {
		// Create a map of article IDs from nearby results for quick lookup
		nearbyMap := make(map[string]models.Article)
		for _, article := range nearbyResults {
			nearbyMap[article.ID] = article
		}

		// Filter input articles to only include those in nearby results
		var filtered []models.Article
		for _, article := range articles {
			if _, exists := nearbyMap[article.ID]; exists {
				filtered = append(filtered, article)
			}
		}

		// Sort by distance (closest first) using Haversine formula
		sort.Slice(filtered, func(i, j int) bool {
			distI := haversineDistance(lat, lon, filtered[i].Latitude, filtered[i].Longitude)
			distJ := haversineDistance(lat, lon, filtered[j].Latitude, filtered[j].Longitude)
			return distI < distJ
		})

		f.logger.Debug("Nearby filter applied with intersection", map[string]interface{}{
			"latitude":       lat,
			"longitude":      lon,
			"radius":         radius,
			"input_count":    len(articles),
			"nearby_count":   len(nearbyResults),
			"filtered_count": len(filtered),
		})

		return filtered, nil
	}

	// If no input articles, return nearby results directly (already sorted by distance from repository)
	f.logger.Debug("Nearby filter applied", map[string]interface{}{
		"latitude":       lat,
		"longitude":      lon,
		"radius":         radius,
		"filtered_count": len(nearbyResults),
	})

	return nearbyResults, nil
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
