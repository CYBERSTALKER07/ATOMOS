package driver

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleDriverReturnComplete_CashReconciliationGate(t *testing.T) {
	gateCalled := false
	svc := &Service{
		cashReconRequired: true,
		cashReconGate: func(ctx context.Context, driverID string) (bool, error) {
			gateCalled = true
			return false, nil
		},
		returnComplete: func(ctx context.Context, driverID string) (ReturnCompleteResult, bool, error) {
			return ReturnCompleteResult{ManifestID: "m1"}, true, nil
		},
		log: slog.Default(),
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/fleet/driver/return-complete", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "driver-1",
		Role:    auth.RoleDriver,
	}))
	rr := httptest.NewRecorder()
	svc.HandleDriverReturnComplete(rr, req)
	if !gateCalled {
		t.Fatal("expected cash reconciliation gate to be called")
	}
	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d want %d body=%s", rr.Code, http.StatusConflict, rr.Body.String())
	}
}
