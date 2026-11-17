package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
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

func TestSMSTransport_PlayMobileHardening(t *testing.T) {
	// 1. Originator precedence: config field overrides PLAYMOBILE_ORIGINATOR env,
	//    env overrides default, and message-id is a valid UUID (never UnixNano).
	var gotOriginator string
	var gotMessageID string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				MessageID string `json:"message-id"`
				SMS       struct {
					Originator string `json:"originator"`
				} `json:"sms"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Messages) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		gotMessageID = body.Messages[0].MessageID
		gotOriginator = body.Messages[0].SMS.Originator
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Request is received"))
	}))
	defer ts.Close()

	t.Setenv("PLAYMOBILE_ORIGINATOR", "PegasusEnv")
	sms := &SMSTransport{
		Provider: "playmobile",
		BaseURL:  ts.URL,
		Login:    "user",
		Password: "pw",
		Client:   ts.Client(),
	}
	if err := sms.Deliver(context.Background(), "+998901112233", "RETAILER", FormattedNotification{Title: "Hi", Body: "Test"}); err != nil {
		t.Fatalf("env originator deliver: %v", err)
	}
	if gotOriginator != "PegasusEnv" {
		t.Fatalf("originator = %q want PegasusEnv", gotOriginator)
	}
	if _, err := uuid.Parse(gotMessageID); err != nil {
		t.Fatalf("expected UUID message-id, got %q: %v", gotMessageID, err)
	}

	sms.Originator = "PegasusCfg"
	firstID := gotMessageID
	if err := sms.Deliver(context.Background(), "+998901112233", "RETAILER", FormattedNotification{Title: "Hi", Body: "Test"}); err != nil {
		t.Fatalf("config originator deliver: %v", err)
	}
	if gotOriginator != "PegasusCfg" {
		t.Fatalf("originator = %q want PegasusCfg", gotOriginator)
	}
	if gotMessageID == firstID {
		t.Fatalf("expected distinct UUID per send, got duplicate %q", gotMessageID)
	}
}

func TestSMSTransport_PlayMobileJSONErrorCodesAndCircuitBreaker(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var respMode atomic.Int32 // 0: JSON error on HTTP 200, 1: nested message error, 2: ok
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		switch respMode.Load() {
		case 0:
			_, _ = w.Write([]byte(`{"error-code":102,"error-description":"Invalid originator"}`))
		case 1:
			_, _ = w.Write([]byte(`{"messages":[{"message-id":"x","status":"REJECTED","error-description":"blocked"}]}`))
		default:
			_, _ = w.Write([]byte(`{"messages":[{"message-id":"x","status":"ACCEPTED"}]}`))
		}
	}))
	defer ts.Close()

	cb := circuit.New("notification", circuit.Config{
		FailureThreshold: 2,
		FailureWindow:    5 * time.Second,
		OpenDuration:     200 * time.Millisecond,
	})
	sms := &SMSTransport{
		Provider:   "playmobile",
		BaseURL:    ts.URL,
		Originator: "PegasusX",
		Login:      "user",
		Password:   "pw",
		Client:     ts.Client(),
	}
	sms.SetCircuitBreaker(cb)

	// 1st failure: JSON error-code on HTTP 200.
	err := sms.Deliver(context.Background(), "+998901112233", "RETAILER", FormattedNotification{Title: "T", Body: "B"})
	if err == nil || !strings.Contains(err.Error(), "Invalid originator") {
		t.Fatalf("expected PlayMobile JSON error-code error, got %v", err)
	}

	// 2nd failure: nested message REJECTED on HTTP 200 -> trips breaker open.
	respMode.Store(1)
	err = sms.Deliver(context.Background(), "+998901112233", "RETAILER", FormattedNotification{Title: "T", Body: "B"})
	if err == nil || !strings.Contains(err.Error(), "REJECTED") {
		t.Fatalf("expected PlayMobile REJECTED error, got %v", err)
	}

	// 3rd call must short-circuit without calling upstream server.
	callsBefore := calls.Load()
	err = sms.Deliver(context.Background(), "+998901112233", "RETAILER", FormattedNotification{Title: "T", Body: "B"})
	if !errors.Is(err, circuit.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable when circuit open, got %v", err)
	}
	if calls.Load() != callsBefore {
		t.Fatalf("upstream server was called while circuit was open")
	}
}
