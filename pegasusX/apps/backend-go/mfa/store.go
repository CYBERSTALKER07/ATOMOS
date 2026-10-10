package mfa

import (
	"context"
	"sync"
	"time"
)

// Record is a platform-admin TOTP enrollment row.
type Record struct {
	Subject   string
	Secret    string
	Enabled   bool
	CreatedAt time.Time
	EnabledAt time.Time
}

// Repository persists MFA enrollments by actor subject.
type Repository interface {
	Get(ctx context.Context, subject string) (Record, bool, error)
	Upsert(ctx context.Context, row Record) error
}

// MemoryRepository is an in-memory store for tests / ALLOW_MEMORY_FALLBACK.
type MemoryRepository struct {
	mu   sync.Mutex
	rows map[string]Record
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{rows: map[string]Record{}}
}

func (r *MemoryRepository) Get(_ context.Context, subject string) (Record, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[subject]
	return row, ok, nil
}

func (r *MemoryRepository) Upsert(_ context.Context, row Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[row.Subject] = row
	return nil
}
