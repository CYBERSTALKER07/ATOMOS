package cashrecon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type handlerTestMockRepo struct {
	driverRecons []CashReconciliation
}

func (m *handlerTestMockRepo) SaveReconciliation(ctx context.Context, cr CashReconciliation, eventType string) error {
	return nil
}
func (m *handlerTestMockRepo) GetReconciliation(ctx context.Context, id string) (*CashReconciliation, error) {
	return nil, nil
}
func (m *handlerTestMockRepo) ListReconciliationsByStatus(ctx context.Context, status ReconciliationStatus) ([]CashReconciliation, error) {
	return nil, nil
}
func (m *handlerTestMockRepo) ListByDriver(ctx context.Context, driverID string, shiftDate time.Time) ([]CashReconciliation, error) {
	return m.driverRecons, nil
}
func (m *handlerTestMockRepo) ListBySupplier(ctx context.Context, supplierID string, status ReconciliationStatus, limit int) ([]CashReconciliation, error) {
	return nil, nil
}
func (m *handlerTestMockRepo) ResolveDriverSupplierID(ctx context.Context, driverID string) (string, error) {
	return "sup-1", nil
}

func TestHandleDriverList_BlindCountMasking(t *testing.T) {
	now := time.Now().UTC()
	mockR := &handlerTestMockRepo{
		driverRecons: []CashReconciliation{
			{
				ReconciliationId:  "cr-pending-1",
				SupplierId:        "sup-1",
				DriverId:          "drv-123",
				ShiftDate:         now,
				ExpectedCashMinor: 45000000,
				DeclaredCashMinor: 0,
				DifferenceMinor:   45000000,
				Status:            ReconciliationStatusPending,
				CreatedAt:         now,
			},
			{
				ReconciliationId:  "cr-resolved-2",
				SupplierId:        "sup-1",
				DriverId:          "drv-123",
				ShiftDate:         now.AddDate(0, 0, -1),
				ExpectedCashMinor: 30000000,
				DeclaredCashMinor: 30000000,
				DifferenceMinor:   0,
				Status:            ReconciliationStatusAccepted,
				CreatedAt:         now.AddDate(0, 0, -1),
			},
		},
	}

	svc := NewService(mockR, &mockCash{expected: 45000000})
	h := &Handlers{Svc: svc}

	req := httptest.NewRequest(http.MethodGet, "/v1/driver/cash-reconciliations", nil)
	ctx := auth.WithClaims(req.Context(), auth.Claims{
		Subject: "drv-123",
		Role:    auth.RoleDriver,
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.HandleDriverList(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Reconciliations []wireReconciliation `json:"reconciliations"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Len(t, resp.Reconciliations, 2)

	// Invariant: Pending reconciliation MUST mask ExpectedCashMinor & DifferenceMinor to 0
	assert.Equal(t, "cr-pending-1", resp.Reconciliations[0].ReconciliationID)
	assert.Equal(t, int64(0), resp.Reconciliations[0].ExpectedCashMinor, "pending turn-in must not leak expected cash")
	assert.Equal(t, int64(0), resp.Reconciliations[0].DifferenceMinor, "pending turn-in must not leak difference")

	// Resolved reconciliation preserves verified amounts
	assert.Equal(t, "cr-resolved-2", resp.Reconciliations[1].ReconciliationID)
	assert.Equal(t, int64(30000000), resp.Reconciliations[1].ExpectedCashMinor)
	assert.Equal(t, int64(0), resp.Reconciliations[1].DifferenceMinor)
}
