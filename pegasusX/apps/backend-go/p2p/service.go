package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// ServiceConfig configures dependencies for the §6.6 P2P & Process Mining Service.
type ServiceConfig struct {
	Repo                     Repository
	GLPoster                 GLPoster
	Approvals                ApprovalChecker
	Audit                    AuditRecorder
	POApprovalThresholdMinor int64
	DefaultToleranceBps      int64
	DefaultToleranceMinor    int64
	Now                      func() time.Time
}

// Service orchestrates UZ process templates, instances, P2P lifecycle (PO, Goods Receipt,
// Vendor Invoice, 3-Way Match with GL clearing), and live Spanner Process Mining.
type Service struct {
	repo                     Repository
	glPoster                 GLPoster
	approvals                ApprovalChecker
	audit                    AuditRecorder
	poApprovalThresholdMinor int64
	defaultToleranceBps      int64
	defaultToleranceMinor    int64
	now                      func() time.Time
}

// NewService constructs a P2P Service with the given repository and config.
func NewService(repo Repository, cfg ServiceConfig) *Service {
	if repo != nil {
		cfg.Repo = repo
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &Service{
		repo:                     cfg.Repo,
		glPoster:                 cfg.GLPoster,
		approvals:                cfg.Approvals,
		audit:                    cfg.Audit,
		poApprovalThresholdMinor: maxInt64(cfg.POApprovalThresholdMinor, 0),
		defaultToleranceBps:      maxInt64(cfg.DefaultToleranceBps, 0),
		defaultToleranceMinor:    maxInt64(cfg.DefaultToleranceMinor, 0),
		now:                      nowFn,
	}
}

// NewSpannerService constructs a Spanner-backed P2P Service.
func NewSpannerService(client *spanner.Client, cfg ServiceConfig) *Service {
	if client != nil && cfg.Repo == nil {
		cfg.Repo = NewSpannerRepository(client)
	}
	return NewService(cfg.Repo, cfg)
}

// Repo returns the configured Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// SetGLPoster attaches the GLPoster implementation (e.g., *gl.Service adapter).
func (s *Service) SetGLPoster(p GLPoster) {
	if s != nil {
		s.glPoster = p
	}
}

// SetApprovalChecker attaches the optional ApprovalChecker implementation.
func (s *Service) SetApprovalChecker(a ApprovalChecker) {
	if s != nil {
		s.approvals = a
	}
}

// SetAuditRecorder attaches the optional AuditRecorder implementation.
func (s *Service) SetAuditRecorder(a AuditRecorder) {
	if s != nil {
		s.audit = a
	}
}

// SetPOApprovalThresholdMinor configures the optional minor-unit threshold above which POs require approval.
func (s *Service) SetPOApprovalThresholdMinor(thresholdMinor int64) {
	if s != nil && thresholdMinor >= 0 {
		s.poApprovalThresholdMinor = thresholdMinor
	}
}

// SetDefaultTolerances sets default 3-way match tolerances in bps and minor units.
func (s *Service) SetDefaultTolerances(bps, minor int64) {
	if s != nil && bps >= 0 && bps <= MaxToleranceBps && minor >= 0 {
		s.defaultToleranceBps = bps
		s.defaultToleranceMinor = minor
	}
}

func (s *Service) currentNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

// ListTemplates returns shipped UZ templates merged with tenant custom templates.
func (s *Service) ListTemplates(ctx context.Context, tenantID, domain string) ([]ProcessTemplate, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	return s.repo.ListTemplates(ctx, tid, domain)
}

// CreateTemplate validates and persists a custom tenant ProcessTemplate.
func (s *Service) CreateTemplate(ctx context.Context, in CreateTemplateInput) (ProcessTemplate, error) {
	if s == nil || s.repo == nil {
		return ProcessTemplate{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	domain := normalizeDomain(in.Domain)
	if tid == "" || code == "" || name == "" || (domain != DomainO2C && domain != DomainP2P && domain != DomainReturns) {
		return ProcessTemplate{}, fmt.Errorf("%w: tenant_id, code, name, and valid domain (O2C|P2P|RETURNS) are required", ErrInvalidInput)
	}
	if len(in.Steps) == 0 {
		return ProcessTemplate{}, fmt.Errorf("%w: at least one process step is required", ErrInvalidInput)
	}
	steps := make([]ProcessStep, 0, len(in.Steps))
	for i, st := range in.Steps {
		sc := strings.ToUpper(strings.TrimSpace(st.Code))
		sn := strings.TrimSpace(st.Name)
		if sc == "" {
			return ProcessTemplate{}, fmt.Errorf("%w: step[%d] code is required", ErrInvalidInput, i)
		}
		if sn == "" {
			sn = sc
		}
		if st.SLASeconds < 0 {
			return ProcessTemplate{}, fmt.Errorf("%w: step[%d] sla_seconds must be >= 0", ErrInvalidInput, i)
		}
		seq := st.Sequence
		if seq <= 0 {
			seq = int64(i + 1)
		}
		sla := st.SLASeconds
		if sla == 0 {
			sla = DefaultMiningSLASec
		}
		steps = append(steps, ProcessStep{
			Code:         sc,
			Name:         sn,
			Sequence:     seq,
			SLASeconds:   sla,
			RequiredRole: strings.ToUpper(strings.TrimSpace(st.RequiredRole)),
		})
	}
	now := s.currentNow()
	tplID := strings.TrimSpace(in.TemplateID)
	if tplID == "" {
		tplID = "ptpl_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = TemplateStatusActive
	}
	tpl := ProcessTemplate{
		TenantID:   tid,
		TemplateID: tplID,
		Code:       code,
		Name:       name,
		Domain:     domain,
		Steps:      steps,
		IsDefault:  in.IsDefault,
		BuiltIn:    false,
		Status:     status,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	saved, err := s.repo.SaveTemplate(ctx, tpl)
	if err != nil {
		return ProcessTemplate{}, err
	}
	s.recordAudit(ctx, tid, in.ActorID, in.ActorType, in.ActorRole, "p2p.template.create", "ProcessTemplate", saved.TemplateID, "", saved)
	return saved, nil
}

// ListInstances returns process instances matching filter.
func (s *Service) ListInstances(ctx context.Context, f ListFilter) ([]ProcessInstance, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	if strings.TrimSpace(f.TenantID) == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	return s.repo.ListInstances(ctx, f)
}

// StartOrAdvanceInstance creates a ProcessInstance or advances an existing one to next_step.
func (s *Service) StartOrAdvanceInstance(ctx context.Context, in StartOrAdvanceInstanceInput) (ProcessInstance, error) {
	if s == nil || s.repo == nil {
		return ProcessInstance{}, ErrRepoUnavailable
	}
	in.TenantID = strings.TrimSpace(in.TenantID)
	in.ObjectType = strings.TrimSpace(in.ObjectType)
	in.ObjectID = strings.TrimSpace(in.ObjectID)
	if in.TenantID == "" {
		return ProcessInstance{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if in.InstanceID == "" && (in.ObjectType == "" || in.ObjectID == "") {
		return ProcessInstance{}, fmt.Errorf("%w: instance_id or (object_type and object_id) is required", ErrInvalidInput)
	}
	templates, err := s.repo.ListTemplates(ctx, in.TenantID, "")
	if err != nil {
		return ProcessInstance{}, err
	}
	defDomain := inferDomainFromTemplate(templates, in.TemplateID, in.TemplateCode, in.ObjectType)
	tpl, ok := ResolveTemplate(templates, in.TemplateID, in.TemplateCode, defDomain)
	if !ok {
		return ProcessInstance{}, fmt.Errorf("%w: process template not found", ErrNotFound)
	}
	inst, err := s.repo.UpsertInstance(ctx, in, tpl, s.currentNow())
	if err != nil {
		return ProcessInstance{}, err
	}
	s.recordAudit(ctx, in.TenantID, in.ActorID, in.ActorType, in.ActorRole, "p2p.process_instance.upsert", "ProcessInstance", inst.InstanceID, "", inst)
	return inst, nil
}

// CreatePurchaseOrder validates multi-line PO totals using pure integer math,
// checks optional policy threshold/CEL approval, persists PO + lines + ProcessInstance, and emits p2p.po.created.
func (s *Service) CreatePurchaseOrder(ctx context.Context, in CreatePurchaseOrderInput) (CreatePurchaseOrderResult, error) {
	if s == nil || s.repo == nil {
		return CreatePurchaseOrderResult{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	vendorID := strings.TrimSpace(in.VendorID)
	if tid == "" || vendorID == "" {
		return CreatePurchaseOrderResult{}, fmt.Errorf("%w: tenant_id and vendor_id are required", ErrInvalidInput)
	}
	if len(in.Lines) == 0 {
		return CreatePurchaseOrderResult{}, fmt.Errorf("%w: at least one purchase order line is required", ErrInvalidInput)
	}
	if ik := strings.TrimSpace(in.IdempotencyKey); ik != "" {
		if existing, found, err := s.repo.FindPurchaseOrderByIdempotency(ctx, tid, ik); err != nil {
			return CreatePurchaseOrderResult{}, err
		} else if found {
			return CreatePurchaseOrderResult{
				PurchaseOrder:     existing,
				ApprovalRequired:  existing.Status == POStatusPendingApproval,
				ApprovalRequestID: existing.ApprovalRequestID,
			}, nil
		}
	}

	poID := strings.TrimSpace(in.PoID)
	if poID == "" {
		poID = "po_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	}
	cur := strings.ToUpper(strings.TrimSpace(in.Currency))
	if cur == "" {
		cur = "UZS"
	}
	if len(cur) != 3 {
		return CreatePurchaseOrderResult{}, fmt.Errorf("%w: currency must be 3-letter ISO code", ErrInvalidInput)
	}

	var subtotalMinor, vatMinor, totalMinor int64
	poLines := make([]PurchaseOrderLine, 0, len(in.Lines))
	for idx, rawLine := range in.Lines {
		sku := strings.TrimSpace(rawLine.Sku)
		if sku == "" {
			return CreatePurchaseOrderResult{}, fmt.Errorf("%w: line[%d] sku is required", ErrInvalidInput, idx)
		}
		vatBps := rawLine.VatBps
		net, vat, tot, err := ComputePOLineTotals(rawLine.QtyOrdered, rawLine.UnitCostMinor, vatBps)
		if err != nil {
			return CreatePurchaseOrderResult{}, fmt.Errorf("line[%d]: %w", idx, err)
		}
		subtotalMinor += net
		vatMinor += vat
		totalMinor += tot
		poLines = append(poLines, PurchaseOrderLine{
			TenantID:       tid,
			PoID:           poID,
			LineNum:        int64(idx + 1),
			Sku:            sku,
			ProductName:    strings.TrimSpace(rawLine.ProductName),
			QtyOrdered:     rawLine.QtyOrdered,
			QtyReceived:    0,
			UnitCostMinor:  rawLine.UnitCostMinor,
			VatBps:         vatBps,
			LineTotalMinor: tot,
		})
	}

	createdBy := coalesce(in.CreatedBy, "system")
	now := s.currentNow()
	status := POStatusApproved
	var approvalReqID string
	var approvalRequired bool

	needsThresholdApproval := s.poApprovalThresholdMinor > 0 && totalMinor > s.poApprovalThresholdMinor
	if s.approvals != nil && (needsThresholdApproval || strings.TrimSpace(in.ApprovalID) != "" || s.poApprovalThresholdMinor == 0) {
		canonPayload := map[string]any{
			"po_id":          poID,
			"tenant_id":      tid,
			"entity_id":      strings.TrimSpace(in.EntityID),
			"vendor_id":      vendorID,
			"currency":       cur,
			"subtotal_minor": subtotalMinor,
			"vat_minor":      vatMinor,
			"total_minor":    totalMinor,
			"line_count":     int64(len(poLines)),
		}
		res, aErr := s.approvals.CheckOrSubmitApproval(ctx, ApprovalCheckInput{
			TenantID:       tid,
			ObjectType:     AggregatePurchaseOrder,
			ObjectID:       poID,
			Action:         DefaultApproveAction,
			AmountMinor:    totalMinor,
			RequestedBy:    createdBy,
			ActorType:      in.ActorType,
			ApproverRole:   in.ActorRole,
			ApprovalID:     strings.TrimSpace(in.ApprovalID),
			Payload:        canonPayload,
			Context:        map[string]any{"vendor_id": vendorID, "currency": cur, "total_minor": totalMinor},
			IdempotencyKey: "p2p:po:" + poID,
			ForceCreate:    needsThresholdApproval && strings.TrimSpace(in.ApprovalID) == "",
		})
		if aErr != nil {
			if strings.TrimSpace(in.ApprovalID) != "" {
				return CreatePurchaseOrderResult{}, fmt.Errorf("%w: %v", ErrApprovalFailed, aErr)
			}
			return CreatePurchaseOrderResult{}, aErr
		}
		if res.Required {
			for _, appID := range res.ApproverIDs {
				if strings.EqualFold(appID, createdBy) {
					return CreatePurchaseOrderResult{}, ErrSoDViolation
				}
			}
			approvalReqID = res.ApprovalRequestID
			if !res.Approved {
				status = POStatusPendingApproval
				approvalRequired = true
			}
		}
	} else if needsThresholdApproval && strings.TrimSpace(in.ApprovalID) == "" {
		status = POStatusPendingApproval
		approvalRequired = true
	}

	po := PurchaseOrder{
		TenantID:          tid,
		PoID:              poID,
		EntityID:          strings.TrimSpace(in.EntityID),
		VendorID:          vendorID,
		VendorName:        strings.TrimSpace(in.VendorName),
		VendorTin:         strings.TrimSpace(in.VendorTin),
		Currency:          cur,
		SubtotalMinor:     subtotalMinor,
		VatMinor:          vatMinor,
		TotalMinor:        totalMinor,
		Status:            status,
		ApprovalRequestID: approvalReqID,
		IdempotencyKey:    strings.TrimSpace(in.IdempotencyKey),
		ExpectedAt:        in.ExpectedAt,
		CreatedBy:         createdBy,
		CreatedAt:         now,
		UpdatedAt:         now,
		Lines:             poLines,
	}

	templates, err := s.repo.ListTemplates(ctx, tid, DomainP2P)
	if err != nil {
		return CreatePurchaseOrderResult{}, err
	}
	tpl, _ := ResolveTemplate(templates, "", coalesce(in.TemplateCode, TemplateP2PStdUZ), DomainP2P)

	afterBytes, _ := json.Marshal(po)
	hooks := TxnHooks{
		Audit: s.audit,
		AuditEvent: &AuditRecordInput{
			TenantID:   tid,
			ActorID:    createdBy,
			ActorType:  coalesce(in.ActorType, "human"),
			Role:       in.ActorRole,
			Action:     "p2p.po.create",
			ObjectType: AggregatePurchaseOrder,
			ObjectID:   poID,
			AfterJSON:  string(afterBytes),
			ApprovalID: approvalReqID,
		},
		OutboxEmit: func(emitCtx context.Context, buf outbox.TxnBuffer, _ string) error {
			return emitP2POutbox(emitCtx, buf, EventP2PPOCreated, AggregatePurchaseOrder, poID, events.P2PEvent{
				BaseEvent: events.BaseEvent{
					Type:      EventP2PPOCreated,
					Version:   1,
					Timestamp: now.Format(time.RFC3339),
				},
				TenantID:          tid,
				SupplierID:        tid,
				EntityID:          po.EntityID,
				PoID:              po.PoID,
				VendorID:          po.VendorID,
				Status:            po.Status,
				SubtotalMinor:     po.SubtotalMinor,
				VatMinor:          po.VatMinor,
				TotalMinor:        po.TotalMinor,
				Currency:          po.Currency,
				LineCount:         int64(len(po.Lines)),
				ApprovalRequestID: po.ApprovalRequestID,
				ActorID:           createdBy,
				OccurredAt:        now.Format(time.RFC3339),
			})
		},
	}

	saved, alreadyExisted, err := s.repo.CreatePurchaseOrder(ctx, po, tpl, hooks)
	if err != nil {
		return CreatePurchaseOrderResult{}, err
	}
	if alreadyExisted {
		approvalRequired = saved.Status == POStatusPendingApproval
		approvalReqID = saved.ApprovalRequestID
	}
	return CreatePurchaseOrderResult{
		PurchaseOrder:     saved,
		ApprovalRequired:  approvalRequired,
		ApprovalRequestID: approvalReqID,
	}, nil
}

// ApprovePurchaseOrder transitions a PENDING_APPROVAL PO to APPROVED while enforcing SoD (CreatedBy != ActorID).
func (s *Service) ApprovePurchaseOrder(ctx context.Context, in ApprovePurchaseOrderInput) (PurchaseOrder, error) {
	if s == nil || s.repo == nil {
		return PurchaseOrder{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	poID := strings.TrimSpace(in.PoID)
	actorID := strings.TrimSpace(in.ActorID)
	if tid == "" || poID == "" || actorID == "" {
		return PurchaseOrder{}, fmt.Errorf("%w: tenant_id, po_id, and actor_id are required", ErrInvalidInput)
	}

	po, found, err := s.repo.GetPurchaseOrder(ctx, tid, poID)
	if err != nil {
		return PurchaseOrder{}, err
	}
	if !found {
		return PurchaseOrder{}, ErrNotFound
	}
	if po.CreatedBy != "" && strings.EqualFold(po.CreatedBy, actorID) {
		s.recordAudit(ctx, tid, actorID, in.ActorType, in.ActorRole, "p2p.po.approve.sod_blocked", AggregatePurchaseOrder, poID, "sod_violation", map[string]any{"created_by": po.CreatedBy, "actor_id": actorID})
		return PurchaseOrder{}, ErrSoDViolation
	}

	if strings.TrimSpace(in.ApprovalID) != "" && s.approvals != nil {
		canonPayload := map[string]any{
			"po_id":          po.PoID,
			"tenant_id":      tid,
			"entity_id":      po.EntityID,
			"vendor_id":      po.VendorID,
			"currency":       po.Currency,
			"subtotal_minor": po.SubtotalMinor,
			"vat_minor":      po.VatMinor,
			"total_minor":    po.TotalMinor,
			"line_count":     int64(len(po.Lines)),
		}
		res, aErr := s.approvals.CheckOrSubmitApproval(ctx, ApprovalCheckInput{
			TenantID:     tid,
			ObjectType:   AggregatePurchaseOrder,
			ObjectID:     poID,
			Action:       DefaultApproveAction,
			AmountMinor:  po.TotalMinor,
			RequestedBy:  po.CreatedBy,
			ActorType:    in.ActorType,
			ApproverRole: in.ActorRole,
			ApprovalID:   strings.TrimSpace(in.ApprovalID),
			Payload:      canonPayload,
		})
		if aErr != nil {
			return PurchaseOrder{}, fmt.Errorf("%w: %v", ErrApprovalFailed, aErr)
		}
		if !res.Approved {
			return PurchaseOrder{}, ErrApprovalRequired
		}
	}

	now := s.currentNow()
	hooks := TxnHooks{
		Audit: s.audit,
		AuditEvent: &AuditRecordInput{
			TenantID:   tid,
			ActorID:    actorID,
			ActorType:  coalesce(in.ActorType, "human"),
			Role:       in.ActorRole,
			Action:     "p2p.po.approve",
			ObjectType: AggregatePurchaseOrder,
			ObjectID:   poID,
			ApprovalID: strings.TrimSpace(in.ApprovalID),
		},
	}
	return s.repo.ApprovePurchaseOrder(ctx, tid, poID, actorID, strings.TrimSpace(in.ApprovalID), now, hooks)
}

// GetPurchaseOrder retrieves one PO with its lines.
func (s *Service) GetPurchaseOrder(ctx context.Context, tenantID, poID string) (PurchaseOrder, error) {
	if s == nil || s.repo == nil {
		return PurchaseOrder{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	pid := strings.TrimSpace(poID)
	if tid == "" || pid == "" {
		return PurchaseOrder{}, fmt.Errorf("%w: tenant_id and po_id are required", ErrInvalidInput)
	}
	po, ok, err := s.repo.GetPurchaseOrder(ctx, tid, pid)
	if err != nil {
		return PurchaseOrder{}, err
	}
	if !ok {
		return PurchaseOrder{}, ErrNotFound
	}
	return po, nil
}

// ListPurchaseOrders lists purchase orders matching f.
func (s *Service) ListPurchaseOrders(ctx context.Context, f ListFilter) ([]PurchaseOrder, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	if strings.TrimSpace(f.TenantID) == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	return s.repo.ListPurchaseOrders(ctx, f)
}

// PostGoodsReceipt receives PO quantities, updates PurchaseOrderLines.QtyReceived & PO status,
// posts a balanced GL entry (Debit INVENTORY / Credit GRNI), and emits p2p.receipt.posted.
func (s *Service) PostGoodsReceipt(ctx context.Context, in PostGoodsReceiptInput) (PostGoodsReceiptResult, error) {
	if s == nil || s.repo == nil {
		return PostGoodsReceiptResult{}, ErrRepoUnavailable
	}
	if s.glPoster == nil {
		return PostGoodsReceiptResult{}, ErrGLPosterUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	poID := strings.TrimSpace(in.PoID)
	whID := strings.TrimSpace(in.WarehouseID)
	if tid == "" || poID == "" || whID == "" {
		return PostGoodsReceiptResult{}, fmt.Errorf("%w: tenant_id, po_id, and warehouse_id are required", ErrInvalidInput)
	}
	if len(in.Lines) == 0 {
		return PostGoodsReceiptResult{}, fmt.Errorf("%w: at least one goods receipt line is required", ErrInvalidInput)
	}
	if ik := strings.TrimSpace(in.IdempotencyKey); ik != "" {
		if existingRec, found, err := s.repo.FindGoodsReceiptByIdempotency(ctx, tid, ik); err != nil {
			return PostGoodsReceiptResult{}, err
		} else if found {
			po, _, pErr := s.repo.GetPurchaseOrder(ctx, tid, existingRec.PoID)
			if pErr != nil {
				return PostGoodsReceiptResult{}, pErr
			}
			return PostGoodsReceiptResult{
				Receipt:        existingRec,
				PurchaseOrder:  po,
				JournalEntryID: existingRec.JournalEntryID,
			}, nil
		}
	}

	po, found, err := s.repo.GetPurchaseOrder(ctx, tid, poID)
	if err != nil {
		return PostGoodsReceiptResult{}, err
	}
	if !found {
		return PostGoodsReceiptResult{}, ErrNotFound
	}
	if po.Status == POStatusPendingApproval || po.Status == POStatusCancelled || po.Status == POStatusClosed {
		return PostGoodsReceiptResult{}, fmt.Errorf("%w: status=%s", ErrPONotApproved, po.Status)
	}

	now := s.currentNow()
	receivedAt := now
	if in.ReceivedAt != nil && !in.ReceivedAt.IsZero() {
		receivedAt = in.ReceivedAt.UTC()
	}
	receiptID := strings.TrimSpace(in.ReceiptID)
	if receiptID == "" {
		receiptID = "grn_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	}

	reqLines := make([]GoodsReceiptLine, 0, len(in.Lines))
	for _, l := range in.Lines {
		if l.QtyReceived <= 0 {
			return PostGoodsReceiptResult{}, fmt.Errorf("%w: qty_received must be > 0", ErrInvalidInput)
		}
		reqLines = append(reqLines, GoodsReceiptLine{
			LineNum:     l.LineNum,
			Sku:         strings.TrimSpace(l.Sku),
			QtyReceived: l.QtyReceived,
		})
	}
	previewLines, _, previewNet, err := applyReceiptLinesToPO(po, reqLines, receivedAt)
	if err != nil {
		return PostGoodsReceiptResult{}, err
	}

	entityID := coalesce(strings.TrimSpace(in.EntityID), po.EntityID)
	receivedBy := coalesce(in.ReceivedBy, "system")
	periodID := strings.TrimSpace(in.PeriodID)
	if periodID == "" {
		periodID = receivedAt.UTC().Format("2006-01")
	}

	receipt := GoodsReceipt{
		TenantID:           tid,
		ReceiptID:          receiptID,
		PoID:               poID,
		EntityID:           entityID,
		WarehouseID:        whID,
		Status:             ReceiptStatusPosted,
		Lines:              previewLines,
		TotalReceivedMinor: previewNet,
		IdempotencyKey:     strings.TrimSpace(in.IdempotencyKey),
		ReceivedBy:         receivedBy,
		ReceivedAt:         receivedAt,
		CreatedAt:          now,
	}

	glIn := &GLPostInput{
		TenantID:       tid,
		EntityID:       entityID,
		PeriodID:       periodID,
		SourceType:     "P2P_RECEIPT",
		SourceID:       receiptID,
		Memo:           "Goods Receipt " + receiptID + " for PO " + poID,
		Currency:       coalesce(po.Currency, "UZS"),
		IdempotencyKey: "p2p:receipt:" + receiptID,
		PostedBy:       receivedBy,
		OccurredAt:     receivedAt,
		Lines: []GLLine{
			{
				AccountCode: AccountInventory,
				Direction:   DirectionDebit,
				AmountMinor: previewNet,
				WarehouseID: whID,
				Description: "Inventory received on GRN " + receiptID,
			},
			{
				AccountCode: AccountGRNI,
				Direction:   DirectionCredit,
				AmountMinor: previewNet,
				PartyType:   "VENDOR",
				PartyID:     po.VendorID,
				WarehouseID: whID,
				Description: "Goods Received Not Invoiced for PO " + poID,
			},
		},
	}

	afterBytes, _ := json.Marshal(receipt)
	hooks := TxnHooks{
		GLPoster: s.glPoster,
		GLInput:  glIn,
		Audit:    s.audit,
		AuditEvent: &AuditRecordInput{
			TenantID:   tid,
			ActorID:    receivedBy,
			ActorType:  coalesce(in.ActorType, "human"),
			Role:       in.ActorRole,
			Action:     "p2p.receipt.post",
			ObjectType: AggregateGoodsReceipt,
			ObjectID:   receiptID,
			AfterJSON:  string(afterBytes),
		},
		OutboxEmit: func(emitCtx context.Context, buf outbox.TxnBuffer, journalEntryID string) error {
			return emitP2POutbox(emitCtx, buf, EventP2PReceiptPosted, AggregateGoodsReceipt, receiptID, events.P2PEvent{
				BaseEvent: events.BaseEvent{
					Type:      EventP2PReceiptPosted,
					Version:   1,
					Timestamp: now.Format(time.RFC3339),
				},
				TenantID:           tid,
				SupplierID:         tid,
				EntityID:           entityID,
				PoID:               poID,
				ReceiptID:          receiptID,
				VendorID:           po.VendorID,
				WarehouseID:        whID,
				Status:             ReceiptStatusPosted,
				TotalReceivedMinor: previewNet,
				Currency:           coalesce(po.Currency, "UZS"),
				JournalEntryID:     journalEntryID,
				ActorID:            receivedBy,
				OccurredAt:         receivedAt.Format(time.RFC3339),
			})
		},
	}

	savedReceipt, updatedPO, _, err := s.repo.PostGoodsReceipt(ctx, receipt, hooks)
	if err != nil {
		return PostGoodsReceiptResult{}, err
	}
	return PostGoodsReceiptResult{
		Receipt:        savedReceipt,
		PurchaseOrder:  updatedPO,
		JournalEntryID: savedReceipt.JournalEntryID,
	}, nil
}

// ListGoodsReceipts lists goods receipts matching f.
func (s *Service) ListGoodsReceipts(ctx context.Context, f ListFilter) ([]GoodsReceipt, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	if strings.TrimSpace(f.TenantID) == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	return s.repo.ListGoodsReceipts(ctx, f)
}

// CreateVendorInvoice records a vendor invoice linked to a PO (+ optional GoodsReceipt and Inbound ESF).
func (s *Service) CreateVendorInvoice(ctx context.Context, in CreateVendorInvoiceInput) (VendorInvoice, error) {
	if s == nil || s.repo == nil {
		return VendorInvoice{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	poID := strings.TrimSpace(in.PoID)
	if tid == "" || poID == "" {
		return VendorInvoice{}, fmt.Errorf("%w: tenant_id and po_id are required", ErrInvalidInput)
	}
	if in.SubtotalMinor <= 0 || in.VatMinor < 0 || in.TotalMinor <= 0 || in.SubtotalMinor+in.VatMinor != in.TotalMinor {
		return VendorInvoice{}, fmt.Errorf("%w: subtotal_minor > 0, vat_minor >= 0, and subtotal_minor + vat_minor == total_minor required", ErrInvalidInput)
	}
	if ik := strings.TrimSpace(in.IdempotencyKey); ik != "" {
		if existingInv, found, err := s.repo.FindVendorInvoiceByIdempotency(ctx, tid, ik); err != nil {
			return VendorInvoice{}, err
		} else if found {
			return existingInv, nil
		}
	}

	invID := strings.TrimSpace(in.VendorInvoiceID)
	if invID == "" {
		invID = "vinv_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	}
	now := s.currentNow()
	createdBy := coalesce(in.CreatedBy, "system")

	inv := VendorInvoice{
		TenantID:        tid,
		VendorInvoiceID: invID,
		PoID:            poID,
		ReceiptID:       strings.TrimSpace(in.ReceiptID),
		EsfDocID:        strings.TrimSpace(in.EsfDocID),
		EntityID:        strings.TrimSpace(in.EntityID),
		VendorID:        strings.TrimSpace(in.VendorID),
		Currency:        strings.ToUpper(strings.TrimSpace(in.Currency)),
		SubtotalMinor:   in.SubtotalMinor,
		VatMinor:        in.VatMinor,
		TotalMinor:      in.TotalMinor,
		MatchStatus:     MatchStatusPending,
		IdempotencyKey:  strings.TrimSpace(in.IdempotencyKey),
		CreatedBy:       createdBy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	afterBytes, _ := json.Marshal(inv)
	hooks := TxnHooks{
		Audit: s.audit,
		AuditEvent: &AuditRecordInput{
			TenantID:   tid,
			ActorID:    createdBy,
			ActorType:  coalesce(in.ActorType, "human"),
			Role:       in.ActorRole,
			Action:     "p2p.vendor_invoice.create",
			ObjectType: AggregateVendorInvoice,
			ObjectID:   invID,
			AfterJSON:  string(afterBytes),
		},
	}

	saved, _, err := s.repo.CreateVendorInvoice(ctx, inv, hooks)
	return saved, err
}

// ListVendorInvoices lists vendor invoices matching f.
func (s *Service) ListVendorInvoices(ctx context.Context, f ListFilter) ([]VendorInvoice, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	if strings.TrimSpace(f.TenantID) == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	return s.repo.ListVendorInvoices(ctx, f)
}

// ThreeWayMatch validates PO <-> GoodsReceipt <-> VendorInvoice (+ optional Inbound ESF) within
// configurable bps/minor tolerances. Enforces SoD (CreatedBy != VerifiedBy), never silently auto-approves
// discrepancies, and posts balanced GL (GRNI + TAX:VAT:INPUT -> AP:VENDOR) on MATCHED_3WAY.
func (s *Service) ThreeWayMatch(ctx context.Context, in ThreeWayMatchInput) (ThreeWayMatchResult, error) {
	if s == nil || s.repo == nil {
		return ThreeWayMatchResult{}, ErrRepoUnavailable
	}
	if s.glPoster == nil {
		return ThreeWayMatchResult{}, ErrGLPosterUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	invID := strings.TrimSpace(in.VendorInvoiceID)
	if tid == "" || invID == "" {
		return ThreeWayMatchResult{}, fmt.Errorf("%w: tenant_id and vendor_invoice_id are required", ErrInvalidInput)
	}
	if in.ToleranceBps < 0 || in.ToleranceBps > MaxToleranceBps || in.ToleranceMinor < 0 {
		return ThreeWayMatchResult{}, fmt.Errorf("%w: tolerance_bps must be in [0,%d] and tolerance_minor >= 0", ErrInvalidInput, MaxToleranceBps)
	}

	inv, found, err := s.repo.GetVendorInvoice(ctx, tid, invID)
	if err != nil {
		return ThreeWayMatchResult{}, err
	}
	if !found {
		return ThreeWayMatchResult{}, fmt.Errorf("%w: vendor invoice %s not found", ErrNotFound, invID)
	}
	if inv.MatchStatus == MatchStatusMatched3Way {
		return ThreeWayMatchResult{}, fmt.Errorf("%w: vendor invoice %s is already MATCHED_3WAY", ErrConflict, invID)
	}

	poID := coalesce(strings.TrimSpace(in.PoID), inv.PoID)
	if poID != inv.PoID {
		return ThreeWayMatchResult{}, fmt.Errorf("%w: po_id mismatch (%s != %s)", ErrInvalidInput, poID, inv.PoID)
	}
	po, poFound, err := s.repo.GetPurchaseOrder(ctx, tid, poID)
	if err != nil {
		return ThreeWayMatchResult{}, err
	}
	if !poFound {
		return ThreeWayMatchResult{}, fmt.Errorf("%w: purchase order %s not found", ErrNotFound, poID)
	}

	verifiedBy := coalesce(strings.TrimSpace(in.VerifiedBy), "system")
	if (inv.CreatedBy != "" && strings.EqualFold(verifiedBy, inv.CreatedBy)) ||
		(po.CreatedBy != "" && strings.EqualFold(verifiedBy, po.CreatedBy)) {
		s.recordAudit(ctx, tid, verifiedBy, in.ActorType, in.ActorRole, "p2p.three_way_match.sod_blocked", AggregateVendorInvoice, invID, "sod_violation", map[string]any{
			"invoice_created_by": inv.CreatedBy,
			"po_created_by":      po.CreatedBy,
			"verified_by":        verifiedBy,
		})
		return ThreeWayMatchResult{}, ErrSoDViolation
	}

	allReceipts, err := s.repo.ListGoodsReceipts(ctx, ListFilter{TenantID: tid, PoID: poID, Limit: MaxListLimit})
	if err != nil {
		return ThreeWayMatchResult{}, err
	}
	receiptID := coalesce(strings.TrimSpace(in.ReceiptID), inv.ReceiptID)
	var receipts []GoodsReceipt
	if receiptID != "" {
		for _, r := range allReceipts {
			if r.ReceiptID == receiptID {
				receipts = append(receipts, r)
				break
			}
		}
		if len(receipts) == 0 {
			return ThreeWayMatchResult{}, fmt.Errorf("%w: goods receipt %s not found for po %s", ErrNotFound, receiptID, poID)
		}
	} else {
		receipts = allReceipts
		if len(receipts) == 1 {
			receiptID = receipts[0].ReceiptID
		}
	}

	esfDocID := coalesce(strings.TrimSpace(in.EsfDocID), inv.EsfDocID)
	var esfPtr *InboundESFDoc
	if esfDocID != "" {
		doc, ok, eErr := s.repo.GetInboundESF(ctx, tid, esfDocID)
		if eErr != nil {
			return ThreeWayMatchResult{}, eErr
		}
		if !ok {
			return ThreeWayMatchResult{}, fmt.Errorf("%w: inbound esf document %s not found", ErrNotFound, esfDocID)
		}
		esfPtr = &doc
	}

	tolBps := in.ToleranceBps
	if tolBps == 0 {
		tolBps = s.defaultToleranceBps
	}
	tolMinor := in.ToleranceMinor
	if tolMinor == 0 {
		tolMinor = s.defaultToleranceMinor
	}

	eval := EvaluateThreeWayMatch(ThreeWayMatchEvalInput{
		PurchaseOrder:  po,
		Receipts:       receipts,
		VendorInvoice:  inv,
		InboundESF:     esfPtr,
		ToleranceBps:   tolBps,
		ToleranceMinor: tolMinor,
	})

	now := s.currentNow()
	matchOK := eval.Matched
	if !matchOK && in.OverrideDiscrepancy {
		if s.approvals == nil || strings.TrimSpace(in.ApprovalID) == "" {
			return ThreeWayMatchResult{}, fmt.Errorf("%w: discrepancy override requires verified approval_id", ErrApprovalRequired)
		}
		appRes, aErr := s.approvals.CheckOrSubmitApproval(ctx, ApprovalCheckInput{
			TenantID:     tid,
			ObjectType:   AggregateVendorInvoice,
			ObjectID:     invID,
			Action:       DefaultOverrideAction,
			AmountMinor:  inv.TotalMinor,
			RequestedBy:  verifiedBy,
			ActorType:    in.ActorType,
			ApproverRole: in.ActorRole,
			ApprovalID:   strings.TrimSpace(in.ApprovalID),
		})
		if aErr != nil {
			return ThreeWayMatchResult{}, fmt.Errorf("%w: %v", ErrApprovalFailed, aErr)
		}
		if !appRes.Approved {
			return ThreeWayMatchResult{}, ErrApprovalRequired
		}
		for _, appID := range appRes.ApproverIDs {
			if strings.EqualFold(appID, inv.CreatedBy) || strings.EqualFold(appID, po.CreatedBy) {
				return ThreeWayMatchResult{}, ErrSoDViolation
			}
		}
		matchOK = true
		eval.VarianceReason = "OVERRIDE_APPROVED: " + eval.VarianceReason
	}

	updatedInv := inv
	updatedInv.ReceiptID = receiptID
	updatedInv.EsfDocID = esfDocID
	updatedInv.VarianceMinor = eval.VarianceMinor
	updatedInv.VarianceReason = eval.VarianceReason
	updatedInv.UpdatedAt = now

	entityID := coalesce(inv.EntityID, po.EntityID)
	periodID := strings.TrimSpace(in.PeriodID)
	if periodID == "" {
		periodID = now.UTC().Format("2006-01")
	}

	var hooks TxnHooks
	hooks.ActorID = verifiedBy
	hooks.Audit = s.audit

	if !matchOK {
		updatedInv.MatchStatus = MatchStatusDiscrepancy
		afterBytes, _ := json.Marshal(updatedInv)
		hooks.AuditEvent = &AuditRecordInput{
			TenantID:   tid,
			ActorID:    verifiedBy,
			ActorType:  coalesce(in.ActorType, "human"),
			Role:       in.ActorRole,
			Action:     "p2p.three_way_match.discrepancy",
			ObjectType: AggregateVendorInvoice,
			ObjectID:   invID,
			AfterJSON:  string(afterBytes),
			ReasonCode: "discrepancy",
		}
		savedInv, _, err := s.repo.ApplyThreeWayMatch(ctx, tid, invID, updatedInv, false, now, hooks)
		if err != nil {
			return ThreeWayMatchResult{}, err
		}
		return ThreeWayMatchResult{
			VendorInvoice:  savedInv,
			Matched:        false,
			MatchStatus:    savedInv.MatchStatus,
			VarianceMinor:  savedInv.VarianceMinor,
			VarianceReason: savedInv.VarianceReason,
			VerifiedBy:     verifiedBy,
		}, nil
	}

	glLines, err := BuildThreeWayMatchGLLines(po, inv, eval.ReceiptNetMinor)
	if err != nil {
		return ThreeWayMatchResult{}, err
	}
	updatedInv.MatchStatus = MatchStatusMatched3Way
	hooks.GLPoster = s.glPoster
	hooks.GLInput = &GLPostInput{
		TenantID:       tid,
		EntityID:       entityID,
		PeriodID:       periodID,
		SourceType:     "P2P_3WAY_MATCH",
		SourceID:       invID,
		Memo:           "3-Way Match AP clearing for Vendor Invoice " + invID + " / PO " + poID,
		Currency:       coalesce(inv.Currency, po.Currency, "UZS"),
		IdempotencyKey: "p2p:3way:" + invID,
		PostedBy:       verifiedBy,
		OccurredAt:     now,
		Lines:          glLines,
	}
	afterBytes, _ := json.Marshal(updatedInv)
	hooks.AuditEvent = &AuditRecordInput{
		TenantID:   tid,
		ActorID:    verifiedBy,
		ActorType:  coalesce(in.ActorType, "human"),
		Role:       in.ActorRole,
		Action:     "p2p.three_way_match.completed",
		ObjectType: AggregateVendorInvoice,
		ObjectID:   invID,
		AfterJSON:  string(afterBytes),
		ApprovalID: strings.TrimSpace(in.ApprovalID),
	}
	hooks.OutboxEmit = func(emitCtx context.Context, buf outbox.TxnBuffer, journalEntryID string) error {
		return emitP2POutbox(emitCtx, buf, EventP2PThreeWayMatchComplete, AggregateVendorInvoice, invID, events.P2PEvent{
			BaseEvent: events.BaseEvent{
				Type:      EventP2PThreeWayMatchComplete,
				Version:   1,
				Timestamp: now.Format(time.RFC3339),
			},
			TenantID:        tid,
			SupplierID:      tid,
			EntityID:        entityID,
			PoID:            poID,
			ReceiptID:       receiptID,
			VendorInvoiceID: invID,
			EsfDocID:        esfDocID,
			VendorID:        coalesce(inv.VendorID, po.VendorID),
			Status:          MatchStatusMatched3Way,
			MatchStatus:     MatchStatusMatched3Way,
			SubtotalMinor:   inv.SubtotalMinor,
			VatMinor:        inv.VatMinor,
			TotalMinor:      inv.TotalMinor,
			VarianceMinor:   eval.VarianceMinor,
			VarianceReason:  eval.VarianceReason,
			Currency:        coalesce(inv.Currency, po.Currency, "UZS"),
			JournalEntryID:  journalEntryID,
			ActorID:         verifiedBy,
			OccurredAt:      now.Format(time.RFC3339),
		})
	}

	savedInv, instPtr, err := s.repo.ApplyThreeWayMatch(ctx, tid, invID, updatedInv, true, now, hooks)
	if err != nil {
		return ThreeWayMatchResult{}, err
	}
	return ThreeWayMatchResult{
		VendorInvoice:   savedInv,
		Matched:         true,
		MatchStatus:     savedInv.MatchStatus,
		VarianceMinor:   savedInv.VarianceMinor,
		VarianceReason:  savedInv.VarianceReason,
		JournalEntryID:  savedInv.JournalEntryID,
		VerifiedBy:      verifiedBy,
		ProcessInstance: instPtr,
	}, nil
}

// GetProcessMining computes live transition edge counts, median/p95 durations, SLA breaches, and
// bottleneck stage over real Spanner tables using ReadOnlyTransaction(). Never fabricates metrics when empty.
func (s *Service) GetProcessMining(ctx context.Context, f MiningFilter) (ProcessMiningReport, error) {
	if s == nil || s.repo == nil {
		return ProcessMiningReport{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(f.TenantID)
	if tid == "" {
		return ProcessMiningReport{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	f.TenantID = tid
	f.EntityID = strings.TrimSpace(f.EntityID)
	f.Domain = normalizeDomain(f.Domain)
	if f.Domain != "" && f.Domain != DomainO2C && f.Domain != DomainP2P && f.Domain != DomainReturns {
		return ProcessMiningReport{}, fmt.Errorf("%w: domain must be O2C, P2P, or RETURNS", ErrInvalidInput)
	}
	templates, err := s.repo.ListTemplates(ctx, tid, "")
	if err != nil {
		return ProcessMiningReport{}, err
	}
	samples, err := s.repo.CollectMiningSamples(ctx, f, templates)
	if err != nil {
		return ProcessMiningReport{}, err
	}
	return AggregateMiningSamples(tid, f.EntityID, f.Domain, samples), nil
}

func (s *Service) recordAudit(ctx context.Context, tenantID, actorID, actorType, role, action, objectType, objectID, reasonCode string, after any) {
	if s == nil || s.audit == nil {
		return
	}
	var afterJSON string
	if after != nil {
		if raw, err := json.Marshal(after); err == nil {
			afterJSON = string(raw)
		}
	}
	_ = s.audit.RecordEvent(ctx, AuditRecordInput{
		TenantID:   tenantID,
		ActorID:    coalesce(actorID, "system"),
		ActorType:  coalesce(actorType, "human"),
		Role:       role,
		Action:     action,
		ObjectType: objectType,
		ObjectID:   objectID,
		AfterJSON:  afterJSON,
		ReasonCode: reasonCode,
	})
}

func emitP2POutbox(ctx context.Context, buf outbox.TxnBuffer, eventType, aggType, aggID string, payload events.P2PEvent) error {
	if buf == nil {
		return nil
	}
	topic := events.DomainTopicForEventType(eventType)
	if strings.TrimSpace(topic) == "" {
		topic = events.TopicMain
	}
	return outbox.EmitJSON(ctx, buf, aggType, aggID, topic, payload)
}
