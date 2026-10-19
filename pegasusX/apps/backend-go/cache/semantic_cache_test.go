package cache

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestCosineSimilarity(t *testing.T) {
	// Identical vectors -> similarity = 1.0
	v1 := []float32{1.0, 2.0, 3.0}
	v2 := []float32{1.0, 2.0, 3.0}
	sim := CosineSimilarity(v1, v2)
	if math.Abs(sim-1.0) > 1e-5 {
		t.Errorf("expected 1.0 for identical vectors, got %f", sim)
	}

	// Orthogonal vectors -> similarity = 0.0
	v3 := []float32{1.0, 0.0, 0.0}
	v4 := []float32{0.0, 1.0, 0.0}
	sim = CosineSimilarity(v3, v4)
	if math.Abs(sim-0.0) > 1e-5 {
		t.Errorf("expected 0.0 for orthogonal vectors, got %f", sim)
	}

	// Close vectors (e.g. 0.98 similarity)
	v5 := []float32{0.9, 0.1, 0.0}
	v6 := []float32{0.95, 0.05, 0.0}
	sim = CosineSimilarity(v5, v6)
	if sim < 0.95 {
		t.Errorf("expected > 0.95 similarity, got %f", sim)
	}
}

func TestSemanticCache_ExactMatchFastPath(t *testing.T) {
	backend := NewInMemoryBackend()
	c := New(backend, nil)
	sc := NewSemanticCache(c, SemanticCacheConfig{
		TaskType:            "dispatch",
		SimilarityThreshold: 0.90,
	})
	ctx := context.Background()

	prompt := "What is the fastest route from Warehouse A to Retailer B?"
	response := "Take Highway M39 north."
	attrs := map[string]string{"tenant": "uz_tashkent"}

	// Set in cache
	err := sc.Set(ctx, prompt, []float32{0.1, 0.2, 0.3}, response, attrs, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to set semantic cache: %v", err)
	}

	// Search exact prompt (with nil embedding) -> fast-path hit!
	entry, hit, err := sc.Search(ctx, prompt, nil, attrs)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !hit || entry == nil {
		t.Fatalf("expected exact match hit, got hit=%v", hit)
	}
	if entry.Response != response {
		t.Errorf("expected response %q, got %q", response, entry.Response)
	}
	if entry.Similarity != 1.0 {
		t.Errorf("expected similarity 1.0 for exact hit, got %f", entry.Similarity)
	}
}

func TestSemanticCache_VectorSimilarityMatch(t *testing.T) {
	backend := NewInMemoryBackend()
	c := New(backend, nil)
	sc := NewSemanticCache(c, SemanticCacheConfig{
		TaskType:            "dispatch",
		SimilarityThreshold: 0.90,
	})
	ctx := context.Background()

	prompt1 := "How do I optimize vehicle route for delivery 100?"
	emb1 := []float32{0.8, 0.6, 0.0}
	resp1 := "Run Dijkstra algorithm with road restrictions."

	err := sc.Set(ctx, prompt1, emb1, resp1, nil, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	// Query with semantically similar prompt and vector (similarity ~0.98)
	prompt2 := "How to calculate best route for delivery 100?"
	emb2 := []float32{0.85, 0.55, 0.0}

	entry, hit, err := sc.Search(ctx, prompt2, emb2, nil)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !hit || entry == nil {
		t.Fatalf("expected semantic hit for similar embedding, got hit=%v", hit)
	}
	if entry.Response != resp1 {
		t.Errorf("expected response %q, got %q", resp1, entry.Response)
	}

	// Query with dissimilar embedding (similarity < 0.90) -> Miss
	prompt3 := "What is the weather tomorrow?"
	emb3 := []float32{0.0, 0.1, 0.99}

	entry, hit, err = sc.Search(ctx, prompt3, emb3, nil)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if hit || entry != nil {
		t.Fatalf("expected miss for dissimilar embedding, got hit=%v", hit)
	}
}

func TestSemanticCache_TaskIsolation(t *testing.T) {
	backend := NewInMemoryBackend()
	c := New(backend, nil)

	dispatchCache := NewSemanticCache(c, SemanticCacheConfig{
		TaskType:            "dispatch",
		SimilarityThreshold: 0.90,
	})
	supportCache := NewSemanticCache(c, SemanticCacheConfig{
		TaskType:            "support",
		SimilarityThreshold: 0.90,
	})
	ctx := context.Background()

	prompt := "How do I cancel this order?"
	emb := []float32{0.5, 0.5, 0.5}

	_ = dispatchCache.Set(ctx, prompt, emb, "Dispatch cancellation initiated.", nil, 1*time.Hour)

	// Support cache should MISS because of task isolation
	entry, hit, err := supportCache.Search(ctx, prompt, emb, nil)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if hit || entry != nil {
		t.Errorf("expected support cache to MISS due to task isolation, but got hit")
	}

	// Dispatch cache should HIT
	entry, hit, err = dispatchCache.Search(ctx, prompt, emb, nil)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if !hit || entry == nil {
		t.Errorf("expected dispatch cache to HIT")
	}
}

func TestSemanticCache_GetOrGenerate(t *testing.T) {
	backend := NewInMemoryBackend()
	c := New(backend, nil)
	sc := NewSemanticCache(c, SemanticCacheConfig{
		TaskType:            "classification",
		SimilarityThreshold: 0.90,
	})
	ctx := context.Background()

	called := 0
	generator := func(ctx context.Context) (string, error) {
		called++
		return "Category: Grocery Wholesale", nil
	}

	prompt := "Classify item: 50kg Sugar Bag"
	emb := []float32{0.7, 0.7, 0.1}

	// 1st call -> Generator called (cache miss)
	res1, err := sc.GetOrGenerate(ctx, prompt, emb, nil, 1*time.Hour, generator)
	if err != nil || res1 != "Category: Grocery Wholesale" {
		t.Fatalf("unexpected result: %s, err: %v", res1, err)
	}
	if called != 1 {
		t.Fatalf("expected generator called once, got %d", called)
	}

	// 2nd call -> Returns from cache without calling generator
	res2, err := sc.GetOrGenerate(ctx, prompt, emb, nil, 1*time.Hour, generator)
	if err != nil || res2 != "Category: Grocery Wholesale" {
		t.Fatalf("unexpected result: %s, err: %v", res2, err)
	}
	if called != 1 {
		t.Fatalf("expected generator NOT called second time, got %d", called)
	}
}
