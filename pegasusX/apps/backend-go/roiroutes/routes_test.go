package roiroutes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
	"github.com/pegasusx/pegasusx/apps/backend-go/roiroutes"
)

func TestROIRoutes_EndToEndMountAndRBAC(t *testing.T) {
	t.Parallel()

	const jwtSecret = "roi-routes-test-secret-32-bytes-min!"
	repo := roi.NewMemoryRepository()
	repo.SetPlatformFeesMinor("sup-roi-rt", 100000)
	svc := roi.NewServiceWithRepository(repo)

	r := chi.NewRouter()
	r.Use(auth.SessionAuth(jwtSecret))
	r.Use(auth.AttachTenantFromClaims)
	roiroutes.RegisterRoutes(r, roiroutes.Deps{
		Service:   svc,
		JWTSecret: jwtSecret,
	})

	issue := func(sub, supRole, actorType string) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:      sub,
			Role:         auth.RoleAdmin,
			SupplierID:   "sup-roi-rt",
			SupplierRole: auth.Role(supRole),
			ActorType:    actorType,
		}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	ownerTok := issue("owner-1", auth.SupplierRoleOwner, auth.ActorTypeHuman)
	auditorTok := issue("aud-1", auth.SupplierRoleAuditor, auth.ActorTypeHuman)

	call := func(method, path string, body any, tok string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	if rec := call(http.MethodGet, "/v1/roi/baseline", nil, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated GET /v1/roi/baseline = %d, want 401", rec.Code)
	}
	if rec := call(http.MethodPost, "/v1/roi/baseline", map[string]any{"window_days": 90}, ownerTok); rec.Code != http.StatusCreated {
		t.Fatalf("POST /v1/roi/baseline = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := call(http.MethodGet, "/v1/roi/baseline", nil, auditorTok); rec.Code != http.StatusOK {
		t.Fatalf("auditor GET /v1/roi/baseline = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := call(http.MethodPost, "/v1/roi/events", map[string]any{"category": "STOCKOUT_AVOIDED", "savings_minor": 350000}, ownerTok); rec.Code != http.StatusCreated {
		t.Fatalf("POST /v1/roi/events = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := call(http.MethodGet, "/v1/roi/events", nil, auditorTok); rec.Code != http.StatusOK {
		t.Fatalf("GET /v1/roi/events = %d", rec.Code)
	}
	sumRec := call(http.MethodGet, "/v1/roi/summary?days=90", nil, auditorTok)
	if sumRec.Code != http.StatusOK || !strings.Contains(sumRec.Body.String(), `"roi_multiple_bps":35000`) {
		t.Fatalf("GET /v1/roi/summary = %d body=%s", sumRec.Code, sumRec.Body.String())
	}
	repJSON := call(http.MethodGet, "/v1/roi/report?format=json&days=90", nil, auditorTok)
	if repJSON.Code != http.StatusOK {
		t.Fatalf("GET /v1/roi/report json = %d", repJSON.Code)
	}
	repCSV := call(http.MethodGet, "/v1/roi/report?format=csv&days=90", nil, auditorTok)
	if repCSV.Code != http.StatusOK || !strings.Contains(repCSV.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("GET /v1/roi/report csv = %d %s", repCSV.Code, repCSV.Header().Get("Content-Type"))
	}
}
