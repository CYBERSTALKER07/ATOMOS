package payment

import (
	"context"
	"sync"
	"time"
)

const (
	paymeTargetTypeOrder        = "ORDER"
	paymeTargetTypeARAllocation = "AR_ALLOCATION"
)

// paymeMerchantTx is the Merchant API transaction record Payme requires for
// Create/Perform/Cancel/Check/GetStatement. Persisted in Spanner
// (PaymeMerchantTransactions) when a Spanner client is attached, with an
// in-memory store retained strictly for nil-DB unit tests.
type paymeMerchantTx struct {
	PaymeID     string
	OrderID     string
	TargetType  string
	AmountMinor int64
	State       int
	CreateTime  int64
	PerformTime int64
	CancelTime  int64
	Reason      *int
	MerchantTx  string
}

type paymeMerchantStore interface {
	snapshot(ctx context.Context, id string) (paymeMerchantTx, bool, error)
	snapshotByOrder(ctx context.Context, orderID string) (paymeMerchantTx, bool, error)
	put(ctx context.Context, tx paymeMerchantTx) error
	statement(ctx context.Context, from, to int64) ([]paymeMerchantTx, error)
}

func normalizePaymeTargetType(raw string) string {
	switch raw {
	case paymeTargetTypeARAllocation:
		return paymeTargetTypeARAllocation
	case "", paymeTargetTypeOrder:
		return paymeTargetTypeOrder
	default:
		return raw
	}
}

type paymeMerchantMemory struct {
	mu      sync.Mutex
	byID    map[string]*paymeMerchantTx
	byOrder map[string]string
}

func newPaymeMerchantMemory() *paymeMerchantMemory {
	return &paymeMerchantMemory{
		byID:    map[string]*paymeMerchantTx{},
		byOrder: map[string]string{},
	}
}

func (m *paymeMerchantMemory) snapshot(_ context.Context, id string) (paymeMerchantTx, bool, error) {
	if m == nil {
		return paymeMerchantTx{}, false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	tx, ok := m.byID[id]
	if !ok || tx == nil {
		return paymeMerchantTx{}, false, nil
	}
	return *tx, true, nil
}

func (m *paymeMerchantMemory) snapshotByOrder(_ context.Context, orderID string) (paymeMerchantTx, bool, error) {
	if m == nil {
		return paymeMerchantTx{}, false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byOrder[orderID]
	if !ok {
		return paymeMerchantTx{}, false, nil
	}
	tx, ok := m.byID[id]
	if !ok || tx == nil {
		return paymeMerchantTx{}, false, nil
	}
	return *tx, true, nil
}

func (m *paymeMerchantMemory) put(_ context.Context, tx paymeMerchantTx) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := tx
	cp.TargetType = normalizePaymeTargetType(cp.TargetType)
	m.byID[tx.PaymeID] = &cp
	if tx.OrderID != "" {
		m.byOrder[tx.OrderID] = tx.PaymeID
	}
	return nil
}

func (m *paymeMerchantMemory) statement(_ context.Context, from, to int64) ([]paymeMerchantTx, error) {
	if m == nil {
		return []paymeMerchantTx{}, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]paymeMerchantTx, 0, len(m.byID))
	for _, tx := range m.byID {
		if tx == nil {
			continue
		}
		if from > 0 && tx.CreateTime < from {
			continue
		}
		if to > 0 && tx.CreateTime > to {
			continue
		}
		out = append(out, *tx)
	}
	return out, nil
}

func (s *Service) paymeNowMilli() int64 {
	if s != nil && s.now != nil {
		return s.now().UnixMilli()
	}
	return time.Now().UTC().UnixMilli()
}

func (s *Service) ensurePaymeTx() paymeMerchantStore {
	if s == nil {
		return newPaymeMerchantMemory()
	}
	s.paymeTxMu.Lock()
	defer s.paymeTxMu.Unlock()
	if s.paymeTx == nil {
		if s.spannerClient != nil {
			s.paymeTx = newPaymeMerchantSpanner(s.spannerClient)
		} else {
			s.paymeTx = newPaymeMerchantMemory()
		}
	}
	return s.paymeTx
}

func paymeCreateResult(tx paymeMerchantTx) map[string]any {
	return map[string]any{
		"create_time": tx.CreateTime,
		"transaction": tx.MerchantTx,
		"state":       tx.State,
	}
}

func paymePerformResult(tx paymeMerchantTx) map[string]any {
	return map[string]any{
		"transaction":  tx.MerchantTx,
		"perform_time": tx.PerformTime,
		"state":        tx.State,
	}
}

func paymeCancelResult(tx paymeMerchantTx) map[string]any {
	return map[string]any{
		"transaction": tx.MerchantTx,
		"cancel_time": tx.CancelTime,
		"state":       tx.State,
	}
}

func paymeCheckResult(tx paymeMerchantTx) map[string]any {
	var reason any
	if tx.Reason != nil {
		reason = *tx.Reason
	}
	return map[string]any{
		"create_time":  tx.CreateTime,
		"perform_time": tx.PerformTime,
		"cancel_time":  tx.CancelTime,
		"transaction":  tx.MerchantTx,
		"state":        tx.State,
		"reason":       reason,
	}
}

func paymeStatementItem(tx paymeMerchantTx) map[string]any {
	var reason any
	if tx.Reason != nil {
		reason = *tx.Reason
	}
	account := map[string]string{"order_id": tx.OrderID}
	return map[string]any{
		"id":           tx.PaymeID,
		"time":         tx.CreateTime,
		"amount":       tx.AmountMinor,
		"account":      account,
		"create_time":  tx.CreateTime,
		"perform_time": tx.PerformTime,
		"cancel_time":  tx.CancelTime,
		"transaction":  tx.MerchantTx,
		"state":        tx.State,
		"reason":       reason,
	}
}
