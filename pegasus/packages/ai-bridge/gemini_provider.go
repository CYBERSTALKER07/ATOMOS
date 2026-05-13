package aibridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	defaultVertexLocation = "us-central1"
	defaultGeminiModel    = "gemini-1.5-pro"
	vertexScope           = "https://www.googleapis.com/auth/cloud-platform"
)

// ProviderError surfaces HTTP-level provider failures (429/5xx/etc).
type ProviderError struct {
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("ai provider request failed: status=%d message=%s", e.StatusCode, e.Message)
}

// IsRateLimited reports whether the provider failed with HTTP 429.
func IsRateLimited(err error) bool {
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		return false
	}
	return providerErr.StatusCode == http.StatusTooManyRequests
}

// GeminiProvider calls Gemini on Vertex AI.
type GeminiProvider struct {
	projectID   string
	location    string
	model       string
	endpoint    string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
}

// NewGeminiProviderFromEnv initializes GeminiProvider from environment.
func NewGeminiProviderFromEnv(ctx context.Context, httpClient *http.Client) (*GeminiProvider, error) {
	projectID := strings.TrimSpace(os.Getenv("VERTEX_PROJECT_ID"))
	if projectID == "" {
		projectID = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	}
	location := strings.TrimSpace(os.Getenv("VERTEX_LOCATION"))
	if location == "" {
		location = defaultVertexLocation
	}
	model := strings.TrimSpace(os.Getenv("VERTEX_GEMINI_MODEL"))
	if model == "" {
		model = defaultGeminiModel
	}
	return NewGeminiProvider(ctx, projectID, location, model, httpClient)
}

// NewGeminiProvider initializes GeminiProvider with ADC credentials.
func NewGeminiProvider(ctx context.Context, projectID, location, model string, httpClient *http.Client) (*GeminiProvider, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, errors.New("vertex project id is required")
	}
	location = strings.TrimSpace(location)
	if location == "" {
		location = defaultVertexLocation
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultGeminiModel
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	creds, err := google.FindDefaultCredentials(ctx, vertexScope)
	if err != nil {
		return nil, fmt.Errorf("load vertex credentials: %w", err)
	}

	return &GeminiProvider{
		projectID:   projectID,
		location:    location,
		model:       model,
		endpoint:    fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s", location, projectID, location, model),
		httpClient:  httpClient,
		tokenSource: oauth2.ReuseTokenSource(nil, creds.TokenSource),
	}, nil
}

// DiscoverSchema sends sample rows + target definitions to Gemini.
func (p *GeminiProvider) DiscoverSchema(ctx context.Context, sampleData SampleData) (DiscoverSchemaResult, error) {
	prompt := buildDiscoveryPrompt(sampleData)
	contents := []map[string]any{
		{
			"role":  "user",
			"parts": []map[string]string{{"text": prompt}},
		},
	}

	result := DiscoverSchemaResult{Model: p.model}
	if promptTokens, err := p.countTokens(ctx, contents); err == nil {
		result.Usage.PromptTokens = promptTokens
	}

	reqBody := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"temperature":      0.1,
			"responseMimeType": "application/json",
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return DiscoverSchemaResult{}, fmt.Errorf("marshal generate request: %w", err)
	}

	respBytes, err := p.doVertexJSONRequest(ctx, p.endpoint+":generateContent", body)
	if err != nil {
		return DiscoverSchemaResult{}, err
	}

	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return DiscoverSchemaResult{}, fmt.Errorf("decode generate response: %w", err)
	}

	result.Usage.PromptTokens = maxInt(result.Usage.PromptTokens, response.UsageMetadata.PromptTokenCount)
	result.Usage.CompletionTokens = response.UsageMetadata.CandidatesTokenCount
	if response.UsageMetadata.TotalTokenCount > 0 {
		result.Usage.TotalTokens = response.UsageMetadata.TotalTokenCount
	} else {
		result.Usage.TotalTokens = result.Usage.PromptTokens + result.Usage.CompletionTokens
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return DiscoverSchemaResult{}, errors.New("gemini returned no candidates")
	}

	rawText := strings.TrimSpace(response.Candidates[0].Content.Parts[0].Text)
	if rawText == "" {
		return DiscoverSchemaResult{}, errors.New("gemini returned empty schema payload")
	}

	if err := decodeJSONPayload(rawText, &result); err != nil {
		return DiscoverSchemaResult{}, fmt.Errorf("decode gemini schema payload: %w", err)
	}
	result.Model = p.model
	return result, nil
}

func (p *GeminiProvider) countTokens(ctx context.Context, contents []map[string]any) (int, error) {
	reqBody := map[string]any{"contents": contents}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}
	respBytes, err := p.doVertexJSONRequest(ctx, p.endpoint+":countTokens", body)
	if err != nil {
		return 0, err
	}
	var response struct {
		TotalTokens int `json:"totalTokens"`
	}
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return 0, err
	}
	return response.TotalTokens, nil
}

func (p *GeminiProvider) doVertexJSONRequest(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
	tok, err := p.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("fetch vertex oauth token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build vertex request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vertex request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read vertex response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(respBytes))
		if len(message) > 800 {
			message = message[:800]
		}
		return nil, &ProviderError{StatusCode: resp.StatusCode, Message: message}
	}
	return respBytes, nil
}

func buildDiscoveryPrompt(sampleData SampleData) string {
	sampleJSON, _ := json.Marshal(sampleData)
	return "You are Pegasus inventory import schema mapper.\n" +
		"Task: map source spreadsheet headers to canonical Pegasus fields and flag anomalies.\n" +
		"Rules:\n" +
		"1) Return valid JSON only, no markdown.\n" +
		"2) JSON object keys: mappings (array), anomalies (array).\n" +
		"3) mappings[].source_column must match source header exactly.\n" +
		"4) mappings[].target_field must be one of target_fields.name values.\n" +
		"5) mappings[].confidence is float in [0,1].\n" +
		"6) anomalies[] should capture suspicious patterns such as extreme price variance or future dates.\n" +
		"Input JSON:\n" + string(sampleJSON)
}

func decodeJSONPayload(raw string, out any) error {
	if err := json.Unmarshal([]byte(raw), out); err == nil {
		return nil
	}
	start := strings.IndexByte(raw, '{')
	end := strings.LastIndexByte(raw, '}')
	if start < 0 || end < start {
		return errors.New("no json object found in model response")
	}
	return json.Unmarshal([]byte(raw[start:end+1]), out)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
