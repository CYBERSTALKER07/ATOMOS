package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
)

type testRetailerRepo struct {
	retailer     Retailer
	found        bool
	pricingRule  SupplierPricingRule
	pricingFound bool
	pricingErr   error
	tracking     []TrackingOrder
	trackingErr  error
	receipts     []TrackingOrder
	receiptsErr  error
}

func (r *testRetailerRepo) CreateRetailer(_ context.Context, _ Retailer, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *testRetailerRepo) FindByPhone(_ context.Context, _ string) (Retailer, bool, error) {
	return Retailer{}, false, nil
}

func (r *testRetailerRepo) GetRetailer(_ context.Context, _ string) (Retailer, bool, error) {
	if !r.found {
		return Retailer{}, false, nil
	}
	return r.retailer, true, nil
}

func (r *testRetailerRepo) UpdateRetailer(_ context.Context, _ Retailer, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *testRetailerRepo) ListRetailersBySupplier(_ context.Context, _ string) ([]Retailer, error) {
	return nil, nil
}

func (r *testRetailerRepo) GetSupplierPricingRule(_ context.Context, _ string) (SupplierPricingRule, bool, error) {
	if r.pricingErr != nil {
		return SupplierPricingRule{}, false, r.pricingErr
	}
	if !r.pricingFound {
		return SupplierPricingRule{}, false, nil
	}
	return r.pricingRule, true, nil
}

func (r *testRetailerRepo) ListTrackingOrders(_ context.Context, retailerID string, limit int) ([]TrackingOrder, error) {
	if r.trackingErr != nil {
		return nil, r.trackingErr
	}
	return r.tracking, nil
}

func (r *testRetailerRepo) ListRecentReceipts(_ context.Context, retailerID string, limit int) ([]TrackingOrder, error) {
	if r.receiptsErr != nil {
		return nil, r.receiptsErr
	}
	return r.receipts, nil
}

type testLocationReader struct {
	locations map[string]telemetry.DriverLocation
	err       error
}

type testOrderLifecycle struct {
	confirmAIReq       order.ConfirmAIOrderRequest
	rejectAIReq        order.RejectAIOrderRequest
	editPreorderReq    order.EditPreorderRequest
	confirmPreorderReq order.ConfirmPreorderRequest
	retailerID         string
	predictions        []order.RetailerAIPrediction
	response           order.RetailerOrderLifecycleResponse
	err                error
}

func (o *testOrderLifecycle) ConfirmAIOrder(_ context.Context, retailerID string, req order.ConfirmAIOrderRequest) (order.RetailerOrderLifecycleResponse, error) {
	o.retailerID = retailerID
	o.confirmAIReq = req
	return o.response, o.err
}

func (o *testOrderLifecycle) RejectAIOrder(_ context.Context, retailerID string, req order.RejectAIOrderRequest) (order.RetailerOrderLifecycleResponse, error) {
	o.retailerID = retailerID
	o.rejectAIReq = req
	return o.response, o.err
}

func (o *testOrderLifecycle) EditPreorder(_ context.Context, retailerID string, req order.EditPreorderRequest) (order.RetailerOrderLifecycleResponse, error) {
	o.retailerID = retailerID
	o.editPreorderReq = req
	return o.response, o.err
}

func (o *testOrderLifecycle) ConfirmPreorder(_ context.Context, retailerID string, req order.ConfirmPreorderRequest) (order.RetailerOrderLifecycleResponse, error) {
	o.retailerID = retailerID
	o.confirmPreorderReq = req
	return o.response, o.err
}

func (o *testOrderLifecycle) ListRetailerAIPredictions(_ context.Context, retailerID string, limit int) ([]order.RetailerAIPrediction, error) {
	o.retailerID = retailerID
	return o.predictions, o.err
}

func (r *testLocationReader) GetDriverLocation(_ context.Context, driverID string) (telemetry.DriverLocation, bool, error) {
	if r.err != nil {
		return telemetry.DriverLocation{}, false, r.err
	}
	location, found := r.locations[driverID]
	return location, found, nil
}

func TestHandleProfileRejectsRetailerScopeMismatch(t *testing.T) {
	repo := &testRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/profile?retailer_id=ret-2", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleProfile(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["error"] != "forbidden" {
		t.Fatalf("error=%q want=%q", payload["error"], "forbidden")
	}
}

func TestHandleProfileBackfillsSupplierID(t *testing.T) {
	repo := &testRetailerRepo{
		found: true,
		retailer: Retailer{
			RetailerID:  "ret-1",
			SupplierID:  "",
			Phone:       "+998901112233",
			Name:        "Retailer One",
			CountryCode: "UZ",
			H3Cell:      "872830828ffffff",
			Lat:         41.31,
			Lng:         69.29,
			CreatedAt:   time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 1, 1, 12, 5, 0, 0, time.UTC),
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/profile", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["id"] != "ret-1" {
		t.Fatalf("id=%v want=%q", payload["id"], "ret-1")
	}
	if payload["company"] != "Retailer One" {
		t.Fatalf("company=%v want=%q", payload["company"], "Retailer One")
	}
	if payload["status"] != "ACTIVE" {
		t.Fatalf("status=%v want=%q", payload["status"], "ACTIVE")
	}
	if payload["supplier_id"] != "sup-1" {
		t.Fatalf("supplier_id=%v want=%q", payload["supplier_id"], "sup-1")
	}
	if payload["retailer_id"] != "ret-1" {
		t.Fatalf("retailer_id=%v want=%q", payload["retailer_id"], "ret-1")
	}
}

func TestHandleProfileReturnsReceivingWindow(t *testing.T) {
	repo := &testRetailerRepo{
		found: true,
		retailer: Retailer{
			RetailerID:           "ret-1",
			Phone:                "+998901112233",
			Name:                 "Retailer One",
			CountryCode:          "UZ",
			ReceivingWindowOpen:  "09:00",
			ReceivingWindowClose: "18:00",
			CreatedAt:            time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt:            time.Date(2026, 1, 1, 12, 5, 0, 0, time.UTC),
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/profile", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["receiving_window_open"] != "09:00" {
		t.Fatalf("receiving_window_open=%v want=%q", payload["receiving_window_open"], "09:00")
	}
	if payload["receiving_window_close"] != "18:00" {
		t.Fatalf("receiving_window_close=%v want=%q", payload["receiving_window_close"], "18:00")
	}
}

func TestHandleProfileUpdateRejectsInvalidReceivingWindow(t *testing.T) {
	repo := &testRetailerRepo{
		found: true,
		retailer: Retailer{
			RetailerID:  "ret-1",
			Phone:       "+998901112233",
			Name:        "Retailer One",
			CountryCode: "UZ",
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(
		http.MethodPut,
		"/v1/retailer/profile",
		strings.NewReader(`{"receiving_window_open":"99:99"}`),
	)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleProfile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestHandleSuppliersIncludesPricingSnapshot(t *testing.T) {
	repo := &testRetailerRepo{
		pricingFound: true,
		pricingRule: SupplierPricingRule{
			SupplierID:          "sup-1",
			BaseMarkupBps:       1200,
			RetailerDiscountBps: 200,
			MinMarginBps:        100,
			Currency:            "UZS",
			RuleVersion:         2,
			UpdatedAt:           time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/suppliers", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleSuppliers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("len(payload)=%d want=1", len(payload))
	}
	pricing, ok := payload[0]["pricing"].(map[string]any)
	if !ok {
		t.Fatalf("pricing missing or invalid: %v", payload[0]["pricing"])
	}
	if pricing["base_markup_bps"] != float64(1200) {
		t.Fatalf("base_markup_bps=%v want=%v", pricing["base_markup_bps"], 1200)
	}
	if pricing["retailer_discount_bps"] != float64(200) {
		t.Fatalf("retailer_discount_bps=%v want=%v", pricing["retailer_discount_bps"], 200)
	}
	if pricing["currency"] != "UZS" {
		t.Fatalf("currency=%v want=%q", pricing["currency"], "UZS")
	}
}

func TestHandlePricingRuleReturnsConfiguredFalseWhenMissing(t *testing.T) {
	repo := &testRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/pricing/rules", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandlePricingRule(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["configured"] != false {
		t.Fatalf("configured=%v want=false", payload["configured"])
	}
}

func TestHandleTrackingReturnsAssignedOrders(t *testing.T) {
	createdAt := time.Date(2026, 5, 22, 9, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 22, 10, 5, 0, 0, time.UTC)
	repo := &testRetailerRepo{tracking: []TrackingOrder{{
		OrderID:               "ord-1",
		SupplierID:            "sup-1",
		RetailerID:            "ret-1",
		DriverID:              "drv-1",
		RouteID:               "route-1",
		Status:                "IN_TRANSIT",
		TrackingStatus:        "assigned",
		TotalMinor:            1500,
		Currency:              "UZS",
		LiveLocationAvailable: false,
		CreatedAt:             createdAt.Format(time.RFC3339Nano),
		UpdatedAt:             updatedAt.Format(time.RFC3339Nano),
		Items:                 []TrackingLineItem{},
	}}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleTracking(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "active" {
		t.Fatalf("status=%v want active", payload["status"])
	}
	orders, ok := payload["orders"].([]any)
	if !ok || len(orders) != 1 {
		t.Fatalf("orders=%v", payload["orders"])
	}
	first := orders[0].(map[string]any)
	if first["driver_id"] != "drv-1" || first["route_id"] != "route-1" {
		t.Fatalf("assignment fields missing: %v", first)
	}
	if first["live_location_available"] != false {
		t.Fatalf("live_location_available=%v want false", first["live_location_available"])
	}
	events, ok := payload["events"].([]any)
	if !ok || len(events) != 2 {
		t.Fatalf("events=%v", payload["events"])
	}
	statusEvent := events[0].(map[string]any)
	createdEvent := events[1].(map[string]any)
	if statusEvent["event_type"] != "ORDER_STATUS_SNAPSHOT" || statusEvent["status"] != "IN_TRANSIT" {
		t.Fatalf("unexpected status event: %v", statusEvent)
	}
	if createdEvent["event_type"] != "ORDER_CREATED" {
		t.Fatalf("unexpected created event: %v", createdEvent)
	}
	if statusEvent["source"] != "ORDER_ROW" || createdEvent["source"] != "ORDER_ROW" {
		t.Fatalf("unexpected event source: %v %v", statusEvent["source"], createdEvent["source"])
	}
	if statusEvent["derived"] != true || createdEvent["derived"] != true {
		t.Fatalf("unexpected derived flags: %v %v", statusEvent["derived"], createdEvent["derived"])
	}
}

func TestHandleTrackingKeepsEventsEmptyForIdleOrders(t *testing.T) {
	repo := &testRetailerRepo{tracking: []TrackingOrder{}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleTracking(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "idle" {
		t.Fatalf("status=%v want idle", payload["status"])
	}
	events, ok := payload["events"].([]any)
	if !ok || len(events) != 0 {
		t.Fatalf("events=%v want empty", payload["events"])
	}
}

func TestHandleTrackingDerivesPaymentStatusEvent(t *testing.T) {
	createdAt := time.Date(2026, 5, 22, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 22, 10, 15, 0, 0, time.UTC)
	repo := &testRetailerRepo{tracking: []TrackingOrder{{
		OrderID:        "ord-2",
		SupplierID:     "sup-1",
		RetailerID:     "ret-1",
		Status:         "PENDING_CASH_COLLECTION",
		TrackingStatus: "assigned",
		CreatedAt:      createdAt.Format(time.RFC3339Nano),
		UpdatedAt:      updatedAt.Format(time.RFC3339Nano),
		Items:          []TrackingLineItem{},
	}}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleTracking(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	events, ok := payload["events"].([]any)
	if !ok || len(events) != 2 {
		t.Fatalf("events=%v", payload["events"])
	}
	statusEvent := events[0].(map[string]any)
	if statusEvent["event_type"] != "ORDER_STATUS_SNAPSHOT" {
		t.Fatalf("event_type=%v want ORDER_STATUS_SNAPSHOT", statusEvent["event_type"])
	}
	if statusEvent["status"] != "PENDING_CASH_COLLECTION" {
		t.Fatalf("status=%v want PENDING_CASH_COLLECTION", statusEvent["status"])
	}
}

func TestHandleTrackingIncludesRecentReceiptsSeparately(t *testing.T) {
	activeCreatedAt := time.Date(2026, 5, 22, 9, 30, 0, 0, time.UTC)
	activeUpdatedAt := time.Date(2026, 5, 22, 10, 5, 0, 0, time.UTC)
	receiptCreatedAt := time.Date(2026, 5, 22, 7, 45, 0, 0, time.UTC)
	receiptUpdatedAt := time.Date(2026, 5, 22, 10, 20, 0, 0, time.UTC)
	repo := &testRetailerRepo{
		tracking: []TrackingOrder{{
			OrderID:        "ord-active",
			SupplierID:     "sup-1",
			RetailerID:     "ret-1",
			DriverID:       "drv-1",
			RouteID:        "route-1",
			Status:         "IN_TRANSIT",
			TrackingStatus: "assigned",
			TotalMinor:     1500,
			Currency:       "UZS",
			CreatedAt:      activeCreatedAt.Format(time.RFC3339Nano),
			UpdatedAt:      activeUpdatedAt.Format(time.RFC3339Nano),
			Items:          []TrackingLineItem{},
		}},
		receipts: []TrackingOrder{{
			OrderID:        "ord-receipt",
			SupplierID:     "sup-1",
			RetailerID:     "ret-1",
			Status:         "COMPLETED",
			TrackingStatus: "assigned",
			TotalMinor:     2200,
			Currency:       "UZS",
			PaymentEvidence: &TrackingPaymentEvidence{
				EntryType:   "WEBHOOK_PAID",
				Gateway:     "ADYEN",
				AmountMinor: 2200,
				Currency:    "UZS",
				ReferenceID: "tx-1",
				OccurredAt:  receiptUpdatedAt.Format(time.RFC3339Nano),
			},
			ReceiptDossier: &TrackingReceiptDossier{
				SessionID: "sess-1",
				PaymentTimeline: []TrackingReceiptPaymentRecord{{
					LedgerEntryID: "ledger-1",
					SessionID:     "sess-1",
					OrderID:       "ord-receipt",
					Gateway:       "ADYEN",
					EntryType:     "WEBHOOK_PAID",
					AmountMinor:   2200,
					Currency:      "UZS",
					ReferenceID:   "tx-1",
					Source:        "PAYMENT_WEBHOOK",
					OccurredAt:    receiptUpdatedAt.Format(time.RFC3339Nano),
					CreatedAt:     receiptUpdatedAt.Format(time.RFC3339Nano),
				}},
				GatewayWebhooks: []TrackingReceiptGatewayWebhook{{
					WebhookID:      "wh-1",
					SessionID:      "sess-1",
					Gateway:        "ADYEN",
					TransactionID:  "txn-1",
					Status:         "PAID",
					AmountMinor:    2200,
					Currency:       "UZS",
					SignatureValid: true,
					ReceivedAt:     receiptUpdatedAt.Format(time.RFC3339Nano),
				}},
				ProofStatus: TrackingReceiptProofStatus{
					PaymentTimelineAvailable: true,
					GatewayWebhooksAvailable: true,
					DeliveryProofAvailable:   false,
					MissingArtifacts:         []string{trackingMissingDeliveryHandoffProof},
				},
			},
			CreatedAt: receiptCreatedAt.Format(time.RFC3339Nano),
			UpdatedAt: receiptUpdatedAt.Format(time.RFC3339Nano),
			Items:     []TrackingLineItem{},
		}},
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleTracking(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "active" {
		t.Fatalf("status=%v want active", payload["status"])
	}
	orders, ok := payload["orders"].([]any)
	if !ok || len(orders) != 1 {
		t.Fatalf("orders=%v", payload["orders"])
	}
	receipts, ok := payload["recent_receipts"].([]any)
	if !ok || len(receipts) != 1 {
		t.Fatalf("recent_receipts=%v", payload["recent_receipts"])
	}
	if orders[0].(map[string]any)["order_id"] != "ord-active" {
		t.Fatalf("unexpected active order: %v", orders[0])
	}
	receipt := receipts[0].(map[string]any)
	if receipt["order_id"] != "ord-receipt" || receipt["status"] != "COMPLETED" {
		t.Fatalf("unexpected receipt: %v", receipt)
	}
	paymentEvidence, ok := receipt["payment_evidence"].(map[string]any)
	if !ok {
		t.Fatalf("payment_evidence missing or invalid: %v", receipt["payment_evidence"])
	}
	if paymentEvidence["entry_type"] != "WEBHOOK_PAID" || paymentEvidence["gateway"] != "ADYEN" {
		t.Fatalf("unexpected payment_evidence: %v", paymentEvidence)
	}
	if paymentEvidence["reference_id"] != "tx-1" {
		t.Fatalf("payment_evidence reference_id=%v want tx-1", paymentEvidence["reference_id"])
	}
	receiptDossier, ok := receipt["receipt_dossier"].(map[string]any)
	if !ok {
		t.Fatalf("receipt_dossier missing or invalid: %v", receipt["receipt_dossier"])
	}
	if receiptDossier["session_id"] != "sess-1" {
		t.Fatalf("receipt_dossier session_id=%v want sess-1", receiptDossier["session_id"])
	}
	proofStatus, ok := receiptDossier["proof_status"].(map[string]any)
	if !ok {
		t.Fatalf("receipt_dossier proof_status missing or invalid: %v", receiptDossier["proof_status"])
	}
	if proofStatus["delivery_proof_available"] != false {
		t.Fatalf("delivery_proof_available=%v want false", proofStatus["delivery_proof_available"])
	}
	if receipt["live_location_available"] != false {
		t.Fatalf("receipt live_location_available=%v want false", receipt["live_location_available"])
	}
	events, ok := payload["events"].([]any)
	if !ok || len(events) != 4 {
		t.Fatalf("events=%v", payload["events"])
	}
	firstEvent := events[0].(map[string]any)
	if firstEvent["order_id"] != "ord-receipt" || firstEvent["status"] != "COMPLETED" {
		t.Fatalf("unexpected leading receipt event: %v", firstEvent)
	}
}

func TestHandleTrackingReturnsReceiptOnlySnapshotWhenActiveOrdersIdle(t *testing.T) {
	receiptCreatedAt := time.Date(2026, 5, 22, 7, 45, 0, 0, time.UTC)
	receiptUpdatedAt := time.Date(2026, 5, 22, 10, 20, 0, 0, time.UTC)
	repo := &testRetailerRepo{receipts: []TrackingOrder{{
		OrderID:        "ord-receipt",
		SupplierID:     "sup-1",
		RetailerID:     "ret-1",
		Status:         "COMPLETED",
		TrackingStatus: "assigned",
		TotalMinor:     2200,
		Currency:       "UZS",
		CreatedAt:      receiptCreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:      receiptUpdatedAt.Format(time.RFC3339Nano),
		Items:          []TrackingLineItem{},
	}}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleTracking(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "idle" {
		t.Fatalf("status=%v want idle", payload["status"])
	}
	orders, ok := payload["orders"].([]any)
	if !ok || len(orders) != 0 {
		t.Fatalf("orders=%v want empty", payload["orders"])
	}
	receipts, ok := payload["recent_receipts"].([]any)
	if !ok || len(receipts) != 1 {
		t.Fatalf("recent_receipts=%v", payload["recent_receipts"])
	}
	events, ok := payload["events"].([]any)
	if !ok || len(events) != 2 {
		t.Fatalf("events=%v", payload["events"])
	}
}

func TestUniqueTrackingOrderIDsDeduplicatesAndTrims(t *testing.T) {
	ids := uniqueTrackingOrderIDs([]TrackingOrder{{OrderID: " ord-1 "}, {OrderID: "ord-1"}, {OrderID: "ord-2"}, {OrderID: ""}})
	if len(ids) != 2 {
		t.Fatalf("len(ids)=%d want=2 ids=%v", len(ids), ids)
	}
	if ids[0] != "ord-1" || ids[1] != "ord-2" {
		t.Fatalf("ids=%v want [ord-1 ord-2]", ids)
	}
}

func TestUniqueTrackingSessionIDsDeduplicatesAndTrims(t *testing.T) {
	ids := uniqueTrackingSessionIDs(map[string]string{
		"ord-1": " sess-1 ",
		"ord-2": "sess-1",
		"ord-3": "sess-2",
		"ord-4": "",
	})
	if len(ids) != 2 {
		t.Fatalf("len(ids)=%d want=2 ids=%v", len(ids), ids)
	}
	if ids[0] != "sess-1" || ids[1] != "sess-2" {
		t.Fatalf("ids=%v want [sess-1 sess-2]", ids)
	}
}

func TestMergeTrackingReceiptTimelineDeduplicatesAndOrdersEntries(t *testing.T) {
	paidAt := time.Date(2026, 5, 23, 9, 0, 0, 0, time.UTC)
	reversalAt := paidAt.Add(10 * time.Minute)

	merged := mergeTrackingReceiptTimeline(
		[]trackingReceiptPaymentRecordSnapshot{{
			Record: TrackingReceiptPaymentRecord{
				LedgerEntryID: "ledger-paid",
				SessionID:     "sess-1",
				OrderID:       "ord-1",
				Gateway:       "ADYEN",
				EntryType:     "WEBHOOK_PAID",
				AmountMinor:   2200,
				Currency:      "UZS",
				ReferenceID:   "tx-1",
				Source:        "PAYMENT_WEBHOOK",
				OccurredAt:    paidAt.Format(time.RFC3339Nano),
				CreatedAt:     paidAt.Format(time.RFC3339Nano),
			},
			OccurredAt: paidAt,
			CreatedAt:  paidAt,
		}},
		[]trackingReceiptPaymentRecordSnapshot{
			{
				Record: TrackingReceiptPaymentRecord{
					LedgerEntryID: "ledger-paid",
					SessionID:     "sess-1",
					OrderID:       "ord-1",
					Gateway:       "ADYEN",
					EntryType:     "WEBHOOK_PAID",
					AmountMinor:   2200,
					Currency:      "UZS",
					ReferenceID:   "tx-1",
					Source:        "PAYMENT_WEBHOOK",
					OccurredAt:    paidAt.Format(time.RFC3339Nano),
					CreatedAt:     paidAt.Format(time.RFC3339Nano),
				},
				OccurredAt: paidAt,
				CreatedAt:  paidAt,
			},
			{
				Record: TrackingReceiptPaymentRecord{
					LedgerEntryID: "ledger-reversal",
					SessionID:     "sess-1",
					Gateway:       "ADYEN",
					EntryType:     "CHARGEBACK_REVERSAL_RECORDED",
					AmountMinor:   0,
					Currency:      "UZS",
					ReferenceID:   "rev-1",
					Source:        "CHARGEBACK_REVERSAL",
					OccurredAt:    reversalAt.Format(time.RFC3339Nano),
					CreatedAt:     reversalAt.Format(time.RFC3339Nano),
				},
				OccurredAt: reversalAt,
				CreatedAt:  reversalAt,
			},
		},
	)

	if len(merged) != 2 {
		t.Fatalf("len(merged)=%d want=2 merged=%v", len(merged), merged)
	}
	if merged[0].LedgerEntryID != "ledger-reversal" {
		t.Fatalf("first ledger_entry_id=%q want ledger-reversal", merged[0].LedgerEntryID)
	}
	if merged[1].LedgerEntryID != "ledger-paid" {
		t.Fatalf("second ledger_entry_id=%q want ledger-paid", merged[1].LedgerEntryID)
	}
}

func TestBuildTrackingReceiptDossiersIncludesSessionHistoryAndProofGap(t *testing.T) {
	paidAt := time.Date(2026, 5, 23, 9, 0, 0, 0, time.UTC)
	reversalAt := paidAt.Add(10 * time.Minute)

	dossiers := buildTrackingReceiptDossiers(
		[]TrackingOrder{{OrderID: "ord-1"}},
		map[string][]trackingReceiptPaymentRecordSnapshot{
			"ord-1": {{
				Record: TrackingReceiptPaymentRecord{
					LedgerEntryID: "ledger-paid",
					SessionID:     "sess-1",
					OrderID:       "ord-1",
					Gateway:       "ADYEN",
					EntryType:     "WEBHOOK_PAID",
					AmountMinor:   2200,
					Currency:      "UZS",
					ReferenceID:   "tx-1",
					Source:        "PAYMENT_WEBHOOK",
					OccurredAt:    paidAt.Format(time.RFC3339Nano),
					CreatedAt:     paidAt.Format(time.RFC3339Nano),
				},
				OccurredAt: paidAt,
				CreatedAt:  paidAt,
			}},
		},
		map[string][]trackingReceiptPaymentRecordSnapshot{
			"sess-1": {{
				Record: TrackingReceiptPaymentRecord{
					LedgerEntryID: "ledger-reversal",
					SessionID:     "sess-1",
					Gateway:       "ADYEN",
					EntryType:     "CHARGEBACK_REVERSAL_RECORDED",
					AmountMinor:   0,
					Currency:      "UZS",
					ReferenceID:   "rev-1",
					Source:        "CHARGEBACK_REVERSAL",
					OccurredAt:    reversalAt.Format(time.RFC3339Nano),
					CreatedAt:     reversalAt.Format(time.RFC3339Nano),
				},
				OccurredAt: reversalAt,
				CreatedAt:  reversalAt,
			}},
		},
		map[string]string{"ord-1": "sess-1"},
		map[string][]TrackingReceiptGatewayWebhook{
			"ord-1": {{
				WebhookID:      "wh-1",
				SessionID:      "sess-1",
				Gateway:        "ADYEN",
				TransactionID:  "txn-1",
				Status:         "PAID",
				AmountMinor:    2200,
				Currency:       "UZS",
				SignatureValid: true,
				ReceivedAt:     paidAt.Format(time.RFC3339Nano),
			}},
		},
		map[string][]TrackingReceiptDeliveryProof{},
		map[string][]TrackingReceiptChargebackRecord{},
		map[string][]TrackingReceiptReversalRecord{},
	)

	dossier, found := dossiers["ord-1"]
	if !found {
		t.Fatalf("receipt dossier missing for ord-1: %v", dossiers)
	}
	if dossier.SessionID != "sess-1" {
		t.Fatalf("session_id=%q want sess-1", dossier.SessionID)
	}
	if len(dossier.PaymentTimeline) != 2 {
		t.Fatalf("len(payment_timeline)=%d want=2", len(dossier.PaymentTimeline))
	}
	if dossier.PaymentTimeline[0].EntryType != "CHARGEBACK_REVERSAL_RECORDED" {
		t.Fatalf("first payment timeline entry=%q want CHARGEBACK_REVERSAL_RECORDED", dossier.PaymentTimeline[0].EntryType)
	}
	if len(dossier.GatewayWebhooks) != 1 {
		t.Fatalf("len(gateway_webhooks)=%d want=1", len(dossier.GatewayWebhooks))
	}
	if !dossier.ProofStatus.PaymentTimelineAvailable {
		t.Fatalf("payment_timeline_available=false want true")
	}
	if !dossier.ProofStatus.GatewayWebhooksAvailable {
		t.Fatalf("gateway_webhooks_available=false want true")
	}
	if dossier.ProofStatus.DeliveryProofAvailable {
		t.Fatalf("delivery_proof_available=true want false")
	}
	if len(dossier.ProofStatus.MissingArtifacts) != 1 || dossier.ProofStatus.MissingArtifacts[0] != trackingMissingDeliveryHandoffProof {
		t.Fatalf("missing_artifacts=%v want [%s]", dossier.ProofStatus.MissingArtifacts, trackingMissingDeliveryHandoffProof)
	}
}

func TestBuildTrackingReceiptDossiersIncludesPersistedProofAndDisputeAuthority(t *testing.T) {
	paidAt := time.Date(2026, 5, 23, 9, 0, 0, 0, time.UTC)
	proofCapturedAt := paidAt.Add(2 * time.Minute)
	chargebackAt := paidAt.Add(5 * time.Minute)
	reversalAt := paidAt.Add(10 * time.Minute)

	dossiers := buildTrackingReceiptDossiers(
		[]TrackingOrder{{OrderID: "ord-1", Currency: "UZS"}},
		map[string][]trackingReceiptPaymentRecordSnapshot{
			"ord-1": {{
				Record: TrackingReceiptPaymentRecord{
					LedgerEntryID: "ledger-paid",
					SessionID:     "sess-1",
					OrderID:       "ord-1",
					Gateway:       "ADYEN",
					EntryType:     "WEBHOOK_PAID",
					AmountMinor:   2200,
					Currency:      "UZS",
					ReferenceID:   "tx-1",
					Source:        "PAYMENT_WEBHOOK",
					OccurredAt:    paidAt.Format(time.RFC3339Nano),
					CreatedAt:     paidAt.Format(time.RFC3339Nano),
				},
				OccurredAt: paidAt,
				CreatedAt:  paidAt,
			}},
		},
		map[string][]trackingReceiptPaymentRecordSnapshot{
			"sess-1": {{
				Record: TrackingReceiptPaymentRecord{
					LedgerEntryID: "ledger-reversal",
					SessionID:     "sess-1",
					Gateway:       "ADYEN",
					EntryType:     "CHARGEBACK_REVERSAL_RECORDED",
					AmountMinor:   0,
					Currency:      "UZS",
					ReferenceID:   "rev-1",
					Source:        "CHARGEBACK_REVERSAL",
					OccurredAt:    reversalAt.Format(time.RFC3339Nano),
					CreatedAt:     reversalAt.Format(time.RFC3339Nano),
				},
				OccurredAt: reversalAt,
				CreatedAt:  reversalAt,
			}},
		},
		map[string]string{"ord-1": "sess-1"},
		map[string][]TrackingReceiptGatewayWebhook{},
		map[string][]TrackingReceiptDeliveryProof{
			"ord-1": {{
				ProofID:                 "proof-1",
				ProofType:               "QR_HANDOFF",
				QRTokenHashPresent:      true,
				ScannedTokenHashPresent: true,
				CapturedAt:              proofCapturedAt.Format(time.RFC3339Nano),
			}},
		},
		map[string][]TrackingReceiptChargebackRecord{
			"ord-1": {{
				ChargebackID: "cb-1",
				Gateway:      "ADYEN",
				AmountMinor:  2200,
				Currency:     "UZS",
				CreatedAt:    chargebackAt.Format(time.RFC3339Nano),
			}},
		},
		map[string][]TrackingReceiptReversalRecord{
			"sess-1": {{
				ReversalID: "rev-1",
				SessionID:  "sess-1",
				CreatedAt:  reversalAt.Format(time.RFC3339Nano),
			}},
		},
	)

	dossier := dossiers["ord-1"]
	if !dossier.ProofStatus.DeliveryProofAvailable {
		t.Fatalf("delivery_proof_available=false want true")
	}
	if len(dossier.ProofStatus.MissingArtifacts) != 0 {
		t.Fatalf("missing_artifacts=%v want empty", dossier.ProofStatus.MissingArtifacts)
	}
	if len(dossier.DeliveryProofs) != 1 {
		t.Fatalf("len(delivery_proofs)=%d want 1", len(dossier.DeliveryProofs))
	}
	if len(dossier.Chargebacks) != 1 {
		t.Fatalf("len(chargebacks)=%d want 1", len(dossier.Chargebacks))
	}
	if len(dossier.Reversals) != 1 {
		t.Fatalf("len(reversals)=%d want 1", len(dossier.Reversals))
	}
	if dossier.Reversals[0].Gateway != "ADYEN" || dossier.Reversals[0].LedgerEntryID != "ledger-reversal" {
		t.Fatalf("reversal enrichment=%+v want gateway ADYEN ledger_entry_id ledger-reversal", dossier.Reversals[0])
	}
}

func TestHandleActiveFulfillmentReturnsTrackedOrders(t *testing.T) {
	now := time.Date(2026, 5, 22, 10, 12, 0, 0, time.UTC)
	repo := &testRetailerRepo{tracking: []TrackingOrder{
		{
			OrderID:        "ord-1",
			SupplierID:     "sup-1",
			RetailerID:     "ret-1",
			DriverID:       "drv-1",
			RouteID:        "route-1",
			Status:         "IN_TRANSIT",
			TrackingStatus: "assigned",
			TotalMinor:     1500,
			Currency:       "UZS",
			Items:          []TrackingLineItem{},
		},
		{
			OrderID:        "ord-2",
			SupplierID:     "sup-1",
			RetailerID:     "ret-1",
			Status:         "AWAITING_PAYMENT",
			TrackingStatus: "unassigned",
			TotalMinor:     800,
			Currency:       "UZS",
			Items:          []TrackingLineItem{},
		},
	}}
	locations := &testLocationReader{locations: map[string]telemetry.DriverLocation{
		"drv-1": {
			DriverID:          "drv-1",
			SupplierID:        "sup-1",
			Lat:               41.31,
			Lng:               69.29,
			Latitude:          41.31,
			Longitude:         69.29,
			ReportedAt:        now.Add(-5 * time.Second),
			ReceivedAt:        now.Add(-4 * time.Second),
			StaleAfterSeconds: 30,
		},
	}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Locations: locations, Now: func() time.Time { return now }})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/active-fulfillment", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleActiveFulfillment(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "active" {
		t.Fatalf("status=%v want active", payload["status"])
	}
	fulfillments, ok := payload["fulfillments"].([]any)
	if !ok || len(fulfillments) != 2 {
		t.Fatalf("fulfillments=%v", payload["fulfillments"])
	}
	first := fulfillments[0].(map[string]any)
	if first["order_id"] != "ord-1" {
		t.Fatalf("unexpected first fulfillment: %v", first)
	}
	if first["live_location_available"] != true {
		t.Fatalf("live_location_available=%v want true", first["live_location_available"])
	}
}

func TestHandlePendingPaymentsFiltersPaymentStates(t *testing.T) {
	repo := &testRetailerRepo{tracking: []TrackingOrder{
		{OrderID: "ord-1", SupplierID: "sup-1", RetailerID: "ret-1", Status: "IN_TRANSIT", TrackingStatus: "assigned", Items: []TrackingLineItem{}},
		{OrderID: "ord-2", SupplierID: "sup-1", RetailerID: "ret-1", Status: "AWAITING_PAYMENT", TrackingStatus: "assigned", Items: []TrackingLineItem{}},
		{OrderID: "ord-3", SupplierID: "sup-1", RetailerID: "ret-1", Status: "PENDING_CASH_COLLECTION", TrackingStatus: "assigned", Items: []TrackingLineItem{}},
	}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/pending-payments", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandlePendingPayments(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "pending" {
		t.Fatalf("status=%v want pending", payload["status"])
	}
	if got, ok := payload["count"].(float64); !ok || int(got) != 2 {
		t.Fatalf("count=%v want 2", payload["count"])
	}
	pending, ok := payload["pending"].([]any)
	if !ok || len(pending) != 2 {
		t.Fatalf("pending=%v", payload["pending"])
	}
	first := pending[0].(map[string]any)
	second := pending[1].(map[string]any)
	if first["status"] != "AWAITING_PAYMENT" || second["status"] != "PENDING_CASH_COLLECTION" {
		t.Fatalf("pending statuses=%v %v", first["status"], second["status"])
	}
}

func TestHandleTrackingIncludesScopedFreshLocation(t *testing.T) {
	now := time.Date(2026, 5, 22, 10, 12, 0, 0, time.UTC)
	velocity := 12.5
	repo := &testRetailerRepo{tracking: []TrackingOrder{{
		OrderID:        "ord-1",
		SupplierID:     "sup-1",
		RetailerID:     "ret-1",
		DriverID:       "drv-1",
		RouteID:        "route-1",
		Status:         "IN_TRANSIT",
		TrackingStatus: "assigned",
		TotalMinor:     1500,
		Currency:       "UZS",
		Items:          []TrackingLineItem{},
	}}}
	locations := &testLocationReader{locations: map[string]telemetry.DriverLocation{
		"drv-1": {
			DriverID:          "drv-1",
			SupplierID:        "sup-1",
			Lat:               41.31,
			Lng:               69.29,
			Latitude:          41.31,
			Longitude:         69.29,
			Velocity:          &velocity,
			ReportedAt:        now.Add(-5 * time.Second),
			ReceivedAt:        now.Add(-4 * time.Second),
			StaleAfterSeconds: 30,
		},
	}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Locations: locations, Now: func() time.Time { return now }})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleTracking(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	orders := payload["orders"].([]any)
	first := orders[0].(map[string]any)
	if first["live_location_available"] != true {
		t.Fatalf("live_location_available=%v want true", first["live_location_available"])
	}
	location := first["driver_location"].(map[string]any)
	if location["driver_id"] != "drv-1" || location["supplier_id"] != "sup-1" {
		t.Fatalf("location scope=%v", location)
	}
	if location["lat"] != 41.31 || location["lng"] != 69.29 {
		t.Fatalf("location coordinate=%v", location)
	}
}

func TestHandleTrackingSuppressesCrossSupplierLocation(t *testing.T) {
	now := time.Date(2026, 5, 22, 10, 12, 0, 0, time.UTC)
	repo := &testRetailerRepo{tracking: []TrackingOrder{{
		OrderID:        "ord-1",
		SupplierID:     "sup-1",
		RetailerID:     "ret-1",
		DriverID:       "drv-1",
		RouteID:        "route-1",
		Status:         "IN_TRANSIT",
		TrackingStatus: "assigned",
		Items:          []TrackingLineItem{},
	}}}
	locations := &testLocationReader{locations: map[string]telemetry.DriverLocation{
		"drv-1": {
			DriverID:          "drv-1",
			SupplierID:        "sup-2",
			Lat:               41.31,
			Lng:               69.29,
			Latitude:          41.31,
			Longitude:         69.29,
			ReportedAt:        now.Add(-5 * time.Second),
			ReceivedAt:        now.Add(-4 * time.Second),
			StaleAfterSeconds: 30,
		},
	}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Locations: locations, Now: func() time.Time { return now }})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleTracking(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	orders := payload["orders"].([]any)
	first := orders[0].(map[string]any)
	if first["live_location_available"] != false {
		t.Fatalf("live_location_available=%v want false", first["live_location_available"])
	}
	if _, ok := first["driver_location"]; ok {
		t.Fatalf("driver_location leaked: %v", first["driver_location"])
	}
}

func TestHandleTrackingSuppressesStaleLocation(t *testing.T) {
	now := time.Date(2026, 5, 22, 10, 12, 0, 0, time.UTC)
	repo := &testRetailerRepo{tracking: []TrackingOrder{{
		OrderID:        "ord-1",
		SupplierID:     "sup-1",
		RetailerID:     "ret-1",
		DriverID:       "drv-1",
		RouteID:        "route-1",
		Status:         "IN_TRANSIT",
		TrackingStatus: "assigned",
		Items:          []TrackingLineItem{},
	}}}
	locations := &testLocationReader{locations: map[string]telemetry.DriverLocation{
		"drv-1": {
			DriverID:          "drv-1",
			SupplierID:        "sup-1",
			Lat:               41.31,
			Lng:               69.29,
			Latitude:          41.31,
			Longitude:         69.29,
			ReportedAt:        now.Add(-65 * time.Second),
			ReceivedAt:        now.Add(-61 * time.Second),
			StaleAfterSeconds: 30,
		},
	}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Locations: locations, Now: func() time.Time { return now }})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleTracking(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	orders := payload["orders"].([]any)
	first := orders[0].(map[string]any)
	if first["live_location_available"] != false {
		t.Fatalf("live_location_available=%v want false", first["live_location_available"])
	}
	if _, ok := first["driver_location"]; ok {
		t.Fatalf("driver_location leaked: %v", first["driver_location"])
	}
}

func TestHandleTrackingRejectsRetailerScopeMismatch(t *testing.T) {
	repo := &testRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking?retailer_id=ret-2", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleTracking(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
}

func TestHandleConfirmAIOrderDelegatesToOrderLifecycle(t *testing.T) {
	orders := &testOrderLifecycle{response: order.RetailerOrderLifecycleResponse{OrderID: "ord-1", ConfirmationStatus: order.ConfirmationStatusConfirmed}}
	svc := NewService(ServiceConfig{Repo: &testRetailerRepo{}, SupplierID: "sup-1", Orders: orders})

	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/orders/confirm-ai", strings.NewReader(`{"order_id":"ord-1"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleConfirmAIOrder(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if orders.retailerID != "ret-1" {
		t.Fatalf("retailerID=%s want ret-1", orders.retailerID)
	}
	if orders.confirmAIReq.OrderID != "ord-1" {
		t.Fatalf("confirmAIReq=%+v", orders.confirmAIReq)
	}
}

func TestHandleEditPreorderMapsConflict(t *testing.T) {
	orders := &testOrderLifecycle{err: order.ErrInvalidStatusTransition}
	svc := NewService(ServiceConfig{Repo: &testRetailerRepo{}, SupplierID: "sup-1", Orders: orders})

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/edit-preorder", strings.NewReader(`{"order_id":"ord-1","requested_delivery_date":"2026-01-03T12:00:00Z","line_items":[{"sku":"sku-1","quantity":1,"unit_price_minor":500}]}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleEditPreorder(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusConflict, rr.Body.String())
	}
}

func TestHandleAIPredictionsReturnsItems(t *testing.T) {
	orders := &testOrderLifecycle{predictions: []order.RetailerAIPrediction{{
		OrderID:               "ord-ai-1",
		Source:                order.OrderSourceAIPreorder,
		ConfirmationStatus:    order.ConfirmationStatusPending,
		RequestedDeliveryDate: "2026-01-08T12:00:00Z",
		AutoConfirmAt:         "2026-01-02T12:00:00Z",
		TotalMinor:            500,
		Currency:              "UZS",
	}}}
	svc := NewService(ServiceConfig{Repo: &testRetailerRepo{}, SupplierID: "sup-1", Orders: orders})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/ai/predictions?limit=10", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleAIPredictions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%v", payload["items"])
	}
}
