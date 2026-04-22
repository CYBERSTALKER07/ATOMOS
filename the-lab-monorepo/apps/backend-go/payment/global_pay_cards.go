// Package payment — Global Pay Cards Service wrapper.
// Enables card tokenization: save a retailer's card once, charge later via
// Payments Service Public without re-entering card details.
//
// DOCS-ONLY: All field names are based on provider documentation and must be
// validated during sandbox testing. Fields are marked with relevant comments.
package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GlobalPayCardsClient wraps the Global Pay Cards Service API.
// Requires GLOBAL_PAY_GATEWAY_BASE_URL to be set (e.g. https://gateway-api-dev.globalpay.uz).
type GlobalPayCardsClient struct {
	gatewayBaseURL string
	httpClient     *http.Client
}

// NewGlobalPayCardsClient creates a Cards Service client.
// Returns nil if the gateway base URL is not configured (feature disabled).
func NewGlobalPayCardsClient() *GlobalPayCardsClient {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("GLOBAL_PAY_GATEWAY_BASE_URL")), "/")
	if baseURL == "" {
		return nil
	}
	return &GlobalPayCardsClient{
		gatewayBaseURL: baseURL,
		httpClient:     &http.Client{Timeout: 20 * time.Second},
	}
}

// CardSaveResult is returned by InitiateCardSave.
type CardSaveResult struct {
	CardToken   string `json:"card_token"`
	RequiresOTP bool   `json:"requires_otp"`
}

// CardConfirmResult is returned by ConfirmCardOTP.
type CardConfirmResult struct {
	Confirmed bool   `json:"confirmed"`
	CardLast4 string `json:"card_last4,omitempty"`
	CardType  string `json:"card_type,omitempty"` // UZCARD | HUMO | VISA | MASTERCARD
}

// InitiateCardSave starts the card tokenization flow via Global Pay Cards Service.
// The retailer provides their phone number; Global Pay sends an OTP to the linked card.
// Returns a temporary card token that must be confirmed with ConfirmCardOTP.
func (c *GlobalPayCardsClient) InitiateCardSave(ctx context.Context, creds GlobalPayCredentials, retailerPhone string) (*CardSaveResult, error) {
	if c == nil {
		return nil, fmt.Errorf("global pay cards service not configured (GLOBAL_PAY_GATEWAY_BASE_URL is empty)")
	}

	accessToken, err := c.authenticate(ctx, creds)
	if err != nil {
		return nil, err
	}

	// DOCS: validate field names with sandbox — "phone", "account" are candidates
	payload := map[string]interface{}{
		"phone":   retailerPhone, // DOCS: validate with sandbox
		"account": retailerPhone, // DOCS: validate with sandbox
	}

	endpoint := c.gatewayBaseURL + "/cards/v1/card" // DOCS: validate with sandbox
	body, err := c.doJSON(ctx, http.MethodPost, endpoint, accessToken, payload)
	if err != nil {
		return nil, fmt.Errorf("card save initiation failed: %w", err)
	}

	cardToken := globalPayLookupString(body,
		"cardToken", "card_token",
		"token", "id",
	)
	if cardToken == "" {
		return nil, fmt.Errorf("card save response missing card token")
	}

	requiresOTP := globalPayLookupBool(body,
		"requiresOtp", "requires_otp",
		"otpRequired", "otp_required",
		"smsRequired", "sms_required",
	)
	// Default to true if provider doesn't explicitly say — OTP is expected for Uzbek cards
	if !requiresOTP {
		requiresOTP = true
	}

	return &CardSaveResult{
		CardToken:   cardToken,
		RequiresOTP: requiresOTP,
	}, nil
}

// ConfirmCardOTP confirms the OTP sent to the retailer's phone, completing
// the card tokenization. On success the card token becomes reusable for charges.
func (c *GlobalPayCardsClient) ConfirmCardOTP(ctx context.Context, creds GlobalPayCredentials, cardToken, otpCode string) (*CardConfirmResult, error) {
	if c == nil {
		return nil, fmt.Errorf("global pay cards service not configured")
	}
	if strings.TrimSpace(cardToken) == "" || strings.TrimSpace(otpCode) == "" {
		return nil, fmt.Errorf("card_token and otp_code are required")
	}

	accessToken, err := c.authenticate(ctx, creds)
	if err != nil {
		return nil, err
	}

	// DOCS: validate endpoint path with sandbox
	endpoint := fmt.Sprintf("%s/cards/v1/card/confirm/%s", c.gatewayBaseURL, cardToken)
	payload := map[string]interface{}{
		"otp":      otpCode, // DOCS: validate with sandbox
		"otp_code": otpCode, // DOCS: validate with sandbox
		"code":     otpCode, // DOCS: validate with sandbox
	}

	body, err := c.doJSON(ctx, http.MethodPost, endpoint, accessToken, payload)
	if err != nil {
		return nil, fmt.Errorf("card OTP confirmation failed: %w", err)
	}

	confirmed := globalPayLookupBool(body,
		"confirmed", "success", "verified",
		"is_confirmed", "isConfirmed",
	)
	cardLast4 := globalPayLookupString(body,
		"cardLast4", "card_last4", "last4",
		"pan_last4", "panLast4",
	)
	cardType := globalPayLookupString(body,
		"cardType", "card_type", "type",
		"paymentSystem", "payment_system",
		"cardName", "card_name",
	)

	return &CardConfirmResult{
		Confirmed: confirmed,
		CardLast4: cardLast4,
		CardType:  strings.ToUpper(cardType),
	}, nil
}

// authenticate obtains an OAuth access token from the gateway auth endpoint.
// Reuses the same authentication pattern as the Checkout Service.
func (c *GlobalPayCardsClient) authenticate(ctx context.Context, creds GlobalPayCredentials) (string, error) {
	// DOCS: validate auth endpoint path with sandbox
	authURL := c.gatewayBaseURL + "/payments/v1/merchant/auth"
	payload := map[string]interface{}{
		"username":    creds.Username,
		"merchant_id": creds.Username,
		"login":       creds.Username,
		"password":    creds.Password,
		"secret_key":  creds.Password,
	}

	body, err := c.doJSON(ctx, http.MethodPost, authURL, "", payload)
	if err != nil {
		return "", fmt.Errorf("global pay cards auth failed: %w", err)
	}

	accessToken := globalPayLookupString(body,
		"accessToken", "access_token",
		"token", "jwt",
	)
	if accessToken == "" {
		return "", fmt.Errorf("global pay cards auth response missing access token")
	}
	return accessToken, nil
}

// doJSON performs a JSON HTTP request and returns the parsed response body.
func (c *GlobalPayCardsClient) doJSON(ctx context.Context, method, target, accessToken string, payload interface{}) (map[string]interface{}, error) {
	var rawBody []byte
	if payload != nil {
		var err error
		rawBody, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	buildReq := func() (*http.Request, error) {
		var bodyReader io.Reader
		if len(rawBody) > 0 {
			bodyReader = bytes.NewReader(rawBody)
		}
		req, err := http.NewRequestWithContext(ctx, method, target, bodyReader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if accessToken != "" {
			req.Header.Set("Authorization", globalPayAuthScheme()+" "+accessToken)
		}
		return req, nil
	}

	resp, err := retryDo(c.httpClient, buildReq, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBytes)))
	}

	var body map[string]interface{}
	if err := json.Unmarshal(responseBytes, &body); err != nil {
		return nil, fmt.Errorf("response parse failed: %w", err)
	}
	return body, nil
}
