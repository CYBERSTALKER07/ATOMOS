package cashrecon

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	saved []CashReconciliation
}

func (m *mockRepo) SaveReconciliation(ctx context.Context, cr CashReconciliation, eventType string) error {
	for i, c := range m.saved {
		if c.ReconciliationId == cr.ReconciliationId {
			m.saved[i] = cr
			return nil
		}
	}
	m.saved = append(m.saved, cr)
	return nil
}

func (m *mockRepo) GetReconciliation(ctx context.Context, id string) (*CashReconciliation, error) {
	for _, cr := range m.saved {
		if cr.ReconciliationId == id {
			return &cr, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) ListReconciliationsByStatus(ctx context.Context, status ReconciliationStatus) ([]CashReconciliation, error) {
	return nil, nil
}

func (m *mockRepo) ListByDriver(ctx context.Context, driverID string, shiftDate time.Time) ([]CashReconciliation, error) {
	return nil, nil
}

func (m *mockRepo) ListBySupplier(ctx context.Context, supplierID string, status ReconciliationStatus, limit int) ([]CashReconciliation, error) {
	return nil, nil
}

func (m *mockRepo) ResolveDriverSupplierID(ctx context.Context, driverID string) (string, error) {
	return "sup-1", nil
}

type mockCash struct {
	expected int64
}

func (m *mockCash) ComputeExpectedCashMinor(ctx context.Context, driverID string, shiftDate time.Time, routeID *string) (int64, error) {
	return m.expected, nil
}

func (m *mockCash) HasAcceptedReconciliation(ctx context.Context, driverID string, shiftDate time.Time) (bool, error) {
	for _, cr := range []CashReconciliation{} {
		_ = cr
	}
	return false, nil
}

func TestSubmitReconciliation_HappyPath_Match(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockCash{expected: 1000})

	req := SubmitReconciliationRequest{
		DriverId:          "d-1",
		DeclaredCashMinor: 1000,
	}

	cr, err := svc.SubmitReconciliation(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "sup-1", cr.SupplierId)
	assert.Equal(t, int64(1000), cr.ExpectedCashMinor)
	assert.Equal(t, int64(0), cr.DifferenceMinor)
	assert.Equal(t, ReconciliationStatusAccepted, cr.Status)
	assert.NotNil(t, cr.ResolvedAt)
	assert.Equal(t, "SYSTEM", *cr.ResolvedBy)
}

func TestSubmitReconciliation_Discrepancy(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockCash{expected: 1000})

	note := "Lost a 500 bill"
	req := SubmitReconciliationRequest{
		DriverId:          "d-1",
		DeclaredCashMinor: 500,
		DriverNote:        &note,
	}

	cr, err := svc.SubmitReconciliation(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, int64(-500), cr.DifferenceMinor)
	assert.Equal(t, ReconciliationStatusPending, cr.Status)
	assert.Nil(t, cr.ResolvedAt)
	assert.Equal(t, "Lost a 500 bill", *cr.DriverNote)
}

func TestAccept_HappyPath(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockCash{expected: 1000})

	req := SubmitReconciliationRequest{
		DriverId:          "d-1",
		DeclaredCashMinor: 500,
	}
	cr, _ := svc.SubmitReconciliation(context.Background(), req)

	err := svc.Accept(context.Background(), cr.ReconciliationId, "fin-1", "Verified")
	assert.NoError(t, err)

	updated, _ := repo.GetReconciliation(context.Background(), cr.ReconciliationId)
	assert.Equal(t, ReconciliationStatusAccepted, updated.Status)
	assert.Equal(t, "Verified", *updated.FinanceNote)
	assert.Equal(t, "fin-1", *updated.ResolvedBy)
}

func TestAccept_InvalidStatus(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockCash{expected: 1000})

	req := SubmitReconciliationRequest{
		DriverId:          "d-1",
		DeclaredCashMinor: 1000,
	}
	cr, _ := svc.SubmitReconciliation(context.Background(), req)

	err := svc.Accept(context.Background(), cr.ReconciliationId, "fin-1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot accept from status ACCEPTED")
}

func TestWriteOff_HappyPath(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockCash{expected: 1000})

	req := SubmitReconciliationRequest{
		DriverId:          "d-1",
		DeclaredCashMinor: 800,
	}
	cr, _ := svc.SubmitReconciliation(context.Background(), req)

	err := svc.WriteOff(context.Background(), cr.ReconciliationId, "fin-1", "Writing off short 200")
	assert.NoError(t, err)

	updated, _ := repo.GetReconciliation(context.Background(), cr.ReconciliationId)
	assert.Equal(t, ReconciliationStatusWriteOff, updated.Status)
}

func TestSubmitReconciliation_ExpectedMismatch(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockCash{expected: 1000})

	req := SubmitReconciliationRequest{
		DriverId:          "d-1",
		ExpectedCashMinor: 900,
		DeclaredCashMinor: 1000,
	}
	_, err := svc.SubmitReconciliation(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected_cash_mismatch")
}
