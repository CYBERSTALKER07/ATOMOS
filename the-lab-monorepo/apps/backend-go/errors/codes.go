package errors

// ── Payment Error Codes ─────────────────────────────────────────────────────
// Wire format: GP_*, CARD_*, AUTH_*, SETTLEMENT_*, PAY_*

const (
	CodeGPOTPInvalid      = "GP_OTP_INVALID"
	CodeGPOTPExpired      = "GP_OTP_EXPIRED"
	CodeGP3DSRequired     = "GP_3DS_REQUIRED"
	CodeGP3DSFailed       = "GP_3DS_FAILED"
	CodeGPAuthDeclined    = "GP_AUTH_DECLINED"
	CodeGPRecipientFailed = "GP_RECIPIENT_FAILED"

	CodeCardExpired           = "CARD_EXPIRED"
	CodeCardDeclined          = "CARD_DECLINED"
	CodeCardInsufficientFunds = "CARD_INSUFFICIENT_FUNDS"
	CodeCardTokenInvalid      = "CARD_TOKEN_INVALID"
	CodeCardNotFound          = "CARD_NOT_FOUND"

	CodeAuthPending       = "AUTH_PENDING"
	CodeAuthHoldFailed    = "AUTH_HOLD_FAILED"
	CodeAuthHoldExpired   = "AUTH_HOLD_EXPIRED"
	CodeAuthCaptureOK     = "AUTH_CAPTURE_OK"
	CodeAuthCaptureFailed = "AUTH_CAPTURE_FAILED"
	CodeAuthVoided        = "AUTH_VOIDED"

	CodeSettlementOK      = "SETTLEMENT_OK"
	CodeSettlementDelayed = "SETTLEMENT_DELAYED"
	CodeSettlementFailed  = "SETTLEMENT_FAILED"

	CodePaymentTimeout        = "PAYMENT_TIMEOUT"
	CodePaymentDuplicate      = "PAYMENT_DUPLICATE"
	CodePaymentAmountMismatch = "PAYMENT_AMOUNT_MISMATCH"
	CodeGatewayUnavailable    = "GATEWAY_UNAVAILABLE"
)

// ── Order Error Codes ───────────────────────────────────────────────────────

const (
	CodeOrderNotFound         = "ORDER_NOT_FOUND"
	CodeOrderStateConflict    = "ORDER_STATE_CONFLICT"
	CodeOrderVersionConflict  = "ORDER_VERSION_CONFLICT"
	CodeOrderFreezeLocked     = "ORDER_FREEZE_LOCKED"
	CodeOrderAlreadyProcessed = "ORDER_ALREADY_PROCESSED"
	CodeOrderCancelForbidden  = "ORDER_CANCEL_FORBIDDEN"
)

// ── Dispatch Error Codes ────────────────────────────────────────────────────

const (
	CodeDispatchNoDriver     = "DISPATCH_NO_DRIVER"
	CodeDispatchCapacityFull = "DISPATCH_CAPACITY_FULL"
	CodeDispatchLocked       = "DISPATCH_LOCKED"
)

// ── System / Tactical Codes ─────────────────────────────────────────────────
// For engineering dashboards, NOT shown to end users.

const (
	CodeSpannerTimeout      = "SPANNER_LOCK_TIMEOUT"
	CodeSpannerAborted      = "SPANNER_TXN_ABORTED"
	CodeKafkaBrokerDown     = "KAFKA_BROKER_UNREACHABLE"
	CodeKafkaProduceTimeout = "KAFKA_PRODUCE_TIMEOUT"
	CodeRedisUnavailable    = "REDIS_UNAVAILABLE"
	CodeCircuitOpen         = "CIRCUIT_BREAKER_OPEN"
	CodeRateLimited         = "RATE_LIMITED"
)
