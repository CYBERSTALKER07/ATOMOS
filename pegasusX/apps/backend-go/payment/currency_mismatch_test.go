package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

type stubOrderReader struct {
	currency string
}

func (s stubOrderReader) CheckoutSnapshot(ctx context.Context, orderID, retailerID string) (int64, string, error) {
	return 1000, s.currency, nil
}

func (s stubOrderReader) CheckoutOrderContext(ctx context.Context, orderID, retailerID string) (order.CheckoutOrderContext, error) {
	return order.CheckoutOrderContext{Currency: s.currency, WarehouseID: "wh-1"}, nil
}

func TestInitCheckoutSessionCurrencyMismatch(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)
	svc.BindOrderCheckoutReader(stubOrderReader{currency: "UZS"})

	_, _, _, err := svc.initCheckoutSession(context.Background(), "CARD", CheckoutRequest{
		OrderID: "ord-1", RetailerID: "ret-1", Gateway: "GLOBAL_PAY",
		Currency: "USD", AmountMinor: 1000,
	})
	if !errors.Is(err, auth.ErrPackCurrencyMismatch) {
		t.Fatalf("err=%v want pack_currency_mismatch", err)
	}
	if repo.createWithAttemptCalls != 0 {
		t.Fatalf("createWithAttemptCalls=%d want 0", repo.createWithAttemptCalls)
	}
}

func TestInitCheckoutSessionEmptyUsesOrderCurrency(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)
	if gp, ok := svc.execution.executors["GLOBAL_PAY"].(*globalpayProviderExecutor); ok {
		gp.allowStub = true
	}
	svc.now = func() time.Time { return time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC) }
	svc.newID = func(prefix string) string { return prefix + "-1" }
	svc.BindOrderCheckoutReader(stubOrderReader{currency: "UZS"})

	session, _, _, err := svc.initCheckoutSession(context.Background(), "CARD", CheckoutRequest{
		OrderID: "ord-2", RetailerID: "ret-1", Gateway: "GLOBAL_PAY",
		AmountMinor: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Currency != "UZS" {
		t.Fatalf("currency=%s want UZS", session.Currency)
	}
}

func TestInitCheckoutSessionOrderCurrencyNotPack(t *testing.T) {
	t.Parallel()
	svc := newPaymentServiceForExecutionTest(&paymentRepoStub{})
	svc.BindOrderCheckoutReader(stubOrderReader{currency: "KZT"})
	_, _, _, err := svc.initCheckoutSession(context.Background(), "CARD", CheckoutRequest{
		OrderID: "ord-3", RetailerID: "ret-1", Gateway: "GLOBAL_PAY",
		AmountMinor: 1000,
	})
	if !errors.Is(err, auth.ErrPackCurrencyMismatch) {
		t.Fatalf("err=%v", err)
	}
}

func TestInitCheckoutSessionStripeForbidden(t *testing.T) {
	t.Parallel()
	svc := newPaymentServiceForExecutionTest(&paymentRepoStub{})
	_, _, _, err := svc.initCheckoutSession(context.Background(), "CARD", CheckoutRequest{
		OrderID: "ord-4", RetailerID: "ret-1", Gateway: "STRIPE",
		Currency: "UZS", AmountMinor: 1000,
	})
	if !errors.Is(err, auth.ErrPackGatewayForbidden) {
		t.Fatalf("err=%v", err)
	}
}

func TestInitCheckoutSessionPegasusAlias(t *testing.T) {
	t.Parallel()
	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)
	if gp, ok := svc.execution.executors["GLOBAL_PAY"].(*globalpayProviderExecutor); ok {
		gp.allowStub = true
	}
	svc.now = func() time.Time { return time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC) }
	svc.newID = func(prefix string) string { return prefix + "-1" }
	session, _, _, err := svc.initCheckoutSession(context.Background(), "CARD", CheckoutRequest{
		OrderID: "ord-5", RetailerID: "ret-1", Gateway: "PEGASUS",
		Currency: "UZS", AmountMinor: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Gateway != "GLOBAL_PAY" {
		t.Fatalf("gateway=%s", session.Gateway)
	}
}
