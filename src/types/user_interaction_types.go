package types

import "news-inshorts/src/models"

// RecordInteractionRequest represents the request body for POST /api/v1/interactions
type RecordInteractionRequest struct {
	UserID    string          `json:"user_id" validate:"required"`
	ArticleID string          `json:"article_id" validate:"required"`
	EventType string          `json:"event_type" validate:"required,oneof=view click"`
	Location  models.Location `json:"location" validate:"required"`
}

// RecordInteractionResponse represents the response for interaction recording endpoint
type RecordInteractionResponse struct {
	Success bool   `json:"success"`
	EventID string `json:"event_id"`
}
