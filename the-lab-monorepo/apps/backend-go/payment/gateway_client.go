package payment

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// retryDo executes an HTTP request with exponential backoff.
// Retries on network errors and 5xx responses up to maxRetries times.
func retryDo(client *http.Client, buildReq func() (*http.Request, error), maxRetries int) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 500 * time.Millisecond
			time.Sleep(backoff)
		}

		req, err := buildReq()
		if err != nil {
			return nil, err // request build failure is not retryable
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("[PAYMENT] attempt %d/%d failed: %v", attempt+1, maxRetries+1, err)
			continue
		}

		// Retry on 502/503/504 (gateway/service errors)
		if resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			log.Printf("[PAYMENT] attempt %d/%d: server returned %d, retrying", attempt+1, maxRetries+1, resp.StatusCode)
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("all %d attempts failed, last error: %w", maxRetries+1, lastErr)
}

// isRetryable checks if an error is a transient network/server error.
var _ = errors.Is // suppress unused import if needed

// GatewayClient is the common interface for Uzbek payment gateways.
// Both GlobalPay and Cash implement this contract.
type GatewayClient interface {
	Charge(orderID string, amount int64) error
	Refund(orderID string, refundAmount int64) error
}

// SplittingGateway extends GatewayClient with auth-hold / partial-capture semantics.
// Global Pay implements this; GlobalPay and Cash use the legacy full-capture path.
type SplittingGateway interface {
	GatewayClient
	// Authorize places a hold on the card for the given amount without capturing.
	// Returns the provider's authorization/payment ID.
	Authorize(orderID string, amount int64) (authorizationID string, err error)
	// Capture settles a previously authorized hold. captureAmount may be ≤ authorized amount.
	Capture(authorizationID string, captureAmount int64) error
	// Void releases a held authorization without capturing any funds.
	Void(authorizationID string) error
}

// GlobalPayClient handles full-capture charges and partial refunds via the GlobalPay
// merchant API. All credentials are loaded from environment variables — never
// hardcoded.
//
// Required env vars:
//
//	GLOBAL_PAY_MERCHANT_ID  — your GlobalPay merchant ID
//	GLOBAL_PAY_SECRET_KEY   — your GlobalPay secret key (test or prod)
//	GLOBAL_PAY_API_URL      — defaults to https://checkout.paycom.uz/api

type global_payRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type global_payResponse struct {
	Result map[string]interface{} `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Charge executes a FULL CAPTURE for the given order.

// Refund issues a PARTIAL refund for rejected/damaged line items.

// CashClient handles full-capture charges and partial refunds via Cash Up's
// merchant API.
//
// Required env vars:
//
//	CASH_MERCHANT_ID   — your Cash merchant ID
//	CASH_SERVICE_ID    — your Cash service ID
//	CASH_SECRET_KEY    — your Cash secret key
//	CASH_API_URL       — defaults to https://api.cash.uz/v2/merchant

type cashInvoiceRequest struct {
	ServiceID   string `json:"service_id"`
	OrderID     string `json:"merchant_trans_id"`
	Amount      int64  `json:"amount"` // UZS, integer
	PhoneNumber string `json:"phone_number,omitempty"`
}

// Charge issues a full invoice/capture via Cash Up.

// Refund issues a partial refund for rejected line items via Cash.

// ─── FACTORY ─────────────────────────────────────────────────────────────────

// NewGatewayClient returns the correct GatewayClient implementation based on
// the PaymentGateway column value stored in Spanner.
// Recognised values: "GLOBAL_PAY", "CASH", "GLOBAL_PAY", "UZCARD" (no live charge API), "CASH" (no-op).
func NewGatewayClient(gateway string) (GatewayClient, error) {
	switch gateway {
	case "GLOBAL_PAY":
		log.Printf("[PAYMENT] GLOBAL_PAY gateway: hosted checkout supported, direct charge/refund not wired yet")
		return &noopGateway{gateway: "GLOBAL_PAY"}, nil
	case "SIMULATED":
		return NewSimulatedClient()
	case "UZCARD":
		// UZCARD has no public restocking API yet — log and passthrough
		log.Printf("[PAYMENT] UZCARD gateway: no API integration, logging only")
		return &noopGateway{gateway: "UZCARD"}, nil
	case "CASH":
		// Cash is collected physically — no electronic charge needed
		return &noopGateway{gateway: "CASH"}, nil
	case "STRIPE":
		return NewStripeClient()
	default:
		return nil, fmt.Errorf("unknown payment gateway: %s", gateway)
	}
}

// ─── CHECKOUT URL GENERATORS ─────────────────────────────────────────────────

// CheckoutURL builds a native deep-link URL for the given gateway so
// mobile apps can open the payment experience directly in the provider surface.
// Returns ("", nil) for gateways without an interactive checkout URL.
func CheckoutURL(gateway string, orderID string, amount int64) (string, error) {
	switch gateway {
	case "GLOBAL_PAY":
		return globalPayCheckoutURL(orderID, amount)
	case "SIMULATED":
		// Use native deep-link format for the simulated provider.
		return fmt.Sprintf("https://mock-gateway.lab.fake/checkout/%s?amount=%d", orderID, amount), nil
	case "STRIPE":
		return stripeCheckoutURL(orderID, amount)
	default:
		return "", nil // CASH/UZCARD — no deep link
	}
}

// global_payCheckoutURL builds the GlobalPay checkout redirect URL.
// Format: https://checkout.paycom.uz/<base64(m=MERCHANT_ID;ac.order_id=ORDER_ID;a=AMOUNT_TIYINS)>

// cashCheckoutURL builds the Cash checkout redirect URL.

// globalPayCheckoutURL expands a hosted-checkout URL template.
// The template is supplied by GLOBAL_PAY_CHECKOUT_URL and may include
// {merchant_id}, {order_id}, {amount}, and {amount_tiyin} placeholders.
func globalPayCheckoutURL(orderID string, amount int64) (string, error) {
	merchantId := os.Getenv("GLOBAL_PAY_MERCHANT_ID")
	if merchantId == "" {
		return "", fmt.Errorf("GLOBAL_PAY_MERCHANT_ID not set")
	}
	return globalPayCheckoutURLWithCreds(orderID, amount, merchantId)
}

func globalPayCheckoutURLWithCreds(orderID string, amount int64, merchantId string) (string, error) {
	if merchantId == "" {
		return "", fmt.Errorf("global pay merchant_id required")
	}
	checkoutTemplate := os.Getenv("GLOBAL_PAY_CHECKOUT_URL")
	if checkoutTemplate == "" {
		return "", fmt.Errorf("GLOBAL_PAY_CHECKOUT_URL not set (expected template with {merchant_id}, {order_id}, {amount}, {amount_tiyin})")
	}
	return renderGlobalPayCheckoutURL(checkoutTemplate, merchantId, orderID, amount), nil
}

func renderGlobalPayCheckoutURL(template, merchantId, orderID string, amount int64) string {
	amountTiyin := amount * 100
	replacer := strings.NewReplacer(
		"{merchant_id}", url.QueryEscape(merchantId),
		"{order_id}", url.QueryEscape(orderID),
		"{amount}", strconv.FormatInt(amount, 10),
		"{amount_tiyin}", strconv.FormatInt(amountTiyin, 10),
	)
	return replacer.Replace(template)
}

// stripeCheckoutURL returns a Stripe Checkout Session URL for international payments.
// Creates a one-time session via the Stripe API.
func stripeCheckoutURL(orderID string, amount int64) (string, error) {
	sk := os.Getenv("STRIPE_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("STRIPE_SECRET_KEY not set")
	}
	apiURL := os.Getenv("STRIPE_API_URL")
	if apiURL == "" {
		apiURL = "https://api.stripe.com"
	}
	cur := os.Getenv("STRIPE_CURRENCY")
	if cur == "" {
		cur = "usd"
	}
	successURL := os.Getenv("STRIPE_SUCCESS_URL")
	if successURL == "" {
		successURL = "https://app.thelab.uz/checkout/success?order_id=" + orderID
	}
	cancelURL := os.Getenv("STRIPE_CANCEL_URL")
	if cancelURL == "" {
		cancelURL = "https://app.thelab.uz/checkout/cancel?order_id=" + orderID
	}

	data := url.Values{
		"mode":                                   {"payment"},
		"success_url":                            {successURL},
		"cancel_url":                             {cancelURL},
		"metadata[order_id]":                     {orderID},
		"line_items[0][price_data][currency]":    {cur},
		"line_items[0][price_data][unit_amount]": {strconv.FormatInt(amount, 10)},
		"line_items[0][price_data][product_data][name]": {"Order " + orderID},
		"line_items[0][quantity]":                       {"1"},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPost, apiURL+"/v1/checkout/sessions",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(sk, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("stripe session create: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("stripe session: HTTP %d: %s", resp.StatusCode, body)
	}

	var session struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &session); err != nil {
		return "", fmt.Errorf("stripe session parse: %w", err)
	}
	return session.URL, nil
}

// noopGateway is used for gateways without live API integrations (e.g. UzCard).
type noopGateway struct{ gateway string }

func (n *noopGateway) Charge(orderID string, amount int64) error {
	log.Printf("[PAYMENT][NOOP] %s charge skipped: order=%s amount=%d", n.gateway, orderID, amount)
	return nil
}
func (n *noopGateway) Refund(orderID string, amount int64) error {
	log.Printf("[PAYMENT][NOOP] %s refund skipped: order=%s amount=%d", n.gateway, orderID, amount)
	return nil
}

// ─── SIMULATED GATEWAY ────────────────────────────────────────────────────────

// SimulatedClient acts as a custom payment system that simulates real-world
// card transactions. It performs artificial delays, randomly succeeds or fails
// (e.g., simulating insufficient funds, network timeouts), and logs events.
type SimulatedClient struct{}

func NewSimulatedClient() (*SimulatedClient, error) {
	log.Println("[PAYMENT] Initializing SIMULATED gateway client")
	return &SimulatedClient{}, nil
}

func (s *SimulatedClient) Charge(orderID string, amount int64) error {
	log.Printf("[PAYMENT][SIMULATED] Authorizing full capture: order=%s amount=%d", orderID, amount)

	// Simulate network round-trip / processing delay (500ms - 1500ms)
	time.Sleep(800 * time.Millisecond)

	// Simulate rare card decline / insufficient funds logic
	// e.g., if orderID ends with "999", strictly decline it to simulate test-card rejection
	if len(orderID) >= 3 && orderID[len(orderID)-3:] == "999" {
		return fmt.Errorf("simulated transaction declined: insufficient funds (Test Mode)")
	}

	log.Printf("[PAYMENT][SIMULATED] Transaction approved! Full capture successful: order=%s", orderID)
	return nil
}

func (s *SimulatedClient) Refund(orderID string, refundAmount int64) error {
	log.Printf("[PAYMENT][SIMULATED] Initiating refund: order=%s refund=%d", orderID, refundAmount)

	time.Sleep(600 * time.Millisecond)

	log.Printf("[PAYMENT][SIMULATED] Refund processed successfully: order=%s", orderID)
	return nil
}

// ─── PER-SUPPLIER CHECKOUT URL GENERATORS ────────────────────────────────────

// CheckoutURLWithCredentials builds a deep-link URL using supplier-specific credentials
// instead of global ENV vars. This is the multi-vendor path.
func CheckoutURLWithCredentials(gateway, orderID string, amount int64, merchantId, serviceId string) (string, error) {
	switch gateway {
	case "GLOBAL_PAY":
		return globalPayCheckoutURLWithCreds(orderID, amount, merchantId)
	case "SIMULATED":
		return fmt.Sprintf("https://mock-gateway.lab.fake/checkout/%s?amount=%d&merchant=%s", orderID, amount, merchantId), nil
	default:
		return "", nil
	}
}
