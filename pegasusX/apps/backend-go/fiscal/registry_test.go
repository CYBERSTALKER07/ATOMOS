package fiscal

import (
	"context"
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

	// Test strategy methods
	ctx := context.Background()
	doc, err := s.FormatDocument(ctx, "doc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(doc) != "UZ_FORMATTED_doc123" {
		t.Fatalf("unexpected doc format: %s", string(doc))
	}

	err = s.SubmitDocument(ctx, "doc123", doc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = s.CancelDocument(ctx, "doc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
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
