package stocklots

import (
	"context"
	"encoding/json"
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

func TestHandleApproveInventoryAdjustment_MakerCheckerSoD_LiveSpanner(t *testing.T) {
	client := openStocklotsSpannerTestClient(t)
	SetCycleCountsEnabled(true)
	t.Cleanup(func() { SetCycleCountsEnabled(false) })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	sid := "sup-" + uuid.NewString()[:8]
	wid := "wh-" + uuid.NewString()[:8]
	pid := "prod-" + uuid.NewString()[:8]
	lid := "lot-" + uuid.NewString()[:8]
	adjID := "adj-" + uuid.NewString()[:8]
	makerID := "user-counter-1"
	checkerID := "user-manager-2"

	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("InventoryAdjustments", spanner.Key{adjID}),
			spanner.Delete("StockLots", spanner.Key{lid}),
			spanner.Delete("SupplierInventoryV2", spanner.KeyRange{
				Start: spanner.Key{sid, wid},
				End:   spanner.Key{sid, wid},
				Kind:  spanner.ClosedClosed,
			}),
		})
	})

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       sid,
			"WarehouseId":      wid,
			"ProductId":        pid,
			"QuantityOnHand":   int64(20),
			"QuantityReserved": int64(0),
			"ReorderThreshold": int64(2),
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId":            lid,
			"SupplierId":       sid,
			"WarehouseId":      wid,
			"ProductId":        pid,
			"LocationId":       "bin-A1",
			"QuantityOnHand":   int64(20),
			"QuantityReserved": int64(0),
			"Status":           "AVAILABLE",
			"ReceivedAt":       spanner.CommitTimestamp,
			"CreatedAt":        spanner.CommitTimestamp,
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
		spanner.InsertMap("InventoryAdjustments", map[string]any{
			"AdjustmentId": adjID,
			"WarehouseId":  wid,
			"ProductId":    pid,
			"LotId":        lid,
			"DeltaQty":     int64(5),
			"ReasonCode":   "CYCLE_VARIANCE",
			"Status":       "PENDING",
			"ActorId":      makerID,
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed adjustment: %v", err)
	}

	var recordedSoD []SoDCheckInput
	h := &Handler{
		Spanner:    client,
		SupplierID: sid,
		SoDCheck: func(_ context.Context, in SoDCheckInput) error {
			recordedSoD = append(recordedSoD, in)
			if in.MakerID != "" && in.MakerID == in.ActorID {
				return ErrSodViolation
			}
			return nil
		},
	}

	r := chi.NewRouter()
	r.Post("/v1/warehouse/ops/inventory-adjustments/{adjustmentID}/approve", h.HandleApproveInventoryAdjustment)

	sendApprove := func(claims auth.Claims) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/inventory-adjustments/"+adjID+"/approve?warehouse_id="+wid, strings.NewReader("{}"))
		cctx := auth.WithClaims(req.Context(), claims)
		cctx = context.WithValue(cctx, auth.WarehouseOpsKey, &auth.WarehouseOps{
			SupplierID:  sid,
			WarehouseID: wid,
		})
		req = req.WithContext(cctx)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	// 1. Maker attempting self-approval fails with 403 sod_violation.
	rrSelf := sendApprove(auth.Claims{
		Subject:      makerID,
		Role:         auth.RoleWarehouseAdmin,
		SupplierRole: auth.SupplierRoleWarehouseManager,
		SupplierID:   sid,
	})
	if rrSelf.Code != http.StatusForbidden {
		t.Fatalf("self-approve status=%d body=%s want 403", rrSelf.Code, rrSelf.Body.String())
	}
	if !strings.Contains(rrSelf.Body.String(), "sod_violation") {
		t.Fatalf("self-approve body=%s want sod_violation", rrSelf.Body.String())
	}

	// 2. Direct ApproveAdjustmentInTxn with same maker also returns ErrSodViolation atomically.
	_, txnErr := client.ReadWriteTransaction(ctx, func(tctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := ApproveAdjustmentInTxn(tctx, txn, adjID, makerID)
		return err
	})
	if txnErr == nil || !strings.Contains(txnErr.Error(), "sod_violation") {
		t.Fatalf("ApproveAdjustmentInTxn self-approve err=%v want sod_violation", txnErr)
	}

	// 3. Different checker approves cleanly -> 200 APPROVED, stock delta applied (+5 => 25).
	rrCheck := sendApprove(auth.Claims{
		Subject:      checkerID,
		Role:         auth.RoleWarehouseAdmin,
		SupplierRole: auth.SupplierRoleWarehouseManager,
		SupplierID:   sid,
	})
	if rrCheck.Code != http.StatusOK {
		t.Fatalf("checker approve status=%d body=%s want 200", rrCheck.Code, rrCheck.Body.String())
	}
	var view InventoryAdjustmentView
	if err := json.Unmarshal(rrCheck.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode view: %v", err)
	}
	if view.Status != "APPROVED" || view.ApprovedBy != checkerID {
		t.Fatalf("unexpected view: %+v", view)
	}
	if len(recordedSoD) != 2 || recordedSoD[0].MakerID != makerID || recordedSoD[1].ActorID != checkerID {
		t.Fatalf("unexpected SoDCheck invocations: %+v", recordedSoD)
	}

	lotRow, err := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "StockLots", spanner.Key{lid}, []string{"QuantityOnHand"})
	if err != nil {
		t.Fatalf("read lot: %v", err)
	}
	var qty int64
	if err := lotRow.Columns(&qty); err != nil {
		t.Fatalf("scan qty: %v", err)
	}
	if qty != 25 {
		t.Fatalf("lot QuantityOnHand=%d want 25", qty)
	}
}
