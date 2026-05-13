package webhookroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/payment"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_GlobalPayUsesPriorityAndLog(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		WebhookSvc:    &payment.WebhookService{},
		Log:           markerMiddleware("X-Log", "global-pay"),
		PriorityGuard: markerMiddleware("X-Priority", "global-pay"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("status = %d, expected route to be mounted", rec.Code)
	}
	if got := rec.Header().Get("X-Log"); got != "global-pay" {
		t.Fatalf("log middleware header = %q, want global-pay", got)
	}
	if got := rec.Header().Get("X-Priority"); got != "global-pay" {
		t.Fatalf("priority middleware header = %q, want global-pay", got)
	}
}

func TestRegisterRoutes_AdyenUsesPriorityAndLog(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		WebhookSvc:    &payment.WebhookService{},
		Log:           markerMiddleware("X-Log", "adyen"),
		PriorityGuard: markerMiddleware("X-Priority", "adyen"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/adyen", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("status = %d, expected route to be mounted", rec.Code)
	}
	if got := rec.Header().Get("X-Log"); got != "adyen" {
		t.Fatalf("log middleware header = %q, want adyen", got)
	}
	if got := rec.Header().Get("X-Priority"); got != "adyen" {
		t.Fatalf("priority middleware header = %q, want adyen", got)
	}
}

func TestRegisterRoutes_StripeUsesPriorityAndLog(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		WebhookSvc:    &payment.WebhookService{},
		Log:           markerMiddleware("X-Log", "stripe"),
		PriorityGuard: markerMiddleware("X-Priority", "stripe"),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("status = %d, expected route to be mounted", rec.Code)
	}
	if got := rec.Header().Get("X-Log"); got != "stripe" {
		t.Fatalf("log middleware header = %q, want stripe", got)
	}
	if got := rec.Header().Get("X-Priority"); got != "stripe" {
		t.Fatalf("priority middleware header = %q, want stripe", got)
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
