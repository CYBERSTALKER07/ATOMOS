package retailerroutes

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"backend-go/auth"
	"backend-go/order"

	"cloud.google.com/go/spanner"
)

func TestAuthorizeRetailerOrdersRejectsNonRetailerClaims(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/retailers/ret-1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.PegasusClaims{
		Role:   "ADMIN",
		UserID: "ret-1",
	}))

	if err := authorizeRetailerOrders(req, "ret-1"); err == nil {
		t.Fatal("expected admin claims to be rejected")
	}
}

func TestAuthorizeRetailerOrdersRejectsMismatchedRetailer(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/retailers/ret-1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.PegasusClaims{
		Role:   "RETAILER",
		UserID: "ret-2",
	}))

	if err := authorizeRetailerOrders(req, "ret-1"); err == nil {
		t.Fatal("expected mismatched retailer claims to be rejected")
	}
}

func TestMapMobileOrdersPreservesSharedMetadata(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(30 * time.Minute)
	autoConfirmAt := createdAt.Add(15 * time.Minute)
	deliverBefore := createdAt.Add(2 * time.Hour)
	orders := []order.Order{{
		ID:             "ord-1",
		RetailerID:     "ret-1",
		SupplierID:     "sup-1",
		SupplierName:   "Pegasus Supplier",
		Amount:         22450,
		Currency:       "UZS",
		PaymentGateway: "GLOBAL_PAY",
		PaymentStatus:  "PENDING",
		State:          "PENDING",
		RouteID:        spanner.NullString{StringVal: "route-1", Valid: true},
		OrderSource:    spanner.NullString{StringVal: "MANUAL", Valid: true},
		AutoConfirmAt:  spanner.NullTime{Time: autoConfirmAt, Valid: true},
		DeliverBefore:  spanner.NullTime{Time: deliverBefore, Valid: true},
		DeliveryToken:  spanner.NullString{StringVal: "tok-1", Valid: true},
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		Version:        7,
		Items: []order.LineItem{{
			LineItemID: "li-1",
			OrderID:    "ord-1",
			SkuID:      "sku-1",
			SkuName:    "Milk",
			Quantity:   2,
			UnitPrice:  11225,
			Currency:   "UZS",
			Status:     "PENDING",
		}},
	}}

	mapped := mapMobileOrders(orders)
	if len(mapped) != 1 {
		t.Fatalf("expected 1 mapped order, got %d", len(mapped))
	}
	if mapped[0].PaymentGateway != "GLOBAL_PAY" {
		t.Fatalf("expected payment gateway to be preserved, got %q", mapped[0].PaymentGateway)
	}
	if mapped[0].PaymentStatus != "PENDING" {
		t.Fatalf("expected payment status to be preserved, got %q", mapped[0].PaymentStatus)
	}
	if mapped[0].RouteID == nil || *mapped[0].RouteID != "route-1" {
		t.Fatalf("expected route id to be preserved, got %#v", mapped[0].RouteID)
	}
	if mapped[0].DeliverBefore == nil || *mapped[0].DeliverBefore != deliverBefore.Format(time.RFC3339) {
		t.Fatalf("expected deliver_before to be preserved, got %#v", mapped[0].DeliverBefore)
	}
	if mapped[0].Version != 7 {
		t.Fatalf("expected version 7, got %d", mapped[0].Version)
	}
	if len(mapped[0].Items) != 1 {
		t.Fatalf("expected 1 mapped line item, got %d", len(mapped[0].Items))
	}
	if mapped[0].Items[0].LineItemID != "li-1" || mapped[0].Items[0].SkuID != "sku-1" {
		t.Fatalf("expected canonical line item aliases to be present, got %#v", mapped[0].Items[0])
	}
}
