package e2ep0

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	adyenhmac "github.com/adyen/adyen-go-api-library/v21/src/hmacvalidator"
	adyenwebhook "github.com/adyen/adyen-go-api-library/v21/src/webhook"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalies"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalyroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvalroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auditroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap/memory"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectorroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectors"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltowerroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/esf"
	"github.com/pegasusx/pegasusx/apps/backend-go/esfroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/exportpackroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/exportpacks"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
	"github.com/pegasusx/pegasusx/apps/backend-go/factoryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/forecastroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/forecastv2"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/glroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/iam"
	"github.com/pegasusx/pegasusx/apps/backend-go/iamroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientity"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientityroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontologyroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/orderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2p"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2proutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/paymentroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/periodroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/periods"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposalroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/recon"
	"github.com/pegasusx/pegasusx/apps/backend-go/reconroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailerroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
	"github.com/pegasusx/pegasusx/apps/backend-go/roiroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/valuation"
	"github.com/pegasusx/pegasusx/apps/backend-go/valuationroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouseroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/webhookroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatif"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatifroutes"
)

const (
	DefaultJWTSecret = "e2e-p0-test-jwt-secret-very-secure-key-32bytes!!"
	DefaultJWTIssuer = "pegasusx-e2e-p0"
	DefaultAdyenHMAC = "44DAC79C4F1804F0F43382A4E5A8B068831F61F03526F3020A4697600504E11E"
	DefaultSupplierA = "sup-apple"
	DefaultSupplierB = "sup-banana"
)

// MemoryTxnBuffer captures outbox events in memory.
type MemoryTxnBuffer struct {
	mu     sync.Mutex
	Events []outbox.Event
}

func (b *MemoryTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Events = append(b.Events, e)
	return nil
}

// MemoryRetailerRepo implements retailer.Repository and kycReviewRepository.
type MemoryRetailerRepo struct {
	mu            sync.RWMutex
	retailers     map[string]retailer.Retailer
	byPhone       map[string]string // phone -> retailerID
	kycDocs       map[string]retailer.KycDocument
	kycReviewedBy map[string]string // docID -> reviewedBy adminID
	pricingRules  map[string]retailer.SupplierPricingRule
	tracking      map[string][]retailer.TrackingOrder
}

func NewMemoryRetailerRepo() *MemoryRetailerRepo {
	return &MemoryRetailerRepo{
		retailers:     make(map[string]retailer.Retailer),
		byPhone:       make(map[string]string),
		kycDocs:       make(map[string]retailer.KycDocument),
		kycReviewedBy: make(map[string]string),
		pricingRules:  make(map[string]retailer.SupplierPricingRule),
		tracking:      make(map[string][]retailer.TrackingOrder),
	}
}

func (r *MemoryRetailerRepo) CreateRetailer(ctx context.Context, ret retailer.Retailer, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ret.RetailerID == "" {
		ret.RetailerID = "ret-" + uuid.NewString()[:8]
	}
	r.retailers[ret.RetailerID] = ret
	if ret.Phone != "" {
		r.byPhone[ret.Phone] = ret.RetailerID
	}
	if emit != nil {
		buf := &MemoryTxnBuffer{}
		_ = emit(buf)
	}
	return nil
}

func (r *MemoryRetailerRepo) FindByPhone(ctx context.Context, phone string) (retailer.Retailer, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byPhone[phone]
	if !ok {
		return retailer.Retailer{}, false, nil
	}
	ret, found := r.retailers[id]
	return ret, found, nil
}

func (r *MemoryRetailerRepo) GetRetailer(ctx context.Context, retailerID string) (retailer.Retailer, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ret, found := r.retailers[retailerID]
	return ret, found, nil
}

func (r *MemoryRetailerRepo) UpdateRetailer(ctx context.Context, ret retailer.Retailer, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retailers[ret.RetailerID] = ret
	if emit != nil {
		buf := &MemoryTxnBuffer{}
		_ = emit(buf)
	}
	return nil
}

func (r *MemoryRetailerRepo) ListRetailersBySupplier(ctx context.Context, supplierID string) ([]retailer.Retailer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []retailer.Retailer
	for _, ret := range r.retailers {
		if ret.SupplierID == supplierID {
			out = append(out, ret)
		}
	}
	return out, nil
}

func (r *MemoryRetailerRepo) GetSupplierPricingRule(ctx context.Context, supplierID string) (retailer.SupplierPricingRule, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.pricingRules[supplierID]
	return rule, ok, nil
}

func (r *MemoryRetailerRepo) ListTrackingOrders(ctx context.Context, retailerID string, limit, offset int) ([]retailer.TrackingOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tracking[retailerID], nil
}

func (r *MemoryRetailerRepo) ListRecentReceipts(ctx context.Context, retailerID string, limit int) ([]retailer.TrackingOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tracking[retailerID], nil
}

// UpdateKycDocumentStatus satisfies the unexported kycReviewRepository interface via structural typing.
func (r *MemoryRetailerRepo) UpdateKycDocumentStatus(ctx context.Context, docID string, status string, reviewedBy string, rejectionReason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	doc, ok := r.kycDocs[docID]
	if !ok {
		return fmt.Errorf("kyc document %s not found", docID)
	}
	doc.Status = status
	doc.RejectionReason = rejectionReason
	r.kycDocs[docID] = doc
	r.kycReviewedBy[docID] = reviewedBy
	return nil
}

func (r *MemoryRetailerRepo) GetKycDocument(ctx context.Context, docID string) (retailer.KycDocument, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, ok := r.kycDocs[docID]
	return doc, ok, nil
}

func (r *MemoryRetailerRepo) AddKycDocument(doc retailer.KycDocument) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.kycDocs[doc.DocumentID] = doc
	if doc.RetailerID != "" {
		if _, exists := r.retailers[doc.RetailerID]; !exists {
			r.retailers[doc.RetailerID] = retailer.Retailer{
				RetailerID: doc.RetailerID,
				SupplierID: DefaultSupplierA,
			}
		}
	}
}

func (r *MemoryRetailerRepo) InsertKycDocument(ctx context.Context, doc retailer.KycDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.kycDocs[doc.DocumentID] = doc
	return nil
}

func (r *MemoryRetailerRepo) ListKycDocumentsByRetailer(ctx context.Context, retailerID string) ([]retailer.KycDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []retailer.KycDocument
	for _, doc := range r.kycDocs {
		if doc.RetailerID == retailerID {
			out = append(out, doc)
		}
	}
	return out, nil
}

func (r *MemoryRetailerRepo) GetKycReviewedBy(docID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.kycReviewedBy[docID]
}

// MemoryControlTowerRepo implements controltower.Repository.
type MemoryControlTowerRepo struct {
	mu         sync.RWMutex
	playbooks  map[string]controltower.Playbook
	runs       map[string]controltower.PlaybookRun
	exceptions []controltower.Exception
}

func NewMemoryControlTowerRepo() *MemoryControlTowerRepo {
	return &MemoryControlTowerRepo{
		playbooks: make(map[string]controltower.Playbook),
		runs:      make(map[string]controltower.PlaybookRun),
	}
}

func (m *MemoryControlTowerRepo) ListActivePlaybooks(ctx context.Context, supplierID string) ([]controltower.Playbook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []controltower.Playbook
	for _, p := range m.playbooks {
		if p.SupplierID == supplierID && p.IsActive {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *MemoryControlTowerRepo) GetPlaybook(ctx context.Context, playbookID string) (controltower.Playbook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.playbooks[playbookID]
	if !ok {
		return controltower.Playbook{}, fmt.Errorf("playbook %s not found", playbookID)
	}
	return p, nil
}

func (m *MemoryControlTowerRepo) CreatePlaybook(ctx context.Context, pb controltower.Playbook) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.playbooks[pb.PlaybookID] = pb
	return nil
}

func (m *MemoryControlTowerRepo) UpdatePlaybook(ctx context.Context, playbookID string, fields map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.playbooks[playbookID]
	if !ok {
		return fmt.Errorf("playbook %s not found", playbookID)
	}
	if name, ok := fields["Name"].(string); ok {
		p.Name = name
	}
	if active, ok := fields["IsActive"].(bool); ok {
		p.IsActive = active
	}
	m.playbooks[playbookID] = p
	return nil
}

func (m *MemoryControlTowerRepo) DeactivatePlaybook(ctx context.Context, playbookID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.playbooks[playbookID]
	if !ok {
		return fmt.Errorf("playbook %s not found", playbookID)
	}
	p.IsActive = false
	m.playbooks[playbookID] = p
	return nil
}

func (m *MemoryControlTowerRepo) ListOpenExceptions(ctx context.Context, supplierID string) ([]controltower.Exception, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []controltower.Exception
	for _, e := range m.exceptions {
		if supplierID == "" || e.SupplierID == supplierID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *MemoryControlTowerRepo) ListSupplierIDsWithOpenExceptions(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	seen := make(map[string]bool)
	var out []string
	for _, e := range m.exceptions {
		if !seen[e.SupplierID] {
			seen[e.SupplierID] = true
			out = append(out, e.SupplierID)
		}
	}
	return out, nil
}

func (m *MemoryControlTowerRepo) HasBlockingRun(ctx context.Context, exceptionID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.runs {
		if r.ExceptionID == exceptionID && r.Status == controltower.RunStatusApproved {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryControlTowerRepo) CreateRun(ctx context.Context, run controltower.PlaybookRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[run.RunID] = run
	return nil
}

func (m *MemoryControlTowerRepo) GetRun(ctx context.Context, runID string) (controltower.PlaybookRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runs[runID]
	if !ok {
		return controltower.PlaybookRun{}, fmt.Errorf("run %s not found", runID)
	}
	return r, nil
}

func (m *MemoryControlTowerRepo) UpdateRun(ctx context.Context, run controltower.PlaybookRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[run.RunID] = run
	return nil
}

func (m *MemoryControlTowerRepo) ListRuns(ctx context.Context, supplierID, status string, limit int) ([]controltower.PlaybookRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []controltower.PlaybookRun
	for _, r := range m.runs {
		if r.SupplierID == supplierID && (status == "" || r.Status == status) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *MemoryControlTowerRepo) ListRunsForException(ctx context.Context, supplierID, exceptionID string) ([]controltower.PlaybookRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []controltower.PlaybookRun
	for _, r := range m.runs {
		if r.SupplierID == supplierID && r.ExceptionID == exceptionID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *MemoryControlTowerRepo) UpdateExceptionStatus(ctx context.Context, exceptionID, status string) error {
	return nil
}

func (m *MemoryControlTowerRepo) UpdateExceptionAssignee(ctx context.Context, exceptionID, role string) error {
	return nil
}

func (m *MemoryControlTowerRepo) RecordIntervention(ctx context.Context, manifestID, supplierID, operatorID, commandType, reasonCode, notes string) error {
	return nil
}

// MemoryPaymentRepo implements payment.Repository.
type MemoryPaymentRepo struct {
	mu          sync.RWMutex
	webhooks    []payment.WebhookRecord
	attempts    []payment.PaymentAttemptRecord
	chargebacks []payment.ChargebackRecord
	sessions    map[string]payment.SessionRecord
	payers      map[string]payment.Payer
}

func NewMemoryPaymentRepo() *MemoryPaymentRepo {
	return &MemoryPaymentRepo{
		sessions: make(map[string]payment.SessionRecord),
		payers:   make(map[string]payment.Payer),
	}
}

func (m *MemoryPaymentRepo) CreateSession(ctx context.Context, s payment.SessionRecord, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.SessionID] = s
	return nil
}

func (m *MemoryPaymentRepo) CreateSessionWithAttempt(ctx context.Context, s payment.SessionRecord, a payment.PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.SessionID] = s
	m.attempts = append(m.attempts, a)
	return nil
}

func (m *MemoryPaymentRepo) SaveAttempt(ctx context.Context, a payment.PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts = append(m.attempts, a)
	return nil
}

func (m *MemoryPaymentRepo) SaveChargeback(ctx context.Context, c payment.ChargebackRecord, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chargebacks = append(m.chargebacks, c)
	return nil
}

func (m *MemoryPaymentRepo) SaveReversal(ctx context.Context, rev payment.ReversalRecord, emit func(outbox.TxnBuffer) error) error {
	return nil
}

func (m *MemoryPaymentRepo) SaveWebhook(ctx context.Context, w payment.WebhookRecord, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhooks = append(m.webhooks, w)
	if emit != nil {
		buf := &MemoryTxnBuffer{}
		_ = emit(buf)
	}
	return nil
}

func (m *MemoryPaymentRepo) FindStuckSessions(ctx context.Context, cutoff time.Time, limit int) ([]payment.SessionRecord, error) {
	return nil, nil
}

func (m *MemoryPaymentRepo) GetSession(ctx context.Context, sessionID string) (payment.SessionRecord, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	return s, ok, nil
}

func (m *MemoryPaymentRepo) GetSessionByOrderID(ctx context.Context, orderID string) (payment.SessionRecord, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.sessions {
		if s.OrderID == orderID {
			return s, true, nil
		}
	}
	return payment.SessionRecord{}, false, nil
}

func (m *MemoryPaymentRepo) HasChargebackForOrder(ctx context.Context, orderID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.chargebacks {
		if c.OrderID == orderID {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryPaymentRepo) ListLedgerEntries(ctx context.Context, q payment.LedgerQuery) ([]payment.LedgerEntryRecord, error) {
	return nil, nil
}

func (m *MemoryPaymentRepo) SummarizeLedgerEntries(ctx context.Context, q payment.SettlementAuthorityQuery) ([]payment.SettlementAuthorityRow, error) {
	return nil, nil
}

func (m *MemoryPaymentRepo) CreatePayer(ctx context.Context, p payment.Payer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.payers[p.PayerID] = p
	return nil
}

func (m *MemoryPaymentRepo) GetPayer(ctx context.Context, payerID string) (payment.Payer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.payers[payerID]
	if !ok {
		return payment.Payer{}, errors.New("payer not found")
	}
	return p, nil
}

func (m *MemoryPaymentRepo) UpdatePayer(ctx context.Context, p payment.Payer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.payers[p.PayerID] = p
	return nil
}

func (m *MemoryPaymentRepo) ListPayers(ctx context.Context, limit, offset int) ([]payment.Payer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []payment.Payer
	for _, p := range m.payers {
		out = append(out, p)
	}
	return out, nil
}

func (m *MemoryPaymentRepo) GetAttempts() []payment.PaymentAttemptRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]payment.PaymentAttemptRecord, len(m.attempts))
	copy(res, m.attempts)
	return res
}

func (m *MemoryPaymentRepo) GetWebhooks() []payment.WebhookRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]payment.WebhookRecord, len(m.webhooks))
	copy(res, m.webhooks)
	return res
}

// MemoryInventoryRepo implements inventory.Repository with CAS version tracking.
type MemoryInventoryRepo struct {
	mu     sync.RWMutex
	levels map[string]*inventory.Level
	holds  map[string]*inventory.Hold
}

func NewMemoryInventoryRepo() *MemoryInventoryRepo {
	return &MemoryInventoryRepo{
		levels: make(map[string]*inventory.Level),
		holds:  make(map[string]*inventory.Hold),
	}
}

func (r *MemoryInventoryRepo) SeedLevel(l inventory.Level) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := l
	r.levels[l.InventoryID] = &cp
}

func (r *MemoryInventoryRepo) ListByWarehouse(ctx context.Context, warehouseID string) ([]inventory.Level, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []inventory.Level
	for _, l := range r.levels {
		if l.WarehouseID == warehouseID {
			out = append(out, *l)
		}
	}
	return out, nil
}

func (r *MemoryInventoryRepo) ListBySupplier(ctx context.Context, supplierID string) ([]inventory.Level, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []inventory.Level
	for _, l := range r.levels {
		if l.SupplierID == supplierID {
			out = append(out, *l)
		}
	}
	return out, nil
}

func (r *MemoryInventoryRepo) Get(ctx context.Context, inventoryID string) (*inventory.Level, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.levels[inventoryID]
	if !ok {
		return nil, fmt.Errorf("inventory %s not found", inventoryID)
	}
	cp := *l
	return &cp, nil
}

func (r *MemoryInventoryRepo) GetByWarehouseProduct(ctx context.Context, warehouseID, productID string) (*inventory.Level, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, l := range r.levels {
		if l.WarehouseID == warehouseID && l.ProductID == productID {
			cp := *l
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("inventory for warehouse %s product %s not found", warehouseID, productID)
}

func (r *MemoryInventoryRepo) AdjustStock(ctx context.Context, inventoryID string, delta int64, expectedVersion int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.levels[inventoryID]
	if !ok {
		return fmt.Errorf("read inventory %s: not found", inventoryID)
	}
	if l.Version != expectedVersion {
		return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, l.Version)
	}
	if l.QuantityOnHand+delta < 0 {
		return fmt.Errorf("insufficient stock on hand for inventory %s", inventoryID)
	}
	l.QuantityOnHand += delta
	l.Version++
	l.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryInventoryRepo) Upsert(ctx context.Context, l inventory.Level) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := l
	r.levels[l.InventoryID] = &cp
	return nil
}

func (r *MemoryInventoryRepo) ReserveForOrder(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.levels[inventoryID]
	if !ok {
		return fmt.Errorf("read inventory %s: not found", inventoryID)
	}
	if l.Version != expectedVersion {
		return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, l.Version)
	}
	available := l.QuantityOnHand - l.QuantityReserved
	if available < quantity {
		return fmt.Errorf("inventory %s insufficient stock: %d available, %d requested", inventoryID, available, quantity)
	}
	l.QuantityReserved += quantity
	l.Version++
	l.UpdatedAt = time.Now().UTC()
	return nil
}

// ReleaseReservation: P0 Item 7 Invariant — preserves QuantityOnHand and subtracts ONLY from QuantityReserved.
func (r *MemoryInventoryRepo) ReleaseReservation(ctx context.Context, inventoryID string, quantity int64, expectedVersion int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.levels[inventoryID]
	if !ok {
		return fmt.Errorf("read inventory %s: not found", inventoryID)
	}
	if l.Version != expectedVersion {
		return fmt.Errorf("inventory %s version conflict: expected %d got %d", inventoryID, expectedVersion, l.Version)
	}
	releaseQty := quantity
	if releaseQty > l.QuantityReserved {
		releaseQty = l.QuantityReserved
	}
	// INVARIANT: l.QuantityOnHand is preserved unchanged!
	l.QuantityReserved -= releaseQty
	l.Version++
	l.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryInventoryRepo) ReserveTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, productID string, quantity int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.levels {
		if l.WarehouseID == warehouseID && l.ProductID == productID {
			available := l.QuantityOnHand - l.QuantityReserved
			if available < quantity {
				return fmt.Errorf("insufficient stock for product %s in warehouse %s", productID, warehouseID)
			}
			l.QuantityReserved += quantity
			l.Version++
			return nil
		}
	}
	return fmt.Errorf("inventory not found for warehouse %s product %s", warehouseID, productID)
}

func (r *MemoryInventoryRepo) ReleaseTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, productID string, quantity int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.levels {
		if l.WarehouseID == warehouseID && l.ProductID == productID {
			releaseQty := quantity
			if releaseQty > l.QuantityReserved {
				releaseQty = l.QuantityReserved
			}
			l.QuantityReserved -= releaseQty
			l.Version++
			return nil
		}
	}
	return fmt.Errorf("inventory not found for warehouse %s product %s", warehouseID, productID)
}

func (r *MemoryInventoryRepo) CreateHold(ctx context.Context, hold inventory.Hold) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := hold
	r.holds[hold.HoldID] = &cp
	return nil
}

func (r *MemoryInventoryRepo) ConfirmHold(ctx context.Context, holdID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.holds[holdID]
	if !ok {
		return fmt.Errorf("hold %s not found", holdID)
	}
	h.Status = inventory.HoldStatusConfirmed
	return nil
}

func (r *MemoryInventoryRepo) ReleaseHold(ctx context.Context, holdID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.holds[holdID]
	if !ok {
		return fmt.Errorf("hold %s not found", holdID)
	}
	h.Status = inventory.HoldStatusReleased
	return nil
}

func (r *MemoryInventoryRepo) ListExpiredHolds(ctx context.Context, now time.Time, limit int) ([]inventory.Hold, error) {
	return nil, nil
}

// MemoryOrderRepo implements order.Repository.
type MemoryOrderRepo struct {
	mu             sync.RWMutex
	orders         map[string]order.Order
	fiscalAttempts map[string]order.FiscalReceiptRow
}

func NewMemoryOrderRepo() *MemoryOrderRepo {
	return &MemoryOrderRepo{
		orders:         make(map[string]order.Order),
		fiscalAttempts: make(map[string]order.FiscalReceiptRow),
	}
}

func (r *MemoryOrderRepo) SeedOrder(o order.Order) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[o.OrderID] = o
}

func (r *MemoryOrderRepo) SeedFiscalReceipt(row order.FiscalReceiptRow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row.AttemptID == "" {
		row.AttemptID = "att-" + uuid.NewString()[:8]
	}
	r.fiscalAttempts[row.AttemptID] = row
}

func (r *MemoryOrderRepo) GetFiscalReceipt(ctx context.Context, orderID string) (order.FiscalReceiptRow, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, f := range r.fiscalAttempts {
		if f.OrderID == orderID {
			return f, true, nil
		}
	}
	return order.FiscalReceiptRow{}, false, nil
}

func (r *MemoryOrderRepo) CreateOrder(ctx context.Context, o *order.Order, emit func(outbox.TxnBuffer) error, stockOpts order.StockReservationOpts) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o == nil {
		return errors.New("nil order")
	}
	if o.OrderID == "" {
		o.OrderID = "ord-" + uuid.NewString()[:8]
	}
	r.orders[o.OrderID] = *o
	if emit != nil {
		buf := &MemoryTxnBuffer{}
		_ = emit(buf)
	}
	return nil
}

func (r *MemoryOrderRepo) CreateOrderWithBackorder(ctx context.Context, o *order.Order, backorder *order.Order, inTxn func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error, stockOpts order.StockReservationOpts) error {
	return r.CreateOrder(ctx, o, emit, stockOpts)
}

func (r *MemoryOrderRepo) UpdateOrder(ctx context.Context, o order.Order, proofs []order.DeliveryProofArtifact, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[o.OrderID] = o
	if emit != nil {
		buf := &MemoryTxnBuffer{}
		_ = emit(buf)
	}
	return nil
}

func (r *MemoryOrderRepo) UpdateOrderWithTxn(ctx context.Context, o order.Order, proofs []order.DeliveryProofArtifact, inTxn func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error) error {
	return r.UpdateOrder(ctx, o, proofs, emit)
}

func (r *MemoryOrderRepo) GetOrder(ctx context.Context, orderID string) (order.Order, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orders[orderID]
	return o, ok, nil
}

func (r *MemoryOrderRepo) GetOrderTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (order.Order, bool, error) {
	return r.GetOrder(ctx, orderID)
}

func (r *MemoryOrderRepo) GetFiscalAttempt(ctx context.Context, orderID, attemptID string) (order.FiscalReceiptRow, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.fiscalAttempts[attemptID]
	return f, ok, nil
}

func (r *MemoryOrderRepo) GetFiscalByReceiptID(ctx context.Context, receiptID string) (order.FiscalReceiptRow, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, f := range r.fiscalAttempts {
		if f.FiscalReceiptID == receiptID {
			return f, true, nil
		}
	}
	return order.FiscalReceiptRow{}, false, nil
}

func (r *MemoryOrderRepo) FindPendingBuyerAcceptance(ctx context.Context, limit int) ([]*order.Order, error) {
	return nil, nil
}

func (r *MemoryOrderRepo) CountFiscalAttemptsByStatus(ctx context.Context, orderID, status string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var count int64
	for _, f := range r.fiscalAttempts {
		if f.OrderID == orderID && f.Status == status {
			count++
		}
	}
	return count, nil
}

func (r *MemoryOrderRepo) ListRetailerOrders(ctx context.Context, retailerID string, limit int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []order.Order
	for _, o := range r.orders {
		if o.RetailerID == retailerID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *MemoryOrderRepo) ListWarehouseOrdersByDeliveryWindow(ctx context.Context, warehouseID string, from, to time.Time, limit int) ([]order.Order, error) {
	return nil, nil
}

func (r *MemoryOrderRepo) ListDueAutoConfirmOrders(ctx context.Context, before time.Time, limit int) ([]order.Order, error) {
	return nil, nil
}

func (r *MemoryOrderRepo) ListManifestOrders(ctx context.Context, manifestID string) ([]order.Order, error) {
	return nil, nil
}

func (r *MemoryOrderRepo) ListWarehousePreorders(ctx context.Context, warehouseID string, limit, offset int) ([]order.Order, error) {
	return nil, nil
}

func (r *MemoryOrderRepo) ListOrdersForStockCommitment(ctx context.Context, warehouseID string, limit int) ([]order.Order, error) {
	return nil, nil
}

func (r *MemoryOrderRepo) ListBackorderedOrders(ctx context.Context, limit int) ([]order.Order, error) {
	return nil, nil
}

func (r *MemoryOrderRepo) ClearBackorder(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error, stockOpts order.StockReservationOpts) error {
	return nil
}

func (r *MemoryOrderRepo) ListOrdersByStatus(ctx context.Context, supplierID, status string, limit int) ([]order.Order, error) {
	return nil, nil
}

func (r *MemoryOrderRepo) CreateConditionReport(ctx context.Context, report order.ConditionReport, emit func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *MemoryOrderRepo) ListConditionReports(ctx context.Context, orderID string) ([]order.ConditionReport, error) {
	return nil, nil
}

func (r *MemoryOrderRepo) FindSiblingDriversForOrder(ctx context.Context, orderID string) ([]string, error) {
	return nil, nil
}

// MemoryFiscalProvider implements order.FiscalProvider and order.CorrectiveFiscalProvider.
type MemoryFiscalProvider struct {
	mu              sync.Mutex
	NameStr         string
	ShouldFail      bool
	LastReq         order.FiscalCorrectiveRequest
	CreatedReceipts []order.FiscalCreateResult
}

func NewMemoryFiscalProvider() *MemoryFiscalProvider {
	return &MemoryFiscalProvider{
		NameStr: "SOLIQ_EIMZO_PROVIDER",
	}
}

func (p *MemoryFiscalProvider) Name() string {
	if p.NameStr != "" {
		return p.NameStr
	}
	return "SOLIQ_EIMZO_PROVIDER"
}

func (p *MemoryFiscalProvider) GetSoliqClient() soliq.SoliqClient {
	return nil
}

func (p *MemoryFiscalProvider) CreateReceipt(ctx context.Context, req order.FiscalCreateRequest) (order.FiscalCreateResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ShouldFail {
		return order.FiscalCreateResult{}, errors.New("soliq_service_unavailable")
	}
	res := order.FiscalCreateResult{
		FiscalReceiptID: "SOLIQ-" + req.AttemptID,
		FiscalQR:        "https://ofd.soliq.uz/check?id=" + req.AttemptID,
		RawPayload:      []byte(`{"soliq_signature":"verified_eimzo"}`),
	}
	p.CreatedReceipts = append(p.CreatedReceipts, res)
	return res, nil
}

func (p *MemoryFiscalProvider) CreateCorrectiveReceipt(ctx context.Context, req order.FiscalCorrectiveRequest) (order.FiscalCreateResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.LastReq = req
	if p.ShouldFail {
		return order.FiscalCreateResult{}, errors.New("soliq_corrective_submission_failed")
	}
	res := order.FiscalCreateResult{
		FiscalReceiptID: "CORR-" + req.OriginalReceiptID + "-" + uuid.NewString()[:6],
		FiscalQR:        "https://ofd.soliq.uz/check?corr=true&id=" + req.AttemptID,
		RawPayload:      []byte(`{"soliq_corrective_signature":"valid"}`),
	}
	return res, nil
}

// TestResponse wraps an http.Response for assertions.
type TestResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func (r *TestResponse) JSON(v any) error {
	return json.Unmarshal(r.Body, v)
}

func (r *TestResponse) String() string {
	return string(r.Body)
}

// TestClient provides helper methods for calling the test server.
type TestClient struct {
	BaseURL    string
	HTTPClient *http.Client
	JWTSecret  string
}

func (c *TestClient) Do(req *http.Request) *TestResponse {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return &TestResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       []byte(err.Error()),
		}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return &TestResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       body,
	}
}

func (c *TestClient) Get(path string, token string) *TestResponse {
	req, _ := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.Do(req)
}

func (c *TestClient) GetWithCookie(path string, cookieName, cookieValue string) *TestResponse {
	req, _ := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: cookieValue})
	return c.Do(req)
}

func (c *TestClient) Post(path string, body any, token string) *TestResponse {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(http.MethodPost, c.BaseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-"+uuid.NewString()[:8])
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.Do(req)
}

func (c *TestClient) PostWithCookie(path string, body any, cookieName, cookieValue string) *TestResponse {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(http.MethodPost, c.BaseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: cookieName, Value: cookieValue})
	return c.Do(req)
}

func (c *TestClient) PostRaw(path string, rawBody []byte, token string, contentType ...string) *TestResponse {
	req, _ := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(rawBody))
	ct := "application/json"
	if len(contentType) > 0 && contentType[0] != "" {
		ct = contentType[0]
	}
	req.Header.Set("Content-Type", ct)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.Do(req)
}

func (c *TestClient) PatchRaw(path string, rawBody []byte, token string) *TestResponse {
	req, _ := http.NewRequest(http.MethodPatch, c.BaseURL+path, bytes.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.Do(req)
}

func (c *TestClient) Patch(path string, body any, token string) *TestResponse {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(http.MethodPatch, c.BaseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.Do(req)
}

func (c *TestClient) Delete(path string, token string) *TestResponse {
	req, _ := http.NewRequest(http.MethodDelete, c.BaseURL+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.Do(req)
}

// TestHarness is the unified opaque-box HTTP test harness.
type TestHarness struct {
	T             testing.TB
	Server        *httptest.Server
	Client        *TestClient
	Router        chi.Router
	JWTSecret     string
	JWTIssuer     string
	AdyenHMACKey  string
	SpannerClient *spanner.Client

	RetailerRepo             *MemoryRetailerRepo
	RetailerSvc              *retailer.Service
	FactorySvc               *factory.Service
	WarehouseSvc             *warehouse.Service
	PartnerKeys              partner.KeyRepository
	PartnerSvc               *partner.Service
	PartnerHandlers          *partner.Handlers
	ControlTowerRepo         *MemoryControlTowerRepo
	ControlTowerSvc          *controltower.Service
	PaymentRepo              *MemoryPaymentRepo
	PaymentSvc               *payment.Service
	InventoryRepo            *MemoryInventoryRepo
	InventorySvc             *inventory.Service
	OrderRepo                *MemoryOrderRepo
	OrderSvc                 *order.Service
	FiscalProvider           *MemoryFiscalProvider
	GLSvc                    *gl.Service
	PeriodsSvc               *periods.Service
	ApprovalsSvc             *approvals.Service
	ValuationSvc             *valuation.Service
	ESFSvc                   *esf.Service
	ReconSvc                 *recon.Service
	AuditSvc                 *audit.Service
	IAMSvc                   *iam.Service
	OntologySvc              *ontology.Service
	CreditSvc                *credit.Service
	CreditPolicySvc          *credit.PolicyService
	ARSvc                    *ar.Service
	ForecastV2Svc            *forecastv2.Service
	ReplenishmentSuggestions *replenishment.SuggestionsAPI
	ProposalsSvc             *proposals.Service
	ROISvc                   *roi.Service
	ConnectorsSvc            *connectors.Service
	ExportPacksSvc           *exportpacks.Service
	FxRatesSvc               *fxrates.Service
	FxRatesRepo              fxrates.Repository
	MultiEntitySvc           *multientity.Service
	P2PSvc                   *p2p.Service
	AnomaliesSvc             *anomalies.Service
	WhatIfSvc                *whatif.Service
}

// HarnessOptions controls optional live Spanner wiring for TestHarness.
type HarnessOptions struct {
	SpannerClient    *spanner.Client
	SCIMBearerToken  string
	SCIMTenantID     string
	SCIMTenantTokens map[string]string
}

// NewTestHarness builds and starts an opaque-box HTTP test server with all domain routes mounted.
func NewTestHarness(t testing.TB) *TestHarness {
	return NewTestHarnessWithOptions(t, HarnessOptions{})
}

// NewTestHarnessWithOptions builds an opaque-box HTTP test server optionally backed by a live Spanner client for GL routes.
func NewTestHarnessWithOptions(t testing.TB, opts HarnessOptions) *TestHarness {
	jwtSecret := DefaultJWTSecret
	jwtIssuer := DefaultJWTIssuer
	adyenKey := DefaultAdyenHMAC

	retailerRepo := NewMemoryRetailerRepo()
	retailerSvc := retailer.NewService(retailer.ServiceConfig{
		Repo:      retailerRepo,
		JWTSecret: jwtSecret,
		JWTIssuer: jwtIssuer,
		Now:       time.Now,
	})

	factorySvc := factory.NewService(factory.ServiceConfig{
		JWTSecret: jwtSecret,
		JWTIssuer: jwtIssuer,
	})

	warehouseSvc := warehouse.NewService(warehouse.ServiceConfig{
		JWTSecret: jwtSecret,
		JWTIssuer: jwtIssuer,
	})

	partnerKeyRepo := partner.NewMemoryKeyRepository()
	partnerSvc := partner.NewService(partnerKeyRepo, nil, nil, nil, nil)
	partnerHandlers := &partner.Handlers{Svc: partnerSvc}

	ctRepo := NewMemoryControlTowerRepo()
	ctEngine := controltower.NewEngine(ctRepo, controltower.NewActionExecutor(ctRepo, controltower.ExecutorDeps{}), nil, controltower.Config{Enabled: true}, nil)
	ctSvc := controltower.NewService(ctRepo, ctEngine, controltower.Config{Enabled: true})
	ctHandlers := controltower.NewHandlers(ctSvc, nil)

	paymentRepo := NewMemoryPaymentRepo()
	paymentSvc := payment.NewService(payment.ServiceConfig{
		SupplierID:         DefaultSupplierA,
		AdyenWebhookSecret: adyenKey,
		Repo:               paymentRepo,
	})

	invRepo := NewMemoryInventoryRepo()
	invSvc := inventory.NewService(invRepo, nil, nil)

	orderRepo := NewMemoryOrderRepo()
	orderSvc := order.NewService(order.ServiceConfig{
		Repo:           orderRepo,
		JWTSecret:      jwtSecret,
		SeedSupplierID: DefaultSupplierA,
	})
	fiscalProvider := NewMemoryFiscalProvider()
	orderSvc.SetFiscalProvider(fiscalProvider)

	var glSvc *gl.Service
	var periodsSvc *periods.Service
	var approvalsSvc *approvals.Service
	var valuationSvc *valuation.Service
	var esfSvc *esf.Service
	var reconSvc *recon.Service
	var auditSvc *audit.Service
	var iamSvc *iam.Service
	var ontSvc *ontology.Service
	var creditSvc *credit.Service
	var creditPolicySvc *credit.PolicyService
	var arSvc *ar.Service
	var forecastV2Svc *forecastv2.Service
	var roiSvc *roi.Service
	var proposalsSvc *proposals.Service
	var fxRepo fxrates.Repository
	if opts.SpannerClient != nil {
		fxRepo = fxrates.NewSpannerRepository(opts.SpannerClient)
		glSvc = gl.NewService(opts.SpannerClient)
		_ = glSvc.SeedDefaults(context.Background(), DefaultSupplierA, "UZ", "UZS")
		periodsSvc = periods.NewService(opts.SpannerClient)
		approvalsSvc = approvals.NewService(opts.SpannerClient)
		valuationSvc = valuation.NewService(opts.SpannerClient)
		esfSvc = esf.NewService(esf.ServiceConfig{Repo: esf.NewSpannerRepository(opts.SpannerClient)})
		reconSvc = recon.NewService(opts.SpannerClient)
		auditSvc = audit.NewService(opts.SpannerClient)
		iamSvc = iam.NewService(opts.SpannerClient)
		ontSvc = ontology.NewService(ontology.NewSpannerRepository(opts.SpannerClient), nil)
		arSvc = ar.NewService(ar.NewSpannerRepository(opts.SpannerClient))
		creditSvc = credit.NewService(credit.NewSpannerRepository(opts.SpannerClient))
		creditSvc.SetScoreMetricsProvider(&credit.ARScoreMetrics{Client: opts.SpannerClient})
		creditPolicySvc = credit.NewPolicyService(credit.NewSpannerPolicyRepository(opts.SpannerClient), creditSvc)
		forecastV2Svc = forecastv2.NewService(opts.SpannerClient)
		roiSvc = roi.NewService(opts.SpannerClient)
		proposalsSvc = proposals.NewService(opts.SpannerClient)
	} else {
		fxRepo = fxrates.NewMemoryRepository()
		glSvc = gl.NewServiceWithRepository(newMemoryGLRepo())
		periodsSvc = periods.NewServiceWithRepository(periods.NewMemoryRepository())
		approvalsSvc = approvals.NewServiceWithRepository(approvals.NewMemoryRepository())
		valuationSvc = valuation.NewServiceWithRepository(valuation.NewMemoryRepository())
		esfSvc = esf.NewService(esf.ServiceConfig{Repo: esf.NewMemoryRepository()})
		reconSvc = recon.NewServiceWithRepository(recon.NewMemoryRepository())
		auditSvc = audit.NewServiceWithRepository(audit.NewMemoryRepository())
		iamSvc = iam.NewServiceWithRepository(iam.NewMemoryRepository())
		ontSvc = ontology.NewService(ontology.NewMemoryRepository(), nil)
		arSvc = ar.NewService(ar.NewMemoryRepository())
		memCreditRepo := memory.NewCreditRepo(nil)
		memPolicyRepo := credit.NewMemoryPolicyRepository()
		creditSvc = credit.NewService(memCreditRepo)
		creditSvc.SetScoreHistoryRepo(credit.NewMemoryScoreHistoryRepository(memCreditRepo, memPolicyRepo))
		creditPolicySvc = credit.NewPolicyService(memPolicyRepo, creditSvc)
		forecastV2Svc = forecastv2.NewServiceWithRepository(forecastv2.NewMemoryRepository())
		roiSvc = roi.NewServiceWithRepository(roi.NewMemoryRepository())
		proposalsSvc = proposals.NewServiceWithRepository(proposals.NewMemoryRepository())
	}
	iamSvc.SetAuditRecorder(bootstrap.NewIAMAuditRecorder(auditSvc))
	orderSvc.SetDraftSoDCheck(bootstrap.CheckOrderDraftSoD(iamSvc))
	waveApp := &bootstrap.App{
		Audit:               auditSvc,
		IAM:                 iamSvc,
		Ontology:            ontSvc,
		GL:                  glSvc,
		OrderService:        orderSvc,
		WarehouseService:    warehouseSvc,
		ControlTowerService: ctSvc,
		CreditService:       creditSvc,
		CreditPolicyService: creditPolicySvc,
		ARService:           arSvc,
		Periods:             periodsSvc,
		Approvals:           approvalsSvc,
		Valuation:           valuationSvc,
		ESF:                 esfSvc,
		ForecastV2:          forecastV2Svc,
		ROI:                 roiSvc,
		Proposals:           proposalsSvc,
		FxRatesRepo:         fxRepo,
		FxRatesService:      fxrates.NewService(fxRepo),
	}
	_ = bootstrap.SetupWave3A(context.Background(), &bootstrap.Config{AllowMemoryFallback: true}, opts.SpannerClient, nil, waveApp, nil)
	_ = bootstrap.SetupWave3B(context.Background(), &bootstrap.Config{AllowMemoryFallback: true}, opts.SpannerClient, waveApp, nil)
	_ = bootstrap.SetupWave3C(context.Background(), &bootstrap.Config{
		AllowMemoryFallback: true,
		JWTSecret:           jwtSecret,
		JWTIssuer:           jwtIssuer,
	}, opts.SpannerClient, waveApp, nil)
	_ = bootstrap.SetupWave4A(context.Background(), &bootstrap.Config{
		AllowMemoryFallback: true,
		JWTSecret:           jwtSecret,
		JWTIssuer:           jwtIssuer,
	}, opts.SpannerClient, waveApp, nil)
	_ = bootstrap.SetupWave4B(context.Background(), &bootstrap.Config{
		AllowMemoryFallback: true,
		JWTSecret:           jwtSecret,
		JWTIssuer:           jwtIssuer,
	}, opts.SpannerClient, waveApp, nil)
	connectorsSvc := waveApp.Connectors
	exportPacksSvc := waveApp.ExportPacks
	fxRatesSvc := waveApp.FxRatesService
	fxRatesRepo := waveApp.FxRatesRepo
	multiEntitySvc := waveApp.MultiEntity
	p2pSvc := waveApp.P2P
	anomaliesSvc := waveApp.Anomalies
	whatifSvc := waveApp.WhatIf
	t.Cleanup(func() {
		gl.SetPeriodLockChecker(nil)
		glSvc.SetLegalEntityOwnershipChecker(nil)
		stocklots.SetPickValuationHook(nil)
		stocklots.SetAdjustmentValuationHook(nil)
	})

	r := chi.NewRouter()
	r.Use(bootstrap.DevCORSMiddleware())
	r.Use(auth.SessionAuth(jwtSecret))
	r.Use(auth.AttachTenantFromClaims)
	r.Use(audit.AuditMiddleware(auditSvc))

	// Mount all required domain routes
	retailerroutes.RegisterRoutes(r, retailerroutes.Deps{
		Service:   retailerSvc,
		JWTSecret: jwtSecret,
		JWTIssuer: jwtIssuer,
	})

	factoryroutes.RegisterRoutes(r, factoryroutes.Deps{
		Service:   factorySvc,
		JWTSecret: jwtSecret,
		JWTIssuer: jwtIssuer,
	})

	wmsHandler := &stocklots.Handler{
		Spanner:     opts.SpannerClient,
		SoDCheck:    bootstrap.CheckStockAdjustmentSoD(iamSvc),
		AuditRecord: bootstrap.NewStocklotsAuditRecorder(auditSvc),
	}
	warehouseroutes.RegisterRoutes(r, warehouseroutes.Deps{
		Service:    warehouseSvc,
		WMSHandler: wmsHandler,
		JWTSecret:  jwtSecret,
		JWTIssuer:  jwtIssuer,
	})

	partner.RegisterAdminKeyRoutes(r, partnerHandlers)
	partner.RegisterPartnerRoutesOpts(r, partner.AuthOptions{Keys: partnerKeyRepo, JWTSecret: jwtSecret}, partnerHandlers)

	controltowerroutes.RegisterRoutes(r, controltowerroutes.Deps{
		Handlers: ctHandlers,
	})

	paymentroutes.RegisterRoutes(r, paymentroutes.Deps{
		Service:   paymentSvc,
		JWTSecret: jwtSecret,
	})

	webhookroutes.RegisterRoutes(r, webhookroutes.Deps{
		Service: paymentSvc,
	})

	orderroutes.RegisterRoutes(r, orderroutes.Deps{
		Service: orderSvc,
	})

	glroutes.RegisterRoutes(r, glroutes.Deps{
		Service:   glSvc,
		JWTSecret: jwtSecret,
	})
	periodroutes.RegisterRoutes(r, periodroutes.Deps{
		Service:   periodsSvc,
		JWTSecret: jwtSecret,
	})
	approvalroutes.RegisterRoutes(r, approvalroutes.Deps{
		Service:   approvalsSvc,
		JWTSecret: jwtSecret,
	})
	valuationroutes.RegisterRoutes(r, valuationroutes.Deps{
		Service:   valuationSvc,
		JWTSecret: jwtSecret,
	})
	esfroutes.RegisterRoutes(r, esfroutes.Deps{
		Service:   esfSvc,
		JWTSecret: jwtSecret,
	})
	reconroutes.RegisterRoutes(r, reconroutes.Deps{
		Service:   reconSvc,
		JWTSecret: jwtSecret,
	})
	auditroutes.RegisterRoutes(r, auditroutes.Deps{
		Service:   auditSvc,
		JWTSecret: jwtSecret,
	})
	iamroutes.RegisterRoutes(r, iamroutes.Deps{
		Service:          iamSvc,
		JWTSecret:        jwtSecret,
		JWTIssuer:        jwtIssuer,
		SCIMBearerToken:  opts.SCIMBearerToken,
		SCIMTenantID:     opts.SCIMTenantID,
		SCIMTenantTokens: opts.SCIMTenantTokens,
	})
	ontologyroutes.RegisterRoutes(r, ontologyroutes.Deps{
		Service:   ontSvc,
		JWTSecret: jwtSecret,
	})
	creditroutes.RegisterRoutes(r, creditroutes.Deps{
		Service:       creditSvc,
		PolicyService: creditPolicySvc,
		ARService:     arSvc,
	})
	forecastroutes.RegisterRoutes(r, forecastroutes.Deps{
		Service:          forecastV2Svc,
		ReplenishmentAPI: waveApp.ReplenishmentSuggestions,
		JWTSecret:        jwtSecret,
	})
	proposalroutes.RegisterRoutes(r, proposalroutes.Deps{
		Service:   proposalsSvc,
		JWTSecret: jwtSecret,
	})
	roiroutes.RegisterRoutes(r, roiroutes.Deps{
		Service:   roiSvc,
		JWTSecret: jwtSecret,
	})
	connectorroutes.RegisterRoutes(r, connectorroutes.Deps{
		Service:   connectorsSvc,
		JWTSecret: jwtSecret,
	})
	exportpackroutes.RegisterRoutes(r, exportpackroutes.Deps{
		Service:   exportPacksSvc,
		JWTSecret: jwtSecret,
	})
	multientityroutes.RegisterRoutes(r, multientityroutes.Deps{
		Service:   multiEntitySvc,
		JWTSecret: jwtSecret,
	})
	p2proutes.RegisterRoutes(r, p2proutes.Deps{
		Service:   p2pSvc,
		JWTSecret: jwtSecret,
	})
	anomalyroutes.RegisterRoutes(r, anomalyroutes.Deps{
		Service:   anomaliesSvc,
		JWTSecret: jwtSecret,
	})
	whatifroutes.RegisterRoutes(r, whatifroutes.Deps{
		Service:   whatifSvc,
		JWTSecret: jwtSecret,
	})
	stocklots.RegisterHACCPComplianceRoute(r, wmsHandler, jwtSecret, false)

	// Protected probe route for backend JWT session testing (Item 10 backend perspective)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer, auth.RolePlatformAdmin, auth.RoleFactory, auth.RoleWarehouse, auth.RoleDriver)).Get(
		"/v1/test/auth-probe",
		func(w http.ResponseWriter, req *http.Request) {
			claims, ok := auth.FromContext(req.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"authenticated": true,
				"subject":       claims.Subject,
				"role":          claims.Role,
				"supplier_id":   claims.SupplierID,
			})
		},
	)

	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
	})

	client := &TestClient{
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
		JWTSecret:  jwtSecret,
	}

	h := &TestHarness{
		T:                        t,
		Server:                   ts,
		Client:                   client,
		Router:                   r,
		JWTSecret:                jwtSecret,
		JWTIssuer:                jwtIssuer,
		AdyenHMACKey:             adyenKey,
		SpannerClient:            opts.SpannerClient,
		RetailerRepo:             retailerRepo,
		RetailerSvc:              retailerSvc,
		FactorySvc:               factorySvc,
		WarehouseSvc:             warehouseSvc,
		PartnerKeys:              partnerKeyRepo,
		PartnerSvc:               partnerSvc,
		PartnerHandlers:          partnerHandlers,
		ControlTowerRepo:         ctRepo,
		ControlTowerSvc:          ctSvc,
		PaymentRepo:              paymentRepo,
		PaymentSvc:               paymentSvc,
		InventoryRepo:            invRepo,
		InventorySvc:             invSvc,
		OrderRepo:                orderRepo,
		OrderSvc:                 orderSvc,
		FiscalProvider:           fiscalProvider,
		GLSvc:                    glSvc,
		PeriodsSvc:               periodsSvc,
		ApprovalsSvc:             approvalsSvc,
		ValuationSvc:             valuationSvc,
		ESFSvc:                   esfSvc,
		ReconSvc:                 reconSvc,
		AuditSvc:                 auditSvc,
		IAMSvc:                   iamSvc,
		OntologySvc:              ontSvc,
		CreditSvc:                creditSvc,
		CreditPolicySvc:          creditPolicySvc,
		ARSvc:                    arSvc,
		ForecastV2Svc:            forecastV2Svc,
		ReplenishmentSuggestions: waveApp.ReplenishmentSuggestions,
		ProposalsSvc:             proposalsSvc,
		ROISvc:                   roiSvc,
		ConnectorsSvc:            connectorsSvc,
		ExportPacksSvc:           exportPacksSvc,
		FxRatesSvc:               fxRatesSvc,
		FxRatesRepo:              fxRatesRepo,
		MultiEntitySvc:           multiEntitySvc,
		P2PSvc:                   p2pSvc,
		AnomaliesSvc:             anomaliesSvc,
		WhatIfSvc:                whatifSvc,
	}

	return h
}

// Token helper methods
func (h *TestHarness) IssueToken(role auth.Role, supplierID, subject string) string {
	tok, err := auth.Issue(auth.Claims{
		Subject:    subject,
		Role:       role,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: h.JWTSecret, TTL: time.Hour, Issuer: h.JWTIssuer})
	if err != nil {
		h.T.Fatalf("failed to issue token: %v", err)
	}
	return tok
}

func (h *TestHarness) IssueAdminToken(supplierID, subject string) string {
	return h.IssueToken(auth.RoleAdmin, supplierID, subject)
}

func (h *TestHarness) IssueRetailerToken(supplierID, subject string) string {
	return h.IssueToken(auth.RoleRetailer, supplierID, subject)
}

func (h *TestHarness) IssueFactoryToken(supplierID, subject string) string {
	return h.IssueToken(auth.RoleFactory, supplierID, subject)
}

func (h *TestHarness) IssueWarehouseToken(supplierID, subject string) string {
	return h.IssueToken(auth.RoleWarehouse, supplierID, subject)
}

func (h *TestHarness) IssueDriverToken(supplierID, subject string) string {
	return h.IssueToken(auth.RoleDriver, supplierID, subject)
}

func (h *TestHarness) IssuePlatformAdminToken(subject string) string {
	return h.IssueToken(auth.RolePlatformAdmin, "", subject)
}

func (h *TestHarness) IssueExpiredToken(role auth.Role, supplierID, subject string) string {
	tok, err := auth.Issue(auth.Claims{
		Subject:    subject,
		Role:       role,
		SupplierID: supplierID,
	}, auth.IssueOptions{
		Secret: h.JWTSecret,
		TTL:    time.Hour,
		Now:    func() time.Time { return time.Now().UTC().Add(-48 * time.Hour) },
		Issuer: h.JWTIssuer,
	})
	if err != nil {
		h.T.Fatalf("failed to issue expired token: %v", err)
	}
	return tok
}

func (h *TestHarness) IssueTamperedToken(role auth.Role, supplierID, subject string) string {
	tok, err := auth.Issue(auth.Claims{
		Subject:    subject,
		Role:       role,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: "completely-wrong-tampered-secret-key-!", TTL: time.Hour, Issuer: h.JWTIssuer})
	if err != nil {
		h.T.Fatalf("failed to issue tampered token: %v", err)
	}
	return tok
}

// GenerateAdyenWebhookPayload produces a cryptographically signed Adyen notification item.
func GenerateAdyenWebhookPayload(t testing.TB, hmacKey string, eventCode string, success bool, pspRef string, amountMinor int64, currency string) []byte {
	now := time.Now().UTC()
	item := adyenwebhook.NotificationRequestItem{
		EventCode:           eventCode,
		Success:             fmt.Sprintf("%v", success),
		PspReference:        pspRef,
		MerchantAccountCode: "PegasusXMerchant",
		MerchantReference:   "ord-e2e-ref",
		Amount: adyenwebhook.Amount{
			Value:    amountMinor,
			Currency: currency,
		},
		EventDate: &now,
	}
	sig, err := adyenhmac.CalculateHmac(item, hmacKey)
	if err != nil {
		t.Fatalf("failed to calculate adyen hmac: %v", err)
	}
	m := make(map[string]interface{})
	m["hmacSignature"] = sig
	item.AdditionalData = &m

	root := adyenwebhook.Webhook{
		Live: "false",
		NotificationItems: &[]adyenwebhook.NotificationItem{
			{NotificationRequestItem: item},
		},
	}
	raw, err := json.Marshal(root)
	if err != nil {
		t.Fatalf("failed to marshal adyen webhook: %v", err)
	}
	return raw
}
