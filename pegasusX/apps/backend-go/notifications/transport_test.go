package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogTransport_Deliver(t *testing.T) {
	t.Parallel()
	lt := &LogTransport{}
	notif := FormattedNotification{Title: "Test", Body: "Test Body", Priority: "normal"}
	if err := lt.Deliver(context.Background(), "user-1", "RETAILER", notif); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSMSTransport_PlayMobile(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sms := &SMSTransport{
		Provider: "playmobile",
		BaseURL:  ts.URL,
		Login:    "test-login",
		Password: "test-password",
		Client:   ts.Client(),
	}

	notif := FormattedNotification{Title: "Order Confirmed", Body: "ORD-123 ready", Priority: "high"}
	if err := sms.Deliver(context.Background(), "+998901234567", "RETAILER", notif); err != nil {
		t.Fatalf("failed to deliver playmobile: %v", err)
	}

	if receivedBody == nil {
		t.Fatal("server received no body")
	}
}

func TestSMSTransport_Twilio(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "AC123" || pass != "auth-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sms := &SMSTransport{
		Provider:   "twilio",
		BaseURL:    ts.URL,
		AccountSID: "AC123",
		AuthToken:  "auth-token",
		FromNumber: "+15550001",
		Client:     ts.Client(),
	}

	notif := FormattedNotification{Title: "Alert", Body: "Truck arrived"}
	if err := sms.Deliver(context.Background(), "+15551234", "DRIVER", notif); err != nil {
		t.Fatalf("failed to deliver twilio: %v", err)
	}
}

func TestEmailTransport_SendGrid(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer SG.test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()

	email := &EmailTransport{
		Provider: "sendgrid",
		APIKey:   "SG.test-key",
		From:     "noreply@pegasusx.com",
		Client:   ts.Client(),
	}

	notif := FormattedNotification{Title: "Monthly Report", Body: "Your invoice is ready"}
	// SendGrid hardcodes endpoint URL in code, so mock test checks recipient validation
	if err := email.Deliver(context.Background(), "invalid-email", "BUYER", notif); err != nil {
		t.Fatalf("expected skip on invalid email, got error: %v", err)
	}
}

func TestCompositeTransport(t *testing.T) {
	t.Parallel()
	lt := &LogTransport{}
	comp := &CompositeTransport{Transports: []Transport{lt}}
	notif := FormattedNotification{Title: "Test"}
	if err := comp.Deliver(context.Background(), "u1", "BUYER", notif); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
