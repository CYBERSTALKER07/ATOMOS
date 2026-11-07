package syncroutes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func syncRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("sync-test-secret"))
	RegisterRoutes(r, deps)
	return r
}

func issueSyncToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "sync-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "sync-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestSyncCRDT_Unauthenticated(t *testing.T) {
	r := syncRouter(Deps{})

	req := httptest.NewRequest(http.MethodPost, "/v1/sync/crdt", strings.NewReader(`{"updates":[]}`))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestSyncCRDT_WrongRole(t *testing.T) {
	r := syncRouter(Deps{})

	tok := issueSyncToken(t, auth.RoleDriver) // Only RoleWarehouse is allowed
	req := httptest.NewRequest(http.MethodPost, "/v1/sync/crdt", strings.NewReader(`{"updates":[]}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for driver role, got %d", rr.Code)
	}
}

func TestSyncCRDT_UnavailableWhenSpannerNil(t *testing.T) {
	r := syncRouter(Deps{Engine: nil})

	tok := issueSyncToken(t, auth.RoleWarehouse)
	req := httptest.NewRequest(http.MethodPost, "/v1/sync/crdt", strings.NewReader(`{"updates":[]}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable when Spanner is nil, got %d: %s", rr.Code, rr.Body.String())
	}

	var res map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}
	if res["error"] != "sync_unavailable" {
		t.Fatalf("expected error='sync_unavailable', got %v", res["error"])
	}
}

func TestSyncCommands_BatchSizeLimit(t *testing.T) {
	r := syncRouter(Deps{})

	tok := issueSyncToken(t, auth.RoleDriver)

	// Create request with 51 commands (exceeding MaxSyncBatchSize = 50)
	commands := make([]SyncCommand, 51)
	for i := range commands {
		commands[i] = SyncCommand{CommandID: "cmd", CommandType: "MARK_DELIVERED", EntityID: "ord-1"}
	}
	body, _ := json.Marshal(SyncBatchRequest{Commands: commands})

	req := httptest.NewRequest(http.MethodPost, "/v1/sync/commands", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for batch > 50, got %d", rr.Code)
	}
}

func TestSyncCommands_EmptyBatch(t *testing.T) {
	r := syncRouter(Deps{})

	tok := issueSyncToken(t, auth.RoleDriver)
	body, _ := json.Marshal(SyncBatchRequest{Commands: []SyncCommand{}})

	req := httptest.NewRequest(http.MethodPost, "/v1/sync/commands", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for empty batch, got %d", rr.Code)
	}

	var resp SyncBatchResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(resp.Results))
	}
}
