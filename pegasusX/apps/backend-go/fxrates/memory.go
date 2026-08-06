package fxrates

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository is an in-memory FxRates store for tests / local.
type MemoryRepository struct {
	mu    sync.RWMutex
	rates []Rate
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{rates: nil}
}

func (r *MemoryRepository) Upsert(_ context.Context, rate Rate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rate.RateID == "" {
		rate.RateID = uuid.NewString()
	}
	if rate.Scale <= 0 {
		rate.Scale = DefaultScale
	}
	if rate.CreatedAt.IsZero() {
		rate.CreatedAt = time.Now().UTC()
	}
	for i, existing := range r.rates {
		if existing.BaseCurrency == rate.BaseCurrency &&
			existing.QuoteCurrency == rate.QuoteCurrency &&
			existing.EffectiveAt.Equal(rate.EffectiveAt) {
			r.rates[i] = rate
			return nil
		}
	}
	r.rates = append(r.rates, rate)
	return nil
}

func (r *MemoryRepository) GetAsOf(_ context.Context, base, quote string, at time.Time) (Rate, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	base = NormalizeCurrency(base)
	quote = NormalizeCurrency(quote)
	var best Rate
	found := false
	for _, rate := range r.rates {
		if rate.BaseCurrency != base || rate.QuoteCurrency != quote {
			continue
		}
		if rate.EffectiveAt.After(at) {
			continue
		}
		if !found || rate.EffectiveAt.After(best.EffectiveAt) {
			best = rate
			found = true
		}
	}
	return best, found, nil
}

func (r *MemoryRepository) ListLatest(_ context.Context, limit int) ([]Rate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	type key struct{ b, q string }
	latest := map[key]Rate{}
	for _, rate := range r.rates {
		k := key{rate.BaseCurrency, rate.QuoteCurrency}
		prev, ok := latest[k]
		if !ok || rate.EffectiveAt.After(prev.EffectiveAt) {
			latest[k] = rate
		}
	}
	out := make([]Rate, 0, len(latest))
	for _, rate := range latest {
		out = append(out, rate)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].BaseCurrency != out[j].BaseCurrency {
			return out[i].BaseCurrency < out[j].BaseCurrency
		}
		return out[i].QuoteCurrency < out[j].QuoteCurrency
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
