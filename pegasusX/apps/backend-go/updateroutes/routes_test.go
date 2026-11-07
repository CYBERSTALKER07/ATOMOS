package updateroutes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func testRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	RegisterRoutes(r, deps)
	return r
}

func TestIOSManifest_UnconfiguredBaseURL(t *testing.T) {
	r := testRouter(Deps{BaseURL: ""})

	req := httptest.NewRequest(http.MethodGet, "/v1/updates/ios/driver-app/manifest.plist", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable, got %d", rr.Code)
	}
}

func TestIOSManifest_Success(t *testing.T) {
	r := testRouter(Deps{
		BaseURL:        "https://cdn.pegasusx.net",
		DefaultVersion: "2.1.0",
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/updates/ios/driver-app/manifest.plist", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/x-apple-aspen-metadata") {
		t.Fatalf("unexpected content type: %s", contentType)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "https://cdn.pegasusx.net/downloads/ios/driver-app.ipa") {
		t.Fatalf("manifest missing expected ipa url: %s", body)
	}
	if !strings.Contains(body, "<string>2.1.0</string>") {
		t.Fatalf("manifest missing expected version 2.1.0: %s", body)
	}
}

func TestDesktopUpdater_UnconfiguredBaseURL(t *testing.T) {
	r := testRouter(Deps{BaseURL: ""})

	req := httptest.NewRequest(http.MethodGet, "/v1/updates/desktop/control-tower/updater.json", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable, got %d", rr.Code)
	}
}

func TestDesktopUpdater_Success(t *testing.T) {
	r := testRouter(Deps{
		BaseURL:        "https://cdn.pegasusx.net",
		DefaultVersion: "3.0.0",
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/updates/desktop/control-tower/updater.json", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	var data struct {
		Version   string `json:"version"`
		Platforms map[string]struct {
			URL       string `json:"url"`
			Signature string `json:"signature"`
		} `json:"platforms"`
	}

	if err := json.NewDecoder(rr.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode updater.json: %v", err)
	}

	if data.Version != "3.0.0" {
		t.Fatalf("expected version '3.0.0', got %q", data.Version)
	}

	darwinArm, ok := data.Platforms["darwin-aarch64"]
	if !ok {
		t.Fatalf("expected darwin-aarch64 platform entry")
	}
	if !strings.HasPrefix(darwinArm.URL, "https://cdn.pegasusx.net/downloads/desktop/control-tower") {
		t.Fatalf("unexpected darwin url: %s", darwinArm.URL)
	}
}
