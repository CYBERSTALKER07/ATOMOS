package esf

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryOutboxEvent records an emitted ESF domain event inside MemoryRepository.
type MemoryOutboxEvent struct {
	EventType string
	EntityID  string
	EsfID     string
	Status    string
}

// MemoryGLReversal records a GL VAT reversal triggered on esf.rejected in MemoryRepository.
type MemoryGLReversal struct {
	EntryID        string
	EntityID       string
	EsfID          string
	OrderID        string
	PeriodID       string
	VatMinor       int64
	IdempotencyKey string
}

// MemoryProductCompliance models SKU compliance metadata in MemoryRepository for ListMissingIKPU tests.
type MemoryProductCompliance struct {
	ProductID   string
	SupplierID  string
	Name        string
	CategoryID  string
	Barcode     string
	IKPUCode    string
	PackageCode string
	MarkingCode string
	IsActive    bool
}

type memGLVAT struct {
	currency       string
	outputVatMinor int64
	inputVatMinor  int64
}

// MemoryRepository is a thread-safe in-memory Repository implementation for unit tests.
type MemoryRepository struct {
	mu          sync.RWMutex
	docs        map[string]map[string]EsfDocument             // entityID -> esfID -> doc
	history     map[string]map[string][]EsfStatusHistoryEntry // entityID -> esfID -> history
	orderSrcs   map[string]OrderFiscalSource                  // orderID -> src
	glVAT       map[string]map[string]memGLVAT                // entityID -> periodID -> gl vat
	skus        map[string]map[string]MemoryProductCompliance // supplierID -> productID -> compliance
	events      []MemoryOutboxEvent
	glReversals []MemoryGLReversal
	glEntrySeq  int64
}

// NewMemoryRepository creates an initialized MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		docs:      make(map[string]map[string]EsfDocument),
		history:   make(map[string]map[string][]EsfStatusHistoryEntry),
		orderSrcs: make(map[string]OrderFiscalSource),
		glVAT:     make(map[string]map[string]memGLVAT),
		skus:      make(map[string]map[string]MemoryProductCompliance),
	}
}

// SeedOrderFiscalSource stores an order fiscal snapshot source for CreateOutbound tests.
func (r *MemoryRepository) SeedOrderFiscalSource(src OrderFiscalSource) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if src.Currency == "" {
		src.Currency = "UZS"
	}
	if src.CreatedAt.IsZero() {
		src.CreatedAt = time.Now().UTC()
	}
	r.orderSrcs[strings.TrimSpace(src.OrderID)] = src
}

// SeedGLVATBalance stores expected GL TAX:VAT:OUTPUT and TAX:VAT:INPUT totals for VAT summary tests.
func (r *MemoryRepository) SeedGLVATBalance(entityID, periodID, currency string, outputVatMinor, inputVatMinor int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	eid := strings.TrimSpace(entityID)
	pid := strings.TrimSpace(periodID)
	cur := strings.TrimSpace(currency)
	if cur == "" {
		cur = "UZS"
	}
	if r.glVAT[eid] == nil {
		r.glVAT[eid] = make(map[string]memGLVAT)
	}
	r.glVAT[eid][pid] = memGLVAT{
		currency:       cur,
		outputVatMinor: outputVatMinor,
		inputVatMinor:  inputVatMinor,
	}
}

// SeedProductCompliance seeds a SKU record for ListMissingIKPU tests.
func (r *MemoryRepository) SeedProductCompliance(rec MemoryProductCompliance) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sid := strings.TrimSpace(rec.SupplierID)
	if r.skus[sid] == nil {
		r.skus[sid] = make(map[string]MemoryProductCompliance)
	}
	r.skus[sid][strings.TrimSpace(rec.ProductID)] = rec
}

// EmittedEvents returns a snapshot of recorded domain events.
func (r *MemoryRepository) EmittedEvents() []MemoryOutboxEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MemoryOutboxEvent, len(r.events))
	copy(out, r.events)
	return out
}

// GLReversals returns a snapshot of GL VAT reversals executed on esf.rejected.
func (r *MemoryRepository) GLReversals() []MemoryGLReversal {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MemoryGLReversal, len(r.glReversals))
	copy(out, r.glReversals)
	return out
}

func (r *MemoryRepository) LoadOrderFiscalSource(_ context.Context, entityID, orderID string) (OrderFiscalSource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	src, ok := r.orderSrcs[strings.TrimSpace(orderID)]
	if !ok {
		return OrderFiscalSource{}, ErrOrderNotFound
	}
	if entityID != "" && src.SupplierID != "" && src.SupplierID != entityID {
		return OrderFiscalSource{}, ErrOrderForbidden
	}
	return src, nil
}

func (r *MemoryRepository) SaveDocumentWithHistory(
	_ context.Context,
	doc EsfDocument,
	hist EsfStatusHistoryEntry,
	eventType string,
	glVatReversal *VATReversalSpec,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.docs[doc.EntityID] == nil {
		r.docs[doc.EntityID] = make(map[string]EsfDocument)
	}
	if _, exists := r.docs[doc.EntityID][doc.EsfID]; exists {
		return ErrAlreadyExists
	}

	if r.glVAT[doc.EntityID] == nil {
		r.glVAT[doc.EntityID] = make(map[string]memGLVAT)
	}

	if glVatReversal != nil && glVatReversal.VatMinor > 0 {
		r.applyGLVATReversalLocked(doc.EntityID, doc.PeriodID, glVatReversal)
	}

	r.docs[doc.EntityID][doc.EsfID] = doc
	if r.history[doc.EntityID] == nil {
		r.history[doc.EntityID] = make(map[string][]EsfStatusHistoryEntry)
	}
	r.history[doc.EntityID][doc.EsfID] = append(r.history[doc.EntityID][doc.EsfID], hist)

	if eventType != "" {
		r.events = append(r.events, MemoryOutboxEvent{
			EventType: eventType,
			EntityID:  doc.EntityID,
			EsfID:     doc.EsfID,
			Status:    doc.Status,
		})
	}
	return nil
}

func (r *MemoryRepository) UpdateDocumentStatusWithHistory(
	_ context.Context,
	entityID, esfID, expectedPrevStatus, nextStatus string,
	updates StatusUpdateFields,
	hist EsfStatusHistoryEntry,
	eventType string,
	glVatReversal *VATReversalSpec,
) (EsfDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entDocs := r.docs[entityID]
	if entDocs == nil {
		return EsfDocument{}, ErrNotFound
	}
	doc, ok := entDocs[esfID]
	if !ok {
		return EsfDocument{}, ErrNotFound
	}
	if expectedPrevStatus != "" && doc.Status != expectedPrevStatus {
		return EsfDocument{}, fmt.Errorf("%w: status is %s, expected %s", ErrInvalidTransition, doc.Status, expectedPrevStatus)
	}

	doc.Status = nextStatus
	if updates.Operator != "" {
		doc.Operator = updates.Operator
	}
	if updates.OperatorRef != "" {
		doc.OperatorRef = updates.OperatorRef
	}
	if updates.SignedPayloadURI != "" {
		doc.SignedPayloadURI = updates.SignedPayloadURI
	}
	if updates.PayloadHash != "" {
		doc.PayloadHash = updates.PayloadHash
	}
	if updates.FailureReason != nil {
		doc.FailureReason = *updates.FailureReason
	}
	if updates.IssuedAt != nil {
		ts := *updates.IssuedAt
		doc.IssuedAt = &ts
	}
	if updates.AcceptedAt != nil {
		ts := *updates.AcceptedAt
		doc.AcceptedAt = &ts
	}
	if !updates.UpdatedAt.IsZero() {
		doc.UpdatedAt = updates.UpdatedAt
	}

	if glVatReversal != nil && glVatReversal.VatMinor > 0 {
		r.applyGLVATReversalLocked(entityID, doc.PeriodID, glVatReversal)
	}

	entDocs[esfID] = doc
	if r.history[entityID] == nil {
		r.history[entityID] = make(map[string][]EsfStatusHistoryEntry)
	}
	r.history[entityID][esfID] = append(r.history[entityID][esfID], hist)

	if eventType != "" {
		r.events = append(r.events, MemoryOutboxEvent{
			EventType: eventType,
			EntityID:  doc.EntityID,
			EsfID:     doc.EsfID,
			Status:    doc.Status,
		})
	}
	return doc, nil
}

func (r *MemoryRepository) applyGLVATReversalLocked(entityID, periodID string, spec *VATReversalSpec) {
	if r.glVAT[entityID] == nil {
		r.glVAT[entityID] = make(map[string]memGLVAT)
	}
	for _, existing := range r.glReversals {
		if existing.EntityID == entityID && existing.IdempotencyKey == spec.IdempotencyKey {
			if spec.GlEntryIDOut != nil {
				*spec.GlEntryIDOut = existing.EntryID
			}
			return
		}
	}
	r.glEntrySeq++
	entryID := fmt.Sprintf("mem-gl-vat-rev-%d", r.glEntrySeq)
	cur := spec.Currency
	if cur == "" {
		cur = "UZS"
	}
	bal := r.glVAT[entityID][periodID]
	if bal.currency == "" {
		bal.currency = cur
	}
	bal.outputVatMinor -= spec.VatMinor
	r.glVAT[entityID][periodID] = bal

	r.glReversals = append(r.glReversals, MemoryGLReversal{
		EntryID:        entryID,
		EntityID:       entityID,
		EsfID:          spec.EsfID,
		OrderID:        spec.OrderID,
		PeriodID:       periodID,
		VatMinor:       spec.VatMinor,
		IdempotencyKey: spec.IdempotencyKey,
	})
	if spec.GlEntryIDOut != nil {
		*spec.GlEntryIDOut = entryID
	}
}

func (r *MemoryRepository) GetDocument(_ context.Context, entityID, esfID string) (EsfDocument, []EsfStatusHistoryEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doc, ok := r.docs[entityID][esfID]
	if !ok {
		return EsfDocument{}, nil, ErrNotFound
	}
	histSrc := r.history[entityID][esfID]
	hist := make([]EsfStatusHistoryEntry, len(histSrc))
	copy(hist, histSrc)
	sort.Slice(hist, func(i, j int) bool {
		return hist[i].CreatedAt.Before(hist[j].CreatedAt)
	})
	return doc, hist, nil
}

func (r *MemoryRepository) FindLatestByOrderOrOperatorRef(_ context.Context, entityID, orderID, operatorRef string) (*EsfDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []EsfDocument
	for eid, m := range r.docs {
		if entityID != "" && eid != entityID {
			continue
		}
		for _, doc := range m {
			if (orderID != "" && doc.OrderID == orderID) || (operatorRef != "" && doc.OperatorRef == operatorRef) {
				all = append(all, doc)
			}
		}
	}
	if len(all) == 0 {
		return nil, ErrNotFound
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})
	d := all[0]
	return &d, nil
}

func (r *MemoryRepository) ListDocuments(_ context.Context, filter ListFilter) ([]EsfDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var out []EsfDocument
	for _, doc := range r.docs[filter.EntityID] {
		if filter.Direction != "" && !strings.EqualFold(doc.Direction, filter.Direction) {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(doc.Status, filter.Status) {
			continue
		}
		if filter.PeriodID != "" && doc.PeriodID != filter.PeriodID {
			continue
		}
		if filter.OrderID != "" && doc.OrderID != filter.OrderID {
			continue
		}
		out = append(out, doc)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	if out == nil {
		out = []EsfDocument{}
	}
	return out, nil
}

func (r *MemoryRepository) GetVATSummary(_ context.Context, entityID, periodID string) (VATSummaryReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	report := VATSummaryReport{
		EntityID: entityID,
		PeriodID: periodID,
		Currency: "UZS",
		ByRate:   []VATRateBreakdown{},
	}
	byRate := make(map[int64]*VATRateBreakdown)
	ensureRate := func(bps int64) *VATRateBreakdown {
		if entry, ok := byRate[bps]; ok {
			return entry
		}
		entry := &VATRateBreakdown{VatRateBps: bps}
		byRate[bps] = entry
		return entry
	}

	for _, doc := range r.docs[entityID] {
		if doc.PeriodID != periodID {
			continue
		}
		if doc.Status == StatusRejected {
			report.RejectedCount++
			continue
		}
		if doc.Status != StatusSubmitted && doc.Status != StatusAccepted {
			continue
		}
		baseMinor := doc.TotalMinor - doc.VatMinor
		if baseMinor < 0 {
			baseMinor = 0
		}
		rb := ensureRate(doc.VatRateBps)
		rb.DocCount++
		if doc.Direction == DirectionOutbound {
			report.OutboundCount++
			report.EsfOutputVatMinor += doc.VatMinor
			rb.OutboundBaseMinor += baseMinor
			rb.OutboundVatMinor += doc.VatMinor
		} else if doc.Direction == DirectionInbound {
			report.InboundCount++
			report.EsfInputVatMinor += doc.VatMinor
			rb.InboundBaseMinor += baseMinor
			rb.InboundVatMinor += doc.VatMinor
		}
	}

	if glBal, ok := r.glVAT[entityID][periodID]; ok {
		if glBal.currency != "" {
			report.Currency = glBal.currency
		}
		report.GlOutputVatMinor = glBal.outputVatMinor
		report.GlInputVatMinor = glBal.inputVatMinor
	}

	report.EsfNetVatMinor = report.EsfOutputVatMinor - report.EsfInputVatMinor
	report.GlNetVatMinor = report.GlOutputVatMinor - report.GlInputVatMinor
	report.OutputDiffMinor = report.EsfOutputVatMinor - report.GlOutputVatMinor
	report.InputDiffMinor = report.EsfInputVatMinor - report.GlInputVatMinor
	report.DiffMinor = report.EsfNetVatMinor - report.GlNetVatMinor
	report.Reconciled = report.OutputDiffMinor == 0 && report.InputDiffMinor == 0 && report.DiffMinor == 0

	var rates []int64
	for bps := range byRate {
		rates = append(rates, bps)
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i] < rates[j] })
	for _, bps := range rates {
		report.ByRate = append(report.ByRate, *byRate[bps])
	}
	return report, nil
}

func (r *MemoryRepository) ListMissingIKPU(_ context.Context, supplierID string) ([]MissingIKPUItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []MissingIKPUItem
	for _, rec := range r.skus[supplierID] {
		if !rec.IsActive {
			continue
		}
		missing := detectMissingComplianceFields(rec.IKPUCode, rec.PackageCode, rec.MarkingCode, rec.Barcode)
		if len(missing) == 0 {
			continue
		}
		out = append(out, MissingIKPUItem{
			ProductID:     rec.ProductID,
			Name:          rec.Name,
			CategoryID:    rec.CategoryID,
			Barcode:       rec.Barcode,
			IKPUCode:      rec.IKPUCode,
			PackageCode:   rec.PackageCode,
			MarkingCode:   rec.MarkingCode,
			MissingFields: missing,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	if out == nil {
		out = []MissingIKPUItem{}
	}
	return out, nil
}
