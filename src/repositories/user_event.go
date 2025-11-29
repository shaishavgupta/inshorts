package repositories

import (
	"fmt"
	"time"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserEventRepository defines the interface for user event data access
type UserEventRepository interface {
	Create(event *models.UserEvent) error
	FindByArticleID(articleID string, since time.Time) ([]models.UserEvent, error)
	FindByLocation(lat, lon, radiusKm float64, since time.Time) ([]models.UserEvent, error)
	GetArticlesFromUserEvents() ([]string, error)
}

// userEventRepository implements UserEventRepository
type userEventRepository struct {
	db  *gorm.DB
	log infra.Logger
}

// NewUserEventRepository creates a new instance of UserEventRepository
func NewUserEventRepository(db *gorm.DB) UserEventRepository {
	return &userEventRepository{
		db:  db,
		log: infra.GetLogger(),
	}
}

// Create stores a new user event in the database
func (r *userEventRepository) Create(event *models.UserEvent) error {
	// Generate UUID if not provided
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	query := `
		INSERT INTO user_events (
			id,
			user_id,
			article_id,
			event_type,
			timestamp,
			latitude,
			longitude
		) VALUES (
			COALESCE(?::uuid, uuid_generate_v4()),
			?,
			?::uuid,
			?,
			?,
			?,
			?
		)
	`

	if err := r.db.Exec(query,
		event.ID,
		event.UserID,
		event.ArticleID,
		event.EventType,
		event.Timestamp,
		event.Latitude,
		event.Longitude,
	).Error; err != nil {
		r.log.Error("Failed to create user event", err, map[string]interface{}{
			"user_id":    event.UserID,
			"article_id": event.ArticleID,
			"event_type": event.EventType,
		})
		return fmt.Errorf("failed to create user event: %w", err)
	}

	r.log.Info("Created user event", map[string]interface{}{
		"event_id":   event.ID,
		"user_id":    event.UserID,
		"article_id": event.ArticleID,
		"event_type": event.EventType,
	})

	return nil
}

// FindByArticleID retrieves user events for a specific article with time filtering
func (r *userEventRepository) FindByArticleID(articleID string, since time.Time) ([]models.UserEvent, error) {
	query := `
		SELECT
			id,
			user_id,
			article_id,
			event_type,
			timestamp,
			latitude,
			longitude
		FROM user_events
		WHERE article_id = ?::uuid
			AND timestamp >= ?
		ORDER BY timestamp DESC
	`

	var events []models.UserEvent
	if err := r.db.Raw(query, articleID, since).Scan(&events).Error; err != nil {
		r.log.Error("Failed to query user events by article ID", err, map[string]interface{}{
			"article_id": articleID,
			"since":      since,
		})
		return nil, fmt.Errorf("failed to query user events by article ID: %w", err)
	}

	r.log.Info("Retrieved user events by article ID", map[string]interface{}{
		"article_id": articleID,
		"since":      since,
		"count":      len(events),
	})

	return events, nil
}

// FindByLocation retrieves user events within a specified radius using PostGIS spatial queries
func (r *userEventRepository) FindByLocation(lat, lon, radiusKm float64, since time.Time) ([]models.UserEvent, error) {
	query := `
		SELECT
			id,
			user_id,
			article_id,
			event_type,
			timestamp,
			latitude,
			longitude,
			ST_Distance(
				ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography,
				ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography
			) / 1000.0 as distance_km
		FROM user_events
		WHERE ST_DWithin(
			ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography,
			ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
			? * 1000
		)
		AND timestamp >= ?
		ORDER BY timestamp DESC
	`

	var events []models.UserEvent
	if err := r.db.Raw(query, lon, lat, lon, lat, radiusKm, since).Scan(&events).Error; err != nil {
		r.log.Error("Failed to query user events by location", err, map[string]interface{}{
			"latitude":  lat,
			"longitude": lon,
			"radius_km": radiusKm,
			"since":     since,
		})
		return nil, fmt.Errorf("failed to query user events by location: %w", err)
	}

	r.log.Info("Retrieved user events by location", map[string]interface{}{
		"latitude":  lat,
		"longitude": lon,
		"radius_km": radiusKm,
		"since":     since,
		"count":     len(events),
	})

	return events, nil
}

// GetArticlesFromUserEvents retrieves all distinct article IDs from user_events
func (r *userEventRepository) GetArticlesFromUserEvents() ([]string, error) {
	query := `
		SELECT DISTINCT article_id
		FROM user_events
		ORDER BY article_id
	`

	var articleIDs []string
	if err := r.db.Raw(query).Scan(&articleIDs).Error; err != nil {
		r.log.Error("Failed to get distinct article IDs from user events", err, nil)
		return nil, fmt.Errorf("failed to get distinct article IDs: %w", err)
	}

	r.log.Info("Retrieved distinct article IDs from user events", map[string]interface{}{
		"count": len(articleIDs),
	})

	return articleIDs, nil
}
