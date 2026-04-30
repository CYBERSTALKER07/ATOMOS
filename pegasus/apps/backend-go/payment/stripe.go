package payment

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// StripeClient implements GatewayClient for international payments via Stripe.
//
// Required env vars:
//
//	STRIPE_SECRET_KEY   — sk_test_* or sk_live_*
//	STRIPE_API_URL      — defaults to https://api.stripe.com
//	STRIPE_CURRENCY     — defaults to "usd"
type StripeClient struct {
	secretKey  string
	apiURL     string
	currency   string
	httpClient *http.Client
}

// NewStripeClient creates a Stripe client from environment variables.
func NewStripeClient() (*StripeClient, error) {
	sk := os.Getenv("STRIPE_SECRET_KEY")
	if sk == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY not set")
	}
	apiURL := os.Getenv("STRIPE_API_URL")
	if apiURL == "" {
		apiURL = "https://api.stripe.com"
	}
	cur := os.Getenv("STRIPE_CURRENCY")
	if cur == "" {
		cur = "usd"
	}
	return &StripeClient{
		secretKey:  sk,
		apiURL:     apiURL,
		currency:   cur,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// Charge creates a Stripe PaymentIntent and confirms it immediately.
// amount is in minor units (cents for USD, tiyins for UZS).
func (c *StripeClient) Charge(orderID string, amount int64) error {
	data := url.Values{
		"amount":                    {fmt.Sprintf("%d", amount)},
		"currency":                  {c.currency},
		"confirm":                   {"true"},
		"metadata[order_id]":        {orderID},
		"automatic_payment_methods": {"[enabled]=true"},
	}

	resp, err := retryDo(c.httpClient, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, c.apiURL+"/v1/payment_intents",
			strings.NewReader(data.Encode()))
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(c.secretKey, "")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req, nil
	}, 2)
	if err != nil {
		return fmt.Errorf("stripe charge %s: %w", orderID, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("stripe charge %s: HTTP %d: %s", orderID, resp.StatusCode, body)
	}

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("stripe charge parse: %w", err)
	}

	log.Printf("[STRIPE] charge OK: pi=%s order=%s status=%s", result.ID, orderID, result.Status)
	return nil
}

// Refund creates a Stripe refund on the PaymentIntent linked to the order.
// Looks up the PI by metadata, then issues a partial/full refund.
func (c *StripeClient) Refund(orderID string, refundAmount int64) error {
	piID, err := c.findPaymentIntent(orderID)
	if err != nil {
		return fmt.Errorf("stripe refund lookup %s: %w", orderID, err)
	}

	data := url.Values{
		"payment_intent": {piID},
		"amount":         {fmt.Sprintf("%d", refundAmount)},
	}

	resp, err := retryDo(c.httpClient, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, c.apiURL+"/v1/refunds",
			strings.NewReader(data.Encode()))
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(c.secretKey, "")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req, nil
	}, 2)
	if err != nil {
		return fmt.Errorf("stripe refund %s: %w", orderID, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("stripe refund %s: HTTP %d: %s", orderID, resp.StatusCode, body)
	}

	log.Printf("[STRIPE] refund OK: order=%s amount=%d", orderID, refundAmount)
	return nil
}

// findPaymentIntent searches Stripe for a PI with metadata[order_id] = orderID.
func (c *StripeClient) findPaymentIntent(orderID string) (string, error) {
	searchURL := fmt.Sprintf("%s/v1/payment_intents/search?query=metadata['order_id']:'%s'",
		c.apiURL, url.QueryEscape(orderID))

	resp, err := retryDo(c.httpClient, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodGet, searchURL, nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(c.secretKey, "")
		return req, nil
	}, 2)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse search: %w", err)
	}
	if len(result.Data) == 0 {
		return "", fmt.Errorf("no payment intent found for order %s", orderID)
	}
	return result.Data[0].ID, nil
}
