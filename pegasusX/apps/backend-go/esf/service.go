package esf

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnote"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/fiscal"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
)

// ServiceConfig configures dependencies for ESF lifecycle management.
type ServiceConfig struct {
	Repo          Repository
	SoliqClient   soliq.SoliqClient
	Signer        fiscal.EDSSigner
	CreditNoteSvc *creditnote.Service
	Now           func() time.Time
}

// Service implements M1.4 first-class ESF documents, Soliq submission,
// status lifecycle, GL VAT reconciliation, and IKPU compliance audits.
type Service struct {
	repo          Repository
	soliqClient   soliq.SoliqClient
	signer        fiscal.EDSSigner
	creditNoteSvc *creditnote.Service
	now           func() time.Time
}

// NewService constructs a Service from explicit config.
func NewService(cfg ServiceConfig) *Service {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &Service{
		repo:          cfg.Repo,
		soliqClient:   cfg.SoliqClient,
		signer:        cfg.Signer,
		creditNoteSvc: cfg.CreditNoteSvc,
		now:           nowFn,
	}
}

// NewServiceWithRepository creates a Service backed by the supplied Repository and config.
func NewServiceWithRepository(repo Repository, cfg ServiceConfig) *Service {
	cfg.Repo = repo
	return NewService(cfg)
}

// Repo returns the underlying ESF Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// CreateOutbound builds an outbound ESF document from Orders + OrderLineFiscalSnapshots + TaxRegimeVersions,
// EDS-signs the canonical payload, submits via SoliqClient (preserving circuit breaker),
// and persists the document, status history, and esf.submitted outbox event.
func (s *Service) CreateOutbound(ctx context.Context, req CreateOutboundRequest) (EsfDocument, error) {
	if s == nil || s.repo == nil {
		return EsfDocument{}, ErrRepoUnavailable
	}
	entityID := strings.TrimSpace(req.EntityID)
	orderID := strings.TrimSpace(req.OrderID)
	if entityID == "" || orderID == "" {
		return EsfDocument{}, fmt.Errorf("%w: entity_id and order_id are required", ErrInvalidInput)
	}

	submit := true
	if req.Submit != nil {
		submit = *req.Submit
	}
	if submit {
		if s.signer == nil {
			return EsfDocument{}, fiscal.ErrEDSSignerNotConfigured
		}
		if s.soliqClient == nil {
			return EsfDocument{}, fiscal.ErrSoliqGatewayNotConfigured
		}
	}

	src, err := s.repo.LoadOrderFiscalSource(ctx, entityID, orderID)
	if err != nil {
		return EsfDocument{}, err
	}

	cpTIN := strings.TrimSpace(req.CounterpartyTIN)
	if cpTIN == "" {
		cpTIN = strings.TrimSpace(src.CounterpartyTIN)
	}
	if cpTIN == "" {
		return EsfDocument{}, fmt.Errorf("%w: counterparty_tin is required (retailer TIN missing on order %s)", ErrInvalidInput, orderID)
	}
	cpName := strings.TrimSpace(req.CounterpartyName)
	if cpName == "" {
		cpName = strings.TrimSpace(src.CounterpartyName)
	}

	now := s.now().UTC()
	esfID := "esf_" + uuid.NewString()
	operator := normalizeOperator(req.Operator)
	actorID := strings.TrimSpace(req.ActorID)
	if actorID == "" {
		actorID = "system:esf"
	}

	totalMinor := src.TotalGrossMinor
	if totalMinor <= 0 {
		totalMinor = src.TotalMinor
	}
	periodID := gl.ComputePeriodID(ctx, entityID, "UZ", now)
	cur := strings.TrimSpace(src.Currency)
	if cur == "" {
		cur = "UZS"
	}

	canonicalPayload := EsfCanonicalPayload{
		EsfID:            esfID,
		EntityID:         entityID,
		Direction:        DirectionOutbound,
		OrderID:          orderID,
		SupplierTIN:      strings.TrimSpace(src.SupplierTIN),
		CounterpartyTIN:  cpTIN,
		CounterpartyName: cpName,
		Operator:         operator,
		PeriodID:         periodID,
		Currency:         cur,
		TotalMinor:       totalMinor,
		VatMinor:         src.TotalVatMinor,
		VatRateBps:       src.DominantVatBps,
		Lines:            src.Lines,
	}

	payloadHash, signedURI, signedBytes, signErr := s.prepareSignedPayload(ctx, esfID, canonicalPayload, submit || s.signer != nil)
	if signErr != nil {
		return EsfDocument{}, signErr
	}

	doc := EsfDocument{
		EntityID:         entityID,
		EsfID:            esfID,
		Direction:        DirectionOutbound,
		OrderID:          orderID,
		CounterpartyTIN:  cpTIN,
		CounterpartyName: cpName,
		TotalMinor:       totalMinor,
		VatMinor:         src.TotalVatMinor,
		VatRateBps:       src.DominantVatBps,
		Status:           StatusDraft,
		Operator:         operator,
		SignedPayloadURI: signedURI,
		PayloadHash:      payloadHash,
		PeriodID:         periodID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	var eventType string
	var submitErr error
	var opPayloadJSON string

	if submit {
		resp, err := s.soliqClient.Submit(ctx, signedBytes, fmt.Sprintf("esf-submit:%s:%s", entityID, esfID))
		ehfID, rawResp, failErr := interpretSoliqSubmit(resp, err)
		opPayloadJSON = rawResp
		if failErr != nil {
			doc.Status = StatusFailed
			doc.FailureReason = failErr.Error()
			submitErr = fmt.Errorf("%w: %v", ErrUpstreamSubmission, failErr)
		} else {
			doc.Status = StatusSubmitted
			doc.OperatorRef = ehfID
			issuedAt := now
			doc.IssuedAt = &issuedAt
			eventType = events.EventEsfSubmitted
		}
	}

	hist := EsfStatusHistoryEntry{
		EntityID:            entityID,
		EsfID:               esfID,
		HistoryID:           "ehist_" + uuid.NewString(),
		FromStatus:          "",
		ToStatus:            doc.Status,
		ActorID:             actorID,
		Reason:              doc.FailureReason,
		OperatorPayloadJSON: opPayloadJSON,
		CreatedAt:           now,
	}

	if saveErr := s.repo.SaveDocumentWithHistory(ctx, doc, hist, eventType, nil); saveErr != nil {
		return EsfDocument{}, saveErr
	}
	if submitErr != nil {
		return doc, submitErr
	}
	return doc, nil
}

// CreateInbound records an incoming vendor/purchase ESF document for input VAT tracking.
func (s *Service) CreateInbound(ctx context.Context, req CreateInboundRequest) (EsfDocument, error) {
	if s == nil || s.repo == nil {
		return EsfDocument{}, ErrRepoUnavailable
	}
	entityID := strings.TrimSpace(req.EntityID)
	cpTIN := strings.TrimSpace(req.CounterpartyTIN)
	if entityID == "" || cpTIN == "" || req.TotalMinor <= 0 || req.VatMinor < 0 || req.VatMinor > req.TotalMinor || req.VatRateBps < 0 {
		return EsfDocument{}, fmt.Errorf("%w: invalid inbound ESF fields", ErrInvalidInput)
	}

	now := s.now().UTC()
	esfID := strings.TrimSpace(req.EsfID)
	if esfID == "" {
		esfID = "esf_in_" + uuid.NewString()
	}
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status == "" {
		status = StatusSubmitted
	}
	if status != StatusDraft && status != StatusSubmitted && status != StatusAccepted {
		return EsfDocument{}, fmt.Errorf("%w: invalid initial inbound status %q", ErrInvalidInput, req.Status)
	}

	issuedAt := now
	if raw := strings.TrimSpace(req.IssuedAt); raw != "" {
		if parsed, pErr := time.Parse(time.RFC3339, raw); pErr == nil {
			issuedAt = parsed.UTC()
		} else if parsedDate, dErr := time.Parse("2006-01-02", raw); dErr == nil {
			issuedAt = parsedDate.UTC()
		} else {
			return EsfDocument{}, fmt.Errorf("%w: invalid issued_at %q (expected RFC3339 or YYYY-MM-DD)", ErrInvalidInput, raw)
		}
	}

	periodID := strings.TrimSpace(req.PeriodID)
	if periodID == "" {
		periodID = gl.ComputePeriodID(ctx, entityID, "UZ", issuedAt)
	}
	actorID := strings.TrimSpace(req.ActorID)
	if actorID == "" {
		actorID = "system:esf-inbound"
	}
	operator := normalizeOperator(req.Operator)

	payloadHash, _, _, err := s.prepareSignedPayload(ctx, esfID, EsfCanonicalPayload{
		EsfID:            esfID,
		EntityID:         entityID,
		Direction:        DirectionInbound,
		OrderID:          strings.TrimSpace(req.OrderID),
		PoID:             strings.TrimSpace(req.PoID),
		CounterpartyTIN:  cpTIN,
		CounterpartyName: strings.TrimSpace(req.CounterpartyName),
		Operator:         operator,
		PeriodID:         periodID,
		Currency:         "UZS",
		TotalMinor:       req.TotalMinor,
		VatMinor:         req.VatMinor,
		VatRateBps:       req.VatRateBps,
	}, false)
	if err != nil {
		return EsfDocument{}, err
	}

	var acceptedAtPtr *time.Time
	eventType := events.EventEsfSubmitted
	if status == StatusAccepted {
		ts := now
		acceptedAtPtr = &ts
		eventType = events.EventEsfAccepted
	} else if status == StatusDraft {
		eventType = ""
	}

	issuedCopy := issuedAt
	doc := EsfDocument{
		EntityID:         entityID,
		EsfID:            esfID,
		Direction:        DirectionInbound,
		OrderID:          strings.TrimSpace(req.OrderID),
		PoID:             strings.TrimSpace(req.PoID),
		CounterpartyTIN:  cpTIN,
		CounterpartyName: strings.TrimSpace(req.CounterpartyName),
		TotalMinor:       req.TotalMinor,
		VatMinor:         req.VatMinor,
		VatRateBps:       req.VatRateBps,
		Status:           status,
		Operator:         operator,
		OperatorRef:      strings.TrimSpace(req.OperatorRef),
		PayloadHash:      payloadHash,
		PeriodID:         periodID,
		IssuedAt:         &issuedCopy,
		AcceptedAt:       acceptedAtPtr,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	hist := EsfStatusHistoryEntry{
		EntityID:   entityID,
		EsfID:      esfID,
		HistoryID:  "ehist_" + uuid.NewString(),
		FromStatus: "",
		ToStatus:   status,
		ActorID:    actorID,
		CreatedAt:  now,
	}

	if err := s.repo.SaveDocumentWithHistory(ctx, doc, hist, eventType, nil); err != nil {
		return EsfDocument{}, err
	}
	return doc, nil
}

// Resend EDS-signs and (re-)submits a DRAFT, FAILED, or SUBMITTED outbound ESF document to Soliq.
func (s *Service) Resend(ctx context.Context, entityID, esfID, operator, actorID string) (EsfDocument, error) {
	if s == nil || s.repo == nil {
		return EsfDocument{}, ErrRepoUnavailable
	}
	if s.signer == nil {
		return EsfDocument{}, fiscal.ErrEDSSignerNotConfigured
	}
	if s.soliqClient == nil {
		return EsfDocument{}, fiscal.ErrSoliqGatewayNotConfigured
	}

	doc, _, err := s.repo.GetDocument(ctx, strings.TrimSpace(entityID), strings.TrimSpace(esfID))
	if err != nil {
		return EsfDocument{}, err
	}
	if doc.Direction != DirectionOutbound {
		return EsfDocument{}, fmt.Errorf("%w: only outbound ESF documents can be submitted/resent", ErrInvalidTransition)
	}
	if doc.Status != StatusDraft && doc.Status != StatusFailed && doc.Status != StatusSubmitted {
		return EsfDocument{}, fmt.Errorf("%w: cannot resend document in status %s", ErrInvalidTransition, doc.Status)
	}

	if strings.TrimSpace(operator) == "" {
		operator = doc.Operator
	}
	operator = normalizeOperator(operator)

	var src OrderFiscalSource
	if doc.OrderID != "" {
		if loaded, lErr := s.repo.LoadOrderFiscalSource(ctx, doc.EntityID, doc.OrderID); lErr == nil {
			src = loaded
		}
	}

	payloadHash, signedURI, signedBytes, err := s.prepareSignedPayload(ctx, doc.EsfID, EsfCanonicalPayload{
		EsfID:            doc.EsfID,
		EntityID:         doc.EntityID,
		Direction:        doc.Direction,
		OrderID:          doc.OrderID,
		PoID:             doc.PoID,
		SupplierTIN:      strings.TrimSpace(src.SupplierTIN),
		CounterpartyTIN:  doc.CounterpartyTIN,
		CounterpartyName: doc.CounterpartyName,
		Operator:         operator,
		PeriodID:         doc.PeriodID,
		Currency:         firstNonEmptyStr(src.Currency, "UZS"),
		TotalMinor:       doc.TotalMinor,
		VatMinor:         doc.VatMinor,
		VatRateBps:       doc.VatRateBps,
		Lines:            src.Lines,
	}, true)
	if err != nil {
		return EsfDocument{}, err
	}

	now := s.now().UTC()
	if strings.TrimSpace(actorID) == "" {
		actorID = "system:esf-resend"
	}

	resp, submitErr := s.soliqClient.Submit(ctx, signedBytes, fmt.Sprintf("esf-resend:%s:%s:%d", doc.EntityID, doc.EsfID, now.UnixNano()))
	ehfID, opPayloadJSON, failErr := interpretSoliqSubmit(resp, submitErr)
	if failErr != nil {
		failMsg := failErr.Error()
		updatedDoc, uErr := s.repo.UpdateDocumentStatusWithHistory(ctx, doc.EntityID, doc.EsfID, doc.Status, StatusFailed, StatusUpdateFields{
			Operator:         operator,
			SignedPayloadURI: signedURI,
			PayloadHash:      payloadHash,
			FailureReason:    &failMsg,
			UpdatedAt:        now,
		}, EsfStatusHistoryEntry{
			EntityID:            doc.EntityID,
			EsfID:               doc.EsfID,
			HistoryID:           "ehist_" + uuid.NewString(),
			FromStatus:          doc.Status,
			ToStatus:            StatusFailed,
			ActorID:             actorID,
			Reason:              failMsg,
			OperatorPayloadJSON: opPayloadJSON,
			CreatedAt:           now,
		}, "", nil)
		if uErr != nil {
			return EsfDocument{}, uErr
		}
		return updatedDoc, fmt.Errorf("%w: %v", ErrUpstreamSubmission, failErr)
	}

	clearErr := ""
	issuedAt := now
	if doc.IssuedAt != nil && !doc.IssuedAt.IsZero() {
		issuedAt = *doc.IssuedAt
	}
	updatedDoc, uErr := s.repo.UpdateDocumentStatusWithHistory(ctx, doc.EntityID, doc.EsfID, doc.Status, StatusSubmitted, StatusUpdateFields{
		Operator:         operator,
		OperatorRef:      ehfID,
		SignedPayloadURI: signedURI,
		PayloadHash:      payloadHash,
		FailureReason:    &clearErr,
		IssuedAt:         &issuedAt,
		UpdatedAt:        now,
	}, EsfStatusHistoryEntry{
		EntityID:            doc.EntityID,
		EsfID:               doc.EsfID,
		HistoryID:           "ehist_" + uuid.NewString(),
		FromStatus:          doc.Status,
		ToStatus:            StatusSubmitted,
		ActorID:             actorID,
		OperatorPayloadJSON: opPayloadJSON,
		CreatedAt:           now,
	}, events.EventEsfSubmitted, nil)
	if uErr != nil {
		return EsfDocument{}, uErr
	}
	return updatedDoc, nil
}

// TransitionStatus executes ACCEPT, REJECT, or CANCEL transitions, appending EsfStatusHistory,
// emitting esf.accepted / esf.rejected / esf.cancelled, and on esf.rejected triggering reverse settlement
// (creditnote creation/issuance or atomic GL DR TAX:VAT:OUTPUT reversal).
func (s *Service) TransitionStatus(ctx context.Context, entityID, esfID string, req TransitionStatusRequest) (TransitionStatusResult, error) {
	if s == nil || s.repo == nil {
		return TransitionStatusResult{}, ErrRepoUnavailable
	}
	targetStatus, err := normalizeActionToTargetStatus(req.Action, req.Status)
	if err != nil {
		return TransitionStatusResult{}, err
	}

	doc, _, err := s.repo.GetDocument(ctx, strings.TrimSpace(entityID), strings.TrimSpace(esfID))
	if err != nil {
		return TransitionStatusResult{}, err
	}
	if !isAllowedStatusTransition(doc.Status, targetStatus) {
		return TransitionStatusResult{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, doc.Status, targetStatus)
	}

	now := s.now().UTC()
	actorID := strings.TrimSpace(req.ActorID)
	if actorID == "" {
		actorID = "system:esf-status"
	}
	reason := strings.TrimSpace(req.Reason)

	updates := StatusUpdateFields{
		OperatorRef: strings.TrimSpace(req.OperatorRef),
		UpdatedAt:   now,
	}
	var eventType string
	switch targetStatus {
	case StatusAccepted:
		ts := now
		updates.AcceptedAt = &ts
		clearReason := ""
		updates.FailureReason = &clearReason
		eventType = events.EventEsfAccepted
	case StatusRejected:
		if reason != "" {
			updates.FailureReason = &reason
		}
		eventType = events.EventEsfRejected
	case StatusCancelled:
		if reason != "" {
			updates.FailureReason = &reason
		}
		eventType = events.EventEsfCancelled
	}

	var creditNoteID string
	var creditNoteHandledVAT bool
	var retailerID string
	var currency = "UZS"

	if targetStatus == StatusRejected && doc.Direction == DirectionOutbound {
		if doc.OrderID != "" {
			if src, sErr := s.repo.LoadOrderFiscalSource(ctx, doc.EntityID, doc.OrderID); sErr == nil {
				retailerID = src.RetailerID
				if src.Currency != "" {
					currency = src.Currency
				}
			}
		}

		if s.creditNoteSvc != nil && doc.OrderID != "" {
			cn, cnErr := s.creditNoteSvc.CreateFromBuyerReject(ctx, doc.OrderID, actorID)
			if cnErr == nil && cn != nil {
				creditNoteID = cn.CreditNoteId
				if cn.Status == creditnote.CreditNoteStatusDraft {
					if iErr := s.creditNoteSvc.Issue(ctx, cn.CreditNoteId, actorID); iErr == nil && cn.TotalVatMinor > 0 {
						creditNoteHandledVAT = true
					}
				} else if cn.TotalVatMinor > 0 {
					creditNoteHandledVAT = true
				}
			} else {
				// If buyer-reject credit note already exists (e.g. from BuyerAcceptancePoller), locate and issue if still DRAFT.
				if list, lErr := s.creditNoteSvc.ListBySupplier(ctx, doc.EntityID, "", 50); lErr == nil {
					for _, existing := range list {
						if existing.OrderId == doc.OrderID && existing.Type == creditnote.CreditNoteTypeBuyerReject && existing.Status != creditnote.CreditNoteStatusCancelled {
							creditNoteID = existing.CreditNoteId
							if existing.Status == creditnote.CreditNoteStatusDraft {
								if iErr := s.creditNoteSvc.Issue(ctx, existing.CreditNoteId, actorID); iErr == nil && existing.TotalVatMinor > 0 {
									creditNoteHandledVAT = true
								}
							} else if existing.TotalVatMinor > 0 {
								creditNoteHandledVAT = true
							}
							break
						}
					}
				}
			}
		}
	}

	var vatReversalSpec *VATReversalSpec
	var glEntryID string
	var vatReversedMinor int64
	if targetStatus == StatusRejected && doc.Direction == DirectionOutbound && doc.VatMinor > 0 && !creditNoteHandledVAT {
		vatReversalSpec = &VATReversalSpec{
			EntityID:       doc.EntityID,
			EsfID:          doc.EsfID,
			OrderID:        doc.OrderID,
			RetailerID:     retailerID,
			Currency:       currency,
			VatMinor:       doc.VatMinor,
			ActorID:        actorID,
			IdempotencyKey: "gl:esf_reject:" + doc.EsfID,
			OccurredAt:     now,
			GlEntryIDOut:   &glEntryID,
		}
		vatReversedMinor = doc.VatMinor
	} else if creditNoteHandledVAT {
		vatReversedMinor = doc.VatMinor
	}

	hist := EsfStatusHistoryEntry{
		EntityID:            doc.EntityID,
		EsfID:               doc.EsfID,
		HistoryID:           "ehist_" + uuid.NewString(),
		FromStatus:          doc.Status,
		ToStatus:            targetStatus,
		ActorID:             actorID,
		Reason:              reason,
		OperatorPayloadJSON: strings.TrimSpace(req.OperatorPayload),
		CreatedAt:           now,
	}

	updatedDoc, err := s.repo.UpdateDocumentStatusWithHistory(ctx, doc.EntityID, doc.EsfID, doc.Status, targetStatus, updates, hist, eventType, vatReversalSpec)
	if err != nil {
		return TransitionStatusResult{}, err
	}
	_, history, hErr := s.repo.GetDocument(ctx, updatedDoc.EntityID, updatedDoc.EsfID)
	if hErr != nil {
		history = []EsfStatusHistoryEntry{hist}
	}

	return TransitionStatusResult{
		Document:          updatedDoc,
		History:           history,
		VatReversedMinor:  vatReversedMinor,
		GlReversalEntryID: glEntryID,
		CreditNoteID:      creditNoteID,
	}, nil
}

// SyncOrderAcceptanceStatus implements order.ESFStatusSyncer so BuyerAcceptancePoller
// updates the corresponding EsfDocument when Soliq reports ACCEPTED or REJECTED.
func (s *Service) SyncOrderAcceptanceStatus(ctx context.Context, orderID, ehfID, status, reason string, rawPayload []byte) error {
	if s == nil || s.repo == nil {
		return nil
	}
	doc, err := s.repo.FindLatestByOrderOrOperatorRef(ctx, "", strings.TrimSpace(orderID), strings.TrimSpace(ehfID))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}

	target := strings.ToUpper(strings.TrimSpace(status))
	var action string
	switch target {
	case StatusAccepted:
		action = ActionAccept
	case StatusRejected, "EXPIRED":
		action = ActionReject
		if reason == "" && target == "EXPIRED" {
			reason = "buyer acceptance deadline expired"
		}
	default:
		return nil
	}
	if doc.Status == StatusAccepted && action == ActionAccept {
		return nil
	}
	if doc.Status == StatusRejected && action == ActionReject {
		return nil
	}

	_, tErr := s.TransitionStatus(ctx, doc.EntityID, doc.EsfID, TransitionStatusRequest{
		EntityID:        doc.EntityID,
		Action:          action,
		Reason:          reason,
		OperatorRef:     strings.TrimSpace(ehfID),
		OperatorPayload: string(rawPayload),
		ActorID:         "system:buyer-accept-poller",
	})
	if errors.Is(tErr, ErrInvalidTransition) {
		return nil
	}
	return tErr
}

// Get returns an EsfDocument alongside its ordered status history.
func (s *Service) Get(ctx context.Context, entityID, esfID string) (EsfDetailResponse, error) {
	if s == nil || s.repo == nil {
		return EsfDetailResponse{}, ErrRepoUnavailable
	}
	doc, hist, err := s.repo.GetDocument(ctx, strings.TrimSpace(entityID), strings.TrimSpace(esfID))
	if err != nil {
		return EsfDetailResponse{}, err
	}
	return EsfDetailResponse{Document: doc, History: hist}, nil
}

// List lists ESF documents matching filter criteria.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]EsfDocument, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	if strings.TrimSpace(filter.EntityID) == "" {
		return nil, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	return s.repo.ListDocuments(ctx, filter)
}

// GetVATSummary compares ESF output/input VAT against GL TAX:VAT:OUTPUT and TAX:VAT:INPUT.
func (s *Service) GetVATSummary(ctx context.Context, entityID, periodID string) (VATSummaryReport, error) {
	if s == nil || s.repo == nil {
		return VATSummaryReport{}, ErrRepoUnavailable
	}
	eid := strings.TrimSpace(entityID)
	pid := strings.TrimSpace(periodID)
	if eid == "" {
		return VATSummaryReport{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	if pid == "" {
		pid = gl.ComputePeriodID(ctx, eid, "UZ", s.now().UTC())
	}
	return s.repo.GetVATSummary(ctx, eid, pid)
}

// ListMissingIKPU returns active SKUs missing Soliq IKPU/MXIK, package, or marking compliance fields.
func (s *Service) ListMissingIKPU(ctx context.Context, supplierID string) (MissingIKPUResponse, error) {
	if s == nil || s.repo == nil {
		return MissingIKPUResponse{}, ErrRepoUnavailable
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return MissingIKPUResponse{}, fmt.Errorf("%w: supplier_id is required", ErrInvalidInput)
	}
	items, err := s.repo.ListMissingIKPU(ctx, sid)
	if err != nil {
		return MissingIKPUResponse{}, err
	}
	return MissingIKPUResponse{
		SupplierID: sid,
		Items:      items,
		Count:      len(items),
	}, nil
}

func (s *Service) prepareSignedPayload(ctx context.Context, esfID string, payload EsfCanonicalPayload, sign bool) (hashHex, signedURI string, signedBytes []byte, err error) {
	canonical, err := fiscal.MarshalCanonical(payload)
	if err != nil {
		return "", "", nil, fmt.Errorf("marshal canonical ESF payload: %w", err)
	}
	if sign {
		if s.signer == nil {
			return "", "", nil, fiscal.ErrEDSSignerNotConfigured
		}
		signedBytes, err = fiscal.AttachSignature(ctx, s.signer, payload)
		if err != nil {
			return "", "", nil, fmt.Errorf("EDS sign ESF %s: %w", esfID, err)
		}
	} else {
		signedBytes = canonical
	}
	sum := sha256.Sum256(signedBytes)
	hashHex = hex.EncodeToString(sum[:])
	signedURI = fmt.Sprintf("esf-payload://%s/%s/%s", payload.EntityID, esfID, hashHex)
	return hashHex, signedURI, signedBytes, nil
}

func isAllowedStatusTransition(from, to string) bool {
	switch from {
	case StatusDraft, StatusFailed:
		return to == StatusCancelled || to == StatusSubmitted
	case StatusSubmitted:
		return to == StatusAccepted || to == StatusRejected || to == StatusCancelled
	default:
		return false
	}
}

func detectMissingComplianceFields(ikpuCode, packageCode, markingCode, barcode string) []string {
	var missing []string
	if strings.TrimSpace(ikpuCode) == "" {
		missing = append(missing, "ikpu_code")
	}
	if strings.TrimSpace(packageCode) == "" {
		missing = append(missing, "package_code")
	}
	if strings.TrimSpace(markingCode) == "" && strings.TrimSpace(barcode) == "" {
		missing = append(missing, "marking_code")
	}
	return missing
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func interpretSoliqSubmit(resp *soliq.SoliqResponse, err error) (ehfID, rawJSON string, failErr error) {
	if resp != nil {
		if b, mErr := json.Marshal(resp); mErr == nil {
			rawJSON = string(b)
		}
	}
	if err != nil {
		return "", rawJSON, err
	}
	if resp == nil {
		return "", rawJSON, errors.New("empty response from soliq gateway")
	}
	if !resp.Success {
		msg := strings.TrimSpace(resp.ErrorMessage)
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		if code := strings.TrimSpace(resp.ErrorCode); code != "" {
			return "", rawJSON, fmt.Errorf("soliq rejected (%s): %s", code, msg)
		}
		return "", rawJSON, fmt.Errorf("soliq rejected: %s", msg)
	}
	return strings.TrimSpace(resp.EhfID), rawJSON, nil
}
