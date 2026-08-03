package pricing

import (
	"context"
	"testing"
	"time"
)

type mockRepo struct {
	price int64
	err   error
}

func (m *mockRepo) GetActiveUnitPriceMinor(ctx context.Context, supplierId, sku string, date time.Time) (int64, error) {
	return m.price, m.err
}

func TestResolvePrice(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{price: 1500, err: nil}
		svc := NewService(repo)

		price, err := svc.ResolvePrice(context.Background(), "s1", "sku1", time.Now())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if price != 1500 {
			t.Errorf("expected 1500, got %d", price)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockRepo{price: 0, err: ErrPriceNotFound}
		svc := NewService(repo)

		_, err := svc.ResolvePrice(context.Background(), "s1", "sku1", time.Now())
		if err != ErrPriceNotFound {
			t.Fatalf("expected ErrPriceNotFound, got %v", err)
		}
	})
}
