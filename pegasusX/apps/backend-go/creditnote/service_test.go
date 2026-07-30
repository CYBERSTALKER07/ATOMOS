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

func (m *mockRepo) SaveCreditNote(ctx context.Context, cn CreditNote) error {
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

func (m *mockRepo) SaveReverseLogisticsTask(ctx context.Context, task ReverseLogisticsTask) error {
	m.savedTasks = append(m.savedTasks, task)
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
	}

	cn, err := svc.CreateManual(context.Background(), req, "user-1")
	assert.NoError(t, err)
	assert.Equal(t, CreditNoteTypeManual, cn.Type)
	assert.Equal(t, "MISTAKE", cn.ReasonCode)
	assert.Equal(t, "wrong item packed", *cn.ReasonText)
}
