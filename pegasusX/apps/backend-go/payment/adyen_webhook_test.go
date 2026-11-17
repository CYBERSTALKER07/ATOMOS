package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	adyenhmac "github.com/adyen/adyen-go-api-library/v21/src/hmacvalidator"
	adyenwebhook "github.com/adyen/adyen-go-api-library/v21/src/webhook"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

const testAdyenHMACKey = "44DAC79C4F1804F0F43382A4E5A8B068831F61F03526F3020A4697600504E11E"

type mockAdyenRepo struct {
	mu           sync.Mutex
	webhooks     []WebhookRecord
	attempts     []PaymentAttemptRecord
	outboxEvents []events.FinanceEvent
}

func (m *mockAdyenRepo) CreateSession(ctx context.Context, s SessionRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockAdyenRepo) CreateSessionWithAttempt(ctx context.Context, s SessionRecord, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockAdyenRepo) SaveAttempt(ctx context.Context, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts = append(m.attempts, a)
	return nil
}
func (m *mockAdyenRepo) SaveChargeback(ctx context.Context, c ChargebackRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockAdyenRepo) SaveReversal(ctx context.Context, rev ReversalRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *mockAdyenRepo) SaveWebhook(ctx context.Context, w WebhookRecord, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhooks = append(m.webhooks, w)
	if emit != nil {
		buf := &testAdyenTxnBuffer{repo: m}
		if err := emit(buf); err != nil {
			return err
		}
	}
	return nil
}
func (m *mockAdyenRepo) FindStuckSessions(ctx context.Context, cutoff time.Time, limit int) ([]SessionRecord, error) {
	return nil, nil
}
func (m *mockAdyenRepo) GetSession(ctx context.Context, sessionID string) (SessionRecord, bool, error) {
	return SessionRecord{}, false, nil
}
func (m *mockAdyenRepo) GetSessionByOrderID(ctx context.Context, orderID string) (SessionRecord, bool, error) {
	return SessionRecord{}, false, nil
}
func (m *mockAdyenRepo) HasChargebackForOrder(ctx context.Context, orderID string) (bool, error) {
	return false, nil
}
func (m *mockAdyenRepo) ListLedgerEntries(ctx context.Context, q LedgerQuery) ([]LedgerEntryRecord, error) {
	return nil, nil
}
func (m *mockAdyenRepo) SummarizeLedgerEntries(ctx context.Context, q SettlementAuthorityQuery) ([]SettlementAuthorityRow, error) {
	return nil, nil
}
func (m *mockAdyenRepo) CreatePayer(ctx context.Context, p Payer) error { return nil }
func (m *mockAdyenRepo) GetPayer(ctx context.Context, payerID string) (Payer, error) {
	return Payer{}, nil
}
func (m *mockAdyenRepo) UpdatePayer(ctx context.Context, p Payer) error { return nil }
func (m *mockAdyenRepo) ListPayers(ctx context.Context, limit, offset int) ([]Payer, error) {
	return nil, nil
}

type testAdyenTxnBuffer struct {
	repo *mockAdyenRepo
}

func (b *testAdyenTxnBuffer) BufferOutbox(ctx context.Context, e outbox.Event) error {
	var fe events.FinanceEvent
	if err := json.Unmarshal(e.Payload, &fe); err == nil {
		b.repo.outboxEvents = append(b.repo.outboxEvents, fe)
	}
	return nil
}

func buildSignedAdyenWebhookPayload(t *testing.T, item adyenwebhook.NotificationRequestItem) []byte {
	sig, err := adyenhmac.CalculateHmac(item, testAdyenHMACKey)
	if err != nil {
		t.Fatalf("failed to calculate hmac: %v", err)
	}
	if item.AdditionalData == nil {
		m := make(map[string]interface{})
		item.AdditionalData = &m
	}
	(*item.AdditionalData)["hmacSignature"] = sig

	root := adyenwebhook.Webhook{
		Live: "false",
		NotificationItems: &[]adyenwebhook.NotificationItem{
			{
				NotificationRequestItem: item,
			},
		},
	}
	raw, err := json.Marshal(root)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	return raw
}

func TestHandleAdyenWebhook_ReturnsLiteralAccepted(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID:         "seed",
		AdyenWebhookSecret: testAdyenHMACKey,
	})

	// Valid payload with dummy HMAC
	payload := `{"notificationItems":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", bytes.NewBufferString(payload))
	w := httptest.NewRecorder()

	svc.HandleAdyenWebhook(w, req)
	// Empty notificationItems returns 422
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for empty items, got %d", w.Code)
	}

	// Method not allowed check
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/webhooks/adyen", nil)
	wGet := httptest.NewRecorder()
	svc.HandleAdyenWebhook(wGet, reqGet)
	if wGet.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET, got %d", wGet.Code)
	}
}

// 1. CHARGEBACK event with success = true -> verifies status is "CHARGEBACK", and EventPaymentCleared is not emitted.
func TestHandleAdyenWebhook_ChargebackDoesNotMarkPaid(t *testing.T) {
	repo := &mockAdyenRepo{}
	svc := NewService(ServiceConfig{
		SupplierID:         "sup-test",
		AdyenWebhookSecret: testAdyenHMACKey,
		Repo:               repo,
	})

	now := time.Now().UTC()
	item := adyenwebhook.NotificationRequestItem{
		EventCode:           "CHARGEBACK",
		Success:             "true",
		MerchantAccountCode: "TestMerchant",
		MerchantReference:   "order-cb-101",
		PspReference:        "PSP-CB-99901",
		PaymentMethod:       "visa",
		EventDate:           &now,
		Amount: adyenwebhook.Amount{
			Currency: "UZS",
			Value:    5000000,
		},
	}

	payload := buildSignedAdyenWebhookPayload(t, item)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	svc.HandleAdyenWebhook(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "[accepted]" {
		t.Fatalf("expected '[accepted]', got %q", w.Body.String())
	}

	if len(repo.webhooks) != 1 {
		t.Fatalf("expected 1 webhook recorded, got %d", len(repo.webhooks))
	}
	wb := repo.webhooks[0]
	if wb.Status != "CHARGEBACK" {
		t.Fatalf("expected status 'CHARGEBACK', got %q", wb.Status)
	}

	// Verify attempt recorded with CHARGEBACK_RECORD action
	if len(repo.attempts) != 1 {
		t.Fatalf("expected 1 attempt recorded, got %d", len(repo.attempts))
	}
	att := repo.attempts[0]
	if att.Status != "CHARGEBACK" {
		t.Fatalf("expected attempt status 'CHARGEBACK', got %q", att.Status)
	}
	if att.ExecutionAction != string(ExecutionActionChargebackRecord) {
		t.Fatalf("expected execution action %q, got %q", ExecutionActionChargebackRecord, att.ExecutionAction)
	}

	// Verify EventPaymentCleared is NEVER emitted for chargeback
	for _, ev := range repo.outboxEvents {
		if ev.Type == events.EventPaymentCleared {
			t.Fatalf("FATAL INVARIANT VIOLATION: EventPaymentCleared emitted for chargeback webhook!")
		}
	}
}

// 2. REFUND event -> verifies status is "REFUNDED".
func TestHandleAdyenWebhook_RefundEventMapsToRefunded(t *testing.T) {
	repo := &mockAdyenRepo{}
	svc := NewService(ServiceConfig{
		SupplierID:         "sup-test",
		AdyenWebhookSecret: testAdyenHMACKey,
		Repo:               repo,
	})

	now := time.Now().UTC()
	item := adyenwebhook.NotificationRequestItem{
		EventCode:           "REFUND",
		Success:             "true",
		MerchantAccountCode: "TestMerchant",
		MerchantReference:   "order-ref-202",
		PspReference:        "PSP-REF-88802",
		PaymentMethod:       "visa",
		EventDate:           &now,
		Amount: adyenwebhook.Amount{
			Currency: "UZS",
			Value:    2500000,
		},
	}

	payload := buildSignedAdyenWebhookPayload(t, item)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	svc.HandleAdyenWebhook(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	if len(repo.webhooks) != 1 {
		t.Fatalf("expected 1 webhook recorded, got %d", len(repo.webhooks))
	}
	wb := repo.webhooks[0]
	if wb.Status != "REFUNDED" {
		t.Fatalf("expected status 'REFUNDED', got %q", wb.Status)
	}

	if len(repo.attempts) != 1 {
		t.Fatalf("expected 1 attempt recorded, got %d", len(repo.attempts))
	}
	att := repo.attempts[0]
	if att.Status != "REFUNDED" {
		t.Fatalf("expected attempt status 'REFUNDED', got %q", att.Status)
	}
	if att.ExecutionAction != string(ExecutionActionRefund) {
		t.Fatalf("expected execution action %q, got %q", ExecutionActionRefund, att.ExecutionAction)
	}
}

// 3. CANCEL_OR_REFUND event -> verifies status is "CANCELLED".
func TestHandleAdyenWebhook_CancelOrRefundMapsToCancelled(t *testing.T) {
	repo := &mockAdyenRepo{}
	svc := NewService(ServiceConfig{
		SupplierID:         "sup-test",
		AdyenWebhookSecret: testAdyenHMACKey,
		Repo:               repo,
	})

	now := time.Now().UTC()
	item := adyenwebhook.NotificationRequestItem{
		EventCode:           "CANCEL_OR_REFUND",
		Success:             "true",
		MerchantAccountCode: "TestMerchant",
		MerchantReference:   "order-cancel-303",
		PspReference:        "PSP-CAN-77703",
		PaymentMethod:       "mastercard",
		EventDate:           &now,
		Amount: adyenwebhook.Amount{
			Currency: "UZS",
			Value:    1200000,
		},
	}

	payload := buildSignedAdyenWebhookPayload(t, item)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	svc.HandleAdyenWebhook(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	if len(repo.webhooks) != 1 {
		t.Fatalf("expected 1 webhook recorded, got %d", len(repo.webhooks))
	}
	wb := repo.webhooks[0]
	if wb.Status != "CANCELLED" {
		t.Fatalf("expected status 'CANCELLED', got %q", wb.Status)
	}

	if len(repo.attempts) != 1 {
		t.Fatalf("expected 1 attempt recorded, got %d", len(repo.attempts))
	}
	att := repo.attempts[0]
	if att.Status != "CANCELLED" {
		t.Fatalf("expected attempt status 'CANCELLED', got %q", att.Status)
	}
	if att.ExecutionAction != "CANCELLATION" {
		t.Fatalf("expected execution action 'CANCELLATION', got %q", att.ExecutionAction)
	}
}

// 4. Unknown event with success = true -> verifies status is "UNHANDLED" (never "PAID").
func TestHandleAdyenWebhook_UnknownEventDoesNotMarkPaid(t *testing.T) {
	repo := &mockAdyenRepo{}
	svc := NewService(ServiceConfig{
		SupplierID:         "sup-test",
		AdyenWebhookSecret: testAdyenHMACKey,
		Repo:               repo,
	})

	now := time.Now().UTC()
	item := adyenwebhook.NotificationRequestItem{
		EventCode:           "OFFER_CLOSED",
		Success:             "true",
		MerchantAccountCode: "TestMerchant",
		MerchantReference:   "order-unhandled-404",
		PspReference:        "PSP-UNH-66604",
		PaymentMethod:       "visa",
		EventDate:           &now,
		Amount: adyenwebhook.Amount{
			Currency: "UZS",
			Value:    300000,
		},
	}

	payload := buildSignedAdyenWebhookPayload(t, item)
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	svc.HandleAdyenWebhook(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	if len(repo.webhooks) != 1 {
		t.Fatalf("expected 1 webhook recorded, got %d", len(repo.webhooks))
	}
	wb := repo.webhooks[0]
	if wb.Status == "PAID" {
		t.Fatalf("FATAL INVARIANT VIOLATION: unknown event code was marked PAID!")
	}
	if wb.Status != "UNHANDLED" {
		t.Fatalf("expected status 'UNHANDLED', got %q", wb.Status)
	}

	// Verify EventPaymentCleared was NOT emitted
	for _, ev := range repo.outboxEvents {
		if ev.Type == events.EventPaymentCleared {
			t.Fatalf("FATAL INVARIANT VIOLATION: EventPaymentCleared emitted for unhandled webhook!")
		}
	}
}

// 5. Pre-fix exploit reproduction showing chargebacks are no longer marked paid.
func TestHandleAdyenWebhook_ExploitReproduction_ChargebackCannotClearPayment(t *testing.T) {
	repo := &mockAdyenRepo{}
	svc := NewService(ServiceConfig{
		SupplierID:         "sup-test",
		AdyenWebhookSecret: testAdyenHMACKey,
		Repo:               repo,
	})

	chargebackEvents := []string{
		"CHARGEBACK",
		"NOTIFICATION_OF_CHARGEBACK",
		"SECOND_CHARGEBACK",
	}

	for _, cbEvent := range chargebackEvents {
		now := time.Now().UTC()
		item := adyenwebhook.NotificationRequestItem{
			EventCode:           cbEvent,
			Success:             "true",
			MerchantAccountCode: "TestMerchant",
			MerchantReference:   "order-dispute-" + cbEvent,
			PspReference:        "PSP-DISPUTE-" + cbEvent,
			PaymentMethod:       "visa",
			EventDate:           &now,
			Amount: adyenwebhook.Amount{
				Currency: "UZS",
				Value:    10000000,
			},
		}

		payload := buildSignedAdyenWebhookPayload(t, item)
		req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", bytes.NewReader(payload))
		w := httptest.NewRecorder()

		svc.HandleAdyenWebhook(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for %s, got %d: %s", cbEvent, w.Code, w.Body.String())
		}
	}

	// Assert that none of the chargeback webhooks resulted in PAID status or EventPaymentCleared
	if len(repo.webhooks) != len(chargebackEvents) {
		t.Fatalf("expected %d webhooks recorded, got %d", len(chargebackEvents), len(repo.webhooks))
	}
	for i, wb := range repo.webhooks {
		if wb.Status == "PAID" {
			t.Fatalf("EXPLOIT DETECTED: Chargeback %s was marked PAID!", chargebackEvents[i])
		}
		if wb.Status != "CHARGEBACK" {
			t.Fatalf("expected status 'CHARGEBACK' for %s, got %q", chargebackEvents[i], wb.Status)
		}
	}

	for _, ev := range repo.outboxEvents {
		if ev.Type == events.EventPaymentCleared {
			t.Fatalf("EXPLOIT DETECTED: EventPaymentCleared emitted for chargeback event!")
		}
	}
}
