package warehouse

import (
	"context"
	"strings"
	"testing"
)

func TestDefaultWarehouseIDForSupplier_MemoryFailClosed(t *testing.T) {
	s := &Service{}
	id, err := s.defaultWarehouseIDForSupplier(context.Background(), "sup-1")
	if id != "" {
		t.Fatalf("memory path must not invent a warehouse id, got %q", id)
	}
	if err == nil || !strings.Contains(err.Error(), "no warehouse for supplier") {
		t.Fatalf("err=%v", err)
	}
}
