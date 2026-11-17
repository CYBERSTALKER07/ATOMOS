package spannerutils

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsRetryableSpannerErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		err     error
		retry   bool
	}{
		{"aborted", status.Error(codes.Aborted, "txn aborted"), true},
		{"unavailable", status.Error(codes.Unavailable, "temp"), true},
		{"invalid", status.Error(codes.InvalidArgument, "bad"), false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isRetryableSpannerErr(tc.err)
			if got != tc.retry {
				t.Fatalf("isRetryableSpannerErr(%v) = %v, want %v", tc.err, got, tc.retry)
			}
		})
	}
}

func TestRunReadWriteTransactionNilClient(t *testing.T) {
	t.Parallel()
	err := RunReadWriteTransaction(t.Context(), nil, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return errors.New("should not run")
	})
	if err == nil {
		t.Fatal("expected error for nil client, got nil")
	}
	if err.Error() != "spanner: nil client" {
		t.Fatalf("expected 'spanner: nil client', got: %v", err)
	}
}

func TestRunChunkedTransactionNilClient(t *testing.T) {
	t.Parallel()
	err := RunChunkedTransaction(t.Context(), nil, []string{"item1"}, 1, func(ctx context.Context, txn *spanner.ReadWriteTransaction, chunk []string) error {
		return errors.New("should not run")
	})
	if err == nil {
		t.Fatal("expected error for nil client in chunked transaction, got nil")
	}
	if err.Error() != "spanner: nil client" {
		t.Fatalf("expected 'spanner: nil client', got: %v", err)
	}
}

func TestRunReadOnlyTransactionNilClient(t *testing.T) {
	t.Parallel()
	err := RunReadOnlyTransaction(t.Context(), nil, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		return nil
	})
	if err == nil || err.Error() != "spanner: nil client" {
		t.Fatalf("expected 'spanner: nil client', got %v", err)
	}
}

func TestReadOnlyTxnFromContext_Nil(t *testing.T) {
	t.Parallel()
	txn := ReadOnlyTxnFromContext(context.Background())
	if txn != nil {
		t.Fatalf("expected nil txn from empty context, got %v", txn)
	}
}
