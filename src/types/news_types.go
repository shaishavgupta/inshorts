package types

import (
	"fmt"

	"news-inshorts/src/models"
)

// QueryNewsRequest represents the request body for POST /api/v1/news/query
type QueryNewsRequest struct {
	Query    string           `json:"query" validate:"required"`
	Location *models.Location `json:"location"`
}

// QueryNewsResponse represents the response for news query endpoint
type QueryNewsResponse struct {
	Articles []models.EnrichedArticle `json:"articles"`
}

// LoadDataRequest represents the request body for POST /api/v1/news/load
type LoadDataRequest struct {
	Filepath string `json:"filepath" validate:"required"`
}

// LoadDataResponse represents the response for data loading endpoint
type LoadDataResponse struct {
	Success          bool     `json:"success"`
	Message          string   `json:"message"`
	TotalArticles    int      `json:"total_articles"`
	SuccessCount     int      `json:"success_count"`
	ErrorCount       int      `json:"error_count"`
	ValidationErrors []string `json:"validation_errors,omitempty"`
}

// FilterArticlesRequest represents the query parameters for GET /api/v1/news/filter
type FilterArticlesRequest struct {
	Category string  `json:"category" query:"category" validate:"omitempty"`
	Source   string  `json:"source" query:"source" validate:"omitempty"`
	Lat      float64 `json:"lat" query:"lat" validate:"omitempty,min=-90,max=90"`
	Lon      float64 `json:"lon" query:"lon" validate:"omitempty,min=-180,max=180"`
	Radius   float64 `json:"radius" query:"radius" validate:"omitempty,min=0"`
}

// Validate validates the FilterArticlesRequest
// At least one filter (category, source, or lat/lon) must be provided
func (r *FilterArticlesRequest) Validate() error {
	// Check that at least one filter is provided
	if r.Category == "" && r.Source == "" && (r.Lat == 0 || r.Lon == 0) {
		return fmt.Errorf("at least one filter parameter must be provided: category, source, or lat/lon")
	}

	// Validate latitude if provided
	if r.Lat != 0 || r.Lon != 0 {
		if r.Lat < -90 || r.Lat > 90 || r.Lon < -180 || r.Lon > 180 {
			return fmt.Errorf("latitude and longitude must be between -90 and 90 and -180 and 180 respectively")
		}
	}

	return nil
}
