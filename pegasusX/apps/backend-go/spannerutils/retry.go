package spannerutils

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

const (
	defaultMaxAttempts = 5
	defaultBaseBackoff = 20 * time.Millisecond
	defaultMaxBackoff  = 500 * time.Millisecond
)

// RunReadWriteTransaction executes fn inside a Spanner RW transaction, retrying
// transient Aborted and Unavailable errors with bounded exponential backoff.
func RunReadWriteTransaction(ctx context.Context, client *spanner.Client, fn func(context.Context, *spanner.ReadWriteTransaction) error) error {
	if client == nil {
		return fmt.Errorf("spanner: nil client")
	}
	backoff := defaultBaseBackoff
	var lastErr error
	for attempt := 1; attempt <= defaultMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, err := client.ReadWriteTransaction(ctx, fn)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetryableSpannerErr(err) || attempt == defaultMaxAttempts {
			return err
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		backoff *= 2
		if backoff > defaultMaxBackoff {
			backoff = defaultMaxBackoff
		}
	}
	return lastErr
}

func isRetryableSpannerErr(err error) bool {
	switch spanner.ErrCode(err) {
	case codes.Aborted, codes.Unavailable:
		return true
	default:
		return false
	}
}

type ctxKeyReadOnlyTxn struct{}

// WithReadOnlyTransaction returns a context that carries a ReadOnlyTransaction.
func WithReadOnlyTransaction(ctx context.Context, txn *spanner.ReadOnlyTransaction) context.Context {
	return context.WithValue(ctx, ctxKeyReadOnlyTxn{}, txn)
}

// ReadOnlyTxnFromContext extracts a ReadOnlyTransaction from the context, if present.
func ReadOnlyTxnFromContext(ctx context.Context) *spanner.ReadOnlyTransaction {
	if txn, ok := ctx.Value(ctxKeyReadOnlyTxn{}).(*spanner.ReadOnlyTransaction); ok {
		return txn
	}
	return nil
}
