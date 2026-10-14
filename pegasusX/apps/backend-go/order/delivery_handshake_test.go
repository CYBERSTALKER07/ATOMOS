package order

import (
	"context"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/packages/handoff"
)

func TestVerifyHandshake_TokenMustMatchPersistedDeliveryToken(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	const stored = "persisted-delivery-token-value"
	orderRecord := deliveryTestOrder(StatusArrived)
	orderRecord.QRToken = stored
	repo := &testRepo{found: true, order: orderRecord}
	svc := NewService(ServiceConfig{
		Repo: repo,
		Handoff: handoff.New(handoff.Config{
			LegacyOrderIDFallback: false,
		}),
	})
	claims := driverClaims()
	base := VerifyHandshakeRequest{
		OrderID:   orderRecord.OrderID,
		Latitude:  orderRecord.Lat,
		Longitude: orderRecord.Lng,
	}

	_, err := svc.VerifyHandshake(context.Background(), claims, base)
	if err == nil || !strings.Contains(err.Error(), "token required") {
		t.Fatalf("empty token err=%v", err)
	}

	shortReq := base
	shortReq.Token = "abc"
	if _, err := svc.VerifyHandshake(context.Background(), claims, shortReq); err == nil {
		t.Fatal("short token was accepted")
	}

	wrongReq := base
	wrongReq.Token = "wrong-token-not-the-persisted-one"
	if _, err := svc.VerifyHandshake(context.Background(), claims, wrongReq); err == nil {
		t.Fatal("mismatched token was accepted")
	}

	matchReq := base
	matchReq.Token = stored
	resp, err := svc.VerifyHandshake(context.Background(), claims, matchReq)
	if err != nil {
		t.Fatalf("matching token: %v", err)
	}
	if !resp.Success {
		t.Fatalf("response=%+v", resp)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("handshake mutated the order, updateCalls=%d", repo.updateCalls)
	}
}
