package services

import (
	"context"
	"fmt"
	"strconv"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
)

// Filter defines the function type for filtering articles
type Filter func(ctx context.Context, in *[]models.Article) (*[]models.Article, error)

// Chain composes multiple filters into a single filter pipeline
func Chain(ctx context.Context, filters ...Filter) ([]models.Article, error) {
	articles := []models.Article{}

	for _, filter := range filters {
		filteredArticles, err := filter(ctx, &articles)
		if err != nil {
			return nil, err
		}
		articles = *filteredArticles
		fmt.Println("Retrived articles after filter: ", len(articles), "with filter: ", filter)
	}
	return articles, nil
}

// FilterFactory is a function that creates a Filter from intent parameters
type FilterFactory func(params map[string]interface{}) Filter

// FilterChain manages and executes a chain of article filters
type FilterChain struct {
	filterRegistry map[string]FilterFactory
	articleRepo    repositories.ArticleRepository
	llmService     LLMService
	logger         infra.Logger
}

// NewFilterChain creates a new FilterChain instance
func NewFilterChain(articleRepo repositories.ArticleRepository, llmService LLMService) *FilterChain {
	chain := &FilterChain{
		filterRegistry: make(map[string]FilterFactory),
		articleRepo:    articleRepo,
		llmService:     llmService,
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
		sources := []string{}
		if s, ok := params["source"].([]string); ok {
			sources = s
		}
		return FilterBySource(fc.articleRepo, sources)

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
		return FilterByTextSearch(fc.articleRepo, fc.llmService, query)
	}
	fc.filterRegistry[models.IntentTypeNearby] = func(params map[string]interface{}) Filter {
		lat := 0.0
		lon := 0.0
		radius := 50.0

		if latitude, err := strconv.ParseFloat(params["latitude"].(string), 64); err == nil {
			lat = latitude
		}
		if longitude, err := strconv.ParseFloat(params["longitude"].(string), 64); err == nil {
			lon = longitude
		}
		if _, ok := params["radius"]; ok {
			if r, err := strconv.ParseFloat(params["radius"].(string), 64); err == nil {
				radius = r
			}
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
			if sources, ok := intent.Values.([]string); ok {
				params["source"] = sources
			} else {
				fc.logger.Error("Invalid source values", nil, map[string]interface{}{"intent": intent.Type})
			}
		case models.IntentTypeNearby:
			values, ok := intent.Values.([]string)
			if !ok || len(values) < 2 {
				fc.logger.Error("Invalid nearby values", nil, map[string]interface{}{"intent": intent.Type})
				continue
			}
			params["latitude"] = values[0]
			params["longitude"] = values[1]
		}

		filter := factory(params)
		filters = append(filters, filter)
	}
	if len(filters) > 0 {
		filters = append(filters, FilterByTextSearch(fc.articleRepo, fc.llmService, entities))
		filters = append(filters, FilterByScore(fc.articleRepo, 0.1))
	}
	ctx := context.Background()
	return Chain(ctx, filters...)
}
