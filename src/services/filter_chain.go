package services

import (
	"fmt"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
)

// ArticleFilter defines the interface for filtering articles based on intent
type ArticleFilter interface {
	// Filter applies the filter logic to a collection of articles
	Filter(articles []models.Article, params map[string]interface{}) ([]models.Article, error)

	// CanHandle determines if this filter can handle the given intent
	CanHandle(intent models.Intent) bool
}

// FilterChain manages and executes a chain of article filters
type FilterChain struct {
	filters []ArticleFilter
	logger  infra.Logger
}

// NewFilterChain creates a new FilterChain instance
func NewFilterChain() *FilterChain {
	return &FilterChain{
		filters: make([]ArticleFilter, 0),
		logger:  infra.GetLogger(),
	}
}

// NewFilterChainWithDefaults creates a FilterChain with all default filters registered
// This is a convenience function that initializes all available filters
func NewFilterChainWithDefaults(articleRepo repositories.ArticleRepository) *FilterChain {
	chain := NewFilterChain()
	chain.RegisterDefaultFilters(articleRepo)
	return chain
}

// RegisterDefaultFilters registers all default filters to the chain
// This method can be used to add all standard filters at once
func (fc *FilterChain) RegisterDefaultFilters(articleRepo repositories.ArticleRepository) {
	fc.RegisterFilter(NewCategoryFilter())
	fc.RegisterFilter(NewSourceFilter())
	fc.RegisterFilter(NewScoreFilter())
	fc.RegisterFilter(NewSearchFilter(articleRepo))
	fc.RegisterFilter(NewNearbyFilter(articleRepo))
}

// RegisterFilter adds a filter to the chain
func (fc *FilterChain) RegisterFilter(filter ArticleFilter) {
	fc.filters = append(fc.filters, filter)
	fc.logger.Debug("Registered filter", map[string]interface{}{
		"filter_count": len(fc.filters),
	})
}

// Execute applies all applicable filters based on the provided intents
func (fc *FilterChain) Execute(articles []models.Article, intents []models.Intent, location *models.Location) ([]models.Article, error) {
	if len(articles) == 0 {
		fc.logger.Debug("No articles to filter", nil)
		return articles, nil
	}

	if len(intents) == 0 {
		fc.logger.Debug("No intents provided, returning all articles", nil)
		return articles, nil
	}

	result := articles

	fc.logger.Info("Starting filter chain execution", map[string]interface{}{
		"initial_count": len(articles),
		"intent_count":  len(intents),
	})

	// Apply filters for each intent
	for _, intent := range intents {
		// Add location to parameters if this is a nearby intent
		params := intent.Parameters
		if params == nil {
			params = make(map[string]interface{})
		}

		if intent.Type == "nearby" && location != nil {
			params["latitude"] = location.Latitude
			params["longitude"] = location.Longitude
		}

		// Find and apply the appropriate filter
		filterApplied := false
		for _, filter := range fc.filters {
			if filter.CanHandle(intent) {
				fc.logger.Debug("Applying filter", map[string]interface{}{
					"intent_type":    intent.Type,
					"articles_count": len(result),
				})

				filtered, err := filter.Filter(result, params)
				if err != nil {
					fc.logger.Error("Filter execution failed", err, map[string]interface{}{
						"intent_type": intent.Type,
					})
					return nil, fmt.Errorf("filter execution failed for intent %s: %w", intent.Type, err)
				}

				result = filtered
				filterApplied = true

				fc.logger.Info("Filter applied", map[string]interface{}{
					"intent_type":     intent.Type,
					"remaining_count": len(result),
				})

				break
			}
		}

		if !filterApplied {
			fc.logger.Warn("No filter found for intent", map[string]interface{}{
				"intent_type": intent.Type,
			})
		}
	}

	fc.logger.Info("Filter chain execution completed", map[string]interface{}{
		"initial_count": len(articles),
		"final_count":   len(result),
	})

	return result, nil
}
