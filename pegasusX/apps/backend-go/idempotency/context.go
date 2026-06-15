package idempotency

import "context"

type claimedKey struct{}

// WithClaimed marks the request context so handler-level Guard calls become no-ops.
// The HTTP middleware sets this after it acquires an idempotency key.
func WithClaimed(ctx context.Context) context.Context {
	return context.WithValue(ctx, claimedKey{}, true)
}

// Claimed reports whether global idempotency middleware already owns this request.
func Claimed(ctx context.Context) bool {
	claimed, _ := ctx.Value(claimedKey{}).(bool)
	return claimed
}
