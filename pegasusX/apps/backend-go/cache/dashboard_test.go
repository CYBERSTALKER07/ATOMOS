package cache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDashboardKey(t *testing.T) {
	t.Parallel()
	got := DashboardKey("supplier", "sup-1")
	if got != "dash:supplier:sup-1:today" {
		t.Fatalf("key=%q", got)
	}
}

func TestWriteJSONWithETag_304(t *testing.T) {
	t.Parallel()
	body := []byte(`{"active_orders":0}`)
	etag := WeakETag(body)

	req := httptest.NewRequest(http.MethodGet, "/dash", nil)
	rr := httptest.NewRecorder()
	WriteJSONWithETag(rr, req, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("first status=%d", rr.Code)
	}
	if rr.Header().Get("ETag") != etag {
		t.Fatalf("etag=%q want=%q", rr.Header().Get("ETag"), etag)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/dash", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	WriteJSONWithETag(rr2, req2, body)
	if rr2.Code != http.StatusNotModified {
		t.Fatalf("second status=%d want 304", rr2.Code)
	}
	if rr2.Body.Len() != 0 {
		t.Fatalf("304 must not write a body: %s", rr2.Body.String())
	}
}

func TestLoadDashboard_GetOrLoadThenInvalidate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := New(NewInMemoryBackend(), nil)
	loads := 0
	loader := func(context.Context) ([]byte, error) {
		loads++
		return []byte(`{"n":1}`), nil
	}
	key := DashboardKey("warehouse", "wh-1")
	first, err := LoadDashboard(c, ctx, key, loader)
	if err != nil || string(first) != `{"n":1}` {
		t.Fatalf("first=%s err=%v", first, err)
	}
	second, err := LoadDashboard(c, ctx, key, loader)
	if err != nil || string(second) != `{"n":1}` {
		t.Fatalf("second=%s err=%v", second, err)
	}
	if loads != 1 {
		t.Fatalf("loads=%d want 1 (singleflight/cache)", loads)
	}
	c.Invalidate(ctx, key)
	third, err := LoadDashboard(c, ctx, key, loader)
	if err != nil || string(third) != `{"n":1}` {
		t.Fatalf("third=%s err=%v", third, err)
	}
	if loads != 2 {
		t.Fatalf("loads=%d want 2 after invalidate", loads)
	}
}
