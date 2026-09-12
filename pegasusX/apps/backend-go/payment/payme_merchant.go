package payment

import (
	"sync"
	"time"
)

// paymeMerchantTx is the Merchant API transaction record Payme requires for
// Create/Perform/Cancel/Check/GetStatement. Replica-local until this adapter
// is wired to Spanner; routes are unmounted so production never hits it.
type paymeMerchantTx struct {
	PaymeID     string
	OrderID     string
	AmountMinor int64
	State       int
	CreateTime  int64
	PerformTime int64
	CancelTime  int64
	Reason      *int
	MerchantTx  string
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

func (m *paymeMerchantMemory) snapshot(id string) (paymeMerchantTx, bool) {
	if m == nil {
		return paymeMerchantTx{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	tx, ok := m.byID[id]
	if !ok || tx == nil {
		return paymeMerchantTx{}, false
	}
	return *tx, true
}

func (m *paymeMerchantMemory) snapshotByOrder(orderID string) (paymeMerchantTx, bool) {
	if m == nil {
		return paymeMerchantTx{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byOrder[orderID]
	if !ok {
		return paymeMerchantTx{}, false
	}
	tx, ok := m.byID[id]
	if !ok || tx == nil {
		return paymeMerchantTx{}, false
	}
	return *tx, true
}

func (m *paymeMerchantMemory) put(tx paymeMerchantTx) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := tx
	m.byID[tx.PaymeID] = &cp
	if tx.OrderID != "" {
		m.byOrder[tx.OrderID] = tx.PaymeID
	}
}

func (m *paymeMerchantMemory) statement(from, to int64) []paymeMerchantTx {
	if m == nil {
		return []paymeMerchantTx{}
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
	return out
}

func (s *Service) paymeNowMilli() int64 {
	if s != nil && s.now != nil {
		return s.now().UnixMilli()
	}
	return time.Now().UTC().UnixMilli()
}

func (s *Service) ensurePaymeTx() *paymeMerchantMemory {
	if s == nil {
		return newPaymeMerchantMemory()
	}
	if s.paymeTx == nil {
		s.paymeTx = newPaymeMerchantMemory()
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
