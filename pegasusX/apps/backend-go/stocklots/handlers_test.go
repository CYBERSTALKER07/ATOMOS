package stocklots

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func recallRequest(method, target, body string, claims auth.Claims) *http.Request {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	ctx := auth.WithClaims(req.Context(), claims)
	if claims.Role == auth.RoleWarehouse || claims.Role == auth.RoleWarehouseAdmin {
		ctx = context.WithValue(ctx, auth.WarehouseOpsKey, &auth.WarehouseOps{
			WarehouseID: claims.HomeNodeID,
			SupplierID:  claims.SupplierID,
			Subject:     claims.Subject,
		})
	}
	return req.WithContext(ctx)
}

func decodeErr(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rr.Body.String(), err)
	}
	return body["error"]
}

func TestHandleRecalls_RejectsBodySupplierMismatch(t *testing.T) {
	h := &Handler{}
	req := recallRequest(http.MethodPost, "/v1/supplier/recalls",
		`{"supplier_id":"sup-other","product_id":"p1","recall_reason":"contamination"}`,
		auth.Claims{Subject: "ceo", Role: auth.RoleAdmin, SupplierID: "sup-a"},
	)
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := decodeErr(t, rr); got != "supplier_scope_forbidden" {
		t.Fatalf("error=%q", got)
	}
}

func TestHandleRecalls_RejectsQuerySupplierMismatch(t *testing.T) {
	h := &Handler{}
	req := recallRequest(http.MethodGet, "/v1/warehouse/ops/recalls?supplier_id=sup-other", "",
		auth.Claims{Subject: "wh", Role: auth.RoleWarehouse, SupplierID: "sup-a", HomeNodeID: "wh-1", HomeNodeType: auth.HomeNodeWarehouse},
	)
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := decodeErr(t, rr); got != "supplier_scope_forbidden" {
		t.Fatalf("error=%q", got)
	}
}

func TestHandleRecalls_BodyDoesNotCreateSupplier(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/recalls",
		strings.NewReader(`{"supplier_id":"sup-other","product_id":"p1","recall_reason":"x"}`))
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := decodeErr(t, rr); got != "supplier_id_required" {
		t.Fatalf("error=%q", got)
	}
}

func TestHandleRecalls_SeedDoesNotAcceptForeignBody(t *testing.T) {
	h := &Handler{SupplierID: "seed-supplier"}
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/recalls",
		strings.NewReader(`{"supplier_id":"sup-other","product_id":"p1","recall_reason":"x"}`))
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := decodeErr(t, rr); got != "supplier_scope_forbidden" {
		t.Fatalf("error=%q", got)
	}
}

func TestHandleRecalls_MatchingSupplierReachesStore(t *testing.T) {
	h := &Handler{}
	req := recallRequest(http.MethodPost, "/v1/supplier/recalls",
		`{"supplier_id":"sup-a","product_id":"p1","recall_reason":"contamination"}`,
		auth.Claims{Subject: "ceo", Role: auth.RoleAdmin, SupplierID: "sup-a"},
	)
	rr := httptest.NewRecorder()
	h.HandleRecalls(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
