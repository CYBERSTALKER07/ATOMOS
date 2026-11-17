package e2ep0

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/recon"
)

func TestE2E_ReconRoutesMountedAndRBACEnforced(t *testing.T) {
	h := NewTestHarness(t)
	if h.ReconSvc == nil {
		t.Fatal("expected TestHarness.ReconSvc to be wired")
	}

	// Unauthenticated -> 401.
	if unauth := h.Client.Get("/v1/finance/recon/runs", ""); unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /v1/finance/recon/runs = %d, want 401", unauth.StatusCode)
	}

	// Non-admin role (RETAILER) -> 403.
	retTok := h.IssueRetailerToken(DefaultSupplierA, "ret-user-1")
	if forbidden := h.Client.Get("/v1/finance/recon/breaks", retTok); forbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("retailer /v1/finance/recon/breaks = %d, want 403", forbidden.StatusCode)
	}

	// Admin -> POST /v1/finance/recon/run -> GET runs & breaks cleanly.
	admTok := h.IssueAdminToken(DefaultSupplierA, "adm-user-1")
	if mem, ok := h.ReconSvc.Repo().(*recon.MemoryRepository); ok {
		mem.SeedSnapshot(recon.Snapshot{
			EntityID:             DefaultSupplierA,
			PeriodID:             "2026-11",
			Currency:             "UZS",
			OpenARBalanceMinor:   100_000,
			GLARRetailerNetMinor: 85_000,
		})
	}

	postResp := h.Client.Post("/v1/finance/recon/run", map[string]any{"period_id": "2026-11"}, admTok)
	if postResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /v1/finance/recon/run = %d body=%s", postResp.StatusCode, string(postResp.Body))
	}
	var out recon.RunOutput
	if err := json.Unmarshal(postResp.Body, &out); err != nil {
		t.Fatalf("decode RunOutput: %v", err)
	}
	if out.Run.BreakCount != 1 || len(out.Breaks) != 1 {
		t.Fatalf("expected 1 break from seeded snapshot, got %+v", out)
	}

	runsResp := h.Client.Get("/v1/finance/recon/runs", admTok)
	if runsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/finance/recon/runs = %d body=%s", runsResp.StatusCode, string(runsResp.Body))
	}
}
