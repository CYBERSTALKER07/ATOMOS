package payment

import (
	"bytes"
	"encoding/base64"
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
// Both Payme and Click implement this contract.
type GatewayClient interface {
	Charge(orderID string, amount int64) error
	Refund(orderID string, refundAmount int64) error
}

// SplittingGateway extends GatewayClient with auth-hold / partial-capture semantics.
// Global Pay implements this; Payme and Click use the legacy full-capture path.
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

// ─── PAYME ────────────────────────────────────────────────────────────────────

// PaymeClient handles full-capture charges and partial refunds via the Payme
// merchant API. All credentials are loaded from environment variables — never
// hardcoded.
//
// Required env vars:
//
//	PAYME_MERCHANT_ID  — your Payme merchant ID
//	PAYME_SECRET_KEY   — your Payme secret key (test or prod)
//	PAYME_API_URL      — defaults to https://checkout.paycom.uz/api
type PaymeClient struct {
	merchantID string
	secretKey  string
	apiURL     string
	httpClient *http.Client
}

func NewPaymeClient() (*PaymeClient, error) {
	mid := os.Getenv("PAYME_MERCHANT_ID")
	key := os.Getenv("PAYME_SECRET_KEY")
	if mid == "" || key == "" {
		return nil, fmt.Errorf("PAYME_MERCHANT_ID and PAYME_SECRET_KEY must be set")
	}
	apiURL := os.Getenv("PAYME_API_URL")
	if apiURL == "" {
		apiURL = "https://checkout.paycom.uz/api"
	}
	return &PaymeClient{
		merchantID: mid,
		secretKey:  key,
		apiURL:     apiURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

type paymeRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type paymeResponse struct {
	Result map[string]interface{} `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (p *PaymeClient) do(method string, params map[string]interface{}) (*paymeResponse, error) {
	payload, _ := json.Marshal(paymeRequest{Method: method, Params: params})
	creds := base64.StdEncoding.EncodeToString([]byte(p.merchantID + ":" + p.secretKey))

	buildReq := func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, p.apiURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic "+creds)
		return req, nil
	}

	resp, err := retryDo(p.httpClient, buildReq, 2)
	if err != nil {
		return nil, fmt.Errorf("payme HTTP error: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var pr paymeResponse
	if err := json.Unmarshal(raw, &pr); err != nil {
		return nil, fmt.Errorf("payme response parse error: %w", err)
	}
	if pr.Error != nil {
		return nil, fmt.Errorf("payme error %d: %s", pr.Error.Code, pr.Error.Message)
	}
	return &pr, nil
}

// Charge executes a FULL CAPTURE for the given order.
func (p *PaymeClient) Charge(orderID string, amount int64) error {
	// Payme expects amount in tiyins (1 UZS = 100 tiyins)
	amountTiyins := amount * 100
	_, err := p.do("receipts.create_p2p", map[string]interface{}{
		"amount":      amountTiyins,
		"order":       map[string]string{"id": orderID},
		"description": fmt.Sprintf("Lab Industries order %s full capture", orderID),
	})
	if err != nil {
		return fmt.Errorf("payme charge failed for order %s: %w", orderID, err)
	}
	log.Printf("[PAYMENT] Payme full capture: order=%s amount=%d", orderID, amount)
	return nil
}

// Refund issues a PARTIAL refund for rejected/damaged line items.
func (p *PaymeClient) Refund(orderID string, refundAmount int64) error {
	refundTiyins := refundAmount * 100
	_, err := p.do("receipts.cancel", map[string]interface{}{
		"amount": refundTiyins,
		"order":  map[string]string{"id": orderID},
		"reason": "PALLET_REJECTED_AT_DELIVERY",
	})
	if err != nil {
		return fmt.Errorf("payme refund failed for order %s: %w", orderID, err)
	}
	log.Printf("[PAYMENT] Payme partial refund: order=%s refund=%d", orderID, refundAmount)
	return nil
}

// ─── CLICK ────────────────────────────────────────────────────────────────────

// ClickClient handles full-capture charges and partial refunds via Click Up's
// merchant API.
//
// Required env vars:
//
//	CLICK_MERCHANT_ID   — your Click merchant ID
//	CLICK_SERVICE_ID    — your Click service ID
//	CLICK_SECRET_KEY    — your Click secret key
//	CLICK_API_URL       — defaults to https://api.click.uz/v2/merchant
type ClickClient struct {
	merchantID string
	serviceID  string
	secretKey  string
	apiURL     string
	httpClient *http.Client
}

func NewClickClient() (*ClickClient, error) {
	mid := os.Getenv("CLICK_MERCHANT_ID")
	sid := os.Getenv("CLICK_SERVICE_ID")
	key := os.Getenv("CLICK_SECRET_KEY")
	if mid == "" || sid == "" || key == "" {
		return nil, fmt.Errorf("CLICK_MERCHANT_ID, CLICK_SERVICE_ID and CLICK_SECRET_KEY must be set")
	}
	apiURL := os.Getenv("CLICK_API_URL")
	if apiURL == "" {
		apiURL = "https://api.click.uz/v2/merchant"
	}
	return &ClickClient{
		merchantID: mid,
		serviceID:  sid,
		secretKey:  key,
		apiURL:     apiURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

type clickInvoiceRequest struct {
	ServiceID   string `json:"service_id"`
	OrderID     string `json:"merchant_trans_id"`
	Amount      int64  `json:"amount"` // UZS, integer
	PhoneNumber string `json:"phone_number,omitempty"`
}

// Charge issues a full invoice/capture via Click Up.
func (c *ClickClient) Charge(orderID string, amount int64) error {
	payload, _ := json.Marshal(clickInvoiceRequest{
		ServiceID: c.serviceID,
		OrderID:   orderID,
		Amount:    amount,
	})

	buildReq := func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, c.apiURL+"/invoice/create", bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Auth", fmt.Sprintf("%s:%s", c.merchantID, c.secretKey))
		return req, nil
	}

	resp, err := retryDo(c.httpClient, buildReq, 2)
	if err != nil {
		return fmt.Errorf("click charge HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("click charge failed (HTTP %d): %s", resp.StatusCode, string(raw))
	}
	log.Printf("[PAYMENT] Click full capture: order=%s amount=%d", orderID, amount)
	return nil
}

// Refund issues a partial refund for rejected line items via Click.
func (c *ClickClient) Refund(orderID string, refundAmount int64) error {
	payload, _ := json.Marshal(map[string]interface{}{
		"service_id":        c.serviceID,
		"merchant_trans_id": orderID,
		"amount":            refundAmount,
		"reason":            "PALLET_REJECTED_AT_DELIVERY",
	})

	buildReq := func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, c.apiURL+"/payment/reverse", bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Auth", fmt.Sprintf("%s:%s", c.merchantID, c.secretKey))
		return req, nil
	}

	resp, err := retryDo(c.httpClient, buildReq, 2)
	if err != nil {
		return fmt.Errorf("click refund HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("click refund failed (HTTP %d): %s", resp.StatusCode, string(raw))
	}
	log.Printf("[PAYMENT] Click partial refund: order=%s refund=%d", orderID, refundAmount)
	return nil
}

// ─── FACTORY ─────────────────────────────────────────────────────────────────

// NewGatewayClient returns the correct GatewayClient implementation based on
// the PaymentGateway column value stored in Spanner.
// Recognised values: "PAYME", "CLICK", "GLOBAL_PAY", "UZCARD" (no live charge API), "CASH" (no-op).
func NewGatewayClient(gateway string) (GatewayClient, error) {
	switch gateway {
	case "PAYME":
		return NewPaymeClient()
	case "CLICK":
		return NewClickClient()
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
	case "PAYME":
		return paymeCheckoutURL(orderID, amount)
	case "CLICK":
		return clickCheckoutURL(orderID, amount)
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

// paymeCheckoutURL builds the Payme checkout redirect URL.
// Format: https://checkout.paycom.uz/<base64(m=MERCHANT_ID;ac.order_id=ORDER_ID;a=AMOUNT_TIYINS)>
func paymeCheckoutURL(orderID string, amount int64) (string, error) {
	mid := os.Getenv("PAYME_MERCHANT_ID")
	if mid == "" {
		return "", fmt.Errorf("PAYME_MERCHANT_ID not set")
	}
	amountTiyins := amount * 100
	raw := fmt.Sprintf("m=%s;ac.order_id=%s;a=%d", mid, orderID, amountTiyins)
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	checkoutBase := os.Getenv("PAYME_CHECKOUT_URL")
	if checkoutBase == "" {
		checkoutBase = "https://checkout.paycom.uz"
	}
	return fmt.Sprintf("%s/%s", checkoutBase, encoded), nil
}

// clickCheckoutURL builds the Click checkout redirect URL.
func clickCheckoutURL(orderID string, amount int64) (string, error) {
	mid := os.Getenv("CLICK_MERCHANT_ID")
	sid := os.Getenv("CLICK_SERVICE_ID")
	if mid == "" || sid == "" {
		return "", fmt.Errorf("CLICK_MERCHANT_ID and CLICK_SERVICE_ID must be set")
	}
	checkoutBase := os.Getenv("CLICK_CHECKOUT_URL")
	if checkoutBase == "" {
		checkoutBase = "https://my.click.uz/services/pay"
	}
	return fmt.Sprintf("%s?service_id=%s&merchant_id=%s&amount=%d&transaction_param=%s",
		checkoutBase, sid, mid, amount, orderID), nil
}

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
	case "PAYME":
		return paymeCheckoutURLWithCreds(orderID, amount, merchantId)
	case "CLICK":
		return clickCheckoutURLWithCreds(orderID, amount, merchantId, serviceId)
	case "GLOBAL_PAY":
		return globalPayCheckoutURLWithCreds(orderID, amount, merchantId)
	case "SIMULATED":
		return fmt.Sprintf("https://mock-gateway.lab.fake/checkout/%s?amount=%d&merchant=%s", orderID, amount, merchantId), nil
	default:
		return "", nil
	}
}

func paymeCheckoutURLWithCreds(orderID string, amount int64, merchantId string) (string, error) {
	if merchantId == "" {
		return "", fmt.Errorf("payme merchant_id required")
	}
	amountTiyins := amount * 100
	raw := fmt.Sprintf("m=%s;ac.order_id=%s;a=%d", merchantId, orderID, amountTiyins)
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	checkoutBase := os.Getenv("PAYME_CHECKOUT_URL")
	if checkoutBase == "" {
		checkoutBase = "https://checkout.paycom.uz"
	}
	return fmt.Sprintf("%s/%s", checkoutBase, encoded), nil
}

func clickCheckoutURLWithCreds(orderID string, amount int64, merchantId, serviceId string) (string, error) {
	if merchantId == "" || serviceId == "" {
		return "", fmt.Errorf("click merchant_id and service_id required")
	}
	checkoutBase := os.Getenv("CLICK_CHECKOUT_URL")
	if checkoutBase == "" {
		checkoutBase = "https://my.click.uz/services/pay"
	}
	return fmt.Sprintf("%s?service_id=%s&merchant_id=%s&amount=%d&transaction_param=%s",
		checkoutBase, serviceId, merchantId, amount, orderID), nil
}
