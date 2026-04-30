package errors

// ── Client Recovery Actions ─────────────────────────────────────────────────
// Clients use these to trigger specific UX flows (re-display OTP pad,
// redirect to 3DS page, show "contact support" banner, etc.).

const (
	ActionReEnterOTP     = "RE_ENTER_OTP"
	ActionRedirect3DS    = "REDIRECT_3DS"
	ActionRetry          = "RETRY"
	ActionContactSupport = "CONTACT_SUPPORT"
	ActionRefreshCard    = "REFRESH_CARD"
	ActionSelectCard     = "SELECT_CARD"
	ActionNone           = ""
)
