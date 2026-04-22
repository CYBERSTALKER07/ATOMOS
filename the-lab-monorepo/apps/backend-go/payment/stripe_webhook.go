package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// HandleStripeWebhook processes Stripe webhook events (payment_intent.succeeded,
// charge.refunded, etc.). Signature verification is the FIRST non-trivial
// statement per Webhook Playbook.
func (s *WebhookService) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body first — needed for both signature verification and parsing.
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Signature verification — FIRST before any parsing or mutation.
	sigHeader := r.Header.Get("Stripe-Signature")
	whSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if whSecret == "" {
		log.Printf("[STRIPE-WH] STRIPE_WEBHOOK_SECRET not configured — rejecting")
		http.Error(w, "webhook not configured", http.StatusInternalServerError)
		return
	}

	if !verifyStripeSignature(body, sigHeader, whSecret) {
		log.Printf("[STRIPE-WH] signature mismatch")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var event stripeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("[STRIPE-WH] parse error: %v", err)
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "payment_intent.succeeded":
		s.handleStripePaymentSucceeded(r, w, event)
	case "charge.refunded":
		s.handleStripeRefund(r, w, event)
	default:
		log.Printf("[STRIPE-WH] unhandled event type: %s", event.Type)
	}

	// Always ACK to Stripe — they will retry on non-2xx.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"received":true}`))
}

type stripeEvent struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data stripeEventData `json:"data"`
}

type stripeEventData struct {
	Object json.RawMessage `json:"object"`
}

type stripePaymentIntent struct {
	ID       string            `json:"id"`
	Amount   int64             `json:"amount"`
	Status   string            `json:"status"`
	Metadata map[string]string `json:"metadata"`
}

func (s *WebhookService) handleStripePaymentSucceeded(r *http.Request, w http.ResponseWriter, event stripeEvent) {
	var pi stripePaymentIntent
	if err := json.Unmarshal(event.Data.Object, &pi); err != nil {
		log.Printf("[STRIPE-WH] parse PI: %v", err)
		return
	}

	orderID := pi.Metadata["order_id"]
	if orderID == "" {
		log.Printf("[STRIPE-WH] no order_id in metadata for PI %s", pi.ID)
		return
	}

	log.Printf("[STRIPE-WH] payment succeeded: pi=%s order=%s amount=%d", pi.ID, orderID, pi.Amount)

	// Transition order payment state via Spanner RW txn.
	_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("Orders",
				[]string{"OrderId", "PaymentStatus", "UpdatedAt"},
				[]interface{}{orderID, "CAPTURED", time.Now().UTC()}),
		})
	})
	if err != nil {
		log.Printf("[STRIPE-WH] order update failed: order=%s err=%v", orderID, err)
	}
}

func (s *WebhookService) handleStripeRefund(r *http.Request, w http.ResponseWriter, event stripeEvent) {
	var obj struct {
		ID              string            `json:"id"`
		Amount          int64             `json:"amount"`
		AmountRefunded  int64             `json:"amount_refunded"`
		PaymentIntentID string            `json:"payment_intent"`
		Metadata        map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(event.Data.Object, &obj); err != nil {
		log.Printf("[STRIPE-WH] parse refund: %v", err)
		return
	}

	orderID := obj.Metadata["order_id"]
	if orderID == "" {
		log.Printf("[STRIPE-WH] no order_id in refund metadata for charge %s", obj.ID)
		return
	}

	log.Printf("[STRIPE-WH] refund processed: charge=%s order=%s refunded=%d", obj.ID, orderID, obj.AmountRefunded)
}

// verifyStripeSignature validates the Stripe-Signature header using HMAC-SHA256.
// Stripe sends: t=<timestamp>,v1=<hex-signature>[,v0=<legacy>]
// We compute HMAC-SHA256(whSecret, "<timestamp>.<body>") and compare with v1.
func verifyStripeSignature(body []byte, sigHeader, whSecret string) bool {
	if sigHeader == "" {
		return false
	}

	var timestamp, sig string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			sig = kv[1]
		}
	}
	if timestamp == "" || sig == "" {
		return false
	}

	// Compute expected signature.
	payload := fmt.Sprintf("%s.%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(whSecret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sig))
}
