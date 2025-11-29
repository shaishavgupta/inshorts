package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
)

// LLMService defines the interface for LLM operations
type LLMService interface {
	ProcessQuery(query string, sources []string, categories []string) (*models.QueryAnalysis, error)
	GenerateSummary(title, description string) (string, error)
	GenerateEmbedding(text string) ([]float64, error)
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
func (s *llmService) ProcessQuery(query string, sources []string, categories []string) (*models.QueryAnalysis, error) {
	prompt := s.buildQueryAnalysisPrompt(query, sources, categories)

	response, err := s.callOpenAI(prompt, 500)
	if err != nil {
		s.logger.Error("Failed to process query with LLM", err, map[string]interface{}{
			"query": query,
		})
		return nil, fmt.Errorf("LLM service unavailable: %w", err)
	}

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
		return "", nil
	}

	s.logger.Debug("Successfully generated summary", map[string]interface{}{
		"title": title,
	})

	return response, nil
}

// GenerateEmbedding generates an embedding vector for the given text using OpenAI embeddings API
func (s *llmService) GenerateEmbedding(text string) ([]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	embeddingRequest := struct {
		Model string `json:"model"`
		Input string `json:"input"`
	}{
		Model: "text-embedding-3-small", // or "text-embedding-ada-002" for 1536 dimensions
		Input: text,
	}

	jsonData, err := json.Marshal(embeddingRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	url := fmt.Sprintf("%s/embeddings", s.config.APIURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI embeddings API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedding response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI embeddings API returned status %d: %s", resp.StatusCode, string(body))
	}

	var embeddingResp struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding response: %w", err)
	}

	if embeddingResp.Error != nil {
		return nil, fmt.Errorf("OpenAI embeddings API error: %s", embeddingResp.Error.Message)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in OpenAI response")
	}

	s.logger.Debug("Successfully generated embedding", map[string]interface{}{
		"dimensions": len(embeddingResp.Data[0].Embedding),
	})

	return embeddingResp.Data[0].Embedding, nil
}

// buildQueryAnalysisPrompt creates the prompt for query analysis
func (s *llmService) buildQueryAnalysisPrompt(query string, sources []string, categories []string) string {
	return fmt.Sprintf(`You are an intelligent query parser for a Contextual News Retrieval System. Your task is to analyze a user's natural-language news query and convert it into structured intent-based filters.

1. INPUTS

You will always receive:

A list of Valid Categories

A list of Allowed Sources

The user's Search Query

2. REQUIRED JSON OUTPUT FORMAT

Respond with ONLY a single valid JSON object, nothing else:

{
"entities": [],
"intent": {
"category": { "values": [] },
"source": { "values": [] },
"nearby": { "lat": null, "lon": null }
}
}

3. CATEGORY MATCHING RULES

Map category values only from the provided Valid Categories list.

Perform fuzzy matching on query tokens (order-agnostic, case-insensitive, mild misspell tolerant).

Output must always be lowercase.

4. SOURCE MATCHING RULES

Map source values only from the provided Allowed Sources list.

Perform fuzzy matching (partial match, abbreviations, mild misspell, different casing).

If the query token refers to a cluster (e.g., "abp" or "ani"), include all matching variants from the list.

Generic nouns like "news", "updates", "articles" must be ignored and excluded from source matches.

5. LOCATION / NEARBY INTENT (Updated)

If the query contains any real place name (city, region, country, landmark), you must:

Activate the nearby intent.

Insert that place name into entities[].

Generate approximate latitude & longitude of that place and populate nearby.lat and nearby.lon.

Example: "Delhi" → 28.61, 77.23

"Mumbai" → 19.07, 72.88

"Palo Alto" → 37.44, -122.14

"Paris" → 48.85, 2.34

If multiple places are present, choose the most relevant one for proximity and still include all in entities.

You may approximate; slight offsets are acceptable, do not be exact.

6. ENTITY EXTRACTION RULES

Extract all key real-world names (people, orgs, places, events, concepts) into entities[].

Do not change case of entities except preserving spelling.

7. PLACEHOLDER SECTION YOU MUST KEEP
Valid Categories: %s

Allowed Sources: %s

8. MATCHING PRIORITY RULES

Do NOT emit new strings in category or source values that do not exist in the allowed lists.

Only the nearby lat/lon may be approximated when a place name is present.

9. EXAMPLES (Follow strictly)

Input Query: "latest news near Paris from ANI"
Allowed Sources: ["ANI","BBC","DW"]
Allowed Categories: ["world","technology","sports","science"]

Output:
{
"entities": ["Paris","ANI","paris"],
"intent": {
"category": { "values": [] },
"source": { "values": ["ANI"] },
"nearby": { "lat": 48.85, "lon": 2.34 }
}
}

Input Query: "technology updates from News18 Mumbai"
Allowed Sources: ["News18","Reuters","DW","BBC"]
Allowed Categories: ["technology","sports","world"]

Output:
{
"entities": ["News18","Mumbai","technology"],
"intent": {
"category": { "values": ["technology"] },
"source": { "values": ["News18"] },
"nearby": { "lat": 19.07, "lon": 72.88 }
}
}

Now analyze the following query:

Input Query: "%s"
`, strings.Join(categories, ", "), strings.Join(sources, ", "), query)
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

	startIdx := bytes.IndexByte([]byte(response), '{')
	endIdx := bytes.LastIndexByte([]byte(response), '}')

	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[startIdx : endIdx+1]

	if err := json.Unmarshal([]byte(jsonStr), &llmResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	analysis := &models.QueryAnalysis{
		Entities: llmResp.Entities,
		Intents:  make([]models.Intent, 0),
	}

	if len(llmResp.Intent.Category.Values) > 0 {
		analysis.Intents = append(analysis.Intents, models.Intent{
			Type:   models.IntentTypeCategory,
			Values: llmResp.Intent.Category.Values,
		})
	}

	if len(llmResp.Intent.Source.Values) > 0 {
		analysis.Intents = append(analysis.Intents, models.Intent{
			Type:   models.IntentTypeSource,
			Values: llmResp.Intent.Source.Values,
		})
	}

	if llmResp.Intent.Nearby.Lat != nil && llmResp.Intent.Nearby.Lon != nil {
		analysis.Intents = append(analysis.Intents, models.Intent{
			Type:   models.IntentTypeNearby,
			Values: []string{fmt.Sprintf("%f", *llmResp.Intent.Nearby.Lat), fmt.Sprintf("%f", *llmResp.Intent.Nearby.Lon)},
		})
	}

	return analysis, nil
}
