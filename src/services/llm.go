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
	return fmt.Sprintf(`Analyze the following news query and extract:
1. Entities (people, organizations, locations, events)
2. Intents (category, score, search, source, nearby)
3. Parameters for each intent

Query: %s

Return ONLY valid JSON in this exact format (no additional text):
{
    "entities": [{"type": "person", "value": "Name"}],
    "intents": [
        {"type": "search", "parameters": {"query": "search terms"}},
        {"type": "category", "parameters": {"category": "Technology"}},
        {"type": "source", "parameters": {"source": "Source Name"}},
        {"type": "score", "parameters": {"threshold": 0.7}},
        {"type": "nearby", "parameters": {"location": "City Name"}}
    ]
}

Valid intent types: category, score, search, source, nearby
Valid entity types: person, organization, location, event
Valid categories: Technology, Business, Sports, General, Politics, Entertainment, Health, Science

If no specific intent is clear, use "search" with the query terms.`, query)
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

// parseQueryAnalysis parses the LLM response into QueryAnalysis
func (s *llmService) parseQueryAnalysis(response string) (*models.QueryAnalysis, error) {
	var analysis models.QueryAnalysis

	// Try to extract JSON from the response (in case LLM adds extra text)
	startIdx := bytes.IndexByte([]byte(response), '{')
	endIdx := bytes.LastIndexByte([]byte(response), '}')

	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[startIdx : endIdx+1]

	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Ensure at least one intent exists
	if len(analysis.Intents) == 0 {
		// Default to search intent if none provided
		analysis.Intents = []models.Intent{
			{
				Type: "search",
				Parameters: map[string]interface{}{
					"query": "general news",
				},
			},
		}
	}

	// Initialize empty slices if nil
	if analysis.Entities == nil {
		analysis.Entities = []models.Entity{}
	}

	return &analysis, nil
}
