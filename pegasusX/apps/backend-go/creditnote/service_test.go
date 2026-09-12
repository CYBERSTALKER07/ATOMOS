package creditnote

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	savedNotes []CreditNote
	savedTasks []ReverseLogisticsTask
}

func (m *mockRepo) SaveCreditNote(ctx context.Context, cn CreditNote, eventType string) error {
	for i, c := range m.savedNotes {
		if c.CreditNoteId == cn.CreditNoteId {
			m.savedNotes[i] = cn
			return nil
		}
	}
	m.savedNotes = append(m.savedNotes, cn)
	return nil
}

func (m *mockRepo) GetCreditNote(ctx context.Context, id string) (*CreditNote, error) {
	for _, n := range m.savedNotes {
		if n.CreditNoteId == id {
			return &n, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) ListCreditNotesByOrder(ctx context.Context, orderId string) ([]CreditNote, error) {
	return nil, nil
}

func (m *mockRepo) ListBySupplier(ctx context.Context, supplierID string, status CreditNoteStatus, limit int) ([]CreditNote, error) {
	return nil, nil
}

func (m *mockRepo) GetDeliveredOrderLines(ctx context.Context, orderId string) ([]CreditNoteLine, error) {
	return []CreditNoteLine{
		{
			OrderLineId:    "mock-line-1",
			Sku:            "sku-1",
			Qty:            2,
			UnitNetMinor:   100,
			LineNetMinor:   200,
			LineVatMinor:   20,
			LineGrossMinor: 220,
		},
	}, nil
}

func (m *mockRepo) OrderOwnedBySupplier(ctx context.Context, orderID, supplierID string) (bool, error) {
	return true, nil
}

func (m *mockRepo) GetClaimOrder(ctx context.Context, claimID string) (string, int64, bool, error) {
	return "order-claim", 5000, true, nil
}

func (m *mockRepo) SaveReverseLogisticsTask(ctx context.Context, task ReverseLogisticsTask, eventType string) error {
	m.savedTasks = append(m.savedTasks, task)
	return nil
}

func (m *mockRepo) GetReverseLogisticsTask(ctx context.Context, taskID string) (*ReverseLogisticsTask, error) {
	for _, t := range m.savedTasks {
		if t.TaskId == taskID {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) ListReverseLogisticsTasks(ctx context.Context, warehouseID, status string, limit int) ([]ReverseLogisticsTask, error) {
	return m.savedTasks, nil
}

func (m *mockRepo) ReceiveReverseLogisticsTask(ctx context.Context, taskID string, warehouseID string, receivedJSON []byte, actor string) error {
	return nil
}

func TestCreateFromBuyerReject_HappyPath(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	cn, err := svc.CreateFromBuyerReject(context.Background(), "order-1", "driver-1")
	assert.NoError(t, err)
	assert.NotNil(t, cn)
	assert.Equal(t, CreditNoteTypeBuyerReject, cn.Type)
	assert.Equal(t, CreditNoteStatusDraft, cn.Status)
	assert.Equal(t, "BUYER_REJECTED_DELIVERY", cn.ReasonCode)

	assert.Len(t, repo.savedNotes, 1)
}

func TestIssue_HappyPath(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	cn, _ := svc.CreateFromBuyerReject(context.Background(), "order-1", "driver-1")

	err := svc.Issue(context.Background(), cn.CreditNoteId, "manager")
	assert.NoError(t, err)

	updated, _ := repo.GetCreditNote(context.Background(), cn.CreditNoteId)
	assert.Equal(t, CreditNoteStatusFiscalPending, updated.Status)
	assert.NotNil(t, updated.IssuedAt)

	assert.Len(t, repo.savedTasks, 1)
	assert.Equal(t, ReverseTaskStatusOpen, repo.savedTasks[0].Status)
	assert.Equal(t, cn.CreditNoteId, repo.savedTasks[0].CreditNoteId)
}

func TestIssue_NotFound(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	err := svc.Issue(context.Background(), "missing-id", "manager")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCreateManual_HappyPath(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	req := CreateManualCreditNoteRequest{
		OrderId:    "order-man",
		ReasonCode: "MISTAKE",
		ReasonText: "wrong item packed",
		Lines: []CreditNoteLineInput{
			{OrderLineId: "mock-line-1", Qty: 1},
		},
	}

	cn, err := svc.CreateManual(context.Background(), req, "user-1")
	assert.NoError(t, err)
	assert.Equal(t, CreditNoteTypeManual, cn.Type)
	assert.Equal(t, "MISTAKE", cn.ReasonCode)
	assert.Equal(t, "wrong item packed", *cn.ReasonText)
}

func TestCreateFromClaim_HappyPath(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	cn, err := svc.CreateFromClaim(context.Background(), "clm-1", "system")
	assert.NoError(t, err)
	assert.Equal(t, "order-claim", cn.OrderId)
	assert.Equal(t, int64(5000), cn.TotalGrossMinor)
}
