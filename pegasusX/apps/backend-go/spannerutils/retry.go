package spannerutils

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

const (
	defaultMaxAttempts = 3
	defaultBaseBackoff = 25 * time.Millisecond
	defaultMaxBackoff  = 500 * time.Millisecond
)

// RunReadWriteTransaction executes fn inside a Spanner RW transaction, retrying
// transient Aborted and Unavailable errors with bounded exponential backoff and full jitter
// to prevent retry amplification storms under high concurrency.
func RunReadWriteTransaction(ctx context.Context, client *spanner.Client, fn func(context.Context, *spanner.ReadWriteTransaction) error) error {
	if client == nil {
		return fmt.Errorf("spanner: nil client")
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
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
		// Full jitter: random duration between 0 and backoff to break contention synchronization
		jitter := time.Duration(rng.Int63n(int64(backoff) + 1))
		timer := time.NewTimer(jitter)
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

// RunReadOnlyTransaction manages the complete lifecycle of a ReadOnlyTransaction,
// injecting it into ctx and guaranteeing that Close() is called on all exit paths.
func RunReadOnlyTransaction(ctx context.Context, client *spanner.Client, fn func(context.Context, *spanner.ReadOnlyTransaction) error) error {
	if client == nil {
		return fmt.Errorf("spanner: nil client")
	}
	txn := client.ReadOnlyTransaction()
	defer txn.Close()
	return fn(WithReadOnlyTransaction(ctx, txn), txn)
}
