package geolocation

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

type countingBackend struct {
	gets atomic.Int64
	sets atomic.Int64
	data map[string][]byte
}

func (b *countingBackend) Get(_ context.Context, key string) ([]byte, bool, error) {
	b.gets.Add(1)
	if b.data == nil {
		return nil, false, nil
	}
	v, ok := b.data[key]
	return v, ok, nil
}

func (b *countingBackend) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	b.sets.Add(1)
	if b.data == nil {
		b.data = make(map[string][]byte)
	}
	b.data[key] = value
	return nil
}

func (b *countingBackend) Delete(context.Context, ...string) error { return nil }
func (b *countingBackend) Publish(context.Context, string, []byte) error {
	return nil
}
func (b *countingBackend) Subscribe(context.Context, string) (<-chan []byte, func(), error) {
	ch := make(chan []byte)
	return ch, func() { close(ch) }, nil
}

func TestForwardGeocodeCacheHitMiss(t *testing.T) {
	backend := &countingBackend{data: make(map[string][]byte)}
	c := cache.New(backend, nil)
	svc := NewService("", c)

	ctx := context.Background()
	loc := ResolvedLocation{Address: "1 Test St", Lat: 1.23, Lng: 4.56, Formatted: "1 Test St"}
	raw, _ := json.Marshal(loc)
	backend.data[forwardCacheKey("uz", "1 test st")] = raw

	got, err := svc.ForwardGeocode(ctx, "1 Test St")
	if err != nil {
		t.Fatalf("ForwardGeocode: %v", err)
	}
	if got.Address != loc.Address {
		t.Fatalf("got address %q want %q", got.Address, loc.Address)
	}
	if backend.gets.Load() < 1 {
		t.Fatalf("expected cache get")
	}
}

func TestAutocompleteCacheRoundTrip(t *testing.T) {
	backend := &countingBackend{data: make(map[string][]byte)}
	c := cache.New(backend, nil)
	svc := NewService("", c)

	ctx := context.Background()
	predictions := []AutocompletePrediction{{PlaceID: "p1", Description: "123 Main"}}
	raw, _ := json.Marshal(predictions)
	key := autocompleteCacheKey("uz", "123 ma")
	backend.data[key] = raw

	got, err := svc.Autocomplete(ctx, "123 ma")
	if err != nil {
		t.Fatalf("Autocomplete: %v", err)
	}
	if len(got) != 1 || got[0].PlaceID != "p1" {
		t.Fatalf("unexpected predictions: %+v", got)
	}
}

func TestCountryNamespacedCacheIsolation(t *testing.T) {
	backend := &countingBackend{data: make(map[string][]byte)}
	c := cache.New(backend, nil)
	svc := NewService("", c)

	ctx := context.Background()
	locUZ := ResolvedLocation{Address: "Tashkent Store", Lat: 41.31, Lng: 69.28}
	rawUZ, _ := json.Marshal(locUZ)
	backend.data[forwardCacheKey("uz", "main street")] = rawUZ

	locKZ := ResolvedLocation{Address: "Almaty Store", Lat: 43.25, Lng: 76.92}
	rawKZ, _ := json.Marshal(locKZ)
	backend.data[forwardCacheKey("kz", "main street")] = rawKZ

	gotUZ, err := svc.ForwardGeocode(ctx, "main street", "UZ")
	if err != nil || gotUZ.Address != "Tashkent Store" {
		t.Fatalf("expected Tashkent Store for UZ, got %v (err: %v)", gotUZ, err)
	}

	gotKZ, err := svc.ForwardGeocode(ctx, "main street", "KZ")
	if err != nil || gotKZ.Address != "Almaty Store" {
		t.Fatalf("expected Almaty Store for KZ, got %v (err: %v)", gotKZ, err)
	}
}

