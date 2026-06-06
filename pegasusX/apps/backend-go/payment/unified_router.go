package payment

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// CartCheckoutHandler serves cart-shaped POST /v1/checkout/unified bodies.
type CartCheckoutHandler interface {
	HandleUnifiedCheckout(http.ResponseWriter, *http.Request)
}

// BindCartCheckout wires the order-service cart checkout handler.
func (s *Service) BindCartCheckout(handler CartCheckoutHandler) {
	s.cartCheckout = handler
}

func isCartUnifiedCheckoutBody(body []byte) bool {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return false
	}
	itemsRaw, ok := raw["items"]
	if !ok || len(itemsRaw) == 0 {
		return false
	}
	var items []json.RawMessage
	if err := json.Unmarshal(itemsRaw, &items); err != nil {
		return false
	}
	return len(items) > 0
}

func requestWithBody(r *http.Request, body []byte) *http.Request {
	clone := r.Clone(r.Context())
	clone.Body = io.NopCloser(bytes.NewReader(body))
	clone.ContentLength = int64(len(body))
	return clone
}
