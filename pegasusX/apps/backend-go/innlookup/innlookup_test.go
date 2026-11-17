package innlookup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type countingAdapter struct {
	inner LookupAdapter
	calls atomic.Int64
}

func (a *countingAdapter) Lookup(ctx context.Context, homeCell, normalizedINN string) (LookupResult, error) {
	a.calls.Add(1)
	return a.inner.Lookup(ctx, homeCell, normalizedINN)
}

func TestNormalizeTaxID(t *testing.T) {
	cases := []struct {
		raw      string
		wantNorm string
		wantType string
		wantErr  bool
	}{
		{"301234567", "301234567", EntityTypeYuridik, false},
		{" INN 301-234 567 ", "301234567", EntityTypeYuridik, false},
		{"STIR: 301 234 567", "301234567", EntityTypeYuridik, false},
		{"PINFL 31234567890123", "31234567890123", EntityTypePinfl, false},
		{"000000000", "", "", true},
		{"12345678", "", "", true},
		{"1234567890123", "", "", true},
		{"30123456A", "", "", true},
		{"", "", "", true},
	}
	for _, tc := range cases {
		norm, typ, err := NormalizeTaxID(tc.raw)
		if tc.wantErr {
			if !errors.Is(err, ErrInvalidTaxID) {
				t.Fatalf("NormalizeTaxID(%q) err=%v want ErrInvalidTaxID", tc.raw, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizeTaxID(%q) unexpected err=%v", tc.raw, err)
		}
		if norm != tc.wantNorm || typ != tc.wantType {
			t.Fatalf("NormalizeTaxID(%q)=(%q,%q) want (%q,%q)", tc.raw, norm, typ, tc.wantNorm, tc.wantType)
		}
	}
}

func TestService_Cache24hAndExpiryAndUnavailableNotCached(t *testing.T) {
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	stub := NewStubAdapter()
	stub.Register("301234567", "Baraka Food LLC", "Tashkent, Yunusabad 11")
	counted := &countingAdapter{inner: stub}
	mem := NewMemoryStore()
	mem.RegisterTenant(TenantMatch{
		HomeCell:      "cell-uz",
		TenantType:    TenantTypeSupplier,
		TenantID:      "sup-existing",
		LegalName:     "Baraka Food LLC",
		NormalizedINN: "301234567",
	})

	svc := NewService(Config{
		Adapter:      counted,
		Cache:        mem,
		Tenants:      mem,
		JoinRequests: mem,
		CacheTTL:     24 * time.Hour,
		Now:          func() time.Time { return now },
	})

	// 1. Cache miss hits adapter once + attaches duplicate tenant match.
	res, err := svc.Lookup(context.Background(), LookupOptions{
		TaxID:      "INN 301-234 567",
		HomeCell:   "cell-uz",
		TenantType: TenantTypeSupplier,
	})
	if err != nil {
		t.Fatalf("first lookup err=%v", err)
	}
	if res.Status != StatusFound || res.Cached || res.ExistingTenantID != "sup-existing" || !res.ExistingJoinEligible {
		t.Fatalf("unexpected first result: %+v", res)
	}
	if counted.calls.Load() != 1 {
		t.Fatalf("calls=%d want 1", counted.calls.Load())
	}

	// 2. Within 24h returns Cached=true without calling adapter.
	now = now.Add(23 * time.Hour)
	res2, err := svc.Lookup(context.Background(), LookupOptions{
		TaxID:      "301234567",
		HomeCell:   "cell-uz",
		TenantType: TenantTypeSupplier,
	})
	if err != nil {
		t.Fatalf("cached lookup err=%v", err)
	}
	if !res2.Cached || res2.LegalName != "Baraka Food LLC" {
		t.Fatalf("expected cached hit, got %+v", res2)
	}
	if counted.calls.Load() != 1 {
		t.Fatalf("calls=%d want 1 after cache hit", counted.calls.Load())
	}

	// 3. Past 24h expires and re-fetches from adapter.
	now = now.Add(2 * time.Hour)
	res3, err := svc.Lookup(context.Background(), LookupOptions{
		TaxID:    "301234567",
		HomeCell: "cell-uz",
	})
	if err != nil || res3.Cached || counted.calls.Load() != 2 {
		t.Fatalf("expected expired refresh calls=%d res=%+v err=%v", counted.calls.Load(), res3, err)
	}

	// 4. UNAVAILABLE is fail-closed and NEVER cached.
	stub.SetForceUnavailable(true)
	_, err = svc.Lookup(context.Background(), LookupOptions{TaxID: "309999999", HomeCell: "cell-uz"})
	if !errors.Is(err, ErrRegistryUnavailable) {
		t.Fatalf("expected ErrRegistryUnavailable, got %v", err)
	}
	stub.SetForceUnavailable(false)
	stub.Register("309999999", "Samarkand Trade", "Samarkand")
	res4, err := svc.Lookup(context.Background(), LookupOptions{TaxID: "309999999", HomeCell: "cell-uz"})
	if err != nil || res4.Status != StatusFound {
		t.Fatalf("expected clean recovery after unavailable, got res=%+v err=%v", res4, err)
	}
}

func TestService_RateLimitAndSoliqHTTPAdapter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("inn") {
		case "301111111":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"found":      true,
				"legal_name": "Soliq Verified LLC",
				"address":    "Tashkent",
			})
		case "302222222":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusBadGateway)
		}
	}))
	defer srv.Close()

	adapter := &SoliqHTTPAdapter{BaseURL: srv.URL, APIKey: "k-1"}
	svc := NewService(Config{
		Adapter:            adapter,
		RateLimitPerMinute: 2,
	})

	r1, err := svc.Lookup(context.Background(), LookupOptions{TaxID: "301111111", ClientKey: "ip:1.2.3.4"})
	if err != nil || r1.Status != StatusFound || r1.LegalName != "Soliq Verified LLC" {
		t.Fatalf("found case failed: %+v err=%v", r1, err)
	}
	r2, err := svc.Lookup(context.Background(), LookupOptions{TaxID: "302222222", ClientKey: "ip:1.2.3.4"})
	if err != nil || r2.Status != StatusNotFound {
		t.Fatalf("not found case failed: %+v err=%v", r2, err)
	}
	_, err = svc.Lookup(context.Background(), LookupOptions{TaxID: "303333333", ClientKey: "ip:1.2.3.4"})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("3rd cache miss should rate limit, got err=%v", err)
	}

	// Test 502 fail-closed from SoliqHTTPAdapter directly.
	if res, err := adapter.Lookup(context.Background(), "cell-uz", "305555555"); !errors.Is(err, ErrRegistryUnavailable) || res.Status != StatusUnavailable {
		t.Fatalf("expected fail-closed unavailable, got res=%+v err=%v", res, err)
	}
	// Test nil/empty BaseURL fail-closed.
	if res, err := (&SoliqHTTPAdapter{}).Lookup(context.Background(), "cell-uz", "301111111"); !errors.Is(err, ErrRegistryUnavailable) || res.Status != StatusUnavailable {
		t.Fatalf("expected empty adapter unavailable, got res=%+v err=%v", res, err)
	}
}

func TestHandleLookup_HTTPEndpointsAndCrossCellGuard(t *testing.T) {
	stub := NewStubAdapter()
	stub.Register("304444444", "Navoi Agro", "Navoi")
	svc := NewService(Config{Adapter: stub})

	r := chi.NewRouter()
	r.Post("/v1/platform/inn/lookup", svc.HandleLookup)
	r.Get("/v1/onboarding/tax-id/{inn}", svc.HandleLookup)

	// POST /v1/platform/inn/lookup
	body, _ := json.Marshal(map[string]string{"tax_id": "304444444", "market_code": "UZ"})
	req := httptest.NewRequest(http.MethodPost, "/v1/platform/inn/lookup", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST status=%d body=%s", rr.Code, rr.Body.String())
	}

	// GET /v1/onboarding/tax-id/{inn} with cross-cell claim mismatch -> 403
	req2 := httptest.NewRequest(http.MethodGet, "/v1/onboarding/tax-id/304444444?home_cell=cell-kz", nil)
	ctx := auth.WithClaims(req2.Context(), auth.Claims{Subject: "u1", HomeCell: "cell-uz"})
	req2 = req2.WithContext(ctx)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("cross-cell status=%d body=%s", rr2.Code, rr2.Body.String())
	}

	// Nil adapter returns 503 fail-closed
	unavailSvc := NewService(Config{})
	r3 := chi.NewRouter()
	r3.Post("/v1/platform/inn/lookup", unavailSvc.HandleLookup)
	req3 := httptest.NewRequest(http.MethodPost, "/v1/platform/inn/lookup", bytes.NewReader(body))
	rr3 := httptest.NewRecorder()
	r3.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable adapter status=%d want 503", rr3.Code)
	}
}

func TestMemoryStore_JoinRequestDedupeAndDecision(t *testing.T) {
	mem := NewMemoryStore()
	now := time.Date(2026, 11, 14, 11, 0, 0, 0, time.UTC)

	jr1, err := mem.CreateJoinRequest(context.Background(), JoinRequest{
		HomeCell:         "cell-uz",
		TargetTenantType: TenantTypeSupplier,
		TargetTenantID:   "sup-1",
		NormalizedINN:    "301234567",
		RequesterPhone:   "+998901234567",
		RequesterName:    "Ali",
		RoleSought:       string(auth.RoleWarehouse),
		CreatedAt:        now,
	}, nil)
	if err != nil {
		t.Fatalf("create jr1: %v", err)
	}
	jr2, err := mem.CreateJoinRequest(context.Background(), JoinRequest{
		HomeCell:         "cell-uz",
		TargetTenantType: TenantTypeSupplier,
		TargetTenantID:   "sup-1",
		NormalizedINN:    "301234567",
		RequesterPhone:   "+998901234567",
	}, nil)
	if err != nil || jr2.RequestID != jr1.RequestID {
		t.Fatalf("expected idempotent pending reuse, got jr1=%s jr2=%s err=%v", jr1.RequestID, jr2.RequestID, err)
	}

	decided, err := mem.DecideJoinRequest(context.Background(), TenantTypeSupplier, "sup-1", jr1.RequestID, JoinStatusApproved, "admin-1", "verified", now.Add(time.Minute), nil)
	if err != nil || decided.Status != JoinStatusApproved {
		t.Fatalf("decide failed: %+v err=%v", decided, err)
	}
	if role, ok := mem.MembershipRole("sup-1", "+998901234567"); !ok || role != string(auth.RoleWarehouse) {
		t.Fatalf("expected membership WAREHOUSE, got %q ok=%v", role, ok)
	}
	if _, err := mem.DecideJoinRequest(context.Background(), TenantTypeSupplier, "sup-1", jr1.RequestID, JoinStatusRejected, "admin-1", "", now.Add(2*time.Minute), nil); !errors.Is(err, ErrJoinRequestDecided) {
		t.Fatalf("expected ErrJoinRequestDecided on double decide, got %v", err)
	}
}
