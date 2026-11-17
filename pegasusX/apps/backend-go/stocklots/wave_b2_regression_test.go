package stocklots

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestHandleTraceLot_EnforcesWarehouseAndSupplierScope(t *testing.T) {
	ctx := context.Background()
	client := openStocklotsSpannerTestClient(t)
	defer client.Close()

	lotID := "lot-trace-scope-" + uuid.NewString()[:8]
	now := time.Now().UTC().Add(-2 * time.Minute)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotID, "SupplierId": "sup-trace-a", "WarehouseId": "wh-trace-a",
			"ProductId": "sku-trace", "LocationId": "bin-1", "LotCode": "LC-1",
			"QuantityOnHand": int64(10), "QuantityReserved": int64(0), "Status": "AVAILABLE",
			"ReceivedAt": now, "CreatedAt": now, "UpdatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed StockLots: %v", err)
	}

	h := &Handler{Spanner: client, SupplierID: "sup-trace-a"}

	// Foreign warehouse -> 403
	reqForeignWh := withChiParam(recallRequest(http.MethodGet, "/v1/warehouse/ops/lots/"+lotID+"/trace", "",
		auth.Claims{Subject: "wh-b", Role: auth.RoleWarehouse, SupplierID: "sup-trace-a", HomeNodeID: "wh-trace-b", HomeNodeType: auth.HomeNodeWarehouse},
	), "lotID", lotID)
	rrWh := httptest.NewRecorder()
	h.HandleTraceLot(rrWh, reqForeignWh)
	if rrWh.Code != http.StatusForbidden {
		t.Fatalf("foreign warehouse trace status=%d body=%s, want 403", rrWh.Code, rrWh.Body.String())
	}

	// Foreign supplier -> 403
	reqForeignSup := withChiParam(recallRequest(http.MethodGet, "/v1/warehouse/ops/lots/"+lotID+"/trace", "",
		auth.Claims{Subject: "wh-a", Role: auth.RoleWarehouse, SupplierID: "sup-trace-other", HomeNodeID: "wh-trace-a", HomeNodeType: auth.HomeNodeWarehouse},
	), "lotID", lotID)
	rrSup := httptest.NewRecorder()
	h.HandleTraceLot(rrSup, reqForeignSup)
	if rrSup.Code != http.StatusForbidden {
		t.Fatalf("foreign supplier trace status=%d body=%s, want 403", rrSup.Code, rrSup.Body.String())
	}

	// Matching warehouse and supplier -> 200
	reqOK := withChiParam(recallRequest(http.MethodGet, "/v1/warehouse/ops/lots/"+lotID+"/trace", "",
		auth.Claims{Subject: "wh-a", Role: auth.RoleWarehouse, SupplierID: "sup-trace-a", HomeNodeID: "wh-trace-a", HomeNodeType: auth.HomeNodeWarehouse},
	), "lotID", lotID)
	rrOK := httptest.NewRecorder()
	h.HandleTraceLot(rrOK, reqOK)
	if rrOK.Code != http.StatusOK {
		t.Fatalf("matching trace status=%d body=%s, want 200", rrOK.Code, rrOK.Body.String())
	}
}

func TestHandleRecallByID_EnforcesSupplierScope(t *testing.T) {
	ctx := context.Background()
	client := openStocklotsSpannerTestClient(t)
	defer client.Close()

	campaignID := "camp-scope-" + uuid.NewString()[:8]
	now := time.Now().UTC().Add(-2 * time.Minute)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("LotRecallCampaigns", map[string]any{
			"CampaignId": campaignID, "SupplierId": "sup-rec-a", "ProductId": "sku-rec",
			"RecallReason": "contamination", "Severity": "HIGH", "Status": "INITIATED",
			"InitiatedBy": "admin-a", "ImpactedLotCount": int64(1), "ImpactedUnitsCount": int64(5),
			"ImpactedOrderCount": int64(0), "CreatedAt": now, "UpdatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed LotRecallCampaigns: %v", err)
	}

	h := &Handler{Spanner: client}

	reqForbidden := withChiParam(recallRequest(http.MethodGet, "/v1/supplier/recalls/"+campaignID, "",
		auth.Claims{Subject: "admin-b", Role: auth.RoleAdmin, SupplierID: "sup-rec-other"},
	), "campaignID", campaignID)
	rrForb := httptest.NewRecorder()
	h.HandleRecallByID(rrForb, reqForbidden)
	if rrForb.Code != http.StatusForbidden {
		t.Fatalf("foreign supplier recall get status=%d body=%s, want 403", rrForb.Code, rrForb.Body.String())
	}

	reqOK := withChiParam(recallRequest(http.MethodGet, "/v1/supplier/recalls/"+campaignID, "",
		auth.Claims{Subject: "admin-a", Role: auth.RoleAdmin, SupplierID: "sup-rec-a"},
	), "campaignID", campaignID)
	rrOK := httptest.NewRecorder()
	h.HandleRecallByID(rrOK, reqOK)
	if rrOK.Code != http.StatusOK {
		t.Fatalf("matching supplier recall get status=%d body=%s, want 200", rrOK.Code, rrOK.Body.String())
	}
}

func TestSubmitCycleCountInTxn_ReturnsSubmittedStateWithoutStaleRead(t *testing.T) {
	ctx := context.Background()
	client := openStocklotsSpannerTestClient(t)
	defer client.Close()

	countID := "cnt-ryw-" + uuid.NewString()[:8]
	now := time.Now().UTC().Add(-2 * time.Minute)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("CycleCounts", map[string]any{
			"CountId": countID, "WarehouseId": "wh-cnt", "LocationId": "bin-1",
			"ProductId": "sku-cnt", "ExpectedQty": int64(10), "Status": "OPEN",
			"CreatedAt": now, "UpdatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed CycleCounts: %v", err)
	}

	var got *CycleCountView
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		v, serr := SubmitCycleCountInTxn(ctx, txn, countID, "counter-1", 8, "DAMAGED")
		if serr != nil {
			return serr
		}
		got = v
		return nil
	})
	if err != nil {
		t.Fatalf("SubmitCycleCountInTxn: %v", err)
	}
	if got == nil || got.Status != "SUBMITTED" || got.CountedQty == nil || *got.CountedQty != 8 || got.VarianceQty == nil || *got.VarianceQty != -2 || got.ReasonCode != "DAMAGED" {
		t.Fatalf("unexpected CycleCountView returned in same txn: %+v", got)
	}
}

func TestHandleConfirmPickTask_DisabledReturns409(t *testing.T) {
	SetPickWavesEnabled(false)
	h := &Handler{Spanner: &spanner.Client{}}
	req := withChiParam(httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/pick-waves/w1/tasks/t1/confirm?warehouse_id=wh1",
		strings.NewReader(`{"quantity_picked":1,"picker_id":"p1"}`)), "waveID", "w1")
	rr := httptest.NewRecorder()
	h.HandleConfirmPickTask(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("disabled pick wave confirm status=%d body=%s want 409", rr.Code, rr.Body.String())
	}
}
