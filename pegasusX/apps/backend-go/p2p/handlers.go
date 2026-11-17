package p2p

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type createTemplateHTTPReq struct {
	TenantID   string        `json:"tenant_id,omitempty"`
	TemplateID string        `json:"template_id,omitempty"`
	Code       string        `json:"code"`
	Name       string        `json:"name"`
	Domain     string        `json:"domain"`
	Steps      []ProcessStep `json:"steps"`
	IsDefault  bool          `json:"is_default,omitempty"`
	Status     string        `json:"status,omitempty"`
}

type upsertInstanceHTTPReq struct {
	TenantID     string `json:"tenant_id,omitempty"`
	InstanceID   string `json:"instance_id,omitempty"`
	TemplateID   string `json:"template_id,omitempty"`
	TemplateCode string `json:"template_code,omitempty"`
	EntityID     string `json:"entity_id,omitempty"`
	ObjectType   string `json:"object_type"`
	ObjectID     string `json:"object_id"`
	NextStep     string `json:"next_step,omitempty"`
	CurrentStep  string `json:"current_step,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Complete     bool   `json:"complete,omitempty"`
}

type createPOLineHTTPReq struct {
	Sku           string `json:"sku"`
	ProductName   string `json:"product_name,omitempty"`
	QtyOrdered    int64  `json:"qty_ordered"`
	UnitCostMinor int64  `json:"unit_cost_minor"`
	VatBps        *int64 `json:"vat_bps,omitempty"`
}

type createPOHTTPReq struct {
	TenantID       string                `json:"tenant_id,omitempty"`
	PoID           string                `json:"po_id,omitempty"`
	EntityID       string                `json:"entity_id,omitempty"`
	VendorID       string                `json:"vendor_id"`
	VendorName     string                `json:"vendor_name,omitempty"`
	VendorTin      string                `json:"vendor_tin,omitempty"`
	Currency       string                `json:"currency,omitempty"`
	ExpectedAt     string                `json:"expected_at,omitempty"`
	ApprovalID     string                `json:"approval_id,omitempty"`
	IdempotencyKey string                `json:"idempotency_key,omitempty"`
	TemplateCode   string                `json:"template_code,omitempty"`
	Lines          []createPOLineHTTPReq `json:"lines"`
}

type approvePOHTTPReq struct {
	TenantID   string `json:"tenant_id,omitempty"`
	ApprovalID string `json:"approval_id,omitempty"`
}

type postReceiptHTTPReq struct {
	TenantID       string                  `json:"tenant_id,omitempty"`
	ReceiptID      string                  `json:"receipt_id,omitempty"`
	PoID           string                  `json:"po_id"`
	EntityID       string                  `json:"entity_id,omitempty"`
	WarehouseID    string                  `json:"warehouse_id"`
	PeriodID       string                  `json:"period_id,omitempty"`
	IdempotencyKey string                  `json:"idempotency_key,omitempty"`
	ReceivedAt     string                  `json:"received_at,omitempty"`
	Lines          []GoodsReceiptLineInput `json:"lines"`
}

type createVendorInvoiceHTTPReq struct {
	TenantID        string `json:"tenant_id,omitempty"`
	VendorInvoiceID string `json:"vendor_invoice_id,omitempty"`
	PoID            string `json:"po_id"`
	ReceiptID       string `json:"receipt_id,omitempty"`
	EsfDocID        string `json:"esf_doc_id,omitempty"`
	EntityID        string `json:"entity_id,omitempty"`
	VendorID        string `json:"vendor_id,omitempty"`
	Currency        string `json:"currency,omitempty"`
	SubtotalMinor   int64  `json:"subtotal_minor"`
	VatMinor        int64  `json:"vat_minor"`
	TotalMinor      int64  `json:"total_minor"`
	IdempotencyKey  string `json:"idempotency_key,omitempty"`
}

type threeWayMatchHTTPReq struct {
	TenantID            string `json:"tenant_id,omitempty"`
	VendorInvoiceID     string `json:"vendor_invoice_id"`
	PoID                string `json:"po_id,omitempty"`
	ReceiptID           string `json:"receipt_id,omitempty"`
	EsfDocID            string `json:"esf_doc_id,omitempty"`
	PeriodID            string `json:"period_id,omitempty"`
	ToleranceBps        int64  `json:"tolerance_bps,omitempty"`
	ToleranceMinor      int64  `json:"tolerance_minor,omitempty"`
	OverrideDiscrepancy bool   `json:"override_discrepancy,omitempty"`
	ApprovalID          string `json:"approval_id,omitempty"`
	VerifiedBy          string `json:"verified_by,omitempty"`
}

// HandleListTemplates serves GET /v1/processes/templates.
func (s *Service) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	items, err := s.ListTemplates(r.Context(), tenantID, r.URL.Query().Get("domain"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if items == nil {
		items = []ProcessTemplate{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"templates": items,
		"count":     len(items),
	})
}

// HandleCreateTemplate serves POST /v1/processes/templates.
func (s *Service) HandleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	var req createTemplateHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, coalesce(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	actorID, actorType, actorRole := callerIdentity(r)
	tpl, err := s.CreateTemplate(r.Context(), CreateTemplateInput{
		TenantID:   tenantID,
		TemplateID: req.TemplateID,
		Code:       req.Code,
		Name:       req.Name,
		Domain:     req.Domain,
		Steps:      req.Steps,
		IsDefault:  req.IsDefault,
		Status:     req.Status,
		ActorID:    actorID,
		ActorType:  actorType,
		ActorRole:  actorRole,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "p2p.template.create",
		ObjectType: "ProcessTemplate",
		ObjectID:   tpl.TemplateID,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"template": tpl})
}

// HandleListInstances serves GET /v1/processes/instances.
func (s *Service) HandleListInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	items, err := s.ListInstances(r.Context(), ListFilter{
		TenantID:   tenantID,
		EntityID:   coalesce(r.URL.Query().Get("entity_id"), r.URL.Query().Get("entity")),
		ObjectType: r.URL.Query().Get("object_type"),
		ObjectID:   r.URL.Query().Get("object_id"),
		Status:     r.URL.Query().Get("status"),
		Limit:      parseQueryLimit(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if items == nil {
		items = []ProcessInstance{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"instances": items,
		"count":     len(items),
	})
}

// HandleUpsertInstance serves POST /v1/processes/instances.
func (s *Service) HandleUpsertInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	var req upsertInstanceHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, coalesce(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	actorID, actorType, actorRole := callerIdentity(r)
	inst, err := s.StartOrAdvanceInstance(r.Context(), StartOrAdvanceInstanceInput{
		TenantID:     tenantID,
		InstanceID:   req.InstanceID,
		TemplateID:   req.TemplateID,
		TemplateCode: req.TemplateCode,
		EntityID:     req.EntityID,
		ObjectType:   req.ObjectType,
		ObjectID:     req.ObjectID,
		NextStep:     req.NextStep,
		CurrentStep:  req.CurrentStep,
		Reason:       req.Reason,
		Complete:     req.Complete,
		ActorID:      actorID,
		ActorType:    actorType,
		ActorRole:    actorRole,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "p2p.process_instance.upsert",
		ObjectType: "ProcessInstance",
		ObjectID:   inst.InstanceID,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"instance": inst})
}

// HandleProcessMining serves GET /v1/processes/mining.
func (s *Service) HandleProcessMining(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	fromPtr, err := parseOptionalTimestamp(r.URL.Query().Get("from"), false)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_from", "from must be RFC3339 or YYYY-MM-DD")
		return
	}
	toPtr, err := parseOptionalTimestamp(r.URL.Query().Get("to"), true)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_to", "to must be RFC3339 or YYYY-MM-DD")
		return
	}
	report, err := s.GetProcessMining(r.Context(), MiningFilter{
		TenantID: tenantID,
		EntityID: coalesce(r.URL.Query().Get("entity_id"), r.URL.Query().Get("entity")),
		Domain:   r.URL.Query().Get("domain"),
		From:     fromPtr,
		To:       toPtr,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// HandleListPurchaseOrders serves GET /v1/p2p/purchase-orders.
func (s *Service) HandleListPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	items, err := s.ListPurchaseOrders(r.Context(), ListFilter{
		TenantID: tenantID,
		EntityID: coalesce(r.URL.Query().Get("entity_id"), r.URL.Query().Get("entity")),
		VendorID: r.URL.Query().Get("vendor_id"),
		Status:   r.URL.Query().Get("status"),
		Limit:    parseQueryLimit(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if items == nil {
		items = []PurchaseOrder{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":       tenantID,
		"purchase_orders": items,
		"count":           len(items),
	})
}

// HandleGetPurchaseOrder serves GET /v1/p2p/purchase-orders/{poID}.
func (s *Service) HandleGetPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	poID := strings.TrimSpace(chi.URLParam(r, "poID"))
	if poID == "" {
		writeJSONError(w, http.StatusBadRequest, "po_id_required", "poID is required")
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	po, err := s.GetPurchaseOrder(r.Context(), tenantID, poID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"purchase_order": po})
}

// HandleCreatePurchaseOrder serves POST /v1/p2p/purchase-orders.
func (s *Service) HandleCreatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	var req createPOHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, coalesce(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	expectedAt, err := parseOptionalTimestamp(req.ExpectedAt, false)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_expected_at", "expected_at must be RFC3339 or YYYY-MM-DD")
		return
	}
	lines := make([]PurchaseOrderLineInput, 0, len(req.Lines))
	for _, rl := range req.Lines {
		vatBps := DefaultVatRateBps
		if rl.VatBps != nil {
			vatBps = *rl.VatBps
		}
		lines = append(lines, PurchaseOrderLineInput{
			Sku:           rl.Sku,
			ProductName:   rl.ProductName,
			QtyOrdered:    rl.QtyOrdered,
			UnitCostMinor: rl.UnitCostMinor,
			VatBps:        vatBps,
		})
	}
	actorID, actorType, actorRole := callerIdentity(r)
	idemKey := coalesce(req.IdempotencyKey, r.Header.Get("Idempotency-Key"))
	res, err := s.CreatePurchaseOrder(r.Context(), CreatePurchaseOrderInput{
		TenantID:       tenantID,
		PoID:           req.PoID,
		EntityID:       req.EntityID,
		VendorID:       req.VendorID,
		VendorName:     req.VendorName,
		VendorTin:      req.VendorTin,
		Currency:       req.Currency,
		ExpectedAt:     expectedAt,
		ApprovalID:     req.ApprovalID,
		IdempotencyKey: idemKey,
		TemplateCode:   req.TemplateCode,
		Lines:          lines,
		CreatedBy:      actorID,
		ActorType:      actorType,
		ActorRole:      actorRole,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "p2p.po.create",
		ObjectType: AggregatePurchaseOrder,
		ObjectID:   res.PurchaseOrder.PoID,
		ApprovalID: res.ApprovalRequestID,
	})
	statusCode := http.StatusCreated
	if res.ApprovalRequired {
		statusCode = http.StatusAccepted
	}
	writeJSON(w, statusCode, res)
}

// HandleApprovePurchaseOrder serves POST /v1/p2p/purchase-orders/{poID}/approve.
func (s *Service) HandleApprovePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	poID := strings.TrimSpace(chi.URLParam(r, "poID"))
	if poID == "" {
		writeJSONError(w, http.StatusBadRequest, "po_id_required", "poID is required")
		return
	}
	var req approvePOHTTPReq
	if r.Body != nil {
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		_ = r.Body.Close()
		if len(bytes.TrimSpace(raw)) > 0 {
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.UseNumber()
			if err := dec.Decode(&req); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
				return
			}
		}
	}
	tenantID, ok := resolveP2PTenantScope(w, r, coalesce(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	actorID, actorType, actorRole := callerIdentity(r)
	po, err := s.ApprovePurchaseOrder(r.Context(), ApprovePurchaseOrderInput{
		TenantID:   tenantID,
		PoID:       poID,
		ApprovalID: req.ApprovalID,
		ActorID:    actorID,
		ActorType:  actorType,
		ActorRole:  actorRole,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "p2p.po.approve",
		ObjectType: AggregatePurchaseOrder,
		ObjectID:   po.PoID,
		ApprovalID: req.ApprovalID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"purchase_order": po})
}

// HandleListReceipts serves GET /v1/p2p/receipts.
func (s *Service) HandleListReceipts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	items, err := s.ListGoodsReceipts(r.Context(), ListFilter{
		TenantID: tenantID,
		PoID:     r.URL.Query().Get("po_id"),
		EntityID: coalesce(r.URL.Query().Get("entity_id"), r.URL.Query().Get("entity")),
		Limit:    parseQueryLimit(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if items == nil {
		items = []GoodsReceipt{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"receipts":  items,
		"count":     len(items),
	})
}

// HandlePostReceipt serves POST /v1/p2p/receipts.
func (s *Service) HandlePostReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	var req postReceiptHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, coalesce(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	receivedAt, err := parseOptionalTimestamp(req.ReceivedAt, false)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_received_at", "received_at must be RFC3339 or YYYY-MM-DD")
		return
	}
	actorID, actorType, actorRole := callerIdentity(r)
	idemKey := coalesce(req.IdempotencyKey, r.Header.Get("Idempotency-Key"))
	res, err := s.PostGoodsReceipt(r.Context(), PostGoodsReceiptInput{
		TenantID:       tenantID,
		ReceiptID:      req.ReceiptID,
		PoID:           req.PoID,
		EntityID:       req.EntityID,
		WarehouseID:    req.WarehouseID,
		PeriodID:       req.PeriodID,
		IdempotencyKey: idemKey,
		ReceivedAt:     receivedAt,
		Lines:          req.Lines,
		ReceivedBy:     actorID,
		ActorType:      actorType,
		ActorRole:      actorRole,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "p2p.receipt.post",
		ObjectType: AggregateGoodsReceipt,
		ObjectID:   res.Receipt.ReceiptID,
	})
	writeJSON(w, http.StatusCreated, res)
}

// HandleListVendorInvoices serves GET /v1/p2p/vendor-invoices.
func (s *Service) HandleListVendorInvoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	items, err := s.ListVendorInvoices(r.Context(), ListFilter{
		TenantID:    tenantID,
		PoID:        r.URL.Query().Get("po_id"),
		EntityID:    coalesce(r.URL.Query().Get("entity_id"), r.URL.Query().Get("entity")),
		MatchStatus: r.URL.Query().Get("match_status"),
		Limit:       parseQueryLimit(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if items == nil {
		items = []VendorInvoice{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":       tenantID,
		"vendor_invoices": items,
		"count":           len(items),
	})
}

// HandleCreateVendorInvoice serves POST /v1/p2p/vendor-invoices.
func (s *Service) HandleCreateVendorInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	var req createVendorInvoiceHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, coalesce(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	actorID, actorType, actorRole := callerIdentity(r)
	idemKey := coalesce(req.IdempotencyKey, r.Header.Get("Idempotency-Key"))
	inv, err := s.CreateVendorInvoice(r.Context(), CreateVendorInvoiceInput{
		TenantID:        tenantID,
		VendorInvoiceID: req.VendorInvoiceID,
		PoID:            req.PoID,
		ReceiptID:       req.ReceiptID,
		EsfDocID:        req.EsfDocID,
		EntityID:        req.EntityID,
		VendorID:        req.VendorID,
		Currency:        req.Currency,
		SubtotalMinor:   req.SubtotalMinor,
		VatMinor:        req.VatMinor,
		TotalMinor:      req.TotalMinor,
		IdempotencyKey:  idemKey,
		CreatedBy:       actorID,
		ActorType:       actorType,
		ActorRole:       actorRole,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "p2p.vendor_invoice.create",
		ObjectType: AggregateVendorInvoice,
		ObjectID:   inv.VendorInvoiceID,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"vendor_invoice": inv})
}

// HandleThreeWayMatch serves POST /v1/p2p/three-way-match.
func (s *Service) HandleThreeWayMatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", "P2P service unavailable")
		return
	}
	var req threeWayMatchHTTPReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tenantID, ok := resolveP2PTenantScope(w, r, coalesce(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	actorID, actorType, actorRole := callerIdentity(r)
	verifiedBy := coalesce(strings.TrimSpace(req.VerifiedBy), actorID)
	res, err := s.ThreeWayMatch(r.Context(), ThreeWayMatchInput{
		TenantID:            tenantID,
		VendorInvoiceID:     req.VendorInvoiceID,
		PoID:                req.PoID,
		ReceiptID:           req.ReceiptID,
		EsfDocID:            req.EsfDocID,
		PeriodID:            req.PeriodID,
		ToleranceBps:        req.ToleranceBps,
		ToleranceMinor:      req.ToleranceMinor,
		OverrideDiscrepancy: req.OverrideDiscrepancy,
		ApprovalID:          req.ApprovalID,
		VerifiedBy:          verifiedBy,
		ActorType:           actorType,
		ActorRole:           actorRole,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "p2p.three_way_match",
		ObjectType: AggregateVendorInvoice,
		ObjectID:   res.VendorInvoice.VendorInvoiceID,
		ApprovalID: req.ApprovalID,
	})
	writeJSON(w, http.StatusOK, res)
}

func rejectAuditorOrAgentMutation(w http.ResponseWriter, r *http.Request) bool {
	if auth.IsAuditorExpiredInContext(r.Context()) {
		writeJSONError(w, http.StatusUnauthorized, "auditor_access_expired", "auditor access has expired")
		return false
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return true
	}
	if auth.IsAuditorExpired(claims, time.Now().UTC()) {
		writeJSONError(w, http.StatusUnauthorized, "auditor_access_expired", "auditor access has expired")
		return false
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return false
	}
	if auth.IsAgentActor(claims) {
		writeJSONError(w, http.StatusForbidden, "agent_approval_forbidden", "AGENT actor cannot execute P2P write mutations")
		return false
	}
	return true
}

func resolveP2PTenantScope(w http.ResponseWriter, r *http.Request, requested string) (string, bool) {
	req := strings.TrimSpace(requested)
	ctx := r.Context()

	var callerTenant string
	if t, ok := auth.TenantFromContext(ctx); ok && strings.TrimSpace(t.SupplierID) != "" {
		callerTenant = strings.TrimSpace(t.SupplierID)
	} else if !auth.TenantContextEnforced() {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			callerTenant = strings.TrimSpace(sid)
		}
	}

	if auth.IsPlatformAdmin(ctx) {
		target := coalesce(req, callerTenant)
		if target == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_required", "tenant_id parameter is required for platform admin")
			return "", false
		}
		return target, true
	}
	if callerTenant == "" {
		writeJSONError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", false
	}
	if req != "" && req != callerTenant {
		writeJSONError(w, http.StatusForbidden, "tenant_scope_mismatch", "requested tenant_id does not match authenticated supplier tenant")
		return "", false
	}
	return callerTenant, true
}

func callerIdentity(r *http.Request) (actorID, actorType, actorRole string) {
	actorID = auth.ActorFromContext(r.Context())
	if c, ok := auth.FromContext(r.Context()); ok {
		actorType = auth.EffectiveActorType(c)
		actorRole = auth.EffectiveSupplierRole(c)
	}
	if actorType == "" {
		actorType = auth.ActorTypeHuman
	}
	return actorID, actorType, actorRole
}

func parseQueryLimit(r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return DefaultListLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return DefaultListLimit
	}
	return clampLimit(n)
}

func parseOptionalTimestamp(raw string, endOfDay bool) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		u := t.UTC()
		return &u, nil
	}
	if t, err := time.Parse("2006-01-02", trimmed); err == nil {
		u := t.UTC()
		if endOfDay {
			u = u.Add(24*time.Hour - time.Nanosecond)
		}
		return &u, nil
	}
	return nil, ErrInvalidInput
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return false
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 512*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	return true
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrSoDViolation):
		writeJSONError(w, http.StatusForbidden, "sod_violation", err.Error())
	case errors.Is(err, ErrForbidden):
		writeJSONError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, ErrApprovalRequired):
		writeJSONError(w, http.StatusForbidden, "approval_required", err.Error())
	case errors.Is(err, ErrApprovalFailed):
		writeJSONError(w, http.StatusConflict, "approval_failed", err.Error())
	case errors.Is(err, ErrQtyExceeded):
		writeJSONError(w, http.StatusUnprocessableEntity, "qty_exceeded", err.Error())
	case errors.Is(err, ErrPONotApproved):
		writeJSONError(w, http.StatusConflict, "po_not_approved", err.Error())
	case errors.Is(err, ErrConflict):
		writeJSONError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, ErrGLPosterUnavailable), errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "p2p_unavailable", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "p2p_internal_error", err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error":   message,
		"code":    code,
		"message": message,
	})
}
