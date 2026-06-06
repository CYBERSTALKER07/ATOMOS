package optimizercontract

// ErrorCode is the wire-stable identifier for solver failures. Both the client
// and the server map to / from these values; do not use raw strings.
type ErrorCode string

const (
	// ErrCodeVersion: SolveRequest.V did not match V.
	ErrCodeVersion ErrorCode = "VERSION_MISMATCH"

	// ErrCodeAuth: missing or wrong X-Internal-Api-Key header.
	ErrCodeAuth ErrorCode = "UNAUTHORIZED"

	// ErrCodeBadRequest: payload failed schema or semantic validation
	// (e.g. negative VolumeVU, malformed HH:MM window, empty stops).
	ErrCodeBadRequest ErrorCode = "BAD_REQUEST"

	// ErrCodeEmptyFleet: vehicles slice was empty.
	ErrCodeEmptyFleet ErrorCode = "EMPTY_FLEET"

	// ErrCodeTimeout: solver did not converge within its internal budget.
	// The client treats this as fallback-trigger.
	ErrCodeTimeout ErrorCode = "TIMEOUT"

	// ErrCodeInternal: unexpected solver-side panic / bug. Always paged.
	ErrCodeInternal ErrorCode = "INTERNAL"
)

// ErrorResponse is the body returned by the optimiser on any non-200 status.
// HTTP status codes used: 400 (BAD_REQUEST, VERSION_MISMATCH, EMPTY_FLEET),
// 401 (UNAUTHORIZED), 504 (TIMEOUT), 500 (INTERNAL).
type ErrorResponse struct {
	V       string    `json:"v"`
	TraceID string    `json:"trace_id"`
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}
