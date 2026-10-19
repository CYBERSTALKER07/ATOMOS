package webhookroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

func TestRegisterRoutes_PaymeAndClickUnmounted(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{Service: &payment.Service{}})

	for _, path := range []string{"/v1/webhooks/payme", "/v1/webhooks/click"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d want 404 (unwired)", path, res.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code == http.StatusNotFound {
		t.Fatal("global-pay must stay mounted")
	}
}
