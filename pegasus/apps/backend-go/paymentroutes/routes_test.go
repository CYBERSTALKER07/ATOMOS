package paymentroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_ChargebackUsesPriorityAndIdempotency(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Checkout:      stubCheckout{},
		Log:           passthroughMiddleware,
		PriorityGuard: markerMiddleware("X-Priority-Guard", "chargeback"),
		Idempotency:   markerMiddleware("X-Idempotency-Guard", "chargeback"),
	})

	req := httptest.NewRequest(http.MethodOptions, "/v1/payment/chargeback", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Priority-Guard"); got != "chargeback" {
		t.Fatalf("priority guard header = %q, want chargeback", got)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "chargeback" {
		t.Fatalf("idempotency guard header = %q, want chargeback", got)
	}
}

func TestRegisterRoutes_ReversalUsesPriorityAndIdempotency(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Checkout:      stubCheckout{},
		Log:           passthroughMiddleware,
		PriorityGuard: markerMiddleware("X-Priority-Guard", "reversal"),
		Idempotency:   markerMiddleware("X-Idempotency-Guard", "reversal"),
	})

	req := httptest.NewRequest(http.MethodOptions, "/v1/payment/chargeback/reversal", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Priority-Guard"); got != "reversal" {
		t.Fatalf("priority guard header = %q, want reversal", got)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "reversal" {
		t.Fatalf("idempotency guard header = %q, want reversal", got)
	}
}

func passthroughMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}

func markerMiddleware(name, value string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(name, value)
			next(w, r)
		}
	}
}

type stubCheckout struct{}

func (stubCheckout) HandleB2BCheckout(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (stubCheckout) HandleUnifiedCheckout(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
