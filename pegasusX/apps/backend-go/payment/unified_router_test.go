package payment

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubCartCheckout struct {
	called bool
}

func (s *stubCartCheckout) HandleUnifiedCheckout(w http.ResponseWriter, r *http.Request) {
	s.called = true
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func TestHandleUnifiedCheckout_RoutesCartPayloadToOrderHandler(t *testing.T) {
	t.Parallel()

	cart := &stubCartCheckout{}
	svc := NewService(ServiceConfig{})
	svc.BindCartCheckout(cart)

	body := `{"retailer_id":"r-1","payment_gateway":"GLOBAL_PAY","items":[{"sku_id":"sku-1","quantity":1,"unit_price":1000}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/checkout/unified", strings.NewReader(body))
	res := httptest.NewRecorder()

	svc.HandleUnifiedCheckout(res, req)
	if !cart.called {
		t.Fatal("cart checkout handler not invoked")
	}
	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusCreated)
	}
}

func TestIsCartUnifiedCheckoutBody(t *testing.T) {
	t.Parallel()

	if !isCartUnifiedCheckoutBody([]byte(`{"items":[{"sku_id":"x","quantity":1,"unit_price":1}]}`)) {
		t.Fatal("expected cart body detection")
	}
	if isCartUnifiedCheckoutBody([]byte(`{"order_id":"o-1","amount_minor":100}`)) {
		t.Fatal("order_id body must not route as cart")
	}
	if isCartUnifiedCheckoutBody([]byte(`{"items":[]}`)) {
		t.Fatal("empty items must not route as cart")
	}
}
