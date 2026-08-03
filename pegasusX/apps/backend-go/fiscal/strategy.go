package fiscal

import "context"

// FiscalStrategy defines the interface for country-specific fiscal operations.
type FiscalStrategy interface {
	FormatDocument(ctx context.Context, docID string) ([]byte, error)
	SubmitDocument(ctx context.Context, docID string, formattedDoc []byte) error
	CancelDocument(ctx context.Context, docID string) error
}
