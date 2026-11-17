package fiscal

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	uzStrategy := NewUzbekistanStrategy()

	// Test Register
	err := r.Register("UZ", uzStrategy)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Test Register duplicate
	err = r.Register("UZ", uzStrategy)
	if err != ErrStrategyAlreadyRegistered {
		t.Fatalf("expected ErrStrategyAlreadyRegistered, got %v", err)
	}

	// Test Get
	s, err := r.Get("UZ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s != uzStrategy {
		t.Fatalf("expected strategy to match")
	}

	// Test Get missing
	_, err = r.Get("US")
	if err != ErrStrategyNotFound {
		t.Fatalf("expected ErrStrategyNotFound, got %v", err)
	}

	// Test strategy methods (unconfigured strategy must fail closed, never return nil success)
	ctx := context.Background()
	doc, err := s.FormatDocument(ctx, "doc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var parsed map[string]any
	if uErr := json.Unmarshal(doc, &parsed); uErr != nil {
		t.Fatalf("expected canonical JSON from FormatDocument, got %q (%v)", string(doc), uErr)
	}
	if parsed["doc_id"] != "doc123" || parsed["country"] != "UZ" {
		t.Fatalf("unexpected formatted doc payload: %+v", parsed)
	}

	if err = s.SubmitDocument(ctx, "doc123", doc); !errors.Is(err, ErrSoliqGatewayNotConfigured) {
		t.Fatalf("expected ErrSoliqGatewayNotConfigured on unconfigured strategy, got %v", err)
	}

	if err = s.CancelDocument(ctx, "doc123"); !errors.Is(err, ErrSoliqGatewayNotConfigured) {
		t.Fatalf("expected ErrSoliqGatewayNotConfigured on unconfigured strategy, got %v", err)
	}
}

func TestGlobalRegistry(t *testing.T) {
	uzStrategy := NewUzbekistanStrategy()

	// Use a unique code to avoid clashing if tests run in parallel or pollute global state
	code := "UZ_GLOBAL_TEST"

	err := Register(code, uzStrategy)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	s, err := Get(code)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s != uzStrategy {
		t.Fatalf("expected strategy to match")
	}
}
