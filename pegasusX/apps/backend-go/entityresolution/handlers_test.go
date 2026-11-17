package entityresolution

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleResolve_Unauthorized(t *testing.T) {
	svc := NewService(fakeRepository{})
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/resolve", strings.NewReader(`{"entity_id":"x"}`))
	rr := httptest.NewRecorder()
	HandleResolve(svc)(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleResolve_DoesNotWrapStatusOK(t *testing.T) {
	svc := NewService(fakeRepository{
		findExact: func(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error) {
			return []EntityRecord{{
				EntityType: EntityTypeOrder,
				EntityID:   "ord-1",
				Label:      "ord-1",
				SearchText: "ord-1",
			}}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/resolve", strings.NewReader(`{"entity_type":"ORDER","entity_id":"ord-1"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleAdmin, Subject: "u1", SupplierID: "sup-1"}))
	rr := httptest.NewRecorder()
	HandleResolve(svc)(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if _, has := m["status"]; has {
		t.Fatal("must not wrap {status:ok}")
	}
	if m["scope_supplier_id"] != "sup-1" {
		t.Fatalf("body=%s", rr.Body.String())
	}
}
