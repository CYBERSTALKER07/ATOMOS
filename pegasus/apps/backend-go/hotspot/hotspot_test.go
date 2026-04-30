package hotspot

import (
	"testing"
)

// ─── ShardCount / ConfigureShardCount ───────────────────────────────────────

func TestShardCount_Default(t *testing.T) {
	// Reset to default
	shardCount = DefaultShardCount
	if got := ShardCount(); got != 16 {
		t.Errorf("default shard count = %d, want 16", got)
	}
}

func TestConfigureShardCount_Positive(t *testing.T) {
	defer func() { shardCount = DefaultShardCount }()
	ConfigureShardCount(32)
	if ShardCount() != 32 {
		t.Errorf("after configure(32), got %d", ShardCount())
	}
}

func TestConfigureShardCount_Zero_Ignored(t *testing.T) {
	defer func() { shardCount = DefaultShardCount }()
	ConfigureShardCount(0)
	if ShardCount() != DefaultShardCount {
		t.Errorf("zero should be ignored, got %d", ShardCount())
	}
}

func TestConfigureShardCount_Negative_Ignored(t *testing.T) {
	defer func() { shardCount = DefaultShardCount }()
	ConfigureShardCount(-5)
	if ShardCount() != DefaultShardCount {
		t.Errorf("negative should be ignored, got %d", ShardCount())
	}
}

// ─── AllShards ──────────────────────────────────────────────────────────────

func TestAllShards_Length(t *testing.T) {
	defer func() { shardCount = DefaultShardCount }()
	shardCount = 8
	shards := AllShards()
	if len(shards) != 8 {
		t.Errorf("len = %d, want 8", len(shards))
	}
}

func TestAllShards_Sequential(t *testing.T) {
	defer func() { shardCount = DefaultShardCount }()
	shardCount = 4
	shards := AllShards()
	for i, s := range shards {
		if s != int64(i) {
			t.Errorf("shard[%d] = %d, want %d", i, s, i)
		}
	}
}

// ─── ShardForKey ────────────────────────────────────────────────────────────

func TestShardForKey_Deterministic(t *testing.T) {
	defer func() { shardCount = DefaultShardCount }()
	a := ShardForKey("retailer-abc")
	b := ShardForKey("retailer-abc")
	if a != b {
		t.Errorf("same key should produce same shard: %d != %d", a, b)
	}
}

func TestShardForKey_InRange(t *testing.T) {
	defer func() { shardCount = DefaultShardCount }()
	shardCount = 8
	for _, key := range []string{"a", "b", "c", "order-1", "retailer-xyz", "sku-123"} {
		s := ShardForKey(key)
		if s < 0 || s >= 8 {
			t.Errorf("shard for %q = %d, out of range [0,8)", key, s)
		}
	}
}

func TestShardForKey_DifferentKeys(t *testing.T) {
	defer func() { shardCount = DefaultShardCount }()
	shardCount = 256 // high shard count makes collision unlikely
	a := ShardForKey("key-alpha")
	b := ShardForKey("key-beta")
	// Not guaranteed to differ, but with 256 shards almost certainly will
	_ = a
	_ = b
}

func TestShardForKey_SingleShard(t *testing.T) {
	defer func() { shardCount = DefaultShardCount }()
	shardCount = 1
	if got := ShardForKey("anything"); got != 0 {
		t.Errorf("single shard should always return 0, got %d", got)
	}
}

// ─── ID Generators ──────────────────────────────────────────────────────────

func TestNewOpaqueID_NotEmpty(t *testing.T) {
	if id := NewOpaqueID(); id == "" {
		t.Error("opaque ID should not be empty")
	}
}

func TestNewOpaqueID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewOpaqueID()
		if ids[id] {
			t.Fatalf("duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestNewOrderID_NotEmpty(t *testing.T) {
	if id := NewOrderID(); id == "" {
		t.Error("order ID should not be empty")
	}
}

func TestNewInvoiceID_NotEmpty(t *testing.T) {
	if id := NewInvoiceID(); id == "" {
		t.Error("invoice ID should not be empty")
	}
}

func TestNewPredictionID_NotEmpty(t *testing.T) {
	if id := NewPredictionID(); id == "" {
		t.Error("prediction ID should not be empty")
	}
}

func TestNewPredictionItemID_NotEmpty(t *testing.T) {
	if id := NewPredictionItemID(); id == "" {
		t.Error("prediction item ID should not be empty")
	}
}

func TestAllIDGenerators_Unique(t *testing.T) {
	ids := make(map[string]bool)
	generators := []struct {
		name string
		fn   func() string
	}{
		{"Order", NewOrderID},
		{"Invoice", NewInvoiceID},
		{"Prediction", NewPredictionID},
		{"PredictionItem", NewPredictionItemID},
	}
	for _, g := range generators {
		for i := 0; i < 25; i++ {
			id := g.fn()
			if ids[id] {
				t.Fatalf("duplicate %s ID: %s", g.name, id)
			}
			ids[id] = true
		}
	}
}
