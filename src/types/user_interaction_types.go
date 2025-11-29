package types

import (
	"fmt"

	"news-inshorts/src/models"
)

// RecordInteractionRequest represents the request body for POST /api/v1/interactions
type RecordInteractionRequest struct {
	UserID    string          `json:"user_id" validate:"required"`
	ArticleID string          `json:"article_id" validate:"required"`
	EventType string          `json:"event_type" validate:"required,oneof=view click"`
	Location  models.Location `json:"location" validate:"required"`
}

// Validate validates the RecordInteractionRequest
func (r *RecordInteractionRequest) Validate() error {
	if r.UserID == "" {
		return fmt.Errorf("user_id field is required")
	}

	if r.ArticleID == "" {
		return fmt.Errorf("article_id field is required")
	}

	if r.EventType == "" {
		return fmt.Errorf("event_type field is required")
	}

	if r.EventType != "view" && r.EventType != "click" {
		return fmt.Errorf("event_type must be either 'view' or 'click'")
	}

	if r.Location.Latitude < -90 || r.Location.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}

	if r.Location.Longitude < -180 || r.Location.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}

	return nil
}

// RecordInteractionResponse represents the response for interaction recording endpoint
type RecordInteractionResponse struct {
	Success bool   `json:"success"`
	EventID string `json:"event_id"`
}
