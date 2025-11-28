package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/repositories"

	"github.com/redis/go-redis/v9"
)

// TrendingService defines the interface for trending news operations
type TrendingService interface {
	ComputeTrendingScore(article models.Article, location models.Location) (float64, error)
	GetCachedTrending(lat, lon float64, limit int) ([]models.Article, bool)
	CacheTrending(lat, lon float64, articles []models.Article)
}

// trendingService implements TrendingService
type trendingService struct {
	userEventRepo repositories.UserEventRepository
	log           infra.Logger
	redisClient   *redis.Client
	cacheTTL      time.Duration
	ctx           context.Context
}

// NewTrendingService creates a new instance of TrendingService
func NewTrendingService(userEventRepo repositories.UserEventRepository, redisClient *redis.Client, cacheTTL time.Duration) TrendingService {
	return &trendingService{
		userEventRepo: userEventRepo,
		log:           infra.GetLogger(),
		redisClient:   redisClient,
		cacheTTL:      cacheTTL,
		ctx:           context.Background(),
	}
}

// ComputeTrendingScore calculates the trending score for an article based on user engagement
// The score is computed using three factors:
// - Interaction volume (40%): Number of user events for the article
// - Recency (40%): How recent the article is
// - Geographic relevance (20%): Proximity to the query location
func (s *trendingService) ComputeTrendingScore(article models.Article, location models.Location) (float64, error) {
	// Query user events for this article from the last 7 days
	since := time.Now().Add(-7 * 24 * time.Hour)
	events, err := s.userEventRepo.FindByArticleID(article.ID, since)
	if err != nil {
		s.log.Error("Failed to retrieve user events for trending score", err, map[string]interface{}{
			"article_id": article.ID,
		})
		return 0, fmt.Errorf("failed to retrieve user events: %w", err)
	}

	// Calculate article age in hours
	articleAge := time.Since(article.PublicationDate)

	// Calculate distance between article location and query location using Haversine formula
	distance := s.calculateDistance(
		article.Latitude,
		article.Longitude,
		location.Latitude,
		location.Longitude,
	)

	// Compute individual score components
	volumeScore := s.computeVolumeScore(len(events))
	recencyScore := s.computeRecencyScore(articleAge)
	geoScore := s.computeGeoScore(distance)

	// Weighted combination: 40% volume, 40% recency, 20% geographic relevance
	trendingScore := (volumeScore * 0.4) + (recencyScore * 0.4) + (geoScore * 0.2)

	s.log.Debug("Computed trending score", map[string]interface{}{
		"article_id":     article.ID,
		"event_count":    len(events),
		"article_age_h":  articleAge.Hours(),
		"distance_km":    distance,
		"volume_score":   volumeScore,
		"recency_score":  recencyScore,
		"geo_score":      geoScore,
		"trending_score": trendingScore,
	})

	return trendingScore, nil
}

// computeVolumeScore calculates the volume component of the trending score
// Normalizes event count with a cap at 100 events
func (s *trendingService) computeVolumeScore(eventCount int) float64 {
	// Normalize to 0-1 range, capping at 100 events
	return math.Min(float64(eventCount)/100.0, 1.0)
}

// computeRecencyScore calculates the recency component of the trending score
// Uses exponential decay based on article age
func (s *trendingService) computeRecencyScore(articleAge time.Duration) float64 {
	// Convert to days
	ageDays := articleAge.Hours() / 24.0

	// Exponential decay: score decreases as article gets older
	// Articles older than ~7 days will have very low recency scores
	return 1.0 / (1.0 + ageDays)
}

// computeGeoScore calculates the geographic relevance component of the trending score
// Uses inverse distance weighting
func (s *trendingService) computeGeoScore(distanceKm float64) float64 {
	// Inverse distance weighting with 10km normalization factor
	// Articles within 10km get high scores, farther articles get lower scores
	return 1.0 / (1.0 + distanceKm/10.0)
}

// calculateDistance computes the distance between two geographic coordinates using Haversine formula
// Returns distance in kilometers
func (s *trendingService) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
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

// GetCachedTrending retrieves cached trending articles for a location
// Returns articles and true if cache hit, nil and false if cache miss
func (s *trendingService) GetCachedTrending(lat, lon float64, limit int) ([]models.Article, bool) {
	cacheKey := s.generateCacheKey(lat, lon, limit)

	// Get data from Redis
	val, err := s.redisClient.Get(s.ctx, cacheKey).Result()
	if err == redis.Nil {
		s.log.Debug("Cache miss for trending articles", map[string]interface{}{
			"cache_key": cacheKey,
		})
		return nil, false
	} else if err != nil {
		s.log.Warn("Failed to get cache from Redis", map[string]interface{}{
			"cache_key": cacheKey,
			"error":     err.Error(),
		})
		return nil, false
	}

	// Deserialize articles from JSON
	var articles []models.Article
	if err := json.Unmarshal([]byte(val), &articles); err != nil {
		s.log.Warn("Failed to unmarshal cached articles", map[string]interface{}{
			"cache_key": cacheKey,
			"error":     err.Error(),
		})
		// Delete invalid cache entry
		s.redisClient.Del(s.ctx, cacheKey)
		return nil, false
	}

	s.log.Debug("Cache hit for trending articles", map[string]interface{}{
		"cache_key": cacheKey,
		"count":     len(articles),
	})

	return articles, true
}

// CacheTrending stores trending articles in the cache with TTL
func (s *trendingService) CacheTrending(lat, lon float64, articles []models.Article) {
	cacheKey := s.generateCacheKey(lat, lon, len(articles))

	// Serialize articles to JSON
	data, err := json.Marshal(articles)
	if err != nil {
		s.log.Warn("Failed to marshal articles for cache", map[string]interface{}{
			"cache_key": cacheKey,
			"error":     err.Error(),
		})
		return
	}

	// Store in Redis with TTL
	if err := s.redisClient.Set(s.ctx, cacheKey, data, s.cacheTTL).Err(); err != nil {
		s.log.Warn("Failed to cache articles in Redis", map[string]interface{}{
			"cache_key": cacheKey,
			"error":     err.Error(),
		})
		return
	}

	s.log.Debug("Cached trending articles", map[string]interface{}{
		"cache_key": cacheKey,
		"count":     len(articles),
		"ttl":       s.cacheTTL,
	})
}

// generateCacheKey creates a cache key from rounded coordinates and limit
// Rounds coordinates to 2 decimal places (~1km precision)
func (s *trendingService) generateCacheKey(lat, lon float64, limit int) string {
	// Round to 2 decimal places for ~1km precision
	latRounded := math.Round(lat*100) / 100
	lonRounded := math.Round(lon*100) / 100

	return fmt.Sprintf("trending:%.2f:%.2f:%d", latRounded, lonRounded, limit)
}
