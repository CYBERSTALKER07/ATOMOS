package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DashboardTTL is the rollup freshness window (GS-UF: 15–30s).
const DashboardTTL = 20 * time.Second

// DashboardKey is dash:{role}:{scope}:today — the command-dashboard rollup key.
func DashboardKey(role, scopeID string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	scopeID = strings.TrimSpace(scopeID)
	if role == "" {
		role = "unknown"
	}
	if scopeID == "" {
		scopeID = "_"
	}
	return fmt.Sprintf("dash:%s:%s:today", role, scopeID)
}

// WeakETag is a short weak validator over the exact response bytes.
func WeakETag(body []byte) string {
	sum := sha256.Sum256(body)
	return `W/"` + hex.EncodeToString(sum[:8]) + `"`
}

// LoadDashboard returns cached rollup bytes or calls loader on miss.
func LoadDashboard(c *Cache, ctx context.Context, key string, loader func(context.Context) ([]byte, error)) ([]byte, error) {
	if c == nil {
		return loader(ctx)
	}
	return c.GetOrLoad(ctx, key, DashboardTTL, loader)
}

// WriteJSONWithETag writes JSON bytes or 304 when If-None-Match matches.
func WriteJSONWithETag(w http.ResponseWriter, r *http.Request, body []byte) {
	etag := WeakETag(body)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func etagMatches(ifNoneMatch, etag string) bool {
	ifNoneMatch = strings.TrimSpace(ifNoneMatch)
	if ifNoneMatch == "" {
		return false
	}
	if ifNoneMatch == "*" {
		return true
	}
	want := unwrapETag(etag)
	for _, part := range strings.Split(ifNoneMatch, ",") {
		cand := strings.TrimSpace(part)
		if cand == etag || unwrapETag(cand) == want {
			return true
		}
	}
	return false
}

func unwrapETag(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "W/")
	return strings.Trim(v, `"`)
}
