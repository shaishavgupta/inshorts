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
func NewFilterChain(articleRepo repositories.ArticleRepository) *FilterChain {
	chain := &FilterChain{
		filterRegistry: make(map[string]FilterFactory),
		articleRepo:    articleRepo,
		logger:         infra.GetLogger(),
	}

	if articleRepo != nil {
		chain.RegisterDefaultFilters()
	}

	return chain
}

// RegisterDefaultFilters registers all default filters to the chain
func (fc *FilterChain) RegisterDefaultFilters() {
	fc.filterRegistry[models.IntentTypeCategory] = func(params map[string]interface{}) Filter {
		var categories []string
		if c, ok := params["category"].(string); ok {
			categories = []string{c}
		} else if cats, ok := params["category"].([]string); ok {
			categories = cats
		}
		return FilterByCategory(fc.articleRepo, categories)
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

	var filters []Filter
	if len(entities) > 0 {
		filters = append(filters, FilterByTextSearch(fc.articleRepo, entities))
	}

	for _, intent := range intents {
		factory, exists := fc.filterRegistry[intent.Type]
		if !exists {
			fc.logger.Error("Unknown intent type", nil, map[string]interface{}{"intent": intent.Type})
			continue
		}

		// Convert intent.Values and location into params map for the factory
		params := make(map[string]interface{})

		switch intent.Type {
		case models.IntentTypeCategory:
			// Handle both single string and []string
			if categories, ok := intent.Values.([]string); ok {
				params["category"] = categories
			} else if category, ok := intent.Values.(string); ok {
				params["category"] = category
			} else {
				fc.logger.Error("Invalid category values", nil, map[string]interface{}{"intent": intent.Type})
				continue
			}
		case models.IntentTypeSource:
			if source, ok := intent.Values.(string); ok {
				params["source"] = source
			} else {
				fc.logger.Error("Invalid source values", nil, map[string]interface{}{"intent": intent.Type})
				continue
			}
		case models.IntentTypeNearby:
			if location != nil {
				params["latitude"] = location.Latitude
				params["longitude"] = location.Longitude
				params["radius"] = 50.0
			} else {
				fc.logger.Error("Location required for nearby intent", nil, map[string]interface{}{"intent": intent.Type})
				continue
			}
		case models.IntentTypeScore:
			if threshold, ok := intent.Values.(float64); ok {
				params["threshold"] = threshold
			} else if threshold, ok := intent.Values.(int); ok {
				params["threshold"] = float64(threshold)
			}
		case models.EntityTypeSearch:
			if query, ok := intent.Values.(string); ok {
				params["query"] = query
			} else if queries, ok := intent.Values.([]string); ok {
				params["query"] = queries
			} else if queries, ok := intent.Values.([]interface{}); ok {
				params["query"] = queries
			}
		}

		filter := factory(params)
		filters = append(filters, filter)
	}
	filters = append(filters, FilterByScore(fc.articleRepo, 0.7))
	fmt.Println("filters", filters)

	ctx := context.Background()
	pipeline := Chain(filters...)
	var articles []models.Article
	result, err := pipeline(ctx, articles)
	if err != nil {
		return nil, fmt.Errorf("filter chain execution failed: %w", err)
	}

	return result, nil
}
