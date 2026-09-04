package factory

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

func TestIotIngestor_FlushBatch(t *testing.T) {
	memBackend := cache.NewInMemoryBackend()
	c := cache.New(memBackend, nil)
	svc := &Service{
		cache: c,
	}
	ingestor := NewIotIngestor(svc, nil)
	defer ingestor.Close()

	batch := map[string]int64{
		"mach-1": 150,
		"mach-2": 42,
	}

	ingestor.flushBatch(batch)

	// Verify counts in cache
	ctx := context.Background()
	val1, ok, err := c.Backend().Get(ctx, "factory:iot:mach-1:units")
	if err != nil || !ok {
		t.Fatalf("expected mach-1 in cache, err: %v", err)
	}
	if string(val1) != "150" {
		t.Fatalf("expected mach-1 units 150, got %s", string(val1))
	}

	val2, ok, err := c.Backend().Get(ctx, "factory:iot:mach-2:units")
	if err != nil || !ok {
		t.Fatalf("expected mach-2 in cache, err: %v", err)
	}
	if string(val2) != "42" {
		t.Fatalf("expected mach-2 units 42, got %s", string(val2))
	}
}
