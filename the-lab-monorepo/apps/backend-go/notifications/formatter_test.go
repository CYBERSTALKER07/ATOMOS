package notifications

import (
	"strings"
	"testing"
)

func TestFormatOrderDispatched(t *testing.T) {
	n := FormatOrderDispatched("ROUTE-001", 3)
	if n.Title != "Order Dispatched" {
		t.Errorf("Title = %q; want Order Dispatched", n.Title)
	}
	if !strings.Contains(n.Body, "ROUTE-001") {
		t.Error("Body should contain route ID")
	}
	if !strings.Contains(n.Body, "3") {
		t.Error("Body should contain order count")
	}
}

func TestFormatDriverDispatched(t *testing.T) {
	n := FormatDriverDispatched("ROUTE-002", 5)
	if n.Title != "New Dispatch Assignment" {
		t.Errorf("Title = %q; want New Dispatch Assignment", n.Title)
	}
	if !strings.Contains(n.Body, "ROUTE-002") {
		t.Error("Body should contain route ID")
	}
}

func TestFormatDriverArrived(t *testing.T) {
	n := FormatDriverArrived("ORD-100")
	if n.Title != "Driver Has Arrived" {
		t.Errorf("Title = %q; want Driver Has Arrived", n.Title)
	}
	if !strings.Contains(n.Body, "ORD-100") {
		t.Error("Body should contain order ID")
	}
}

func TestFormatOrderStatusChanged(t *testing.T) {
	n := FormatOrderStatusChanged("ORD-200", "IN_TRANSIT", "ARRIVED")
	if n.Title != "Order Status Updated" {
		t.Errorf("Title = %q", n.Title)
	}
	if !strings.Contains(n.Body, "IN_TRANSIT") || !strings.Contains(n.Body, "ARRIVED") {
		t.Error("Body should contain both old and new states")
	}
}

func TestFormatPayloadReadyToSeal(t *testing.T) {
	n := FormatPayloadReadyToSeal("ROUTE-300", 4)
	if n.Title != "Orders Ready to Seal" {
		t.Errorf("Title = %q", n.Title)
	}
	if !strings.Contains(n.Body, "4") {
		t.Error("Body should contain order count")
	}
}

func TestFormatPayloadSealed(t *testing.T) {
	n := FormatPayloadSealed("ORD-400", "TERM-01")
	if n.Title != "Payload Sealed" {
		t.Errorf("Title = %q", n.Title)
	}
	if !strings.Contains(n.Body, "TERM-01") {
		t.Error("Body should contain terminal ID")
	}
}

func TestFormatPaymentSettled(t *testing.T) {
	n := FormatPaymentSettled("ORD-500", "CLICK", 150000)
	if n.Title != "Payment Received" {
		t.Errorf("Title = %q", n.Title)
	}
	if !strings.Contains(n.Body, "150000") {
		t.Error("Body should contain amount")
	}
	if !strings.Contains(n.Body, "CLICK") {
		t.Error("Body should contain gateway")
	}
}

func TestFormatPaymentFailed(t *testing.T) {
	n := FormatPaymentFailed("ORD-600", "PAYME", "insufficient funds")
	if n.Title != "Payment Failed" {
		t.Errorf("Title = %q", n.Title)
	}
	if !strings.Contains(n.Body, "insufficient funds") {
		t.Error("Body should contain reason")
	}
}

func TestFormatTelegram(t *testing.T) {
	n := FormattedNotification{Title: "Test Title", Body: "Test body text"}
	text := FormatTelegram(n)
	if !strings.HasPrefix(text, "*Test Title*") {
		t.Error("Telegram text should start with bold title")
	}
	if !strings.Contains(text, "Test body text") {
		t.Error("Telegram text should contain body")
	}
}
