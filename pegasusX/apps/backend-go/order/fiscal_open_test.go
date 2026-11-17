package order

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
)

func TestEvaluateOpenFiscalOrders_PolicyGatedNonBlocking(t *testing.T) {
	candidates := []OpenFiscalCandidate{
		{
			OrderID:            "ord-retry-ok",
			Status:             string(StatusFiscalFailed),
			FiscalStatus:       "FISCAL_RETRY_QUEUED",
			HasPaymentRecorded: true,
			HasCryptoOrQRPoD:   true,
		},
		{
			OrderID:            "ord-pending-ok",
			Status:             string(StatusFiscalizing),
			FiscalStatus:       FiscalStatusPending,
			HasPaymentRecorded: true,
			HasCryptoOrQRPoD:   true,
		},
	}

	snap := EvaluateOpenFiscalOrders(false, candidates)
	if snap.Frozen || snap.Count != 0 {
		t.Fatalf("expected non-blocking shift end when cash + PoD recorded, got Frozen=%v Count=%d", snap.Frozen, snap.Count)
	}
	if snap.RetryQueuedCount != 2 || snap.FiscalIssueCount != 2 || len(snap.RetryOrderIDs) != 2 {
		t.Fatalf("unexpected retry counts: %+v", snap)
	}

	// Missing QR/crypto PoD OR unrecorded cash MUST freeze shift-end.
	missingProof := EvaluateOpenFiscalOrders(false, []OpenFiscalCandidate{
		{
			OrderID:            "ord-no-pod",
			Status:             string(StatusFiscalFailed),
			FiscalStatus:       FiscalStatusFailed,
			HasPaymentRecorded: true,
			HasCryptoOrQRPoD:   false,
		},
	})
	if !missingProof.Frozen || missingProof.Count != 1 {
		t.Fatalf("missing QR PoD must freeze bag, got %+v", missingProof)
	}

	uncollected := EvaluateOpenFiscalOrders(false, []OpenFiscalCandidate{
		{
			OrderID:            "ord-unpaid",
			Status:             string(StatusFiscalizing),
			FiscalStatus:       FiscalStatusPending,
			HasPaymentRecorded: false,
			HasCryptoOrQRPoD:   true,
		},
	})
	if !uncollected.Frozen || uncollected.Count != 1 {
		t.Fatalf("unrecorded cash must freeze bag, got %+v", uncollected)
	}

	// Strict immediate mode forces blocking even with PoD + payment recorded.
	strict := EvaluateOpenFiscalOrders(true, candidates)
	if !strict.Frozen || strict.Count != 2 || strict.RetryQueuedCount != 0 {
		t.Fatalf("strict mode must freeze, got %+v", strict)
	}
}

type capturingFiscalNotifier struct {
	mu       sync.Mutex
	notified []FiscalReceiptRow
}

func (c *capturingFiscalNotifier) NotifyFiscalReceiptQR(_ context.Context, _ Order, r FiscalReceiptRow) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notified = append(c.notified, r)
	return nil
}

func TestFiscalFailureRaisesCriticalTicket_AndSuccessTriggersAsyncSMSAndTelegram(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusFiscalizing)
	o.OrderID = "ord-d10-1"
	o.FiscalStatus = FiscalStatusPending
	o.TotalMinor = 12500000 // 125 000 so'm
	o.Currency = "UZS"
	o.LatestFiscalAttemptID = "att-fail-1"

	repo := &testRepo{
		found: true,
		order: o,
		fiscalAttempts: map[string]FiscalReceiptRow{
			"ord-d10-1:att-fail-1": {
				OrderID:       "ord-d10-1",
				AttemptID:     "att-fail-1",
				SupplierID:    o.SupplierID,
				RetailerID:    o.RetailerID,
				Status:        FiscalAttemptPending,
				AmountMinor:   12500000,
				Currency:      "UZS",
				PaymentMethod: "CASH",
			},
		},
	}

	svc := newTestService(repo, now)
	// 1. Failure queues retry + raises CRITICAL Handler #1 ExceptionTicket.
	svc.SetFiscalProvider(hardFailProvider{reason: "ofd_timeout"})
	if err := svc.ApplyFiscalWorkerResult(context.Background(), "ord-d10-1", "att-fail-1"); err != nil {
		t.Fatalf("ApplyFiscalWorkerResult fail path err: %v", err)
	}
	if len(repo.exceptionTickets) != 1 {
		t.Fatalf("expected 1 ExceptionTicket raised on fiscal failure, got %d", len(repo.exceptionTickets))
	}
	ticket := repo.exceptionTickets[0]
	if ticket.Severity != "CRITICAL" || ticket.Type != "FISCAL_FAILED" || ticket.AssignedRole != "FINANCE" {
		t.Fatalf("unexpected fiscal exception ticket: %+v", ticket)
	}

	// 2. Subsequent SubmitFiscalReceipt recovery succeeds & triggers async whole-so'm SMS + Telegram QR.
	var smsBody string
	smsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read sms body: %v", err)
		}
		smsBody = string(raw)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer smsSrv.Close()

	var tgBody string
	tgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read tg body: %v", err)
		}
		tgBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer tgSrv.Close()

	capNotifier := &capturingFiscalNotifier{}
	defaultNotif := &DefaultFiscalSuccessNotifier{
		SMS: &notifications.SMSTransport{
			Provider: "playmobile",
			BaseURL:  smsSrv.URL,
			Client:   smsSrv.Client(),
		},
		Telegram: &notifications.TelegramBotSender{
			BotToken: "bot-tok",
			BaseURL:  tgSrv.URL,
			Client:   tgSrv.Client(),
		},
	}

	svc.SetFiscalProvider(FakeFiscalProvider{})
	svc.SetFiscalNotifier(capNotifier)
	claims := driverClaims()
	ctx := auth.WithTenant(context.Background(), auth.TenantContext{SupplierID: o.SupplierID, Source: "jwt"})
	resp, err := svc.SubmitFiscalReceipt(ctx, claims, "ord-d10-1", "")
	if err != nil {
		t.Fatalf("SubmitFiscalReceipt: %v", err)
	}
	if resp.State != StatusCompleted {
		t.Fatalf("state=%s want COMPLETED", resp.State)
	}
	svc.FlushFiscalNotifications()

	if len(capNotifier.notified) != 1 || capNotifier.notified[0].FiscalQR == "" {
		t.Fatalf("expected 1 notified fiscal receipt with QR, got %+v", capNotifier.notified)
	}

	msg := notifications.FormatFiscalReceiptQR("ord-d10-1", capNotifier.notified[0].FiscalReceiptID, capNotifier.notified[0].FiscalQR, 12500000, "UZS", "uz")
	if err := defaultNotif.SMS.Deliver(ctx, "+998901234567", "RETAILER", msg); err != nil {
		t.Fatalf("SMS deliver: %v", err)
	}
	if err := defaultNotif.Telegram.SendFiscalReceiptQR(ctx, "777888", "uz", msg, msg.DeepLink); err != nil {
		t.Fatalf("Telegram deliver: %v", err)
	}
	if !strings.Contains(smsBody, "125 000 so'm") || strings.Contains(smsBody, "12500000") {
		t.Fatalf("SMS body must show whole so'm without tiyin: %s", smsBody)
	}
	if !strings.Contains(tgBody, "125 000 so'm") || !strings.Contains(tgBody, "fake.ofd.local") {
		t.Fatalf("Telegram body must contain whole so'm and QR URL: %s", tgBody)
	}
}

// TestQueryOpenFiscalForDriver_FailedAndPendingReceiptsDoNotCountAsPaid verifies that
// failed or unpaid pending fiscal receipts do not evaluate HasPaymentRecorded to true,
// preventing drivers from bypassing shift-end cash freeze.
func TestQueryOpenFiscalForDriver_FailedAndPendingReceiptsDoNotCountAsPaid(t *testing.T) {
	// A driver completed delivery but fiscal failed and cash was never collected (HasPaymentRecorded=false)
	candidateWithFailedReceipt := OpenFiscalCandidate{
		OrderID:            "ord-fisc-failed-unpaid",
		Status:             string(StatusFiscalFailed),
		FiscalStatus:       "FAILED",
		HasPaymentRecorded: false, // Correct: FAILED fiscal receipt must NOT count as payment recorded!
		HasCryptoOrQRPoD:   true,
	}

	snap := EvaluateOpenFiscalOrders(false, []OpenFiscalCandidate{candidateWithFailedReceipt})
	if !snap.Frozen {
		t.Fatalf("shift-end must FREEZE when fiscal failed and cash is uncollected, but snap.Frozen was false")
	}
	if snap.Count != 1 {
		t.Fatalf("expected snap.Count=1 (blocking), got %d", snap.Count)
	}

	// Conversely, only when real payment was recorded (Status=SUCCESS or CASH_COLLECTION_GEOFENCE)
	// can a driver with PoD queue for background retry without shift freeze.
	candidateWithSuccessPayment := OpenFiscalCandidate{
		OrderID:            "ord-fisc-success-paid",
		Status:             string(StatusFiscalFailed),
		FiscalStatus:       "FISCAL_RETRY_QUEUED",
		HasPaymentRecorded: true, // Verified cash collected / successful tender
		HasCryptoOrQRPoD:   true,
	}

	snapPaid := EvaluateOpenFiscalOrders(false, []OpenFiscalCandidate{candidateWithSuccessPayment})
	if snapPaid.Frozen {
		t.Fatalf("shift-end should not freeze when cash was legitimately collected and PoD exists")
	}
	if snapPaid.Count != 0 || snapPaid.RetryQueuedCount != 1 {
		t.Fatalf("expected non-blocking retry queued, got Count=%d RetryCount=%d", snapPaid.Count, snapPaid.RetryQueuedCount)
	}
}
