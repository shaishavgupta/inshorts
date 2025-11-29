package models

import (
	"encoding/json"
	"time"
)

// Location represents geographic coordinates
type Location struct {
	Latitude  float64 `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" validate:"required,min=-180,max=180"`
}

// Intent type constants
const (
	IntentTypeCategory = "category"
	IntentTypeScore    = "score"
	EntityTypeSearch   = "search"
	IntentTypeSource   = "source"
	IntentTypeNearby   = "nearby"
)

// Intent represents the determined purpose or retrieval strategy for a user query
type Intent struct {
	Type   string      `json:"type" validate:"required,oneof=category source nearby"`
	Values interface{} `json:"values" validate:"required,min=1"`
}

// QueryAnalysis represents the result of LLM query processing
type QueryAnalysis struct {
	Entities []string `json:"entities"`
	Intents  []Intent `json:"intents" validate:"required,min=1"`
}

// Article represents a news article stored in the database
type Article struct {
	ID                string    `json:"id" db:"id"`
	Title             string    `json:"title" db:"title" validate:"required"`
	Description       string    `json:"description" db:"description"`
	URL               string    `json:"url" db:"url" validate:"required,url"`
	PublicationDate   time.Time `json:"publication_date" db:"publication_date" validate:"required"`
	SourceName        string    `json:"source_name" db:"source_name" validate:"required"`
	Category          []string  `json:"category" db:"category" validate:"required,min=1"`
	RelevanceScore    float64   `json:"relevance_score" db:"relevance_score" validate:"required,min=0,max=1"`
	Latitude          float64   `json:"latitude" db:"latitude" validate:"required,min=-90,max=90"`
	Longitude         float64   `json:"longitude" db:"longitude" validate:"required,min=-180,max=180"`
	Summary           string    `json:"summary" db:"summary"`
	DescriptionVector []float64 `json:"-" db:"description_vector"`
}

// UnmarshalJSON implements json.Unmarshaler for Article
func (a *Article) UnmarshalJSON(data []byte) error {
	type Alias Article
	aux := &struct {
		PublicationDate string `json:"publication_date"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.PublicationDate != "" {
		parsedTime, err := time.Parse("2006-01-02T15:04:05", aux.PublicationDate)
		if err != nil {
			return err
		}
		a.PublicationDate = parsedTime
	}

	return nil
}

// UserEvent represents a user interaction with an article
type UserEvent struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id" validate:"required"`
	ArticleID string    `json:"article_id" db:"article_id" validate:"required"`
	EventType string    `json:"event_type" db:"event_type" validate:"required,oneof=view click"`
	Timestamp time.Time `json:"timestamp" db:"timestamp" validate:"required"`
	Latitude  float64   `json:"latitude" db:"latitude" validate:"required,min=-90,max=90"`
	Longitude float64   `json:"longitude" db:"longitude" validate:"required,min=-180,max=180"`
}

// GetLocation returns the Location for a UserEvent
func (ue *UserEvent) GetLocation() Location {
	return Location{
		Latitude:  ue.Latitude,
		Longitude: ue.Longitude,
	}
}

// GetLocation returns the Location for an Article
func (a *Article) GetLocation() Location {
	return Location{
		Latitude:  a.Latitude,
		Longitude: a.Longitude,
	}
}

// HasIntent checks if QueryAnalysis contains a specific intent type
func (qa *QueryAnalysis) HasIntent(intentType string) bool {
	for _, intent := range qa.Intents {
		if intent.Type == intentType {
			return true
		}
	}
	return false
}

// GetIntent retrieves the first intent of a specific type from QueryAnalysis
func (qa *QueryAnalysis) GetIntent(intentType string) *Intent {
	for i := range qa.Intents {
		if qa.Intents[i].Type == intentType {
			return &qa.Intents[i]
		}
	}
	return nil
}
