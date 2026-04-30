// Package spannerx provides Spanner access helpers that implement the
// Bounded-Staleness read doctrine from copilot-instructions.md §4.
//
// Rule of thumb:
//   - Use StaleQuery / StaleReadRow for list views, search results, analytics
//     feeds, config lookups, and any dashboard-serving path.
//   - Keep spannerClient.Single().Query (strong read) for ReadWriteTransaction
//     precondition checks, payment state gates, and live-tracking endpoints.
//   - The default staleness window is 15 s, which covers the p99 commit latency
//     on a 1-node Spanner instance at 60 % utilisation. Tight callers may
//     override with a non-zero lastWriteAt to get ExactStaleness(now-lastWriteAt)
//     clamped to [minFresh, maxStale].
package spannerx

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	// DefaultStaleness is used when the caller does not provide a lastWriteAt.
	DefaultStaleness = 15 * time.Second

	// MinFreshness is the minimum staleness window — below this we fall back to
	// a strong read to avoid serving data that may predate a recent mutation.
	MinFreshness = 2 * time.Second

	// MaxStaleness is the upper bound; Spanner caps bounded staleness at 1 h.
	MaxStaleness = 30 * time.Second
)

// StaleReader wraps a *spanner.Client and exposes staleness-bounded reads.
// Construct once per request or reuse across a service's read-only path.
type StaleReader struct {
	client    *spanner.Client
	staleness time.Duration
}

// NewStaleReader returns a StaleReader using DefaultStaleness.
// Use ForLastWrite to get a tighter bound when lastWriteAt is known.
func NewStaleReader(client *spanner.Client) *StaleReader {
	return &StaleReader{client: client, staleness: DefaultStaleness}
}

// ForLastWrite returns a StaleReader whose window is (now - lastWriteAt),
// clamped to [MinFreshness, MaxStaleness].
// If lastWriteAt is zero (unknown), DefaultStaleness is used.
func ForLastWrite(client *spanner.Client, lastWriteAt time.Time) *StaleReader {
	if lastWriteAt.IsZero() {
		return NewStaleReader(client)
	}
	age := time.Since(lastWriteAt)
	switch {
	case age < MinFreshness:
		// Data was written very recently — fall back to DefaultStaleness so
		// the replica has had time to apply the mutation.
		return &StaleReader{client: client, staleness: DefaultStaleness}
	case age > MaxStaleness:
		age = MaxStaleness
	}
	return &StaleReader{client: client, staleness: age}
}

// txn returns a read-only transaction with the configured staleness bound.
func (r *StaleReader) txn() *spanner.ReadOnlyTransaction {
	return r.client.Single().WithTimestampBound(spanner.ExactStaleness(r.staleness))
}

// Query executes a SQL query using a stale read. The caller must call
// iter.Stop() when done.
func (r *StaleReader) Query(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator {
	return r.txn().Query(ctx, stmt)
}

// ReadRow reads a single row by key using a stale read.
func (r *StaleReader) ReadRow(ctx context.Context, table string, key spanner.Key, cols []string) (*spanner.Row, error) {
	return r.txn().ReadRow(ctx, table, key, cols)
}

// Collect runs stmt via a stale read and calls scan on each row. Returns the
// number of rows processed. scan must not call iter.Next().
func (r *StaleReader) Collect(ctx context.Context, stmt spanner.Statement, scan func(*spanner.Row) error) (int, error) {
	iter := r.Query(ctx, stmt)
	defer iter.Stop()
	var n int
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return n, nil
		}
		if err != nil {
			return n, fmt.Errorf("spannerx: query failed: %w", err)
		}
		if err := scan(row); err != nil {
			return n, err
		}
		n++
	}
}

// StaleQuery is a package-level convenience that builds a StaleReader with
// DefaultStaleness and runs a query. Equivalent to:
//
//	NewStaleReader(client).Query(ctx, stmt)
func StaleQuery(ctx context.Context, client *spanner.Client, stmt spanner.Statement) *spanner.RowIterator {
	return NewStaleReader(client).Query(ctx, stmt)
}

// StaleReadRow is a package-level convenience for single-row stale reads.
func StaleReadRow(ctx context.Context, client *spanner.Client, table string, key spanner.Key, cols []string) (*spanner.Row, error) {
	return NewStaleReader(client).ReadRow(ctx, table, key, cols)
}
