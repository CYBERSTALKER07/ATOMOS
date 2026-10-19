package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// SemanticEntry represents a cached LLM / AI prompt-response pair.
type SemanticEntry struct {
	ID         string            `json:"id"`
	TaskType   string            `json:"task_type"`
	Prompt     string            `json:"prompt"`
	Response   string            `json:"response"`
	Embedding  []float32         `json:"embedding,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Similarity float64           `json:"similarity,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// SemanticCacheConfig defines configuration for task-isolated semantic caching.
type SemanticCacheConfig struct {
	TaskType            string        // Workload isolation (e.g. "dispatch", "customer_support", "routing")
	SimilarityThreshold float64       // Minimum cosine similarity (0.95: strict, 0.90: balanced default, 0.80: loose)
	DefaultTTL          time.Duration // Cache retention TTL (e.g. TTLSemanticCache = 24h)
}

// DefaultSimilarityThreshold is the balanced production default per redis-semantic-cache skill.
const DefaultSimilarityThreshold = 0.90

// SemanticCache implements an enterprise semantic cache-aside layer for AI/LLM workloads.
// It features:
// 1. Task-type namespace isolation preventing cross-task response pollution.
// 2. O(1) SHA-256 exact match fast-path.
// 3. Normalized cosine-similarity vector matching for semantic equivalents.
// 4. Attribute-based filtering (tenant isolation, category routing).
type SemanticCache struct {
	cache  *Cache
	config SemanticCacheConfig
	mu     sync.RWMutex
}

// NewSemanticCache initializes a SemanticCache for a specific AI task domain.
func NewSemanticCache(cache *Cache, cfg SemanticCacheConfig) *SemanticCache {
	if cfg.TaskType == "" {
		cfg.TaskType = "general"
	}
	if cfg.SimilarityThreshold <= 0 {
		cfg.SimilarityThreshold = DefaultSimilarityThreshold
	}
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = TTLSemanticCache
	}
	return &SemanticCache{
		cache:  cache,
		config: cfg,
	}
}

// Search searches for a semantically similar cached response.
// Returns (entry, hit=true, nil) if an exact or similarity match >= threshold is found.
func (s *SemanticCache) Search(ctx context.Context, prompt string, embedding []float32, attrs map[string]string) (*SemanticEntry, bool, error) {
	if s == nil || s.cache == nil {
		return nil, false, nil
	}

	// 1. Fast-Path: Exact match lookup via SHA-256 hash
	exactKey := s.exactMatchKey(prompt, attrs)
	if val, found, err := s.cache.Get(ctx, exactKey); err == nil && found {
		var entry SemanticEntry
		if jsonErr := json.Unmarshal(val, &entry); jsonErr == nil {
			entry.Similarity = 1.0
			RecordHit(exactKey)
			return &entry, true, nil
		}
	}

	// If no embedding is supplied, only exact match can be performed
	if len(embedding) == 0 {
		RecordMiss(exactKey)
		return nil, false, nil
	}

	// 2. Semantic Search: Scan task entries and compute cosine similarity
	// In clustered Redis with hash tags, entries use {ai:semantic:task}:entry:*
	matchPattern := fmt.Sprintf("{%s%s}:entry:*", PrefixSemanticCache, s.config.TaskType)
	var cursor uint64
	var bestMatch *SemanticEntry
	highestSim := 0.0

	for {
		keys, nextCursor, err := s.cache.Scan(ctx, cursor, matchPattern, 100)
		if err != nil {
			break
		}

		for _, k := range keys {
			data, found, getErr := s.cache.Get(ctx, k)
			if getErr != nil || !found {
				continue
			}

			var candidate SemanticEntry
			if err := json.Unmarshal(data, &candidate); err != nil {
				continue
			}

			// Validate attributes filter (e.g. tenant or category must match)
			if !matchAttributes(candidate.Attributes, attrs) {
				continue
			}

			if len(candidate.Embedding) != len(embedding) {
				continue
			}

			sim := CosineSimilarity(candidate.Embedding, embedding)
			if sim >= s.config.SimilarityThreshold && sim > highestSim {
				highestSim = sim
				candidateCopy := candidate
				candidateCopy.Similarity = sim
				bestMatch = &candidateCopy
			}
		}

		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	if bestMatch != nil {
		RecordHit(exactKey)
		return bestMatch, true, nil
	}

	RecordMiss(exactKey)
	return nil, false, nil
}

// Set stores a new prompt, embedding, and response in the semantic cache.
func (s *SemanticCache) Set(ctx context.Context, prompt string, embedding []float32, response string, attrs map[string]string, ttl time.Duration) error {
	if s == nil || s.cache == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = s.config.DefaultTTL
	}

	id := generateEntryID(prompt, attrs)
	entry := SemanticEntry{
		ID:         id,
		TaskType:   s.config.TaskType,
		Prompt:     prompt,
		Response:   response,
		Embedding:  embedding,
		Attributes: attrs,
		CreatedAt:  time.Now(),
	}

	bytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("semantic cache marshal failed: %w", err)
	}

	// Store full entry for vector/semantic scanning
	entryKey := fmt.Sprintf("{%s%s}:entry:%s", PrefixSemanticCache, s.config.TaskType, id)
	if err := s.cache.Set(ctx, entryKey, bytes, ttl); err != nil {
		return err
	}

	// Store fast-path exact match key
	exactKey := s.exactMatchKey(prompt, attrs)
	return s.cache.Set(ctx, exactKey, bytes, ttl)
}

// GetOrGenerate implements the cache-aside pattern: searches the cache, and on miss,
// calls generator and caches the result.
func (s *SemanticCache) GetOrGenerate(
	ctx context.Context,
	prompt string,
	embedding []float32,
	attrs map[string]string,
	ttl time.Duration,
	generator func(ctx context.Context) (string, error),
) (string, error) {
	entry, hit, err := s.Search(ctx, prompt, embedding, attrs)
	if err == nil && hit && entry != nil {
		return entry.Response, nil
	}

	response, err := generator(ctx)
	if err != nil {
		return "", err
	}

	// Asynchronously or synchronously cache result
	_ = s.Set(ctx, prompt, embedding, response, attrs, ttl)
	return response, nil
}

// InvalidateTask purges all entries in this semantic cache task namespace.
func (s *SemanticCache) InvalidateTask(ctx context.Context) error {
	if s == nil || s.cache == nil {
		return nil
	}
	matchPattern := fmt.Sprintf("{%s%s}:*", PrefixSemanticCache, s.config.TaskType)
	var cursor uint64
	for {
		keys, nextCursor, err := s.cache.Scan(ctx, cursor, matchPattern, 100)
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			s.cache.Invalidate(ctx, keys...)
		}
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
	return nil
}

// CosineSimilarity computes the cosine similarity between two float vectors.
// Returns a value between -1.0 and 1.0 (1.0 = identical angle/direction).
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		ai := float64(a[i])
		bi := float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA <= 0 || normB <= 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (s *SemanticCache) exactMatchKey(prompt string, attrs map[string]string) string {
	h := sha256.New()
	h.Write([]byte(strings.TrimSpace(prompt)))
	if len(attrs) > 0 {
		keys := make([]string, 0, len(attrs))
		for k := range attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h.Write([]byte(k))
			h.Write([]byte(attrs[k]))
		}
	}
	hashStr := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("{%s%s}:exact:%s", PrefixSemanticCache, s.config.TaskType, hashStr)
}

func generateEntryID(prompt string, attrs map[string]string) string {
	h := sha256.New()
	h.Write([]byte(prompt))
	for k, v := range attrs {
		h.Write([]byte(k + "=" + v))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func matchAttributes(stored, query map[string]string) bool {
	if len(query) == 0 {
		return true
	}
	if len(stored) < len(query) {
		return false
	}
	for k, expected := range query {
		if stored[k] != expected {
			return false
		}
	}
	return true
}
