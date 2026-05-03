package factory

import (
	"net/http/httptest"
	"testing"

	"backend-go/auth"
)

func TestResolveSupplyLaneAction_UsesSupplierScopeFromClaims(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		claims            *auth.PegasusClaims
		wantLaneID        string
		wantSupplierID    string
		wantTransitUpdate bool
		wantErr           bool
	}{
		{
			name: "supplier scoped token prefers supplier id",
			path: "/v1/supplier/supply-lanes/lane-123",
			claims: &auth.PegasusClaims{
				UserID:     "user-123",
				SupplierID: "supplier-456",
			},
			wantLaneID:        "lane-123",
			wantSupplierID:    "supplier-456",
			wantTransitUpdate: false,
		},
		{
			name: "legacy token falls back to user id",
			path: "/v1/supplier/supply-lanes/lane-123/transit",
			claims: &auth.PegasusClaims{
				UserID: "legacy-user",
			},
			wantLaneID:        "lane-123",
			wantSupplierID:    "legacy-user",
			wantTransitUpdate: true,
		},
		{
			name: "missing lane id returns error",
			path: "/v1/supplier/supply-lanes/",
			claims: &auth.PegasusClaims{
				UserID:     "user-123",
				SupplierID: "supplier-456",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PATCH", tt.path, nil)
			action, err := resolveSupplyLaneAction(req, tt.claims)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveSupplyLaneAction() error = %v", err)
			}
			if action.laneID != tt.wantLaneID {
				t.Fatalf("laneID = %q, want %q", action.laneID, tt.wantLaneID)
			}
			if action.supplierID != tt.wantSupplierID {
				t.Fatalf("supplierID = %q, want %q", action.supplierID, tt.wantSupplierID)
			}
			if action.isTransitUpdate != tt.wantTransitUpdate {
				t.Fatalf("isTransitUpdate = %v, want %v", action.isTransitUpdate, tt.wantTransitUpdate)
			}
		})
	}
}
