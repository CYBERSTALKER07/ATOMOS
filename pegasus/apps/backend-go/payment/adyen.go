package payment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/adyen/adyen-go-api-library/v21/src/adyen"
	"github.com/adyen/adyen-go-api-library/v21/src/checkout"
	adyencommon "github.com/adyen/adyen-go-api-library/v21/src/common"
)

var (
	// ErrAdyenDirectOperationUnsupported marks operations that cannot be safely
	// expressed through the generic GatewayClient Charge/Refund contract.
	ErrAdyenDirectOperationUnsupported = errors.New("adyen direct charge/refund unsupported; use hosted checkout flow")
)

// AdyenCredentials stores the minimal merchant credentials required for API calls.
type AdyenCredentials struct {
	MerchantAccount        string
	APIKey                 string
	LiveEndpointURLPrefix  string
	Environment            adyencommon.Environment
	DirectExecutionEnabled bool
}

// AdyenCheckoutRequest is the hosted checkout init payload.
type AdyenCheckoutRequest struct {
	OrderID   string
	InvoiceID string
	SessionID string
	AttemptID string
	Amount    int64
	Currency  string
	ReturnURL string
	ExpiresAt *time.Time
}

// AdyenCheckoutResult contains the redirect URL and provider reference.
type AdyenCheckoutResult struct {
	RedirectURL       string
	ProviderReference string
	ExpiresAt         *time.Time
}

// adyenGateway intentionally exposes only provider identity in the generic
// resolver. Hosted checkout and webhook settlement are handled in dedicated paths.
type adyenGateway struct{}

func (a *adyenGateway) Charge(orderID string, amount int64) error {
	return fmt.Errorf("%w for order %s", ErrAdyenDirectOperationUnsupported, orderID)
}

func (a *adyenGateway) Refund(orderID string, refundAmount int64) error {
	return fmt.Errorf("%w for order %s", ErrAdyenDirectOperationUnsupported, orderID)
}

// ResolveAdyenCredentials merges supplier vault values with environment fallbacks.
func ResolveAdyenCredentials(merchantID, liveEndpointPrefix, secretKey string) (AdyenCredentials, error) {
	envRaw := firstNonEmpty(os.Getenv("ADYEN_ENVIRONMENT"), "TEST")
	creds := AdyenCredentials{
		MerchantAccount:        firstNonEmpty(merchantID, os.Getenv("ADYEN_MERCHANT_ACCOUNT")),
		APIKey:                 firstNonEmpty(secretKey, os.Getenv("ADYEN_API_KEY")),
		LiveEndpointURLPrefix:  firstNonEmpty(liveEndpointPrefix, os.Getenv("ADYEN_LIVE_ENDPOINT_PREFIX")),
		Environment:            resolveAdyenEnvironment(envRaw),
		DirectExecutionEnabled: strings.ToLower(firstNonEmpty(os.Getenv("ADYEN_DIRECT_EXECUTION_ENABLED"), "false")) == "true",
	}

	if creds.MerchantAccount == "" || creds.APIKey == "" {
		return AdyenCredentials{}, fmt.Errorf("adyen credentials incomplete: merchant_id and secret_key(api key) are required")
	}
	if creds.Environment == adyencommon.LiveEnv && creds.LiveEndpointURLPrefix == "" {
		return AdyenCredentials{}, fmt.Errorf("adyen live endpoint prefix required for LIVE environment")
	}
	return creds, nil
}

// CreateAdyenHostedCheckout creates a Pay by Link session using the official Adyen SDK.
func CreateAdyenHostedCheckout(ctx context.Context, creds AdyenCredentials, req AdyenCheckoutRequest) (*AdyenCheckoutResult, error) {
	if strings.TrimSpace(req.OrderID) == "" {
		return nil, fmt.Errorf("adyen checkout requires order_id")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("adyen checkout requires positive amount")
	}

	currency := normalizeCurrencyCode(req.Currency)
	if currency == "" {
		currency = "UZS"
	}

	client := adyen.NewClient(&adyencommon.Config{
		ApiKey:                creds.APIKey,
		Environment:           creds.Environment,
		LiveEndpointURLPrefix: creds.LiveEndpointURLPrefix,
		Log4XXError:           true,
		Log5XXError:           true,
	})
	service := client.Checkout()

	metadata := map[string]string{
		"order_id":   req.OrderID,
		"invoice_id": req.InvoiceID,
		"session_id": req.SessionID,
		"attempt_id": req.AttemptID,
	}
	cleanMetadata := make(map[string]string, len(metadata))
	for key, value := range metadata {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			cleanMetadata[key] = trimmed
		}
	}

	amountMinor := req.Amount * 100
	paymentLinkReq := checkout.NewPaymentLinkRequest(checkout.Amount{
		Currency: currency,
		Value:    amountMinor,
	}, creds.MerchantAccount, req.OrderID)
	description := fmt.Sprintf("Pegasus order %s", req.OrderID)
	paymentLinkReq.Description = &description
	if len(cleanMetadata) > 0 {
		paymentLinkReq.Metadata = &cleanMetadata
	}
	if req.ExpiresAt != nil {
		paymentLinkReq.ExpiresAt = req.ExpiresAt
	}
	if returnURL := strings.TrimSpace(req.ReturnURL); returnURL != "" {
		paymentLinkReq.ReturnUrl = &returnURL
	}

	input := service.PaymentLinksApi.PaymentLinksInput().PaymentLinkRequest(*paymentLinkReq)
	if idempotencyKey := strings.TrimSpace(firstNonEmpty(req.SessionID, req.AttemptID, req.OrderID)); idempotencyKey != "" {
		if len(idempotencyKey) > 64 {
			idempotencyKey = idempotencyKey[:64]
		}
		input = input.IdempotencyKey(idempotencyKey)
	}

	res, _, err := service.PaymentLinksApi.PaymentLinks(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("adyen payment link create failed: %w", err)
	}
	if strings.TrimSpace(res.Url) == "" || strings.TrimSpace(res.Id) == "" {
		return nil, fmt.Errorf("adyen payment link response missing redirect URL or reference")
	}

	return &AdyenCheckoutResult{
		RedirectURL:       strings.TrimSpace(res.Url),
		ProviderReference: strings.TrimSpace(res.Id),
		ExpiresAt:         res.ExpiresAt,
	}, nil
}

func resolveAdyenEnvironment(raw string) adyencommon.Environment {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "LIVE":
		return adyencommon.LiveEnv
	default:
		return adyencommon.TestEnv
	}
}

// AdyenDirectClient wraps strict execution endpoints for the direct execution path.
type AdyenDirectClient struct {
	client          *adyen.APIClient
	merchantAccount string
}

// CreateAdyenDirectClient creates an AdyenClient equipped for Refunds and direct charges.
func CreateAdyenDirectClient(creds AdyenCredentials) (*AdyenDirectClient, error) {
	client := adyen.NewClient(&adyencommon.Config{
		ApiKey:                creds.APIKey,
		Environment:           creds.Environment,
		LiveEndpointURLPrefix: creds.LiveEndpointURLPrefix,
		Log4XXError:           true,
		Log5XXError:           true,
	})
	return &AdyenDirectClient{
		client:          client,
		merchantAccount: creds.MerchantAccount,
	}, nil
}

// Refund calls Adyen's Modifications API to refund a captured payment.
func (a *AdyenDirectClient) Refund(ctx context.Context, req ProviderRefundRequest) (*ProviderRefundResult, error) {
	if strings.TrimSpace(req.PaymentID) == "" {
		return nil, fmt.Errorf("adyen refund requires provider payment id (psp reference)")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("adyen refund requires positive amount")
	}

	currency := normalizeCurrencyCode(req.Currency)
	if currency == "" {
		currency = "UZS"
	}

	service := a.client.Checkout()
	amountMinor := req.Amount * 100
	refundReq := checkout.NewPaymentRefundRequest(checkout.Amount{
		Currency: currency,
		Value:    amountMinor,
	}, a.merchantAccount)

	// In Adyen, the original payment ID is the pspReference of the payment to refund.
	pspReference := strings.TrimSpace(req.PaymentID)
	input := service.ModificationsApi.RefundCapturedPaymentInput(pspReference).PaymentRefundRequest(*refundReq)

	// Ensure idempotency for the refund
	if req.OrderID != "" {
		key := "refund_" + req.OrderID + "_" + pspReference
		if len(key) > 64 {
			key = key[:64]
		}
		input = input.IdempotencyKey(key)
	}

	res, _, err := service.ModificationsApi.RefundCapturedPayment(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("adyen refund request failed: %w", err)
	}

	return &ProviderRefundResult{
		ProviderRefundID:  res.PspReference,
		ProviderReference: res.PaymentPspReference,
		Status:            res.Status,
	}, nil
}
