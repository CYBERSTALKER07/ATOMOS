package fiscal

import "context"

// UzbekistanStrategy implements FiscalStrategy for Uzbekistan (UZ).
type UzbekistanStrategy struct{}

// NewUzbekistanStrategy creates a new instance of UzbekistanStrategy.
func NewUzbekistanStrategy() *UzbekistanStrategy {
	return &UzbekistanStrategy{}
}

func (s *UzbekistanStrategy) FormatDocument(ctx context.Context, docID string) ([]byte, error) {
	return []byte("UZ_FORMATTED_" + docID), nil
}

func (s *UzbekistanStrategy) SubmitDocument(ctx context.Context, docID string, formattedDoc []byte) error {
	// Mock success
	return nil
}

func (s *UzbekistanStrategy) CancelDocument(ctx context.Context, docID string) error {
	// Mock success
	return nil
}
