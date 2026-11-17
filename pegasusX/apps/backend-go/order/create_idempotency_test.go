package order

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/packages/handoff"
)

func TestHandleCreate_RequiresIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(`{"line_items":[]}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1",
		Role:    auth.RoleRetailer,
	}))
	rec := httptest.NewRecorder()
	svc.HandleCreate(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
	var errResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil || errResp["error"] != "idempotency_key_required" {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestCustodyMutations_RequireIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	cases := []struct {
		name string
		path string
		body string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"arrive", "/v1/delivery/arrive", `{"order_id":"ord-1"}`, svc.HandleMarkArrived},
		{"deliver", "/v1/order/deliver", `{"order_id":"ord-1"}`, svc.HandleSubmitDelivery},
		{"offload", "/v1/order/confirm-offload", `{"order_id":"ord-1"}`, svc.HandleConfirmOffload},
		{"complete", "/v1/order/complete", `{"order_id":"ord-1"}`, svc.HandleCompleteOrder},
		{"shop-closed", "/v1/delivery/shop-closed", `{"order_id":"ord-1"}`, svc.HandleReportShopClosed},
		{"partial-offload", "/v1/delivery/partial-offload", `{"order_id":"ord-1"}`, svc.HandlePartialOffload},
		{"sync-batch", "/v1/sync/batch", `{"deliveries":[]}`, svc.HandleSyncBatch},
		{"credit-leave", "/v1/driver/orders/ord-1/credit-leave", `{}`, svc.HandleCreditLeave},
		{"bypass-offload", "/v1/delivery/bypass-offload", `{"order_id":"ord-1"}`, svc.HandleBypassOffload},
		{"proximity", "/v1/delivery/proximity-unlock", `{"order_id":"ord-1"}`, svc.HandleProximityUnlock},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
				Subject: "drv-1",
				Role:    auth.RoleDriver,
			}))
			rec := httptest.NewRecorder()
			tc.call(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "idempotency_key_required") {
				t.Fatalf("body=%s", rec.Body.String())
			}
		})
	}
}

func TestHandleCollectCash_RequiresIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/order/collect-cash", strings.NewReader(`{"order_id":"ord-1"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "drv-1",
		Role:    auth.RoleDriver,
	}))
	rec := httptest.NewRecorder()
	svc.HandleCollectCash(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleCreate_CrossTenantIdempotencyIsolated(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{}
	svc := NewService(ServiceConfig{
		Repo:      repo,
		Warehouse: &testWarehouseResolver{warehouseID: "wh-1"},
		Idem:      idempotency.NewInMemoryStore(),
		Now:       func() time.Time { return now },
		Log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		NewID:     func() string { return "ord-idem-test" },
	})

	doCreate := func(orgID, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(body))
		req.Header.Set("Idempotency-Key", "shared-create-key")
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject:       "multi-org-buyer",
			Role:          auth.RoleRetailer,
			RetailerOrgID: orgID,
		}))
		rec := httptest.NewRecorder()
		svc.HandleCreate(rec, req)
		return rec
	}

	bodyA := `{"supplier_id":"sup-1","h3_cell":"872830828ffffff","lat":41.3,"lng":69.2,"line_items":[{"sku":"sku-1","quantity":1,"unit_price_minor":5000}]}`
	bodyB := `{"supplier_id":"sup-1","h3_cell":"872830828ffffff","lat":41.3,"lng":69.2,"line_items":[{"sku":"sku-2","quantity":2,"unit_price_minor":3000}]}`

	rec1 := doCreate("ret-org-a", bodyA)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("org-a status=%d body=%s", rec1.Code, rec1.Body.String())
	}
	if repo.created.RetailerID != "ret-org-a" || !strings.Contains(rec1.Body.String(), `"total_minor":5000`) {
		t.Fatalf("unexpected org-a response=%s created=%+v", rec1.Body.String(), repo.created)
	}

	// Same Subject + same Idempotency-Key, different RetailerOrgID + different body:
	// must NOT collide with 409 Conflict or replay org-a's response.
	rec2 := doCreate("ret-org-b", bodyB)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("org-b status=%d want 201 body=%s", rec2.Code, rec2.Body.String())
	}
	if repo.created.RetailerID != "ret-org-b" || !strings.Contains(rec2.Body.String(), `"total_minor":6000`) {
		t.Fatalf("expected org-b order (6000) in response, got %s created=%+v", rec2.Body.String(), repo.created)
	}
}

func TestSyncBatch_CrossTenantScopedIdempotency(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	signer := handoff.FromEnv()
	idem := idempotency.NewInMemoryStore()
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-batch-1",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			DriverID:   "drv-a",
			QRToken:    "qr-secret-1",
			Status:     StatusArrived,
			TotalMinor: 10000,
		},
	}
	svc := NewService(ServiceConfig{
		Repo:    repo,
		Idem:    idem,
		Handoff: signer,
		Now:     func() time.Time { return now },
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	tok := svc.publicDeliveryToken(repo.order)
	sig := signer.HashToken(tok)

	// Record cached batch success under driver A's tenant scope.
	idA, err := svc.processBatchDelivery(context.Background(), auth.Claims{
		Subject:    "drv-a",
		Role:       auth.RoleDriver,
		SupplierID: "sup-1",
	}, BatchDelivery{OrderID: "ord-batch-1", Signature: sig})
	if err != nil || idA != "ord-batch-1" {
		t.Fatalf("drv-a err=%v id=%q", err, idA)
	}

	// Driver B in another supplier must NOT hit drv-a's cached idempotency record
	// (it reaches ownership check and gets ErrForbidden instead of silent hit).
	_, errB := svc.processBatchDelivery(context.Background(), auth.Claims{
		Subject:    "drv-b",
		Role:       auth.RoleDriver,
		SupplierID: "sup-2",
	}, BatchDelivery{OrderID: "ord-batch-1", Signature: sig})
	if errB == nil {
		t.Fatal("expected driver B cross-tenant batch delivery to be rejected, not idempotently replayed")
	}
}
