package services

import (
	"context"
	"fmt"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
)

// Filter defines the function type for filtering articles
type Filter func(ctx context.Context, in []models.Article) ([]models.Article, error)

// Chain composes multiple filters into a single filter pipeline
func Chain(filters ...Filter) Filter {
	return func(ctx context.Context, articles []models.Article) ([]models.Article, error) {
		var err error
		for _, f := range filters {
			articles, err = f(ctx, articles)
			if err != nil {
				return nil, err
			}
		}
		return articles, nil
	}
}

// FilterFactory is a function that creates a Filter from intent parameters
type FilterFactory func(params map[string]interface{}) Filter

// FilterChain manages and executes a chain of article filters
type FilterChain struct {
	filterRegistry map[string]FilterFactory
	articleRepo    repositories.ArticleRepository
	logger         infra.Logger
}

// NewFilterChain creates a new FilterChain instance
// If articleRepo is provided, it will register all default filters
// If articleRepo is nil, returns an empty FilterChain without default filters
func NewFilterChain(articleRepo repositories.ArticleRepository) *FilterChain {
	chain := &FilterChain{
		filterRegistry: make(map[string]FilterFactory),
		articleRepo:    articleRepo,
	}

	if articleRepo != nil {
		chain.RegisterDefaultFilters()
	}

	return chain
}

// RegisterDefaultFilters registers all default filters to the chain
// This method maps intent types (keywords) to their filter factory functions
func (fc *FilterChain) RegisterDefaultFilters() {
	fc.filterRegistry[models.IntentTypeCategory] = func(params map[string]interface{}) Filter {
		category := ""
		if c, ok := params["category"].(string); ok {
			category = c
		}
		return FilterByCategory(fc.articleRepo, category)
	}
	fc.filterRegistry[models.IntentTypeSource] = func(params map[string]interface{}) Filter {
		source := ""
		if s, ok := params["source"].(string); ok {
			source = s
		}
		return FilterBySource(fc.articleRepo, source)
	}
	fc.filterRegistry[models.IntentTypeScore] = func(params map[string]interface{}) Filter {
		threshold := 0.7
		if t, ok := params["threshold"]; ok {
			switch v := t.(type) {
			case float64:
				threshold = v
			case int:
				threshold = float64(v)
			}
		}
		return FilterByScore(fc.articleRepo, threshold)
	}
	fc.filterRegistry[models.EntityTypeSearch] = func(params map[string]interface{}) Filter {
		var query []string
		if q, ok := params["query"]; ok {
			switch v := q.(type) {
			case string:
				if v != "" {
					query = []string{v}
				}
			case []string:
				query = v
			case []interface{}:
				query = make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok && str != "" {
						query = append(query, str)
					}
				}
			}
		}
		return FilterByTextSearch(fc.articleRepo, query)
	}
	fc.filterRegistry[models.IntentTypeNearby] = func(params map[string]interface{}) Filter {
		lat := 0.0
		lon := 0.0
		radius := 50.0

		if latitude, ok := params["latitude"].(float64); ok {
			lat = latitude
		}
		if longitude, ok := params["longitude"].(float64); ok {
			lon = longitude
		}
		if r, ok := params["radius"].(float64); ok {
			radius = r
		}
		return FilterByRadius(fc.articleRepo, lat, lon, radius)
	}
}

// Execute applies all applicable filters based on the provided intents
func (fc *FilterChain) Execute(intents []models.Intent, entities []string, location *models.Location) ([]models.Article, error) {
	if len(intents) == 0 && len(entities) == 0 && location == nil {
		return fc.articleRepo.FindAll()
	}

	// Build filter chain from intents
	var filters []Filter
	if len(entities) > 0 {
		filters = append(filters, FilterByTextSearch(fc.articleRepo, entities))
	}

	for _, intent := range intents {
		// Add location to parameters if this is a nearby intent
		params := intent.Parameters
		if params == nil {
			params = make(map[string]interface{})
		}

		// Add entities to parameters if this is a category intent
		if intent.Type == models.IntentTypeCategory {
			params["entities"] = entities
		}

		if intent.Type == models.IntentTypeNearby && location != nil {
			params["latitude"] = location.Latitude
			params["longitude"] = location.Longitude
		}

		// Get filter factory for this intent type
		factory, exists := fc.filterRegistry[intent.Type]
		if !exists {
			fc.logger.Error("Filter factory not found for intent type", nil, map[string]interface{}{"intent": intent.Type})
			continue
		}

		// Create filter from factory
		filter := factory(params)
		filters = append(filters, filter)
	}
	filters = append(filters, FilterByScore(fc.articleRepo, 0.7))
	fc.logger.Info("Filters", map[string]interface{}{"count": len(filters)})

	// Execute the filter chain
	ctx := context.Background()
	pipeline := Chain(filters...)
	var articles []models.Article
	result, err := pipeline(ctx, articles)
	if err != nil {
		return nil, fmt.Errorf("filter chain execution failed: %w", err)
	}

	return result, nil
}
