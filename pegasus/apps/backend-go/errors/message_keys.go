package errors

// ── i18n Message Keys ───────────────────────────────────────────────────────
// Dot-notation keys mapped 1:1 with native string tables.
// iOS:     Localizable.strings → "payment.error.otp_mismatch" = "..."
// Android: strings.xml         → <string name="payment_error_otp_mismatch">...</string>
// Admin:   i18n JSON           → { "payment.error.otp_mismatch": "..." }

const (
	MsgKeyBadRequest         = "error.bad_request"
	MsgKeyNotFound           = "error.not_found"
	MsgKeyConflict           = "error.conflict"
	MsgKeyMethodNotAllowed   = "error.method_not_allowed"
	MsgKeyServiceUnavailable = "error.service_unavailable"
	MsgKeyOTPMismatch        = "payment.error.otp_mismatch"
	MsgKeyOTPExpired         = "payment.error.otp_expired"
	MsgKey3DSRequired        = "payment.error.3ds_required"
	MsgKey3DSFailed          = "payment.error.3ds_failed"
	MsgKeyCardExpired        = "payment.error.card_expired"
	MsgKeyCardDeclined       = "payment.error.card_declined"
	MsgKeyInsufficientFunds  = "payment.error.insufficient_funds"
	MsgKeyCardInvalid        = "payment.error.card_invalid"
	MsgKeyPaymentTimeout     = "payment.error.timeout"
	MsgKeyPaymentGeneric     = "payment.error.generic"
	MsgKeyAuthPending        = "payment.status.auth_pending"
	MsgKeyAuthSuccess        = "payment.status.auth_success"
	MsgKeySettlementOK       = "payment.status.settlement_ok"
	MsgKeySettlementDelayed  = "payment.status.settlement_delayed"
	MsgKeyGatewayDown        = "payment.error.gateway_unavailable"

	MsgKeyOrderNotFound      = "order.error.not_found"
	MsgKeyOrderStateConflict = "order.error.state_conflict"
	MsgKeyOrderCancelDenied  = "order.error.cancel_denied"
	MsgKeyOrderAlreadyDone   = "order.error.already_processed"

	MsgKeyInternalError = "error.internal"
	MsgKeyRateLimited   = "error.rate_limited"
	MsgKeyUnauthorized  = "error.unauthorized"
	MsgKeyForbidden     = "error.forbidden"
)
