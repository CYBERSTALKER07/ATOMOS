package payment

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/simulator"
)

// Phase-1 exit gate: refund happy path proven against the Global Pay
// simulator's mirrored URL surface (auth -> backoffice perform RF).
func TestGlobalPayRefundAgainstSimulator(t *testing.T) {
	var sawAction string
	r := chi.NewRouter()
	sim := simulator.NewHandler("whsec-test", "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.Route("/sim/globalpay", func(r chi.Router) {
		simulator.RegisterRoutes(r, sim)
	})
	// Wrap perform to assert RF wire format without replacing the simulator.
	inner := r
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, "/perform") && req.Method == http.MethodPost {
			body, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(strings.NewReader(string(body)))
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			if a, ok := payload["action"].(string); ok {
				sawAction = a
			}
		}
		inner.ServeHTTP(w, req)
	}))
	defer ts.Close()

	exec := newGlobalPayProviderExecutorWithSimulator("dev", "svc-1", "user", "pass", ts.URL)
	res, err := exec.Execute(context.Background(), ExecutionRequest{
		Gateway:     "GLOBAL_PAY",
		Action:      ExecutionActionRefund,
		OrderID:     "ord-sim-refund-1",
		SessionID:   "pay-sim-1",
		AmountMinor: 4000,
		Currency:    "UZS",
	})
	if err != nil {
		t.Fatalf("simulator refund: %v", err)
	}
	if res.ProviderRef != "pay-sim-1" {
		t.Fatalf("provider ref = %q, want pay-sim-1", res.ProviderRef)
	}
	if res.ResolvedGateway != "GLOBAL_PAY" {
		t.Fatalf("resolved gateway = %q", res.ResolvedGateway)
	}
	if sawAction != "RF" {
		t.Fatalf("perform action = %q, want RF (override via GLOBAL_PAY_REFUND_ACTION)", sawAction)
	}
}

// Capture against the simulator remains green alongside refund (regression).
func TestGlobalPayCaptureAgainstSimulator(t *testing.T) {
	r := chi.NewRouter()
	sim := simulator.NewHandler("whsec-test", "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.Route("/sim/globalpay", func(r chi.Router) {
		simulator.RegisterRoutes(r, sim)
	})
	ts := httptest.NewServer(r)
	defer ts.Close()

	exec := newGlobalPayProviderExecutorWithSimulator("dev", "svc-1", "user", "pass", ts.URL)
	res, err := exec.Execute(context.Background(), ExecutionRequest{
		Gateway:     "GLOBAL_PAY",
		Action:      ExecutionActionCheckoutCapture,
		OrderID:     "ord-sim-capture-1",
		SessionID:   "pay-sim-1",
		AmountMinor: 10000,
		Currency:    "UZS",
	})
	if err != nil {
		t.Fatalf("simulator capture: %v", err)
	}
	if res.ProviderRef == "" {
		t.Fatal("capture must return a provider ref")
	}
}
