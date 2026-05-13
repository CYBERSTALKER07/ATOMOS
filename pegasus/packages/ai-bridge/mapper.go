package aibridge
package aibridge

import "context"

// InventoryMapper is the provider-agnostic schema discovery seam.
type InventoryMapper interface {
	DiscoverSchema(ctx context.Context, sampleData SampleData) (DiscoverSchemaResult, error)
}

// SampleData is the normalized sample payload sent to an AI provider.
type SampleData struct {
	Headers      []string          `json:"headers"`
	Rows         []map[string]any  `json:"rows"`
	TargetFields []FieldDefinition `json:"target_fields"`
}

// FieldDefinition describes one canonical target field in Pegasus.
type FieldDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	Required    bool     `json:"required,omitempty"`
}

// MappingCandidate is one source-column to target-field match.
type MappingCandidate struct {
	SourceColumn string  `json:"source_column"`
	TargetField  string  `json:"target_field"`
	Confidence   float64 `json:"confidence"`
	Reason       string  `json:"reason,omitempty"`
	Deterministic bool   `json:"deterministic,omitempty"`
}

// Anomaly describes one suspicious data pattern detected in the sample.
type Anomaly struct {
	Kind     string `json:"kind"`
	Column   string `json:"column,omitempty"`
	Detail   string `json:"detail"`
	Severity string `json:"severity,omitempty"`
}

// TokenUsage carries provider token accounting telemetry.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// DiscoverSchemaResult is the normalized provider output.
type DiscoverSchemaResult struct {
	Mappings  []MappingCandidate `json:"mappings"`
	Anomalies []Anomaly          `json:"anomalies,omitempty"`
	Usage     TokenUsage         `json:"usage,omitempty"`
	Model     string             `json:"model,omitempty"`
}
