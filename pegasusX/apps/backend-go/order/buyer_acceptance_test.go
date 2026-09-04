package order

import (
	"os"
	"testing"
	"time"
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
