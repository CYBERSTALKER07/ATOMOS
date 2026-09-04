package warehouse

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestPublishSupplierPerimeter(t *testing.T) {
	// Skip for now since it needs Spanner to fetch ListWarehouses.
	// (Or we could mock the proximity.CoverageStore, but it's tightly coupled to Spanner).
}

func TestCheckSupplierPerimeter(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when starting miniredis", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	svc := &Service{
		redisClient: rdb,
	}

	supplierID := "sup-123"
	key := "perimeter:supplier:" + supplierID
	targetCell := "8720a52d2ffffff"
	missingCell := "8720a52d3ffffff"

	mr.SAdd(key, targetCell)
	mr.SetTTL(key, 24*time.Hour)

	// Check existing cell
	supported, err := svc.CheckSupplierPerimeter(context.Background(), supplierID, targetCell)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !supported {
		t.Fatalf("expected cell to be supported")
	}

	// Check missing cell
	supported, err = svc.CheckSupplierPerimeter(context.Background(), supplierID, missingCell)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if supported {
		t.Fatalf("expected cell to not be supported")
	}

	// Check missing key (redis client returns false on SIsMember if key does not exist, but Service defaults to true)
	mr.Del(key)
	supported, err = svc.CheckSupplierPerimeter(context.Background(), supplierID, targetCell)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !supported {
		t.Fatalf("expected cell to be supported when key is missing (fallback to true)")
	}
}
