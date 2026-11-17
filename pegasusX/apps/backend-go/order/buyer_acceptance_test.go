package order

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
)

func TestStampBuyerAcceptancePending_MySoliqOnly(t *testing.T) {
	now := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)

	pegasus := &Order{OrderID: "o1"}
	if stampBuyerAcceptancePending(pegasus, FiscalProviderPegasus, now) {
		t.Fatal("PEGASUS must not stamp buyer acceptance")
	}
	if pegasus.BuyerAcceptanceStatus != "" {
		t.Fatalf("status = %q, want empty", pegasus.BuyerAcceptanceStatus)
	}

	mysoliq := &Order{OrderID: "o2"}
	if !stampBuyerAcceptancePending(mysoliq, FiscalProviderMySoliq, now) {
		t.Fatal("MY_SOLIQ must stamp PENDING")
	}
	if mysoliq.BuyerAcceptanceStatus != BuyerAcceptancePending {
		t.Fatalf("status = %q, want PENDING", mysoliq.BuyerAcceptanceStatus)
	}
	if mysoliq.BuyerAcceptanceDeadline == nil {
		t.Fatal("deadline must be set")
	}
	wantDeadline := now.Add(BuyerAcceptanceWindowDefault)
	if !mysoliq.BuyerAcceptanceDeadline.Equal(wantDeadline) {
		t.Fatalf("deadline = %v, want %v", mysoliq.BuyerAcceptanceDeadline, wantDeadline)
	}
}

func TestStampBuyerAcceptancePending_DoesNotReopenResolved(t *testing.T) {
	now := time.Now().UTC()
	o := &Order{OrderID: "o3", BuyerAcceptanceStatus: BuyerAcceptanceRejected}
	if stampBuyerAcceptancePending(o, FiscalProviderMySoliq, now) {
		t.Fatal("must not reopen REJECTED")
	}
	if o.BuyerAcceptanceStatus != BuyerAcceptanceRejected {
		t.Fatalf("status changed to %q", o.BuyerAcceptanceStatus)
	}
}

func TestBuyerAcceptanceWindow_EnvOverride(t *testing.T) {
	t.Setenv("BUYER_ACCEPTANCE_DAYS", "3")
	if got := buyerAcceptanceWindow(); got != 3*24*time.Hour {
		t.Fatalf("window = %v, want 3d", got)
	}
	_ = os.Unsetenv("BUYER_ACCEPTANCE_DAYS")
	if got := buyerAcceptanceWindow(); got != BuyerAcceptanceWindowDefault {
		t.Fatalf("default window = %v, want %v", got, BuyerAcceptanceWindowDefault)
	}
}

func TestNewBuyerAcceptancePoller_CreditNoteDefaultsOn(t *testing.T) {
	t.Setenv("CREDIT_NOTE_AUTO_FROM_BUYER_REJECT", "")
	p := NewBuyerAcceptancePoller(nil, nil, nopLogger{}, nil)
	if !p.autoCreditNoteOnBuyerReject {
		t.Fatal("auto credit-note must default ON (P1-6)")
	}
	t.Setenv("CREDIT_NOTE_AUTO_FROM_BUYER_REJECT", "false")
	p2 := NewBuyerAcceptancePoller(nil, nil, nopLogger{}, nil)
	if p2.autoCreditNoteOnBuyerReject {
		t.Fatal("CREDIT_NOTE_AUTO_FROM_BUYER_REJECT=false must disable")
	}
}

type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

type pollTestRepo struct {
	*testRepo
	pending []*Order
}

func (r *pollTestRepo) FindPendingBuyerAcceptance(_ context.Context, _ int) ([]*Order, error) {
	return r.pending, nil
}

type spyESFSyncer struct {
	calls []struct {
		orderID string
		ehfID   string
		status  string
		reason  string
	}
}

func (s *spyESFSyncer) SyncOrderAcceptanceStatus(_ context.Context, orderID, ehfID, status, reason string, _ []byte) error {
	s.calls = append(s.calls, struct {
		orderID string
		ehfID   string
		status  string
		reason  string
	}{orderID: orderID, ehfID: ehfID, status: status, reason: reason})
	return nil
}

func TestBuyerAcceptancePoller_SyncsESFStatus(t *testing.T) {
	for _, tc := range []struct {
		soliqStatus string
		wantStatus  string
		wantOrderSt string
	}{
		{soliqStatus: "ACCEPTED", wantStatus: "ACCEPTED", wantOrderSt: BuyerAcceptanceAccepted},
		{soliqStatus: "REJECTED", wantStatus: "REJECTED", wantOrderSt: BuyerAcceptanceRejected},
	} {
		t.Run(tc.soliqStatus, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{"status": tc.soliqStatus},
				})
			}))
			defer srv.Close()

			sc := soliq.NewClient(soliq.SoliqConfig{BaseURL: srv.URL, Operator: "direct", Timeout: 5 * time.Second})
			ord := &Order{
				OrderID:               "ord-poll-" + tc.soliqStatus,
				SupplierID:            "sup-poll",
				RetailerID:            "ret-poll",
				LatestFiscalReceiptID: "ehf-poll-1",
				BuyerAcceptanceStatus: BuyerAcceptancePending,
			}
			repo := &pollTestRepo{testRepo: &testRepo{order: *ord, found: true}, pending: []*Order{ord}}
			poller := NewBuyerAcceptancePoller(sc, repo, nopLogger{}, nil)
			spy := &spyESFSyncer{}
			poller.SetESFSyncer(spy)

			poller.poll(context.Background())

			if repo.captured.BuyerAcceptanceStatus != tc.wantOrderSt {
				t.Fatalf("order BuyerAcceptanceStatus = %q, want %q", repo.captured.BuyerAcceptanceStatus, tc.wantOrderSt)
			}
			if len(spy.calls) != 1 {
				t.Fatalf("esfSyncer calls = %d, want 1", len(spy.calls))
			}
			if spy.calls[0].orderID != ord.OrderID || spy.calls[0].ehfID != "ehf-poll-1" || spy.calls[0].status != tc.wantStatus {
				t.Fatalf("unexpected sync call: %+v", spy.calls[0])
			}
		})
	}
}
