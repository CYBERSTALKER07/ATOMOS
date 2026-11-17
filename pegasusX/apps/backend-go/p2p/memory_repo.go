package p2p

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type memTxnBuffer struct {
	events []outbox.Event
}

func (b *memTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

// MemoryOrderCase allows unit tests to seed O2C Orders + OrderStatusTransitions into MemoryRepository.
type MemoryOrderCase struct {
	TenantID    string
	OrderID     string
	CreatedAt   time.Time
	Transitions []MemoryOrderTransition
}

// MemoryOrderTransition mirrors one OrderStatusTransitions row in MemoryRepository.
type MemoryOrderTransition struct {
	PreviousStatus string
	NewStatus      string
	CreatedAt      time.Time
}

// MemoryRepository implements Repository in memory for fast deterministic unit tests.
type MemoryRepository struct {
	mu             sync.RWMutex
	templates      map[string]map[string]ProcessTemplate // tenantID -> templateID -> tpl
	instances      map[string]map[string]ProcessInstance // tenantID -> instanceID -> inst
	pos            map[string]map[string]PurchaseOrder   // tenantID -> poID -> PO
	receipts       map[string]map[string]GoodsReceipt    // tenantID -> receiptID -> receipt
	vendorInvoices map[string]map[string]VendorInvoice   // tenantID -> invID -> inv
	inboundESF     map[string]map[string]InboundESFDoc   // tenantID -> esfID -> doc
	orderCases     map[string][]MemoryOrderCase          // tenantID -> cases
	outboxEvents   []outbox.Event
}

// NewMemoryRepository creates an isolated in-memory P2P repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		templates:      make(map[string]map[string]ProcessTemplate),
		instances:      make(map[string]map[string]ProcessInstance),
		pos:            make(map[string]map[string]PurchaseOrder),
		receipts:       make(map[string]map[string]GoodsReceipt),
		vendorInvoices: make(map[string]map[string]VendorInvoice),
		inboundESF:     make(map[string]map[string]InboundESFDoc),
		orderCases:     make(map[string][]MemoryOrderCase),
	}
}

// SeedInboundESF registers a mock inbound ESF document for 3-way match unit tests.
func (m *MemoryRepository) SeedInboundESF(doc InboundESFDoc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inboundESF[doc.EntityID] == nil {
		m.inboundESF[doc.EntityID] = make(map[string]InboundESFDoc)
	}
	m.inboundESF[doc.EntityID][doc.EsfID] = doc
}

// SeedOrderCase registers O2C order transitions for process mining unit tests.
func (m *MemoryRepository) SeedOrderCase(c MemoryOrderCase) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orderCases[c.TenantID] = append(m.orderCases[c.TenantID], c)
}

// OutboxEvents returns a copy of buffered outbox events captured in memory.
func (m *MemoryRepository) OutboxEvents() []outbox.Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]outbox.Event, len(m.outboxEvents))
	copy(out, m.outboxEvents)
	return out
}

func (m *MemoryRepository) ListTemplates(_ context.Context, tenantID, domain string) ([]ProcessTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var custom []ProcessTemplate
	for _, tpl := range m.templates[tenantID] {
		custom = append(custom, tpl)
	}
	return MergeTemplates(tenantID, custom, domain), nil
}

func (m *MemoryRepository) SaveTemplate(_ context.Context, tpl ProcessTemplate) (ProcessTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.templates[tpl.TenantID] == nil {
		m.templates[tpl.TenantID] = make(map[string]ProcessTemplate)
	}
	for _, existing := range m.templates[tpl.TenantID] {
		if strings.EqualFold(existing.Code, tpl.Code) && existing.TemplateID != tpl.TemplateID {
			return ProcessTemplate{}, fmt.Errorf("%w: template code %s already exists", ErrConflict, tpl.Code)
		}
	}
	m.templates[tpl.TenantID][tpl.TemplateID] = tpl
	return tpl, nil
}

func (m *MemoryRepository) ListInstances(_ context.Context, f ListFilter) ([]ProcessInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []ProcessInstance
	for _, inst := range m.instances[f.TenantID] {
		if f.EntityID != "" && inst.EntityID != f.EntityID {
			continue
		}
		if f.ObjectType != "" && !strings.EqualFold(inst.ObjectType, f.ObjectType) {
			continue
		}
		if f.ObjectID != "" && inst.ObjectID != f.ObjectID {
			continue
		}
		if f.Status != "" && !strings.EqualFold(inst.Status, f.Status) {
			continue
		}
		out = append(out, inst)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	limit := clampLimit(f.Limit)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepository) UpsertInstance(_ context.Context, in StartOrAdvanceInstanceInput, tpl ProcessTemplate, now time.Time) (ProcessInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.instances[in.TenantID] == nil {
		m.instances[in.TenantID] = make(map[string]ProcessInstance)
	}
	inst, found := m.findInstanceLocked(in.TenantID, in.InstanceID, in.ObjectType, in.ObjectID)
	if found && in.NextStep != "" {
		return m.advanceInstanceLocked(inst, tpl, in.NextStep, in.ActorID, in.Reason, in.Complete, now), nil
	}
	if found && in.NextStep == "" {
		return inst, nil
	}

	firstStep := coalesce(in.CurrentStep, in.NextStep)
	if firstStep == "" && len(tpl.Steps) > 0 {
		firstStep = tpl.Steps[0].Code
	}
	if firstStep == "" {
		firstStep = "CREATED"
	}
	instID := strings.TrimSpace(in.InstanceID)
	if instID == "" {
		instID = "pinst_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	}
	inst = ProcessInstance{
		TenantID:     in.TenantID,
		InstanceID:   instID,
		TemplateID:   tpl.TemplateID,
		TemplateCode: tpl.Code,
		EntityID:     strings.TrimSpace(in.EntityID),
		ObjectType:   strings.TrimSpace(in.ObjectType),
		ObjectID:     strings.TrimSpace(in.ObjectID),
		CurrentStep:  strings.ToUpper(firstStep),
		Status:       InstanceStatusActive,
		Transitions: []ProcessTransition{
			{FromStep: "START", ToStep: strings.ToUpper(firstStep), ActorID: in.ActorID, Reason: coalesce(in.Reason, "started"), OccurredAt: now},
		},
		StartedAt: now,
		UpdatedAt: now,
	}
	if in.Complete || (len(tpl.Steps) > 0 && strings.EqualFold(inst.CurrentStep, tpl.Steps[len(tpl.Steps)-1].Code)) {
		inst.Status = InstanceStatusCompleted
		doneAt := now
		inst.CompletedAt = &doneAt
	}
	m.instances[in.TenantID][inst.InstanceID] = inst
	return inst, nil
}

func (m *MemoryRepository) findInstanceLocked(tenantID, instanceID, objectType, objectID string) (ProcessInstance, bool) {
	if instanceID != "" {
		inst, ok := m.instances[tenantID][instanceID]
		return inst, ok
	}
	var latest ProcessInstance
	var found bool
	for _, inst := range m.instances[tenantID] {
		if strings.EqualFold(inst.ObjectType, objectType) && inst.ObjectID == objectID {
			if !found || inst.StartedAt.After(latest.StartedAt) {
				latest = inst
				found = true
			}
		}
	}
	return latest, found
}

func (m *MemoryRepository) advanceInstanceLocked(inst ProcessInstance, tpl ProcessTemplate, nextStep, actorID, reason string, complete bool, now time.Time) ProcessInstance {
	nextUpper := strings.ToUpper(strings.TrimSpace(nextStep))
	if nextUpper == "" {
		return inst
	}
	if !strings.EqualFold(inst.CurrentStep, nextUpper) {
		inst.Transitions = append(inst.Transitions, ProcessTransition{
			FromStep:   inst.CurrentStep,
			ToStep:     nextUpper,
			ActorID:    actorID,
			Reason:     reason,
			OccurredAt: now,
		})
		inst.CurrentStep = nextUpper
	}
	if complete || (len(tpl.Steps) > 0 && strings.EqualFold(nextUpper, tpl.Steps[len(tpl.Steps)-1].Code)) {
		inst.Status = InstanceStatusCompleted
		doneAt := now
		inst.CompletedAt = &doneAt
	}
	inst.UpdatedAt = now
	if m.instances[inst.TenantID] == nil {
		m.instances[inst.TenantID] = make(map[string]ProcessInstance)
	}
	m.instances[inst.TenantID][inst.InstanceID] = inst
	return inst
}

func (m *MemoryRepository) CreatePurchaseOrder(ctx context.Context, po PurchaseOrder, tpl ProcessTemplate, hooks TxnHooks) (PurchaseOrder, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pos[po.TenantID] == nil {
		m.pos[po.TenantID] = make(map[string]PurchaseOrder)
	}
	if po.IdempotencyKey != "" {
		for _, existing := range m.pos[po.TenantID] {
			if existing.IdempotencyKey == po.IdempotencyKey {
				return clonePO(existing), true, nil
			}
		}
	}
	buf := &memTxnBuffer{}
	if _, err := runNonSpannerGLAndAudit(ctx, nil, buf, hooks); err != nil {
		return PurchaseOrder{}, false, err
	}
	if hooks.OutboxEmit != nil {
		if err := hooks.OutboxEmit(ctx, buf, ""); err != nil {
			return PurchaseOrder{}, false, err
		}
	}

	m.pos[po.TenantID][po.PoID] = clonePO(po)
	if tpl.TemplateID != "" {
		if m.instances[po.TenantID] == nil {
			m.instances[po.TenantID] = make(map[string]ProcessInstance)
		}
		instID := "pinst_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
		transitions := []ProcessTransition{
			{FromStep: "START", ToStep: "PO_CREATED", ActorID: po.CreatedBy, Reason: "po.created", OccurredAt: po.CreatedAt},
		}
		curr := "PO_CREATED"
		if po.Status == POStatusApproved {
			transitions = append(transitions, ProcessTransition{
				FromStep: "PO_CREATED", ToStep: "PO_APPROVED", ActorID: po.CreatedBy, Reason: "auto_approved_below_threshold", OccurredAt: po.CreatedAt,
			})
			curr = "PO_APPROVED"
		}
		m.instances[po.TenantID][instID] = ProcessInstance{
			TenantID:     po.TenantID,
			InstanceID:   instID,
			TemplateID:   tpl.TemplateID,
			TemplateCode: tpl.Code,
			EntityID:     po.EntityID,
			ObjectType:   AggregatePurchaseOrder,
			ObjectID:     po.PoID,
			CurrentStep:  curr,
			Status:       InstanceStatusActive,
			Transitions:  transitions,
			StartedAt:    po.CreatedAt,
			UpdatedAt:    po.CreatedAt,
		}
	}
	m.outboxEvents = append(m.outboxEvents, buf.events...)
	return clonePO(po), false, nil
}

func (m *MemoryRepository) ApprovePurchaseOrder(ctx context.Context, tenantID, poID, actorID, approvalID string, now time.Time, hooks TxnHooks) (PurchaseOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	po, ok := m.pos[tenantID][poID]
	if !ok {
		return PurchaseOrder{}, ErrNotFound
	}
	if po.CreatedBy != "" && strings.EqualFold(po.CreatedBy, actorID) {
		return PurchaseOrder{}, ErrSoDViolation
	}
	if po.Status != POStatusPendingApproval {
		return PurchaseOrder{}, fmt.Errorf("%w: purchase order status is %s", ErrConflict, po.Status)
	}
	if _, err := runNonSpannerGLAndAudit(ctx, nil, nil, hooks); err != nil {
		return PurchaseOrder{}, err
	}
	po.Status = POStatusApproved
	if approvalID != "" {
		po.ApprovalRequestID = approvalID
	}
	po.UpdatedAt = now
	m.pos[tenantID][poID] = clonePO(po)
	if inst, found := m.findInstanceLocked(tenantID, "", AggregatePurchaseOrder, poID); found {
		m.advanceInstanceLocked(inst, ProcessTemplate{}, "PO_APPROVED", actorID, "po.approved", false, now)
	}
	return clonePO(po), nil
}

func (m *MemoryRepository) FindPurchaseOrderByIdempotency(_ context.Context, tenantID, idempotencyKey string) (PurchaseOrder, bool, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return PurchaseOrder{}, false, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, po := range m.pos[tenantID] {
		if po.IdempotencyKey == idempotencyKey {
			return clonePO(po), true, nil
		}
	}
	return PurchaseOrder{}, false, nil
}

func (m *MemoryRepository) GetPurchaseOrder(_ context.Context, tenantID, poID string) (PurchaseOrder, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	po, ok := m.pos[tenantID][poID]
	if !ok {
		return PurchaseOrder{}, false, nil
	}
	return clonePO(po), true, nil
}

func (m *MemoryRepository) ListPurchaseOrders(_ context.Context, f ListFilter) ([]PurchaseOrder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []PurchaseOrder
	for _, po := range m.pos[f.TenantID] {
		if f.EntityID != "" && po.EntityID != f.EntityID {
			continue
		}
		if f.VendorID != "" && po.VendorID != f.VendorID {
			continue
		}
		if f.Status != "" && !strings.EqualFold(po.Status, f.Status) {
			continue
		}
		out = append(out, clonePO(po))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	limit := clampLimit(f.Limit)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepository) PostGoodsReceipt(ctx context.Context, receipt GoodsReceipt, hooks TxnHooks) (GoodsReceipt, PurchaseOrder, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.receipts[receipt.TenantID] == nil {
		m.receipts[receipt.TenantID] = make(map[string]GoodsReceipt)
	}
	if receipt.IdempotencyKey != "" {
		for _, ex := range m.receipts[receipt.TenantID] {
			if ex.IdempotencyKey == receipt.IdempotencyKey {
				po := m.pos[receipt.TenantID][ex.PoID]
				return ex, clonePO(po), true, nil
			}
		}
	}

	po, ok := m.pos[receipt.TenantID][receipt.PoID]
	if !ok {
		return GoodsReceipt{}, PurchaseOrder{}, false, ErrNotFound
	}
	if po.Status == POStatusPendingApproval || po.Status == POStatusCancelled || po.Status == POStatusClosed {
		return GoodsReceipt{}, PurchaseOrder{}, false, fmt.Errorf("%w: status=%s", ErrPONotApproved, po.Status)
	}

	resolvedLines, updatedPO, totalNet, err := applyReceiptLinesToPO(po, receipt.Lines, receipt.ReceivedAt)
	if err != nil {
		return GoodsReceipt{}, PurchaseOrder{}, false, err
	}
	receipt.Lines = resolvedLines
	receipt.TotalReceivedMinor = totalNet
	if receipt.EntityID == "" {
		receipt.EntityID = po.EntityID
	}

	buf := &memTxnBuffer{}
	if hooks.GLInput != nil {
		hooks.GLInput.EntityID = coalesce(hooks.GLInput.EntityID, receipt.EntityID)
		for i := range hooks.GLInput.Lines {
			hooks.GLInput.Lines[i].AmountMinor = totalNet
			if hooks.GLInput.Lines[i].AccountCode == AccountGRNI {
				hooks.GLInput.Lines[i].PartyType = "VENDOR"
				hooks.GLInput.Lines[i].PartyID = po.VendorID
			}
		}
	}
	journalEntryID, err := runNonSpannerGLAndAudit(ctx, nil, buf, hooks)
	if err != nil {
		return GoodsReceipt{}, PurchaseOrder{}, false, err
	}
	receipt.JournalEntryID = journalEntryID

	if hooks.OutboxEmit != nil {
		if err := hooks.OutboxEmit(ctx, buf, journalEntryID); err != nil {
			return GoodsReceipt{}, PurchaseOrder{}, false, err
		}
	}

	m.pos[receipt.TenantID][po.PoID] = clonePO(updatedPO)
	m.receipts[receipt.TenantID][receipt.ReceiptID] = receipt
	if inst, found := m.findInstanceLocked(receipt.TenantID, "", AggregatePurchaseOrder, po.PoID); found {
		m.advanceInstanceLocked(inst, ProcessTemplate{}, "GOODS_RECEIVED", receipt.ReceivedBy, "receipt.posted", false, receipt.ReceivedAt)
	}
	m.outboxEvents = append(m.outboxEvents, buf.events...)
	return receipt, clonePO(updatedPO), false, nil
}

func (m *MemoryRepository) FindGoodsReceiptByIdempotency(_ context.Context, tenantID, idempotencyKey string) (GoodsReceipt, bool, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return GoodsReceipt{}, false, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.receipts[tenantID] {
		if r.IdempotencyKey == idempotencyKey {
			return r, true, nil
		}
	}
	return GoodsReceipt{}, false, nil
}

func (m *MemoryRepository) ListGoodsReceipts(_ context.Context, f ListFilter) ([]GoodsReceipt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []GoodsReceipt
	for _, r := range m.receipts[f.TenantID] {
		if f.PoID != "" && r.PoID != f.PoID {
			continue
		}
		if f.EntityID != "" && r.EntityID != f.EntityID {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ReceivedAt.After(out[j].ReceivedAt) })
	limit := clampLimit(f.Limit)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepository) CreateVendorInvoice(ctx context.Context, inv VendorInvoice, hooks TxnHooks) (VendorInvoice, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.vendorInvoices[inv.TenantID] == nil {
		m.vendorInvoices[inv.TenantID] = make(map[string]VendorInvoice)
	}
	if inv.IdempotencyKey != "" {
		for _, ex := range m.vendorInvoices[inv.TenantID] {
			if ex.IdempotencyKey == inv.IdempotencyKey {
				return ex, true, nil
			}
		}
	}
	po, ok := m.pos[inv.TenantID][inv.PoID]
	if !ok {
		return VendorInvoice{}, false, fmt.Errorf("%w: purchase order %s not found", ErrNotFound, inv.PoID)
	}
	if inv.ReceiptID != "" {
		rec, ok := m.receipts[inv.TenantID][inv.ReceiptID]
		if !ok || rec.PoID != inv.PoID {
			return VendorInvoice{}, false, fmt.Errorf("%w: goods receipt %s not found for po %s", ErrNotFound, inv.ReceiptID, inv.PoID)
		}
	}
	if inv.VendorID == "" {
		inv.VendorID = po.VendorID
	}
	if inv.EntityID == "" {
		inv.EntityID = po.EntityID
	}
	if inv.Currency == "" {
		inv.Currency = po.Currency
	}

	if _, err := runNonSpannerGLAndAudit(ctx, nil, nil, hooks); err != nil {
		return VendorInvoice{}, false, err
	}

	m.vendorInvoices[inv.TenantID][inv.VendorInvoiceID] = inv
	if inst, found := m.findInstanceLocked(inv.TenantID, "", AggregatePurchaseOrder, po.PoID); found {
		m.advanceInstanceLocked(inst, ProcessTemplate{}, "INVOICE_RECEIVED", inv.CreatedBy, "vendor_invoice.created", false, inv.CreatedAt)
	}
	return inv, false, nil
}

func (m *MemoryRepository) FindVendorInvoiceByIdempotency(_ context.Context, tenantID, idempotencyKey string) (VendorInvoice, bool, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return VendorInvoice{}, false, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, inv := range m.vendorInvoices[tenantID] {
		if inv.IdempotencyKey == idempotencyKey {
			return inv, true, nil
		}
	}
	return VendorInvoice{}, false, nil
}

func (m *MemoryRepository) GetVendorInvoice(_ context.Context, tenantID, vendorInvoiceID string) (VendorInvoice, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inv, ok := m.vendorInvoices[tenantID][vendorInvoiceID]
	return inv, ok, nil
}

func (m *MemoryRepository) ListVendorInvoices(_ context.Context, f ListFilter) ([]VendorInvoice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []VendorInvoice
	for _, inv := range m.vendorInvoices[f.TenantID] {
		if f.PoID != "" && inv.PoID != f.PoID {
			continue
		}
		if f.EntityID != "" && inv.EntityID != f.EntityID {
			continue
		}
		if f.MatchStatus != "" && !strings.EqualFold(inv.MatchStatus, f.MatchStatus) {
			continue
		}
		out = append(out, inv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	limit := clampLimit(f.Limit)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepository) GetInboundESF(_ context.Context, tenantID, esfDocID string) (InboundESFDoc, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	doc, ok := m.inboundESF[tenantID][esfDocID]
	return doc, ok, nil
}

func (m *MemoryRepository) ApplyThreeWayMatch(ctx context.Context, tenantID, vendorInvoiceID string, update VendorInvoice, matchOK bool, now time.Time, hooks TxnHooks) (VendorInvoice, *ProcessInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.vendorInvoices[tenantID][vendorInvoiceID]
	if !ok {
		return VendorInvoice{}, nil, ErrNotFound
	}
	if existing.MatchStatus == MatchStatusMatched3Way {
		return VendorInvoice{}, nil, fmt.Errorf("%w: vendor invoice %s is already MATCHED_3WAY", ErrConflict, vendorInvoiceID)
	}

	buf := &memTxnBuffer{}
	journalEntryID, err := runNonSpannerGLAndAudit(ctx, nil, buf, hooks)
	if err != nil {
		return VendorInvoice{}, nil, err
	}
	if matchOK {
		update.JournalEntryID = journalEntryID
	}
	if hooks.OutboxEmit != nil {
		if err := hooks.OutboxEmit(ctx, buf, journalEntryID); err != nil {
			return VendorInvoice{}, nil, err
		}
	}
	update.UpdatedAt = now
	m.vendorInvoices[tenantID][vendorInvoiceID] = update

	var instPtr *ProcessInstance
	if matchOK {
		if inst, found := m.findInstanceLocked(tenantID, "", AggregatePurchaseOrder, update.PoID); found {
			updated := m.advanceInstanceLocked(inst, ProcessTemplate{}, "MATCHED_3WAY", hooks.ActorID, "three_way_match.completed", true, now)
			instPtr = &updated
		}
	}
	m.outboxEvents = append(m.outboxEvents, buf.events...)
	return update, instPtr, nil
}

func (m *MemoryRepository) CollectMiningSamples(_ context.Context, f MiningFilter, templates []ProcessTemplate) ([]MiningSample, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var samples []MiningSample

	// 1. OrderStatusTransitions + Orders (O2C)
	for _, oc := range m.orderCases[f.TenantID] {
		if f.Domain != "" && !strings.EqualFold(f.Domain, DomainO2C) {
			continue
		}
		trs := append([]MemoryOrderTransition(nil), oc.Transitions...)
		sort.Slice(trs, func(i, j int) bool { return trs[i].CreatedAt.Before(trs[j].CreatedAt) })
		prevTS := oc.CreatedAt
		for _, tr := range trs {
			if f.From != nil && tr.CreatedAt.Before(*f.From) {
				prevTS = tr.CreatedAt
				continue
			}
			if f.To != nil && tr.CreatedAt.After(*f.To) {
				continue
			}
			from := normalizeStepName(tr.PreviousStatus)
			to := normalizeStepName(tr.NewStatus)
			dur := int64(tr.CreatedAt.Sub(prevTS).Seconds())
			if dur < 0 {
				dur = 0
			}
			samples = append(samples, MiningSample{
				CaseID:      oc.OrderID,
				Domain:      DomainO2C,
				FromStep:    from,
				ToStep:      to,
				DurationSec: dur,
				SLATarget:   LookupStepSLASeconds(templates, DomainO2C, to),
			})
			prevTS = tr.CreatedAt
		}
	}

	// 2. ProcessInstances
	coveredPOs := make(map[string]struct{})
	for _, inst := range m.instances[f.TenantID] {
		if f.EntityID != "" && inst.EntityID != f.EntityID {
			continue
		}
		if strings.EqualFold(inst.ObjectType, AggregatePurchaseOrder) {
			coveredPOs[inst.ObjectID] = struct{}{}
		}
		dom := inferDomainFromTemplate(templates, inst.TemplateID, inst.TemplateCode, inst.ObjectType)
		if f.Domain != "" && !strings.EqualFold(f.Domain, dom) {
			continue
		}
		prevTS := inst.StartedAt
		for _, tr := range inst.Transitions {
			if strings.EqualFold(tr.FromStep, "START") {
				prevTS = tr.OccurredAt
				continue
			}
			if f.From != nil && tr.OccurredAt.Before(*f.From) {
				prevTS = tr.OccurredAt
				continue
			}
			if f.To != nil && tr.OccurredAt.After(*f.To) {
				continue
			}
			dur := int64(tr.OccurredAt.Sub(prevTS).Seconds())
			if dur < 0 {
				dur = 0
			}
			samples = append(samples, MiningSample{
				CaseID:      inst.ObjectID,
				Domain:      dom,
				FromStep:    strings.ToUpper(tr.FromStep),
				ToStep:      strings.ToUpper(tr.ToStep),
				DurationSec: dur,
				SLATarget:   LookupStepSLASeconds(templates, dom, tr.ToStep),
			})
			prevTS = tr.OccurredAt
		}
	}

	// 3. Direct PurchaseOrders -> GoodsReceipts -> VendorInvoices not already covered by ProcessInstances.
	if f.Domain == "" || strings.EqualFold(f.Domain, DomainP2P) {
		for _, po := range m.pos[f.TenantID] {
			if _, ok := coveredPOs[po.PoID]; ok {
				continue
			}
			if f.EntityID != "" && po.EntityID != f.EntityID {
				continue
			}
			var earliestRec *GoodsReceipt
			for _, rec := range m.receipts[f.TenantID] {
				if rec.PoID == po.PoID && (earliestRec == nil || rec.ReceivedAt.Before(earliestRec.ReceivedAt)) {
					rcp := rec
					earliestRec = &rcp
				}
			}
			if earliestRec != nil {
				dur := int64(earliestRec.ReceivedAt.Sub(po.CreatedAt).Seconds())
				if dur < 0 {
					dur = 0
				}
				samples = append(samples, MiningSample{
					CaseID:      po.PoID,
					Domain:      DomainP2P,
					FromStep:    "PO_CREATED",
					ToStep:      "GOODS_RECEIVED",
					DurationSec: dur,
					SLATarget:   LookupStepSLASeconds(templates, DomainP2P, "GOODS_RECEIVED"),
				})
				for _, inv := range m.vendorInvoices[f.TenantID] {
					if inv.PoID == po.PoID && inv.MatchStatus == MatchStatusMatched3Way {
						mdur := int64(inv.UpdatedAt.Sub(earliestRec.ReceivedAt).Seconds())
						if mdur < 0 {
							mdur = 0
						}
						samples = append(samples, MiningSample{
							CaseID:      po.PoID,
							Domain:      DomainP2P,
							FromStep:    "GOODS_RECEIVED",
							ToStep:      "MATCHED_3WAY",
							DurationSec: mdur,
							SLATarget:   LookupStepSLASeconds(templates, DomainP2P, "MATCHED_3WAY"),
						})
						break
					}
				}
			}
		}
	}

	return samples, nil
}

func inferDomainFromTemplate(templates []ProcessTemplate, templateID, templateCode, objectType string) string {
	for _, tpl := range templates {
		if (templateID != "" && tpl.TemplateID == templateID) || (templateCode != "" && strings.EqualFold(tpl.Code, templateCode)) {
			return tpl.Domain
		}
	}
	switch strings.ToUpper(strings.TrimSpace(objectType)) {
	case "PURCHASEORDER", "PO", "VENDORINVOICE":
		return DomainP2P
	case "RETURN", "RTV":
		return DomainReturns
	default:
		return DomainO2C
	}
}

func applyReceiptLinesToPO(po PurchaseOrder, reqLines []GoodsReceiptLine, now time.Time) ([]GoodsReceiptLine, PurchaseOrder, int64, error) {
	if len(reqLines) == 0 {
		return nil, PurchaseOrder{}, 0, fmt.Errorf("%w: at least one receipt line is required", ErrInvalidInput)
	}
	updated := clonePO(po)
	var outLines []GoodsReceiptLine
	var totalNet int64

	for _, rl := range reqLines {
		if rl.QtyReceived <= 0 {
			return nil, PurchaseOrder{}, 0, fmt.Errorf("%w: qty_received must be > 0", ErrInvalidInput)
		}
		idx := -1
		for i, pl := range updated.Lines {
			if rl.LineNum > 0 && pl.LineNum == rl.LineNum {
				idx = i
				break
			}
			if rl.LineNum == 0 && strings.EqualFold(pl.Sku, strings.TrimSpace(rl.Sku)) {
				idx = i
				break
			}
		}
		if idx < 0 {
			return nil, PurchaseOrder{}, 0, fmt.Errorf("%w: po line not found for sku=%q line_num=%d", ErrNotFound, rl.Sku, rl.LineNum)
		}
		pl := &updated.Lines[idx]
		if pl.QtyReceived+rl.QtyReceived > pl.QtyOrdered {
			return nil, PurchaseOrder{}, 0, fmt.Errorf("%w: sku=%s ordered=%d already_received=%d requested=%d",
				ErrQtyExceeded, pl.Sku, pl.QtyOrdered, pl.QtyReceived, rl.QtyReceived)
		}
		lineNet, err := ComputeNetMinor(rl.QtyReceived, pl.UnitCostMinor)
		if err != nil {
			return nil, PurchaseOrder{}, 0, err
		}
		pl.QtyReceived += rl.QtyReceived
		totalNet += lineNet
		outLines = append(outLines, GoodsReceiptLine{
			LineNum:       pl.LineNum,
			Sku:           pl.Sku,
			QtyReceived:   rl.QtyReceived,
			UnitCostMinor: pl.UnitCostMinor,
			VatBps:        pl.VatBps,
			LineNetMinor:  lineNet,
		})
	}

	allReceived := len(updated.Lines) > 0
	for _, pl := range updated.Lines {
		if pl.QtyReceived < pl.QtyOrdered {
			allReceived = false
			break
		}
	}
	if allReceived {
		updated.Status = POStatusReceived
	} else {
		updated.Status = POStatusPartialReceived
	}
	updated.UpdatedAt = now
	return outLines, updated, totalNet, nil
}

func clonePO(po PurchaseOrder) PurchaseOrder {
	cp := po
	if len(po.Lines) > 0 {
		cp.Lines = make([]PurchaseOrderLine, len(po.Lines))
		copy(cp.Lines, po.Lines)
	}
	return cp
}
