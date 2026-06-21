package order

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleRetailerRequestCancel_Disabled(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/request-cancel", strings.NewReader(`{"order_id":"ord-1"}`))
	rr := httptest.NewRecorder()

	svc.HandleRetailerRequestCancel(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOrderPayableAtDelivery(t *testing.T) {
	if !OrderPayableAtDelivery(StatusArrived) || !OrderPayableAtDelivery(StatusAwaitingPayment) {
		t.Fatal("delivery payment statuses should be payable")
	}
	if OrderPayableAtDelivery(StatusPending) || OrderPayableAtDelivery(StatusInTransit) {
		t.Fatal("pre-delivery statuses must not be payable")
	}
}
