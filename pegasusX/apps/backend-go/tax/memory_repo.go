package tax

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
)

type MemoryRepository struct{}

func NewMemoryRepository() *MemoryRepository { return &MemoryRepository{} }
func (m *MemoryRepository) GetActiveRegime(ctx context.Context, txn *spanner.ReadWriteTransaction, countryCode string, ts time.Time) (TaxRegimeVersion, bool, error) { return TaxRegimeVersion{}, false, nil }
func (m *MemoryRepository) GetRegime(ctx context.Context, id string) (TaxRegimeVersion, bool, error) { return TaxRegimeVersion{}, false, nil }
func (m *MemoryRepository) ListRegimes(ctx context.Context, countryCode string, limit int) ([]TaxRegimeVersion, error) { return nil, nil }
func (m *MemoryRepository) CreateRegime(ctx context.Context, regime TaxRegimeVersion) error { return nil }
func (m *MemoryRepository) InsertLineSnapshot(ctx context.Context, txn *spanner.ReadWriteTransaction, snapshot OrderLineFiscalSnapshot) error { return nil }
