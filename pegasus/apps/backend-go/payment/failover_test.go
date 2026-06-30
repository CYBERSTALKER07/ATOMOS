package payment

import (
	"context"
	"fmt"
	"testing"
)

func TestIsFailoverRetryable(t *testing.T) {
	if !IsFailoverRetryable(fmt.Errorf("gateway timeout after 30s")) {
		t.Fatal("expected timeout to be retryable")
	}
	if IsFailoverRetryable(fmt.Errorf("card declined")) {
		t.Fatal("expected card decline to be non-retryable")
	}
}

func TestSelectHealthyGateway_Empty(t *testing.T) {
	if got := SelectHealthyGateway(context.Background(), nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
