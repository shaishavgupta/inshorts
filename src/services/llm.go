package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
)

// LLMService defines the interface for LLM operations
type LLMService interface {
	ProcessQuery(query string) (*models.QueryAnalysis, error)
	GenerateSummary(title, description string) (string, error)
}

// llmService implements the LLMService interface
type llmService struct {
	config     *infra.LLMConfig
	httpClient *http.Client
	logger     infra.Logger
}

// NewLLMService creates a new LLM service instance
func NewLLMService(cfg *infra.LLMConfig) LLMService {
	return &llmService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: infra.GetLogger(),
	}
}

// openAIRequest represents the request structure for OpenAI API
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

// openAIMessage represents a message in the OpenAI API request
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse represents the response structure from OpenAI API
type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// ProcessQuery analyzes a user query using LLM to extract entities and intents
func (s *llmService) ProcessQuery(query string) (*models.QueryAnalysis, error) {
	prompt := s.buildQueryAnalysisPrompt(query)

	response, err := s.callOpenAI(prompt, 500)
	if err != nil {
		s.logger.Error("Failed to process query with LLM", err, map[string]interface{}{
			"query": query,
		})
		return nil, fmt.Errorf("LLM service unavailable: %w", err)
	}

	// Parse the LLM response to extract QueryAnalysis
	analysis, err := s.parseQueryAnalysis(response)
	if err != nil {
		s.logger.Error("Failed to parse LLM response", err, map[string]interface{}{
			"response": response,
		})
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	s.logger.Info("Successfully processed query", map[string]interface{}{
		"query":          query,
		"entities_count": len(analysis.Entities),
		"intents_count":  len(analysis.Intents),
	})

	return analysis, nil
}

// GenerateSummary generates a summary for an article using LLM
func (s *llmService) GenerateSummary(title, description string) (string, error) {
	prompt := s.buildSummaryPrompt(title, description)

	response, err := s.callOpenAI(prompt, 150)
	if err != nil {
		s.logger.Warn("Failed to generate summary with LLM", map[string]interface{}{
			"title": title,
			"error": err.Error(),
		})
		// Return empty string instead of error to handle gracefully
		return "", nil
	}

	s.logger.Debug("Successfully generated summary", map[string]interface{}{
		"title": title,
	})

	return response, nil
}

// buildQueryAnalysisPrompt creates the prompt for query analysis
func (s *llmService) buildQueryAnalysisPrompt(query string) string {
	return fmt.Sprintf(`You are an entity and intent extraction model for a news retrieval backend.

Analyze the user's query and return ONLY a valid JSON object with this schema:

{
  "entities": [],        // list of entity names as strings (people, organizations, places, events)
  "intent": {
    "category": {
      "values": []      // list of news categories/topics to search for
    },
    "source": {
      "values": []      // list of news sources/publishers to consider
    },
    "nearby": {
      "lat": null,      // latitude if location intent exists, else null
      "lon": null       // longitude if location intent exists, else null
    }
  }
}

### Rules:
- Extract ALL important named entities into entities[] as plain strings.
  (Examples: people, publishers, companies, places, events like acquisition, war, election, etc.)
- If the user asks for specific news topics like Technology, Sports, Business, etc., put them in category.values.
- If the user mentions specific publishers/sources, put them in source.values.
- If the user asks for news “near/around a location” or “near me”, classify geo intent and include the place name inside entities[], but do NOT generate coordinates.
  If coordinates are already given by the user, copy them into nearby.lat and nearby.lon.
- If no intent is present for a key, keep the list empty or null values instead of removing the field.
- Do NOT add explanations, code blocks, or extra text outside JSON.

### Examples (strict format):
Input: "Latest updates on AI companies like OpenAI and Google near Delhi"
Output:
{
  "entities": ["OpenAI", "Google", "Delhi", "AI companies"],
  "intent": {
    "category": { "values": ["technology", "artificial intelligence"] },
    "source": { "values": [] },
    "nearby": { "lat": null, "lon": null }
  }
}

Input: "Breaking sports news near me"
Output:
{
  "entities": ["sports", "near me"],
  "intent": {
    "category": { "values": ["sports"] },
    "source": { "values": [] },
    "nearby": { "lat": null, "lon": null }
  }
}

Input: "Top tech news from Reuters and DW"
Output:
{
  "entities": ["Reuters", "DW", "tech news"],
  "intent": {
    "category": { "values": ["technology"] },
    "source": { "values": ["Reuters", "DW"] },
    "nearby": { "lat": null, "lon": null }
  }
}

Now analyze the following query:

Input: "%s"
`, query)
}

// buildSummaryPrompt creates the prompt for article summarization
func (s *llmService) buildSummaryPrompt(title, description string) string {
	return fmt.Sprintf(`Summarize the following news article in 2-3 sentences:

Title: %s
Description: %s

Summary:`, title, description)
}

// callOpenAI makes a request to the OpenAI API
func (s *llmService) callOpenAI(prompt string, maxTokens int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	reqBody := openAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []openAIMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", s.config.APIURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// llmQueryResponse represents the raw JSON response structure from LLM
// This is a temporary structure used only for parsing the LLM JSON response
type llmQueryResponse struct {
	Entities []string `json:"entities"`
	Intent   struct {
		Category struct {
			Values []string `json:"values"`
		} `json:"category"`
		Source struct {
			Values []string `json:"values"`
		} `json:"source"`
		Nearby struct {
			Lat *float64 `json:"lat"`
			Lon *float64 `json:"lon"`
		} `json:"nearby"`
	} `json:"intent"`
}

// parseQueryAnalysis parses the LLM response into QueryAnalysis
func (s *llmService) parseQueryAnalysis(response string) (*models.QueryAnalysis, error) {
	var llmResp llmQueryResponse

	// Try to extract JSON from the response (in case LLM adds extra text)
	startIdx := bytes.IndexByte([]byte(response), '{')
	endIdx := bytes.LastIndexByte([]byte(response), '}')

	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[startIdx : endIdx+1]

	if err := json.Unmarshal([]byte(jsonStr), &llmResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Convert LLM response to QueryAnalysis model
	analysis := &models.QueryAnalysis{
		Entities: llmResp.Entities,
		Intents:  make([]models.Intent, 0),
	}

	// Convert intent structure to Intent array
	// Category intent
	if len(llmResp.Intent.Category.Values) > 0 {
		analysis.Intents = append(analysis.Intents, models.Intent{
			Type: models.IntentTypeCategory,
			Parameters: map[string]interface{}{
				"values": llmResp.Intent.Category.Values,
			},
		})
	}

	// Source intent
	if len(llmResp.Intent.Source.Values) > 0 {
		analysis.Intents = append(analysis.Intents, models.Intent{
			Type: models.IntentTypeSource,
			Parameters: map[string]interface{}{
				"values": llmResp.Intent.Source.Values,
			},
		})
	}

	// Nearby intent
	if llmResp.Intent.Nearby.Lat != nil && llmResp.Intent.Nearby.Lon != nil {
		analysis.Intents = append(analysis.Intents, models.Intent{
			Type: models.IntentTypeNearby,
			Parameters: map[string]interface{}{
				"lat": *llmResp.Intent.Nearby.Lat,
				"lon": *llmResp.Intent.Nearby.Lon,
			},
		})
	}

	return analysis, nil
}
