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
	// executed because direct execution is disabled for the running environment.
	ErrAdyenDirectOperationUnsupported = errors.New("adyen direct execution disabled; use hosted checkout flow")
)

const (
	adyenDirectMaxAttempts = 3
	adyenDirectBaseBackoff = 250 * time.Millisecond
	adyenDirectMaxJitter   = 150 * time.Millisecond
	adyenDefaultReturnURL  = "https://example.com/adyen/return"
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

// ChargeStoredMethod performs an immediate stored-method payment using Adyen's
// /payments endpoint.
func (a *AdyenDirectClient) ChargeStoredMethod(ctx context.Context, req StoredMethodChargeRequest) (*StoredMethodChargeResult, error) {
	input := adyenStoredMethodPaymentInput{
		CardToken:                req.CardToken,
		Amount:                   req.Amount,
		Currency:                 req.Currency,
		OrderID:                  req.OrderID,
		SessionID:                req.SessionID,
		AttemptID:                req.AttemptID,
		ExternalID:               req.ExternalID,
		ReturnURL:                req.ReturnURL,
		ShopperReference:         req.ShopperReference,
		RecurringProcessingModel: req.RecurringProcessingModel,
	}

	response, err := a.executeStoredMethodPayment(ctx, input, false)
	if err != nil {
		return nil, err
	}

	resultCode := strings.TrimSpace(response.GetResultCode())
	if isAdyenFailureResultCode(resultCode) {
		return nil, adyenPaymentFailure(resultCode, response.GetRefusalReason())
	}

	providerRef := strings.TrimSpace(response.GetPspReference())
	result := &StoredMethodChargeResult{
		ProviderPaymentID: providerRef,
		ProviderReference: providerRef,
		Status:            resultCode,
		StoredMethodRef:   strings.TrimSpace(req.CardToken),
		RecurringCapable:  strings.TrimSpace(req.CardToken) != "",
	}
	if action := mapAdyenExecutionAction(response.Action); action != nil {
		result.Action = action
		return result, nil
	}

	result.Paid = isAdyenAcceptedResultCode(resultCode)
	return result, nil
}

// AuthorizeStoredMethod places an authorization hold and keeps capture as a
// later step via Modifications API.
func (a *AdyenDirectClient) AuthorizeStoredMethod(ctx context.Context, req StoredMethodAuthorizationRequest) (*StoredMethodAuthorizationResult, error) {
	input := adyenStoredMethodPaymentInput{
		CardToken:                req.CardToken,
		Amount:                   req.Amount,
		Currency:                 req.Currency,
		OrderID:                  req.OrderID,
		SessionID:                req.SessionID,
		AttemptID:                req.AttemptID,
		ExternalID:               req.ExternalID,
		ReturnURL:                req.ReturnURL,
		ShopperReference:         req.ShopperReference,
		RecurringProcessingModel: req.RecurringProcessingModel,
	}

	response, err := a.executeStoredMethodPayment(ctx, input, true)
	if err != nil {
		return nil, err
	}

	resultCode := strings.TrimSpace(response.GetResultCode())
	if isAdyenFailureResultCode(resultCode) {
		return nil, adyenPaymentFailure(resultCode, response.GetRefusalReason())
	}

	providerRef := strings.TrimSpace(response.GetPspReference())
	result := &StoredMethodAuthorizationResult{
		AuthorizationID:   providerRef,
		ProviderReference: providerRef,
		Status:            resultCode,
	}
	if action := mapAdyenExecutionAction(response.Action); action != nil {
		result.Action = action
		return result, nil
	}

	result.Authorized = isAdyenAcceptedResultCode(resultCode)
	return result, nil
}

// CaptureAuthorization captures an existing authorization in full or in part.
func (a *AdyenDirectClient) CaptureAuthorization(ctx context.Context, req CaptureAuthorizationRequest) (*CaptureAuthorizationResult, error) {
	authorizationID := strings.TrimSpace(req.AuthorizationID)
	if authorizationID == "" {
		return nil, fmt.Errorf("adyen capture requires authorization id")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("adyen capture requires positive amount")
	}

	currency := normalizeCurrencyCode(req.Currency)
	amountMinor := req.Amount * 100
	service := a.client.Checkout()

	captureRequest := checkout.NewPaymentCaptureRequest(checkout.Amount{
		Currency: currency,
		Value:    amountMinor,
	}, a.merchantAccount)
	reference := adyenReference("capture", authorizationID, fmt.Sprintf("%d", req.Amount), currency)
	captureRequest.Reference = &reference

	idempotencyKey := adyenIdempotencyKey("capture", authorizationID, fmt.Sprintf("%d", req.Amount), currency)
	response, err := retryAdyenDirectCall(ctx, "capture", func(callCtx context.Context) (checkout.PaymentCaptureResponse, error) {
		input := service.ModificationsApi.CaptureAuthorisedPaymentInput(authorizationID).PaymentCaptureRequest(*captureRequest)
		if idempotencyKey != "" {
			input = input.IdempotencyKey(idempotencyKey)
		}
		res, _, callErr := service.ModificationsApi.CaptureAuthorisedPayment(callCtx, input)
		return res, callErr
	})
	if err != nil {
		return nil, err
	}

	status := strings.TrimSpace(response.Status)
	providerPaymentID := strings.TrimSpace(response.PaymentPspReference)
	providerReference := strings.TrimSpace(response.PspReference)
	if providerReference == "" {
		providerReference = providerPaymentID
	}

	return &CaptureAuthorizationResult{
		ProviderPaymentID: providerPaymentID,
		ProviderReference: providerReference,
		Status:            status,
		Captured:          strings.EqualFold(status, "received"),
	}, nil
}

// VoidAuthorization cancels an authorization that has not yet been captured.
func (a *AdyenDirectClient) VoidAuthorization(ctx context.Context, authorizationID string) error {
	trimmedAuthorizationID := strings.TrimSpace(authorizationID)
	if trimmedAuthorizationID == "" {
		return fmt.Errorf("adyen void requires authorization id")
	}

	service := a.client.Checkout()
	cancelRequest := checkout.NewStandalonePaymentCancelRequest(a.merchantAccount, trimmedAuthorizationID)
	reference := adyenReference("void", trimmedAuthorizationID)
	cancelRequest.Reference = &reference

	idempotencyKey := adyenIdempotencyKey("void", trimmedAuthorizationID)
	response, err := retryAdyenDirectCall(ctx, "void", func(callCtx context.Context) (checkout.StandalonePaymentCancelResponse, error) {
		input := service.ModificationsApi.CancelAuthorisedPaymentInput().StandalonePaymentCancelRequest(*cancelRequest)
		if idempotencyKey != "" {
			input = input.IdempotencyKey(idempotencyKey)
		}
		res, _, callErr := service.ModificationsApi.CancelAuthorisedPayment(callCtx, input)
		return res, callErr
	})
	if err != nil {
		return err
	}

	if !strings.EqualFold(strings.TrimSpace(response.Status), "received") {
		return fmt.Errorf("adyen void was not accepted for authorization %s", trimmedAuthorizationID)
	}

	return nil
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
	reference := adyenReference("refund", req.OrderID, req.PaymentID, fmt.Sprintf("%d", req.Amount), currency)
	refundReq.Reference = &reference

	// In Adyen, the original payment ID is the pspReference of the payment to refund.
	pspReference := strings.TrimSpace(req.PaymentID)
	input := service.ModificationsApi.RefundCapturedPaymentInput(pspReference).PaymentRefundRequest(*refundReq)

	// Ensure idempotency for the refund
	idempotencyKey := adyenIdempotencyKey("refund", req.OrderID, pspReference, fmt.Sprintf("%d", req.Amount), currency)
	if idempotencyKey != "" {
		input = input.IdempotencyKey(idempotencyKey)
	}

	res, err := retryAdyenDirectCall(ctx, "refund", func(callCtx context.Context) (checkout.PaymentRefundResponse, error) {
		response, _, callErr := service.ModificationsApi.RefundCapturedPayment(callCtx, input)
		return response, callErr
	})
	if err != nil {
		return nil, fmt.Errorf("adyen refund request failed: %w", err)
	}

	return &ProviderRefundResult{
		ProviderRefundID:  res.PspReference,
		ProviderReference: res.PaymentPspReference,
		Status:            res.Status,
	}, nil
}

type adyenStoredMethodPaymentInput struct {
	CardToken                string
	Amount                   int64
	Currency                 string
	OrderID                  string
	SessionID                string
	AttemptID                string
	ExternalID               string
	ReturnURL                string
	ShopperReference         string
	RecurringProcessingModel string
}

func (a *AdyenDirectClient) executeStoredMethodPayment(ctx context.Context, req adyenStoredMethodPaymentInput, preAuthorize bool) (*checkout.PaymentResponse, error) {
	cardToken := strings.TrimSpace(req.CardToken)
	if cardToken == "" {
		return nil, fmt.Errorf("adyen stored-method payment requires card token")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("adyen stored-method payment requires positive amount")
	}

	currency := normalizeCurrencyCode(req.Currency)
	amountMinor := req.Amount * 100

	paymentMethodDetails := checkout.NewStoredPaymentMethodDetails()
	paymentMethodDetails.SetStoredPaymentMethodId(cardToken)
	paymentMethodDetails.SetType("scheme")
	if attemptID := strings.TrimSpace(req.AttemptID); attemptID != "" {
		paymentMethodDetails.SetCheckoutAttemptId(attemptID)
	}

	reference := adyenReference("pay", req.ExternalID, req.OrderID, req.SessionID, req.AttemptID)
	returnURL := resolveAdyenReturnURL(req.ReturnURL)
	paymentRequest := checkout.NewPaymentRequest(checkout.Amount{
		Currency: currency,
		Value:    amountMinor,
	}, a.merchantAccount, checkout.StoredPaymentMethodDetailsAsCheckoutPaymentMethod(paymentMethodDetails), reference, returnURL)

	shopperInteraction := "ContAuth"
	paymentRequest.ShopperInteraction = &shopperInteraction
	recurringModel := firstNonEmpty(strings.TrimSpace(req.RecurringProcessingModel), "CardOnFile")
	paymentRequest.RecurringProcessingModel = &recurringModel
	if shopperReference := strings.TrimSpace(req.ShopperReference); shopperReference != "" {
		paymentRequest.ShopperReference = &shopperReference
	}
	if orderID := strings.TrimSpace(req.OrderID); orderID != "" {
		paymentRequest.MerchantOrderReference = &orderID
	}

	metadata := map[string]string{}
	if orderID := strings.TrimSpace(req.OrderID); orderID != "" {
		metadata["order_id"] = orderID
	}
	if sessionID := strings.TrimSpace(req.SessionID); sessionID != "" {
		metadata["session_id"] = sessionID
	}
	if attemptID := strings.TrimSpace(req.AttemptID); attemptID != "" {
		metadata["attempt_id"] = attemptID
	}
	if len(metadata) > 0 {
		paymentRequest.Metadata = &metadata
	}

	if preAuthorize {
		additionalData := map[string]string{"authorisationType": "PreAuth"}
		paymentRequest.AdditionalData = &additionalData
	}

	service := a.client.Checkout()
	idempotencyKey := adyenIdempotencyKey(
		"payments",
		reference,
		fmt.Sprintf("%d", req.Amount),
		currency,
	)

	response, err := retryAdyenDirectCall(ctx, "payments", func(callCtx context.Context) (checkout.PaymentResponse, error) {
		input := service.PaymentsApi.PaymentsInput().PaymentRequest(*paymentRequest)
		if idempotencyKey != "" {
			input = input.IdempotencyKey(idempotencyKey)
		}
		res, _, callErr := service.PaymentsApi.Payments(callCtx, input)
		return res, callErr
	})
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func retryAdyenDirectCall[T any](ctx context.Context, operation string, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	var lastErr error
	attempts := 0

	for attempt := 1; attempt <= adyenDirectMaxAttempts; attempt++ {
		attempts = attempt
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isAdyenRetryableError(err) || attempt == adyenDirectMaxAttempts {
			break
		}

		backoff := adyenDirectBaseBackoff * time.Duration(1<<(attempt-1))
		jitter := time.Duration(time.Now().UnixNano() % int64(adyenDirectMaxJitter))
		waitFor := backoff + jitter

		timer := time.NewTimer(waitFor)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, fmt.Errorf("adyen %s canceled while retrying: %w", operation, ctx.Err())
		case <-timer.C:
		}
	}

	if lastErr == nil {
		return zero, fmt.Errorf("adyen %s failed", operation)
	}
	if attempts > 1 {
		return zero, fmt.Errorf("adyen %s failed after %d attempts: %w", operation, attempts, lastErr)
	}
	return zero, fmt.Errorf("adyen %s failed: %w", operation, lastErr)
}

func isAdyenRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var apiErr adyencommon.APIError
	if errors.As(err, &apiErr) {
		status := int(apiErr.Status)
		if status == 429 || status >= 500 {
			return true
		}
		return false
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))
	retryableSignatures := []string{
		"timeout",
		"temporarily unavailable",
		"connection reset",
		"connection refused",
		"broken pipe",
		"eof",
		"http 429",
		"http 502",
		"http 503",
		"http 504",
		"too many requests",
	}
	for _, signature := range retryableSignatures {
		if strings.Contains(message, signature) {
			return true
		}
	}

	return false
}

func mapAdyenExecutionAction(action *checkout.PaymentResponseAction) *ExecutionAction {
	if action == nil {
		return nil
	}

	if redirect := action.CheckoutRedirectAction; redirect != nil {
		payload := map[string]any{}
		if method, ok := redirect.GetMethodOk(); ok && strings.TrimSpace(*method) != "" {
			payload["method"] = strings.ToUpper(strings.TrimSpace(*method))
		}
		if data, ok := redirect.GetDataOk(); ok && len(*data) > 0 {
			payload["data"] = *data
		}
		return &ExecutionAction{
			Type:        strings.ToUpper(strings.TrimSpace(redirect.GetType())),
			RedirectURL: strings.TrimSpace(redirect.GetUrl()),
			Payload:     payloadOrNil(payload),
		}
	}

	if native := action.CheckoutNativeRedirectAction; native != nil {
		payload := map[string]any{}
		if method, ok := native.GetMethodOk(); ok && strings.TrimSpace(*method) != "" {
			payload["method"] = strings.ToUpper(strings.TrimSpace(*method))
		}
		if data, ok := native.GetDataOk(); ok && len(*data) > 0 {
			payload["data"] = *data
		}
		if nativeRedirectData, ok := native.GetNativeRedirectDataOk(); ok && strings.TrimSpace(*nativeRedirectData) != "" {
			payload["native_redirect_data"] = strings.TrimSpace(*nativeRedirectData)
		}
		return &ExecutionAction{
			Type:        strings.ToUpper(strings.TrimSpace(native.GetType())),
			RedirectURL: strings.TrimSpace(native.GetUrl()),
			Payload:     payloadOrNil(payload),
		}
	}

	if threeDS2 := action.CheckoutThreeDS2Action; threeDS2 != nil {
		payload := map[string]any{}
		if token, ok := threeDS2.GetTokenOk(); ok && strings.TrimSpace(*token) != "" {
			payload["token"] = strings.TrimSpace(*token)
		}
		if paymentData, ok := threeDS2.GetPaymentDataOk(); ok && strings.TrimSpace(*paymentData) != "" {
			payload["payment_data"] = strings.TrimSpace(*paymentData)
		}
		if subtype, ok := threeDS2.GetSubtypeOk(); ok && strings.TrimSpace(*subtype) != "" {
			payload["subtype"] = strings.TrimSpace(*subtype)
		}
		if authorizationToken, ok := threeDS2.GetAuthorisationTokenOk(); ok && strings.TrimSpace(*authorizationToken) != "" {
			payload["authorisation_token"] = strings.TrimSpace(*authorizationToken)
		}
		return &ExecutionAction{
			Type:        strings.ToUpper(strings.TrimSpace(threeDS2.GetType())),
			RedirectURL: strings.TrimSpace(threeDS2.GetUrl()),
			Payload:     payloadOrNil(payload),
		}
	}

	return &ExecutionAction{Type: "ACTION_REQUIRED"}
}

func adyenPaymentFailure(resultCode, refusalReason string) error {
	trimmedCode := strings.ToUpper(strings.TrimSpace(resultCode))
	trimmedReason := strings.TrimSpace(refusalReason)
	if trimmedReason == "" {
		trimmedReason = "payment was not authorized"
	}
	if trimmedCode == "" {
		return fmt.Errorf("adyen payment failed: %s", trimmedReason)
	}
	return fmt.Errorf("adyen payment failed with result %s: %s", trimmedCode, trimmedReason)
}

func isAdyenFailureResultCode(resultCode string) bool {
	switch strings.ToUpper(strings.TrimSpace(resultCode)) {
	case "REFUSED", "ERROR", "CANCELLED":
		return true
	default:
		return false
	}
}

func isAdyenAcceptedResultCode(resultCode string) bool {
	switch strings.ToUpper(strings.TrimSpace(resultCode)) {
	case "AUTHORISED", "RECEIVED", "PENDING", "PARTIALLYAUTHORISED":
		return true
	default:
		return false
	}
}

func payloadOrNil(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}
	return payload
}

func resolveAdyenReturnURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}

	fallback := strings.TrimSpace(os.Getenv("ADYEN_RETURN_URL_DEFAULT"))
	if fallback != "" {
		return fallback
	}

	return adyenDefaultReturnURL
}

func adyenIdempotencyKey(parts ...string) string {
	joined := strings.Join(filterNonEmpty(parts), "-")
	sanitized := sanitizeAdyenToken(joined)
	if sanitized == "" {
		return ""
	}
	if len(sanitized) > 64 {
		sanitized = sanitized[:64]
	}
	return sanitized
}

func adyenReference(parts ...string) string {
	reference := adyenIdempotencyKey(parts...)
	if reference == "" {
		reference = fmt.Sprintf("adyen-%d", time.Now().UnixNano())
	}
	if len(reference) > 80 {
		reference = reference[:80]
	}
	return reference
}

func filterNonEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

func sanitizeAdyenToken(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(value))
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}

		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}
