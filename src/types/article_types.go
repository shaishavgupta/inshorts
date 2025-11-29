package types

import (
	"fmt"

	"news-inshorts/src/models"
)

// QueryArticlesRequest represents the query parameters for GET /api/v1/news/query
type QueryArticlesRequest struct {
	Query    string           `query:"query" validate:"required"`
	Lat      float64          `query:"lat" validate:"omitempty,min=-90,max=90"`
	Lon      float64          `query:"lon" validate:"omitempty,min=-180,max=180"`
	Location *models.Location `json:"-"` // Computed field, not from query params
}

func (r *QueryArticlesRequest) Validate() error {
	// Validate required fields
	if r.Query == "" {
		return fmt.Errorf("query parameter is required")
	}

	// Build Location object if lat/lon are provided
	// Check if at least one is provided (non-zero)
	hasLat := r.Lat != 0
	hasLon := r.Lon != 0

	if hasLat || hasLon {
		// Both lat and lon must be provided together
		if !hasLat || !hasLon {
			return fmt.Errorf("both lat and lon must be provided together")
		}
		if r.Lat < -90 || r.Lat > 90 {
			return fmt.Errorf("latitude must be between -90 and 90")
		}
		if r.Lon < -180 || r.Lon > 180 {
			return fmt.Errorf("longitude must be between -180 and 180")
		}
		r.Location = &models.Location{
			Latitude:  r.Lat,
			Longitude: r.Lon,
		}
	}
	return nil
}

// QueryArticlesResponse represents the response for news query endpoint
type QueryArticlesResponse struct {
	Articles []models.Article `json:"articles"`
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
	Category       string  `json:"category" query:"category" validate:"omitempty"`
	Source         string  `json:"source" query:"source" validate:"omitempty"`
	Lat            float64 `json:"lat" query:"lat" validate:"omitempty,min=-90,max=90"`
	Lon            float64 `json:"lon" query:"lon" validate:"omitempty,min=-180,max=180"`
	Radius         float64 `json:"radius" query:"radius" validate:"omitempty,min=0"`
	ScoreThreshold float64 `json:"score_threshold" query:"score_threshold" validate:"omitempty,min=0,max=1"`
}

// Validate validates the FilterArticlesRequest
// At least one filter (category, source, lat/lon, or score_threshold) must be provided
func (r *FilterArticlesRequest) Validate() error {
	// Check that at least one filter is provided
	if r.Category == "" && r.Source == "" && (r.Lat == 0 || r.Lon == 0) && r.ScoreThreshold == 0 {
		return fmt.Errorf("at least one filter parameter must be provided: category, source, lat/lon, or score_threshold")
	}

	// Validate latitude if provided
	if r.Lat != 0 || r.Lon != 0 {
		if r.Lat < -90 || r.Lat > 90 || r.Lon < -180 || r.Lon > 180 {
			return fmt.Errorf("latitude and longitude must be between -90 and 90 and -180 and 180 respectively")
		}
	}

	// Validate score threshold if provided
	if r.ScoreThreshold != 0 {
		if r.ScoreThreshold < 0 || r.ScoreThreshold > 1 {
			return fmt.Errorf("score_threshold must be between 0 and 1")
		}
	}

	return nil
}

// FilterArticlesResponse represents the response for the filter articles endpoint
type FilterArticlesResponse struct {
	Articles []models.Article `json:"articles"`
}

// CreateArticleRequest represents the request body for POST /api/v1/news
type CreateArticleRequest struct {
	Title           string   `json:"title" validate:"required"`
	Description     string   `json:"description"`
	URL             string   `json:"url" validate:"required,url"`
	PublicationDate string   `json:"publication_date" validate:"required"`
	SourceName      string   `json:"source_name" validate:"required"`
	Category        []string `json:"category" validate:"required,min=1"`
	RelevanceScore  float64  `json:"relevance_score" validate:"required,min=0,max=1"`
	Latitude        float64  `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude       float64  `json:"longitude" validate:"required,min=-180,max=180"`
	Summary         string   `json:"summary"`
}

// Validate validates the CreateArticleRequest
func (r *CreateArticleRequest) Validate() error {
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if r.URL == "" {
		return fmt.Errorf("url is required")
	}
	if r.SourceName == "" {
		return fmt.Errorf("source_name is required")
	}
	if len(r.Category) == 0 {
		return fmt.Errorf("at least one category is required")
	}
	if r.RelevanceScore < 0 || r.RelevanceScore > 1 {
		return fmt.Errorf("relevance_score must be between 0 and 1")
	}
	if r.Latitude < -90 || r.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if r.Longitude < -180 || r.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	if r.PublicationDate == "" {
		return fmt.Errorf("publication_date is required")
	}
	return nil
}

// CreateArticleResponse represents the response for the create article endpoint
type CreateArticleResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Article models.Article `json:"article"`
}

// GetTrendingRequest represents the query parameters for GET /api/v1/news/trending
type GetTrendingRequest struct {
	Lat   float64 `query:"lat" validate:"omitempty,min=-90,max=90"`
	Lon   float64 `query:"lon" validate:"omitempty,min=-180,max=180"`
	Limit int     `query:"limit" validate:"omitempty,min=1,max=100"`
}

// ErrorResponse represents a standardized error response with error code
type ErrorResponse struct {
	ErrorCode string `json:"error_code"`
	Error     string `json:"error"`
}

// Validate validates the GetTrendingRequest
func (r *GetTrendingRequest) Validate() error {
	// Validate latitude if provided
	if r.Lat != 0 {
		if r.Lat < -90 || r.Lat > 90 {
			return fmt.Errorf("latitude must be between -90 and 90")
		}
	}

	// Validate longitude if provided
	if r.Lon != 0 {
		if r.Lon < -180 || r.Lon > 180 {
			return fmt.Errorf("longitude must be between -180 and 180")
		}
	}

	// Set default limit if not provided
	if r.Limit == 0 {
		r.Limit = 10
	}

	// Validate limit
	if r.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0")
	}

	// Cap limit at 100
	if r.Limit > 100 {
		r.Limit = 100
	}

	return nil
}
