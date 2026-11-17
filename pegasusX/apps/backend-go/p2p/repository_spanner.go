package p2p

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SpannerRepository implements Repository over Cloud Spanner using StrongRead /
// ReadOnlyTransaction queries and atomic ReadWriteTransaction mutations.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed P2P Repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

type tenantStampedOutboxBuf struct {
	inner    outbox.TxnBuffer
	tenantID string
}

func (b *tenantStampedOutboxBuf) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.tenantID) != "" {
		e.SupplierID = outbox.ResolveSupplierID(b.tenantID, e.Payload)
	}
	return b.inner.BufferOutbox(ctx, e)
}

func (r *SpannerRepository) ListTemplates(ctx context.Context, tenantID, domain string) ([]ProcessTemplate, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, TemplateId, Code, Name, Domain, StepsJson, IsDefault, Status, CreatedAt, UpdatedAt
		      FROM ProcessTemplates
		      WHERE TenantId = @tenantId
		      ORDER BY Domain ASC, Code ASC`,
		Params: map[string]any{"tenantId": tid},
	})
	defer iter.Stop()

	var custom []ProcessTemplate
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ProcessTemplates: %w", err)
		}
		var (
			tpl      ProcessTemplate
			stepsRaw string
		)
		if err := row.Columns(
			&tpl.TenantID, &tpl.TemplateID, &tpl.Code, &tpl.Name, &tpl.Domain,
			&stepsRaw, &tpl.IsDefault, &tpl.Status, &tpl.CreatedAt, &tpl.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ProcessTemplate: %w", err)
		}
		if err := json.Unmarshal([]byte(stepsRaw), &tpl.Steps); err != nil {
			tpl.Steps = nil
		}
		custom = append(custom, tpl)
	}
	return MergeTemplates(tid, custom, domain), nil
}

func (r *SpannerRepository) SaveTemplate(ctx context.Context, tpl ProcessTemplate) (ProcessTemplate, error) {
	if r == nil || r.client == nil {
		return ProcessTemplate{}, ErrRepoUnavailable
	}
	stepsRaw, err := json.Marshal(tpl.Steps)
	if err != nil {
		return ProcessTemplate{}, fmt.Errorf("%w: marshal steps: %v", ErrInvalidInput, err)
	}
	err = spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		iter := txn.Query(txCtx, spanner.Statement{
			SQL: `SELECT TemplateId FROM ProcessTemplates WHERE TenantId = @tenantId AND UPPER(Code) = UPPER(@code) LIMIT 1`,
			Params: map[string]any{
				"tenantId": tpl.TenantID,
				"code":     tpl.Code,
			},
		})
		defer iter.Stop()
		if row, qErr := iter.Next(); qErr == nil {
			var existingID string
			if sErr := row.Column(0, &existingID); sErr == nil && existingID != tpl.TemplateID {
				return fmt.Errorf("%w: template code %s already exists", ErrConflict, tpl.Code)
			}
		} else if !errors.Is(qErr, iterator.Done) {
			return fmt.Errorf("check ProcessTemplates duplicate code: %w", qErr)
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("ProcessTemplates", map[string]any{
				"TenantId":   tpl.TenantID,
				"TemplateId": tpl.TemplateID,
				"Code":       tpl.Code,
				"Name":       tpl.Name,
				"Domain":     tpl.Domain,
				"StepsJson":  string(stepsRaw),
				"IsDefault":  tpl.IsDefault,
				"Status":     tpl.Status,
				"CreatedAt":  spanner.CommitTimestamp,
				"UpdatedAt":  spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		return ProcessTemplate{}, err
	}
	return tpl, nil
}

func (r *SpannerRepository) ListInstances(ctx context.Context, f ListFilter) ([]ProcessInstance, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	params := map[string]any{
		"tenantId": strings.TrimSpace(f.TenantID),
		"limit":    int64(clampLimit(f.Limit)),
	}
	sql := `SELECT TenantId, InstanceId, TemplateId, EntityId, ObjectType, ObjectId,
	               CurrentStep, Status, TransitionHistoryJson, StartedAt, CompletedAt, UpdatedAt
	        FROM ProcessInstances
	        WHERE TenantId = @tenantId`
	if eid := strings.TrimSpace(f.EntityID); eid != "" {
		sql += ` AND EntityId = @entityId`
		params["entityId"] = eid
	}
	if ot := strings.TrimSpace(f.ObjectType); ot != "" {
		sql += ` AND LOWER(ObjectType) = LOWER(@objectType)`
		params["objectType"] = ot
	}
	if oid := strings.TrimSpace(f.ObjectID); oid != "" {
		sql += ` AND ObjectId = @objectId`
		params["objectId"] = oid
	}
	if st := strings.TrimSpace(f.Status); st != "" {
		sql += ` AND UPPER(Status) = UPPER(@status)`
		params["status"] = st
	}
	sql += ` ORDER BY StartedAt DESC LIMIT @limit`

	iter := roTxn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var out []ProcessInstance
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ProcessInstances: %w", err)
		}
		inst, err := scanProcessInstanceRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, inst)
	}
	return out, nil
}

func (r *SpannerRepository) UpsertInstance(ctx context.Context, in StartOrAdvanceInstanceInput, tpl ProcessTemplate, now time.Time) (ProcessInstance, error) {
	if r == nil || r.client == nil {
		return ProcessInstance{}, ErrRepoUnavailable
	}
	var out ProcessInstance
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		existing, found, err := loadProcessInstanceInTxn(txCtx, txn, in.TenantID, in.InstanceID, in.ObjectType, in.ObjectID)
		if err != nil {
			return err
		}
		if found && strings.TrimSpace(in.NextStep) != "" {
			updated, mut, mErr := buildAdvanceInstanceMutation(existing, tpl, in.NextStep, in.ActorID, in.Reason, in.Complete, now)
			if mErr != nil {
				return mErr
			}
			out = updated
			if mut == nil {
				return nil
			}
			return txn.BufferWrite([]*spanner.Mutation{mut})
		}
		if found && strings.TrimSpace(in.NextStep) == "" {
			out = existing
			return nil
		}

		firstStep := coalesce(in.CurrentStep, in.NextStep)
		if firstStep == "" && len(tpl.Steps) > 0 {
			firstStep = tpl.Steps[0].Code
		}
		if firstStep == "" {
			firstStep = "CREATED"
		}
		firstStep = strings.ToUpper(firstStep)
		instID := strings.TrimSpace(in.InstanceID)
		if instID == "" {
			instID = "pinst_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
		}
		inst := ProcessInstance{
			TenantID:     in.TenantID,
			InstanceID:   instID,
			TemplateID:   tpl.TemplateID,
			TemplateCode: tpl.Code,
			EntityID:     strings.TrimSpace(in.EntityID),
			ObjectType:   strings.TrimSpace(in.ObjectType),
			ObjectID:     strings.TrimSpace(in.ObjectID),
			CurrentStep:  firstStep,
			Status:       InstanceStatusActive,
			Transitions: []ProcessTransition{
				{FromStep: "START", ToStep: firstStep, ActorID: in.ActorID, Reason: coalesce(in.Reason, "started"), OccurredAt: now},
			},
			StartedAt: now,
			UpdatedAt: now,
		}
		if in.Complete || (len(tpl.Steps) > 0 && strings.EqualFold(inst.CurrentStep, tpl.Steps[len(tpl.Steps)-1].Code)) {
			inst.Status = InstanceStatusCompleted
			doneAt := now
			inst.CompletedAt = &doneAt
		}
		histRaw, mErr := json.Marshal(inst.Transitions)
		if mErr != nil {
			return fmt.Errorf("marshal transition history: %w", mErr)
		}
		out = inst
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("ProcessInstances", map[string]any{
				"TenantId":              inst.TenantID,
				"InstanceId":            inst.InstanceID,
				"TemplateId":            inst.TemplateID,
				"EntityId":              nullStr(inst.EntityID),
				"ObjectType":            inst.ObjectType,
				"ObjectId":              inst.ObjectID,
				"CurrentStep":           inst.CurrentStep,
				"Status":                inst.Status,
				"TransitionHistoryJson": string(histRaw),
				"StartedAt":             inst.StartedAt,
				"CompletedAt":           nullTime(inst.CompletedAt),
				"UpdatedAt":             spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		return ProcessInstance{}, err
	}
	return out, nil
}

func (r *SpannerRepository) CreatePurchaseOrder(ctx context.Context, po PurchaseOrder, tpl ProcessTemplate, hooks TxnHooks) (PurchaseOrder, bool, error) {
	if r == nil || r.client == nil {
		return PurchaseOrder{}, false, ErrRepoUnavailable
	}
	var out PurchaseOrder
	var already bool
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if po.IdempotencyKey != "" {
			if existing, ok, err := findPOByIdempotencyInTxn(txCtx, txn, po.TenantID, po.IdempotencyKey); err != nil {
				return err
			} else if ok {
				out = existing
				already = true
				return nil
			}
		}

		muts := make([]*spanner.Mutation, 0, len(po.Lines)+2)
		muts = append(muts, spanner.InsertMap("PurchaseOrders", map[string]any{
			"TenantId":          po.TenantID,
			"PoId":              po.PoID,
			"EntityId":          nullStr(po.EntityID),
			"VendorId":          po.VendorID,
			"VendorName":        nullStr(po.VendorName),
			"VendorTin":         nullStr(po.VendorTin),
			"Currency":          po.Currency,
			"SubtotalMinor":     po.SubtotalMinor,
			"VatMinor":          po.VatMinor,
			"TotalMinor":        po.TotalMinor,
			"Status":            po.Status,
			"ApprovalRequestId": nullStr(po.ApprovalRequestID),
			"IdempotencyKey":    nullStr(po.IdempotencyKey),
			"ExpectedAt":        nullTime(po.ExpectedAt),
			"CreatedBy":         po.CreatedBy,
			"CreatedAt":         spanner.CommitTimestamp,
			"UpdatedAt":         spanner.CommitTimestamp,
		}))

		for _, l := range po.Lines {
			muts = append(muts, spanner.InsertMap("PurchaseOrderLines", map[string]any{
				"TenantId":       po.TenantID,
				"PoId":           po.PoID,
				"LineNum":        l.LineNum,
				"Sku":            l.Sku,
				"ProductName":    nullStr(l.ProductName),
				"QtyOrdered":     l.QtyOrdered,
				"QtyReceived":    l.QtyReceived,
				"UnitCostMinor":  l.UnitCostMinor,
				"VatBps":         l.VatBps,
				"LineTotalMinor": l.LineTotalMinor,
			}))
		}

		if tpl.TemplateID != "" {
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
			if histRaw, mErr := json.Marshal(transitions); mErr == nil {
				muts = append(muts, spanner.InsertMap("ProcessInstances", map[string]any{
					"TenantId":              po.TenantID,
					"InstanceId":            instID,
					"TemplateId":            tpl.TemplateID,
					"EntityId":              nullStr(po.EntityID),
					"ObjectType":            AggregatePurchaseOrder,
					"ObjectId":              po.PoID,
					"CurrentStep":           curr,
					"Status":                InstanceStatusActive,
					"TransitionHistoryJson": string(histRaw),
					"StartedAt":             po.CreatedAt,
					"CompletedAt":           nil,
					"UpdatedAt":             spanner.CommitTimestamp,
				}))
			}
		}

		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("buffer PurchaseOrder: %w", err)
		}

		rawBuf := outbox.NewSpannerTxnBuffer(txn)
		buf := &tenantStampedOutboxBuf{inner: rawBuf, tenantID: po.TenantID}
		if _, err := runNonSpannerGLAndAudit(txCtx, txn, buf, hooks); err != nil {
			return err
		}
		if hooks.OutboxEmit != nil {
			if err := hooks.OutboxEmit(txCtx, buf, ""); err != nil {
				return err
			}
		}
		if err := rawBuf.Flush(txCtx); err != nil {
			return fmt.Errorf("flush PurchaseOrder outbox: %w", err)
		}
		out = po
		return nil
	})
	if err != nil {
		return PurchaseOrder{}, false, err
	}
	return out, already, nil
}

func (r *SpannerRepository) ApprovePurchaseOrder(ctx context.Context, tenantID, poID, actorID, approvalID string, now time.Time, hooks TxnHooks) (PurchaseOrder, error) {
	if r == nil || r.client == nil {
		return PurchaseOrder{}, ErrRepoUnavailable
	}
	var out PurchaseOrder
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		po, ok, err := loadPurchaseOrderInTxn(txCtx, txn, tenantID, poID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if po.CreatedBy != "" && strings.EqualFold(po.CreatedBy, actorID) {
			return ErrSoDViolation
		}
		if po.Status != POStatusPendingApproval {
			return fmt.Errorf("%w: purchase order status is %s", ErrConflict, po.Status)
		}

		upd := map[string]any{
			"TenantId":  tenantID,
			"PoId":      poID,
			"Status":    POStatusApproved,
			"UpdatedAt": spanner.CommitTimestamp,
		}
		if approvalID != "" {
			upd["ApprovalRequestId"] = approvalID
			po.ApprovalRequestID = approvalID
		}
		muts := []*spanner.Mutation{spanner.UpdateMap("PurchaseOrders", upd)}
		if inst, found, iErr := loadProcessInstanceInTxn(txCtx, txn, tenantID, "", AggregatePurchaseOrder, poID); iErr == nil && found {
			if _, instMut, mErr := buildAdvanceInstanceMutation(inst, ProcessTemplate{}, "PO_APPROVED", actorID, "po.approved", false, now); mErr == nil && instMut != nil {
				muts = append(muts, instMut)
			}
		}
		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("buffer PurchaseOrder approve: %w", err)
		}

		rawBuf := outbox.NewSpannerTxnBuffer(txn)
		buf := &tenantStampedOutboxBuf{inner: rawBuf, tenantID: tenantID}
		if _, err := runNonSpannerGLAndAudit(txCtx, txn, buf, hooks); err != nil {
			return err
		}
		if err := rawBuf.Flush(txCtx); err != nil {
			return err
		}
		po.Status = POStatusApproved
		po.UpdatedAt = now
		out = po
		return nil
	})
	if err != nil {
		return PurchaseOrder{}, err
	}
	return out, nil
}

func (r *SpannerRepository) FindPurchaseOrderByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (PurchaseOrder, bool, error) {
	if r == nil || r.client == nil {
		return PurchaseOrder{}, false, ErrRepoUnavailable
	}
	ik := strings.TrimSpace(idempotencyKey)
	if ik == "" {
		return PurchaseOrder{}, false, nil
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL:    `SELECT PoId FROM PurchaseOrders WHERE TenantId = @tenantId AND IdempotencyKey = @ik LIMIT 1`,
		Params: map[string]any{"tenantId": strings.TrimSpace(tenantID), "ik": ik},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return PurchaseOrder{}, false, nil
	}
	if err != nil {
		return PurchaseOrder{}, false, err
	}
	var poID string
	if err := row.Column(0, &poID); err != nil {
		return PurchaseOrder{}, false, err
	}
	return loadPurchaseOrderWithReader(ctx, roTxn, strings.TrimSpace(tenantID), poID)
}

func (r *SpannerRepository) GetPurchaseOrder(ctx context.Context, tenantID, poID string) (PurchaseOrder, bool, error) {
	if r == nil || r.client == nil {
		return PurchaseOrder{}, false, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()
	return loadPurchaseOrderWithReader(ctx, roTxn, strings.TrimSpace(tenantID), strings.TrimSpace(poID))
}

func (r *SpannerRepository) ListPurchaseOrders(ctx context.Context, f ListFilter) ([]PurchaseOrder, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	params := map[string]any{
		"tenantId": strings.TrimSpace(f.TenantID),
		"limit":    int64(clampLimit(f.Limit)),
	}
	sql := `SELECT TenantId, PoId, EntityId, VendorId, VendorName, VendorTin, Currency,
	               SubtotalMinor, VatMinor, TotalMinor, Status, ApprovalRequestId, IdempotencyKey,
	               ExpectedAt, CreatedBy, CreatedAt, UpdatedAt
	        FROM PurchaseOrders
	        WHERE TenantId = @tenantId`
	if eid := strings.TrimSpace(f.EntityID); eid != "" {
		sql += ` AND EntityId = @entityId`
		params["entityId"] = eid
	}
	if vid := strings.TrimSpace(f.VendorID); vid != "" {
		sql += ` AND VendorId = @vendorId`
		params["vendorId"] = vid
	}
	if st := strings.TrimSpace(f.Status); st != "" {
		sql += ` AND UPPER(Status) = UPPER(@status)`
		params["status"] = st
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @limit`

	iter := roTxn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var pos []PurchaseOrder
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query PurchaseOrders: %w", err)
		}
		po, err := scanPurchaseOrderRow(row)
		if err != nil {
			return nil, err
		}
		pos = append(pos, po)
	}
	if len(pos) > 0 {
		poIDs := make([]string, len(pos))
		for i, po := range pos {
			poIDs[i] = po.PoID
		}
		lineIter := roTxn.Query(ctx, spanner.Statement{
			SQL: `SELECT TenantId, PoId, LineNum, Sku, ProductName, QtyOrdered, QtyReceived, UnitCostMinor, VatBps, LineTotalMinor
			      FROM PurchaseOrderLines
			      WHERE TenantId = @tenantId AND PoId IN UNNEST(@poIds)
			      ORDER BY PoId ASC, LineNum ASC`,
			Params: map[string]any{"tenantId": strings.TrimSpace(f.TenantID), "poIds": poIDs},
		})
		defer lineIter.Stop()
		linesByPO := make(map[string][]PurchaseOrderLine, len(pos))
		for {
			row, err := lineIter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("query PurchaseOrderLines batch: %w", err)
			}
			var (
				l    PurchaseOrderLine
				name spanner.NullString
			)
			if err := row.Columns(&l.TenantID, &l.PoID, &l.LineNum, &l.Sku, &name, &l.QtyOrdered, &l.QtyReceived, &l.UnitCostMinor, &l.VatBps, &l.LineTotalMinor); err != nil {
				return nil, fmt.Errorf("scan PurchaseOrderLine: %w", err)
			}
			l.ProductName = name.StringVal
			linesByPO[l.PoID] = append(linesByPO[l.PoID], l)
		}
		for i := range pos {
			pos[i].Lines = linesByPO[pos[i].PoID]
		}
	}
	return pos, nil
}

func (r *SpannerRepository) FindGoodsReceiptByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (GoodsReceipt, bool, error) {
	if r == nil || r.client == nil {
		return GoodsReceipt{}, false, ErrRepoUnavailable
	}
	ik := strings.TrimSpace(idempotencyKey)
	if ik == "" {
		return GoodsReceipt{}, false, nil
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, ReceiptId, PoId, EntityId, WarehouseId, Status, LinesJson,
		             TotalReceivedMinor, JournalEntryId, IdempotencyKey, ReceivedBy, ReceivedAt, CreatedAt
		      FROM GoodsReceipts WHERE TenantId = @tenantId AND IdempotencyKey = @ik LIMIT 1`,
		Params: map[string]any{"tenantId": strings.TrimSpace(tenantID), "ik": ik},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return GoodsReceipt{}, false, nil
	}
	if err != nil {
		return GoodsReceipt{}, false, err
	}
	rec, err := scanGoodsReceiptRow(row)
	return rec, err == nil, err
}

func (r *SpannerRepository) PostGoodsReceipt(ctx context.Context, receipt GoodsReceipt, hooks TxnHooks) (GoodsReceipt, PurchaseOrder, bool, error) {
	if r == nil || r.client == nil {
		return GoodsReceipt{}, PurchaseOrder{}, false, ErrRepoUnavailable
	}
	var prePostedJournalID string
	if hooks.GLInput != nil {
		if hooks.GLPoster == nil {
			return GoodsReceipt{}, PurchaseOrder{}, false, ErrGLPosterUnavailable
		}
		if _, isTxnPoster := hooks.GLPoster.(TransactionalGLPoster); !isTxnPoster {
			if receipt.IdempotencyKey != "" {
				if exRec, ok, err := r.FindGoodsReceiptByIdempotency(ctx, receipt.TenantID, receipt.IdempotencyKey); err != nil {
					return GoodsReceipt{}, PurchaseOrder{}, false, err
				} else if ok {
					po, _, pErr := r.GetPurchaseOrder(ctx, receipt.TenantID, exRec.PoID)
					if pErr != nil {
						return GoodsReceipt{}, PurchaseOrder{}, false, pErr
					}
					return exRec, po, true, nil
				}
			}
			// Preflight validation before non-transactional GL post so Spanner RW txn never nests.
			po, ok, err := r.GetPurchaseOrder(ctx, receipt.TenantID, receipt.PoID)
			if err != nil {
				return GoodsReceipt{}, PurchaseOrder{}, false, err
			}
			if !ok {
				return GoodsReceipt{}, PurchaseOrder{}, false, ErrNotFound
			}
			if po.Status == POStatusPendingApproval || po.Status == POStatusCancelled || po.Status == POStatusClosed {
				return GoodsReceipt{}, PurchaseOrder{}, false, fmt.Errorf("%w: status=%s", ErrPONotApproved, po.Status)
			}
			if _, _, totalNet, err := applyReceiptLinesToPO(po, receipt.Lines, receipt.ReceivedAt); err != nil {
				return GoodsReceipt{}, PurchaseOrder{}, false, err
			} else {
				glCopy := *hooks.GLInput
				glCopy.EntityID = coalesce(glCopy.EntityID, po.EntityID)
				for i := range glCopy.Lines {
					glCopy.Lines[i].AmountMinor = totalNet
					if glCopy.Lines[i].AccountCode == AccountGRNI {
						glCopy.Lines[i].PartyType = "VENDOR"
						glCopy.Lines[i].PartyID = po.VendorID
					}
				}
				res, gErr := hooks.GLPoster.PostJournal(ctx, glCopy)
				if gErr != nil {
					return GoodsReceipt{}, PurchaseOrder{}, false, gErr
				}
				prePostedJournalID = res.EntryID
				hooks.GLInput = nil
			}
		}
	}

	var outRec GoodsReceipt
	var outPO PurchaseOrder
	var already bool

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if receipt.IdempotencyKey != "" {
			if exRec, ok, err := findReceiptByIdempotencyInTxn(txCtx, txn, receipt.TenantID, receipt.IdempotencyKey); err != nil {
				return err
			} else if ok {
				po, _, pErr := loadPurchaseOrderInTxn(txCtx, txn, receipt.TenantID, exRec.PoID)
				if pErr != nil {
					return pErr
				}
				outRec = exRec
				outPO = po
				already = true
				return nil
			}
		}

		po, ok, err := loadPurchaseOrderInTxn(txCtx, txn, receipt.TenantID, receipt.PoID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if po.Status == POStatusPendingApproval || po.Status == POStatusCancelled || po.Status == POStatusClosed {
			return fmt.Errorf("%w: status=%s", ErrPONotApproved, po.Status)
		}

		resolvedLines, updatedPO, totalNet, err := applyReceiptLinesToPO(po, receipt.Lines, receipt.ReceivedAt)
		if err != nil {
			return err
		}
		receipt.Lines = resolvedLines
		receipt.TotalReceivedMinor = totalNet
		if receipt.EntityID == "" {
			receipt.EntityID = po.EntityID
		}
		linesRaw, err := json.Marshal(receipt.Lines)
		if err != nil {
			return fmt.Errorf("marshal receipt lines: %w", err)
		}

		rawBuf := outbox.NewSpannerTxnBuffer(txn)
		buf := &tenantStampedOutboxBuf{inner: rawBuf, tenantID: receipt.TenantID}
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
		journalEntryID, err := runNonSpannerGLAndAudit(txCtx, txn, buf, hooks)
		if err != nil {
			return err
		}
		if journalEntryID == "" {
			journalEntryID = prePostedJournalID
		}
		receipt.JournalEntryID = journalEntryID

		muts := make([]*spanner.Mutation, 0, len(updatedPO.Lines)+4)
		for _, pl := range updatedPO.Lines {
			muts = append(muts, spanner.UpdateMap("PurchaseOrderLines", map[string]any{
				"TenantId":    updatedPO.TenantID,
				"PoId":        updatedPO.PoID,
				"LineNum":     pl.LineNum,
				"QtyReceived": pl.QtyReceived,
			}))
		}
		muts = append(muts, spanner.UpdateMap("PurchaseOrders", map[string]any{
			"TenantId":  updatedPO.TenantID,
			"PoId":      updatedPO.PoID,
			"Status":    updatedPO.Status,
			"UpdatedAt": spanner.CommitTimestamp,
		}))
		muts = append(muts, spanner.InsertMap("GoodsReceipts", map[string]any{
			"TenantId":           receipt.TenantID,
			"ReceiptId":          receipt.ReceiptID,
			"PoId":               receipt.PoID,
			"EntityId":           nullStr(receipt.EntityID),
			"WarehouseId":        receipt.WarehouseID,
			"Status":             receipt.Status,
			"LinesJson":          string(linesRaw),
			"TotalReceivedMinor": receipt.TotalReceivedMinor,
			"JournalEntryId":     nullStr(receipt.JournalEntryID),
			"IdempotencyKey":     nullStr(receipt.IdempotencyKey),
			"ReceivedBy":         receipt.ReceivedBy,
			"ReceivedAt":         receipt.ReceivedAt,
			"CreatedAt":          spanner.CommitTimestamp,
		}))
		if inst, found, iErr := loadProcessInstanceInTxn(txCtx, txn, receipt.TenantID, "", AggregatePurchaseOrder, po.PoID); iErr == nil && found {
			if _, instMut, mErr := buildAdvanceInstanceMutation(inst, ProcessTemplate{}, "GOODS_RECEIVED", receipt.ReceivedBy, "receipt.posted", false, receipt.ReceivedAt); mErr == nil && instMut != nil {
				muts = append(muts, instMut)
			}
		}

		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("buffer GoodsReceipt: %w", err)
		}
		if hooks.OutboxEmit != nil {
			if err := hooks.OutboxEmit(txCtx, buf, journalEntryID); err != nil {
				return err
			}
		}
		if err := rawBuf.Flush(txCtx); err != nil {
			return fmt.Errorf("flush GoodsReceipt outbox: %w", err)
		}

		outRec = receipt
		outPO = updatedPO
		return nil
	})
	if err != nil {
		return GoodsReceipt{}, PurchaseOrder{}, false, err
	}
	return outRec, outPO, already, nil
}

func (r *SpannerRepository) ListGoodsReceipts(ctx context.Context, f ListFilter) ([]GoodsReceipt, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	params := map[string]any{
		"tenantId": strings.TrimSpace(f.TenantID),
		"limit":    int64(clampLimit(f.Limit)),
	}
	sql := `SELECT TenantId, ReceiptId, PoId, EntityId, WarehouseId, Status, LinesJson,
	               TotalReceivedMinor, JournalEntryId, IdempotencyKey, ReceivedBy, ReceivedAt, CreatedAt
	        FROM GoodsReceipts
	        WHERE TenantId = @tenantId`
	if pid := strings.TrimSpace(f.PoID); pid != "" {
		sql += ` AND PoId = @poId`
		params["poId"] = pid
	}
	if eid := strings.TrimSpace(f.EntityID); eid != "" {
		sql += ` AND EntityId = @entityId`
		params["entityId"] = eid
	}
	sql += ` ORDER BY ReceivedAt DESC LIMIT @limit`

	iter := roTxn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var out []GoodsReceipt
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query GoodsReceipts: %w", err)
		}
		rec, err := scanGoodsReceiptRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *SpannerRepository) CreateVendorInvoice(ctx context.Context, inv VendorInvoice, hooks TxnHooks) (VendorInvoice, bool, error) {
	if r == nil || r.client == nil {
		return VendorInvoice{}, false, ErrRepoUnavailable
	}
	var out VendorInvoice
	var already bool
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if inv.IdempotencyKey != "" {
			if ex, ok, err := findVendorInvoiceByIdempotencyInTxn(txCtx, txn, inv.TenantID, inv.IdempotencyKey); err != nil {
				return err
			} else if ok {
				out = ex
				already = true
				return nil
			}
		}

		po, ok, err := loadPurchaseOrderInTxn(txCtx, txn, inv.TenantID, inv.PoID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: purchase order %s not found", ErrNotFound, inv.PoID)
		}
		if inv.ReceiptID != "" {
			recRow, rErr := txn.ReadRow(txCtx, "GoodsReceipts", spanner.Key{inv.TenantID, inv.ReceiptID}, []string{"PoId"})
			if rErr != nil {
				if isNotFound(rErr) {
					return fmt.Errorf("%w: goods receipt %s not found", ErrNotFound, inv.ReceiptID)
				}
				return rErr
			}
			var recPO string
			if err := recRow.Column(0, &recPO); err != nil {
				return err
			}
			if recPO != inv.PoID {
				return fmt.Errorf("%w: goods receipt %s belongs to po %s", ErrInvalidInput, inv.ReceiptID, recPO)
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

		muts := []*spanner.Mutation{
			spanner.InsertMap("VendorInvoices", map[string]any{
				"TenantId":        inv.TenantID,
				"VendorInvoiceId": inv.VendorInvoiceID,
				"PoId":            inv.PoID,
				"ReceiptId":       nullStr(inv.ReceiptID),
				"EsfDocId":        nullStr(inv.EsfDocID),
				"EntityId":        nullStr(inv.EntityID),
				"VendorId":        inv.VendorID,
				"Currency":        inv.Currency,
				"SubtotalMinor":   inv.SubtotalMinor,
				"VatMinor":        inv.VatMinor,
				"TotalMinor":      inv.TotalMinor,
				"MatchStatus":     inv.MatchStatus,
				"VarianceMinor":   inv.VarianceMinor,
				"VarianceReason":  nullStr(inv.VarianceReason),
				"JournalEntryId":  nullStr(inv.JournalEntryID),
				"IdempotencyKey":  nullStr(inv.IdempotencyKey),
				"CreatedBy":       inv.CreatedBy,
				"CreatedAt":       spanner.CommitTimestamp,
				"UpdatedAt":       spanner.CommitTimestamp,
			}),
		}
		if inst, found, iErr := loadProcessInstanceInTxn(txCtx, txn, inv.TenantID, "", AggregatePurchaseOrder, po.PoID); iErr == nil && found {
			if _, instMut, mErr := buildAdvanceInstanceMutation(inst, ProcessTemplate{}, "INVOICE_RECEIVED", inv.CreatedBy, "vendor_invoice.created", false, inv.CreatedAt); mErr == nil && instMut != nil {
				muts = append(muts, instMut)
			}
		}
		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("buffer VendorInvoice: %w", err)
		}
		rawBuf := outbox.NewSpannerTxnBuffer(txn)
		buf := &tenantStampedOutboxBuf{inner: rawBuf, tenantID: inv.TenantID}
		if _, err := runNonSpannerGLAndAudit(txCtx, txn, buf, hooks); err != nil {
			return err
		}
		if err := rawBuf.Flush(txCtx); err != nil {
			return err
		}
		out = inv
		return nil
	})
	if err != nil {
		return VendorInvoice{}, false, err
	}
	return out, already, nil
}

func (r *SpannerRepository) FindVendorInvoiceByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (VendorInvoice, bool, error) {
	if r == nil || r.client == nil {
		return VendorInvoice{}, false, ErrRepoUnavailable
	}
	ik := strings.TrimSpace(idempotencyKey)
	if ik == "" {
		return VendorInvoice{}, false, nil
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, VendorInvoiceId, PoId, ReceiptId, EsfDocId, EntityId, VendorId, Currency,
		             SubtotalMinor, VatMinor, TotalMinor, MatchStatus, VarianceMinor, VarianceReason,
		             JournalEntryId, IdempotencyKey, CreatedBy, CreatedAt, UpdatedAt
		      FROM VendorInvoices WHERE TenantId = @tenantId AND IdempotencyKey = @ik LIMIT 1`,
		Params: map[string]any{"tenantId": strings.TrimSpace(tenantID), "ik": ik},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return VendorInvoice{}, false, nil
	}
	if err != nil {
		return VendorInvoice{}, false, err
	}
	inv, err := scanVendorInvoiceRow(row)
	return inv, err == nil, err
}

func (r *SpannerRepository) GetVendorInvoice(ctx context.Context, tenantID, vendorInvoiceID string) (VendorInvoice, bool, error) {
	if r == nil || r.client == nil {
		return VendorInvoice{}, false, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()
	return loadVendorInvoiceWithReader(ctx, roTxn, strings.TrimSpace(tenantID), strings.TrimSpace(vendorInvoiceID))
}

func (r *SpannerRepository) ListVendorInvoices(ctx context.Context, f ListFilter) ([]VendorInvoice, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	params := map[string]any{
		"tenantId": strings.TrimSpace(f.TenantID),
		"limit":    int64(clampLimit(f.Limit)),
	}
	sql := `SELECT TenantId, VendorInvoiceId, PoId, ReceiptId, EsfDocId, EntityId, VendorId, Currency,
	               SubtotalMinor, VatMinor, TotalMinor, MatchStatus, VarianceMinor, VarianceReason,
	               JournalEntryId, IdempotencyKey, CreatedBy, CreatedAt, UpdatedAt
	        FROM VendorInvoices
	        WHERE TenantId = @tenantId`
	if pid := strings.TrimSpace(f.PoID); pid != "" {
		sql += ` AND PoId = @poId`
		params["poId"] = pid
	}
	if eid := strings.TrimSpace(f.EntityID); eid != "" {
		sql += ` AND EntityId = @entityId`
		params["entityId"] = eid
	}
	if ms := strings.TrimSpace(f.MatchStatus); ms != "" {
		sql += ` AND UPPER(MatchStatus) = UPPER(@matchStatus)`
		params["matchStatus"] = ms
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @limit`

	iter := roTxn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var out []VendorInvoice
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query VendorInvoices: %w", err)
		}
		inv, err := scanVendorInvoiceRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, nil
}

func (r *SpannerRepository) GetInboundESF(ctx context.Context, tenantID, esfDocID string) (InboundESFDoc, bool, error) {
	if r == nil || r.client == nil {
		return InboundESFDoc{}, false, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	row, err := roTxn.ReadRow(ctx, "EsfDocuments",
		spanner.Key{strings.TrimSpace(tenantID), strings.TrimSpace(esfDocID)},
		[]string{"EntityId", "EsfId", "Direction", "PoId", "CounterpartyTin", "TotalMinor", "VatMinor", "VatRateBps", "Status"},
	)
	if err != nil {
		if isNotFound(err) {
			return InboundESFDoc{}, false, nil
		}
		return InboundESFDoc{}, false, fmt.Errorf("read EsfDocuments: %w", err)
	}
	var doc InboundESFDoc
	var poID spanner.NullString
	if err := row.Columns(&doc.EntityID, &doc.EsfID, &doc.Direction, &poID, &doc.CounterpartyTin, &doc.TotalMinor, &doc.VatMinor, &doc.VatRateBps, &doc.Status); err != nil {
		return InboundESFDoc{}, false, fmt.Errorf("scan EsfDocuments: %w", err)
	}
	doc.PoID = poID.StringVal
	return doc, true, nil
}

func (r *SpannerRepository) ApplyThreeWayMatch(ctx context.Context, tenantID, vendorInvoiceID string, update VendorInvoice, matchOK bool, now time.Time, hooks TxnHooks) (VendorInvoice, *ProcessInstance, error) {
	if r == nil || r.client == nil {
		return VendorInvoice{}, nil, ErrRepoUnavailable
	}
	var prePostedJournalID string
	if matchOK && hooks.GLInput != nil {
		if hooks.GLPoster == nil {
			return VendorInvoice{}, nil, ErrGLPosterUnavailable
		}
		if _, isTxnPoster := hooks.GLPoster.(TransactionalGLPoster); !isTxnPoster {
			ex, ok, err := r.GetVendorInvoice(ctx, tenantID, vendorInvoiceID)
			if err != nil {
				return VendorInvoice{}, nil, err
			}
			if !ok {
				return VendorInvoice{}, nil, ErrNotFound
			}
			if ex.MatchStatus == MatchStatusMatched3Way {
				return VendorInvoice{}, nil, fmt.Errorf("%w: vendor invoice %s is already MATCHED_3WAY", ErrConflict, vendorInvoiceID)
			}
			res, gErr := hooks.GLPoster.PostJournal(ctx, *hooks.GLInput)
			if gErr != nil {
				return VendorInvoice{}, nil, gErr
			}
			prePostedJournalID = res.EntryID
			hooks.GLInput = nil
		}
	}

	var outInv VendorInvoice
	var outInst *ProcessInstance

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		existing, ok, err := loadVendorInvoiceWithReader(txCtx, txn, tenantID, vendorInvoiceID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if existing.MatchStatus == MatchStatusMatched3Way {
			return fmt.Errorf("%w: vendor invoice %s is already MATCHED_3WAY", ErrConflict, vendorInvoiceID)
		}

		rawBuf := outbox.NewSpannerTxnBuffer(txn)
		buf := &tenantStampedOutboxBuf{inner: rawBuf, tenantID: tenantID}
		journalEntryID, err := runNonSpannerGLAndAudit(txCtx, txn, buf, hooks)
		if err != nil {
			return err
		}
		if journalEntryID == "" {
			journalEntryID = prePostedJournalID
		}
		if matchOK {
			update.JournalEntryID = journalEntryID
		}

		muts := []*spanner.Mutation{
			spanner.UpdateMap("VendorInvoices", map[string]any{
				"TenantId":        tenantID,
				"VendorInvoiceId": vendorInvoiceID,
				"ReceiptId":       nullStr(update.ReceiptID),
				"EsfDocId":        nullStr(update.EsfDocID),
				"MatchStatus":     update.MatchStatus,
				"VarianceMinor":   update.VarianceMinor,
				"VarianceReason":  nullStr(update.VarianceReason),
				"JournalEntryId":  nullStr(update.JournalEntryID),
				"UpdatedAt":       spanner.CommitTimestamp,
			}),
		}

		if matchOK {
			if inst, found, iErr := loadProcessInstanceInTxn(txCtx, txn, tenantID, "", AggregatePurchaseOrder, update.PoID); iErr == nil && found {
				if advInst, instMut, mErr := buildAdvanceInstanceMutation(inst, ProcessTemplate{}, "MATCHED_3WAY", hooks.ActorID, "three_way_match.completed", true, now); mErr == nil {
					outInst = &advInst
					if instMut != nil {
						muts = append(muts, instMut)
					}
				}
			}
		}

		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("buffer VendorInvoice match: %w", err)
		}
		if hooks.OutboxEmit != nil {
			if err := hooks.OutboxEmit(txCtx, buf, journalEntryID); err != nil {
				return err
			}
		}
		if err := rawBuf.Flush(txCtx); err != nil {
			return fmt.Errorf("flush ThreeWayMatch outbox: %w", err)
		}

		update.UpdatedAt = now
		outInv = update
		return nil
	})
	if err != nil {
		return VendorInvoice{}, nil, err
	}
	return outInv, outInst, nil
}

func (r *SpannerRepository) CollectMiningSamples(ctx context.Context, f MiningFilter, templates []ProcessTemplate) ([]MiningSample, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	var samples []MiningSample

	// 1. Real O2C Orders + OrderStatusTransitions
	if f.Domain == "" || strings.EqualFold(f.Domain, DomainO2C) {
		o2cSamples, err := collectO2CTransitionSamples(ctx, roTxn, f, templates)
		if err != nil {
			return nil, err
		}
		samples = append(samples, o2cSamples...)
	}

	// 2. ProcessInstances
	coveredPOs := make(map[string]struct{})
	instParams := map[string]any{
		"tenantId": f.TenantID,
		"limit":    int64(MiningQueryRowCap),
	}
	instSQL := `SELECT TenantId, InstanceId, TemplateId, EntityId, ObjectType, ObjectId,
	                   CurrentStep, Status, TransitionHistoryJson, StartedAt, CompletedAt, UpdatedAt
	            FROM ProcessInstances
	            WHERE TenantId = @tenantId`
	if f.EntityID != "" {
		instSQL += ` AND EntityId = @entityId`
		instParams["entityId"] = f.EntityID
	}
	instSQL += ` ORDER BY StartedAt DESC LIMIT @limit`

	instIter := roTxn.Query(ctx, spanner.Statement{SQL: instSQL, Params: instParams})
	defer instIter.Stop()
	for {
		row, err := instIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ProcessInstances for mining: %w", err)
		}
		inst, err := scanProcessInstanceRow(row)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(inst.ObjectType, AggregatePurchaseOrder) {
			coveredPOs[inst.ObjectID] = struct{}{}
		}
		dom := inferDomainFromTemplate(templates, inst.TemplateID, "", inst.ObjectType)
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

	// 3. Direct PurchaseOrders, GoodsReceipts, VendorInvoices (for any PO without a ProcessInstance)
	if f.Domain == "" || strings.EqualFold(f.Domain, DomainP2P) {
		p2pSamples, err := collectDirectP2PSamples(ctx, roTxn, f, templates, coveredPOs)
		if err != nil {
			return nil, err
		}
		samples = append(samples, p2pSamples...)
	}

	return samples, nil
}

func collectO2CTransitionSamples(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, f MiningFilter, templates []ProcessTemplate) ([]MiningSample, error) {
	params := map[string]any{
		"tenantId": f.TenantID,
		"limit":    int64(MiningQueryRowCap),
	}
	sql := `SELECT o.OrderId, o.CreatedAt, t.PreviousStatus, t.NewStatus, t.CreatedAt
	        FROM OrderStatusTransitions t
	        JOIN Orders o ON o.OrderId = t.OrderId
	        WHERE o.SupplierId = @tenantId`
	if f.From != nil {
		sql += ` AND t.CreatedAt >= @from`
		params["from"] = *f.From
	}
	if f.To != nil {
		sql += ` AND t.CreatedAt <= @to`
		params["to"] = *f.To
	}
	sql += ` ORDER BY o.OrderId ASC, t.CreatedAt ASC, t.TransitionId ASC LIMIT @limit`

	iter := roTxn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	type rawOrderTrans struct {
		orderID        string
		orderCreatedAt time.Time
		prevStatus     string
		newStatus      string
		createdAt      time.Time
	}
	var rows []rawOrderTrans
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query OrderStatusTransitions for mining: %w", err)
		}
		var (
			oid, newSt string
			prevSt     spanner.NullString
			oTS, tTS   time.Time
		)
		if err := row.Columns(&oid, &oTS, &prevSt, &newSt, &tTS); err != nil {
			return nil, fmt.Errorf("scan OrderStatusTransitions for mining: %w", err)
		}
		rows = append(rows, rawOrderTrans{
			orderID:        oid,
			orderCreatedAt: oTS,
			prevStatus:     prevSt.StringVal,
			newStatus:      newSt,
			createdAt:      tTS,
		})
	}

	samples := make([]MiningSample, 0, len(rows))
	var curOrder string
	var prevTS time.Time
	for _, r := range rows {
		if r.orderID != curOrder {
			curOrder = r.orderID
			prevTS = r.orderCreatedAt
		}
		from := normalizeStepName(r.prevStatus)
		to := normalizeStepName(r.newStatus)
		dur := int64(r.createdAt.Sub(prevTS).Seconds())
		if dur < 0 {
			dur = 0
		}
		samples = append(samples, MiningSample{
			CaseID:      r.orderID,
			Domain:      DomainO2C,
			FromStep:    from,
			ToStep:      to,
			DurationSec: dur,
			SLATarget:   LookupStepSLASeconds(templates, DomainO2C, to),
		})
		prevTS = r.createdAt
	}
	return samples, nil
}

func collectDirectP2PSamples(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, f MiningFilter, templates []ProcessTemplate, coveredPOs map[string]struct{}) ([]MiningSample, error) {
	poParams := map[string]any{"tenantId": f.TenantID, "limit": int64(MiningQueryRowCap)}
	poSQL := `SELECT PoId, CreatedAt FROM PurchaseOrders WHERE TenantId = @tenantId`
	if f.EntityID != "" {
		poSQL += ` AND EntityId = @entityId`
		poParams["entityId"] = f.EntityID
	}
	poSQL += ` ORDER BY CreatedAt DESC LIMIT @limit`

	poCreated := make(map[string]time.Time)
	poIter := roTxn.Query(ctx, spanner.Statement{SQL: poSQL, Params: poParams})
	defer poIter.Stop()
	for {
		row, err := poIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query PurchaseOrders for mining: %w", err)
		}
		var poID string
		var createdAt time.Time
		if err := row.Columns(&poID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan PurchaseOrders for mining: %w", err)
		}
		if _, covered := coveredPOs[poID]; !covered {
			poCreated[poID] = createdAt
		}
	}
	if len(poCreated) == 0 {
		return nil, nil
	}

	recIter := roTxn.Query(ctx, spanner.Statement{
		SQL:    `SELECT PoId, ReceivedAt FROM GoodsReceipts WHERE TenantId = @tenantId ORDER BY ReceivedAt ASC LIMIT @limit`,
		Params: map[string]any{"tenantId": f.TenantID, "limit": int64(MiningQueryRowCap)},
	})
	defer recIter.Stop()
	earliestRec := make(map[string]time.Time)
	for {
		row, err := recIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query GoodsReceipts for mining: %w", err)
		}
		var poID string
		var recAt time.Time
		if err := row.Columns(&poID, &recAt); err != nil {
			return nil, fmt.Errorf("scan GoodsReceipts for mining: %w", err)
		}
		if _, ok := poCreated[poID]; !ok {
			continue
		}
		if prev, exists := earliestRec[poID]; !exists || recAt.Before(prev) {
			earliestRec[poID] = recAt
		}
	}

	invIter := roTxn.Query(ctx, spanner.Statement{
		SQL:    `SELECT PoId, MatchStatus, UpdatedAt FROM VendorInvoices WHERE TenantId = @tenantId ORDER BY UpdatedAt ASC LIMIT @limit`,
		Params: map[string]any{"tenantId": f.TenantID, "limit": int64(MiningQueryRowCap)},
	})
	defer invIter.Stop()
	matchedInvAt := make(map[string]time.Time)
	for {
		row, err := invIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query VendorInvoices for mining: %w", err)
		}
		var poID, ms string
		var updAt time.Time
		if err := row.Columns(&poID, &ms, &updAt); err != nil {
			return nil, fmt.Errorf("scan VendorInvoices for mining: %w", err)
		}
		if ms == MatchStatusMatched3Way {
			if prev, exists := matchedInvAt[poID]; !exists || updAt.Before(prev) {
				matchedInvAt[poID] = updAt
			}
		}
	}

	var samples []MiningSample
	for poID, poTS := range poCreated {
		recTS, hasRec := earliestRec[poID]
		if !hasRec {
			continue
		}
		if (f.From == nil || !recTS.Before(*f.From)) && (f.To == nil || !recTS.After(*f.To)) {
			dur := int64(recTS.Sub(poTS).Seconds())
			if dur < 0 {
				dur = 0
			}
			samples = append(samples, MiningSample{
				CaseID:      poID,
				Domain:      DomainP2P,
				FromStep:    "PO_CREATED",
				ToStep:      "GOODS_RECEIVED",
				DurationSec: dur,
				SLATarget:   LookupStepSLASeconds(templates, DomainP2P, "GOODS_RECEIVED"),
			})
		}
		if matchTS, hasMatch := matchedInvAt[poID]; hasMatch {
			if (f.From == nil || !matchTS.Before(*f.From)) && (f.To == nil || !matchTS.After(*f.To)) {
				dur := int64(matchTS.Sub(recTS).Seconds())
				if dur < 0 {
					dur = 0
				}
				samples = append(samples, MiningSample{
					CaseID:      poID,
					Domain:      DomainP2P,
					FromStep:    "GOODS_RECEIVED",
					ToStep:      "MATCHED_3WAY",
					DurationSec: dur,
					SLATarget:   LookupStepSLASeconds(templates, DomainP2P, "MATCHED_3WAY"),
				})
			}
		}
	}
	return samples, nil
}

type spannerReader interface {
	ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error)
	Query(ctx context.Context, statement spanner.Statement) *spanner.RowIterator
}

func loadPurchaseOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tenantID, poID string) (PurchaseOrder, bool, error) {
	return loadPurchaseOrderWithReader(ctx, txn, tenantID, poID)
}

func loadPurchaseOrderWithReader(ctx context.Context, rdr spannerReader, tenantID, poID string) (PurchaseOrder, bool, error) {
	row, err := rdr.ReadRow(ctx, "PurchaseOrders", spanner.Key{tenantID, poID}, []string{
		"TenantId", "PoId", "EntityId", "VendorId", "VendorName", "VendorTin", "Currency",
		"SubtotalMinor", "VatMinor", "TotalMinor", "Status", "ApprovalRequestId", "IdempotencyKey",
		"ExpectedAt", "CreatedBy", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		if isNotFound(err) {
			return PurchaseOrder{}, false, nil
		}
		return PurchaseOrder{}, false, fmt.Errorf("read PurchaseOrder: %w", err)
	}
	po, err := scanPurchaseOrderRow(row)
	if err != nil {
		return PurchaseOrder{}, false, err
	}
	lines, err := loadPurchaseOrderLinesWithReader(ctx, rdr, tenantID, poID)
	if err != nil {
		return PurchaseOrder{}, false, err
	}
	po.Lines = lines
	return po, true, nil
}

func loadPurchaseOrderLinesWithReader(ctx context.Context, rdr spannerReader, tenantID, poID string) ([]PurchaseOrderLine, error) {
	iter := rdr.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, PoId, LineNum, Sku, ProductName, QtyOrdered, QtyReceived, UnitCostMinor, VatBps, LineTotalMinor
		      FROM PurchaseOrderLines
		      WHERE TenantId = @tenantId AND PoId = @poId
		      ORDER BY LineNum ASC`,
		Params: map[string]any{"tenantId": tenantID, "poId": poID},
	})
	defer iter.Stop()
	var lines []PurchaseOrderLine
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query PurchaseOrderLines: %w", err)
		}
		var (
			l    PurchaseOrderLine
			name spanner.NullString
		)
		if err := row.Columns(&l.TenantID, &l.PoID, &l.LineNum, &l.Sku, &name, &l.QtyOrdered, &l.QtyReceived, &l.UnitCostMinor, &l.VatBps, &l.LineTotalMinor); err != nil {
			return nil, fmt.Errorf("scan PurchaseOrderLine: %w", err)
		}
		l.ProductName = name.StringVal
		lines = append(lines, l)
	}
	return lines, nil
}

func loadVendorInvoiceWithReader(ctx context.Context, rdr spannerReader, tenantID, vendorInvoiceID string) (VendorInvoice, bool, error) {
	row, err := rdr.ReadRow(ctx, "VendorInvoices", spanner.Key{tenantID, vendorInvoiceID}, []string{
		"TenantId", "VendorInvoiceId", "PoId", "ReceiptId", "EsfDocId", "EntityId", "VendorId", "Currency",
		"SubtotalMinor", "VatMinor", "TotalMinor", "MatchStatus", "VarianceMinor", "VarianceReason",
		"JournalEntryId", "IdempotencyKey", "CreatedBy", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		if isNotFound(err) {
			return VendorInvoice{}, false, nil
		}
		return VendorInvoice{}, false, fmt.Errorf("read VendorInvoice: %w", err)
	}
	inv, err := scanVendorInvoiceRow(row)
	if err != nil {
		return VendorInvoice{}, false, err
	}
	return inv, true, nil
}

func loadProcessInstanceInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tenantID, instanceID, objectType, objectID string) (ProcessInstance, bool, error) {
	if instanceID != "" {
		row, err := txn.ReadRow(ctx, "ProcessInstances", spanner.Key{tenantID, instanceID}, []string{
			"TenantId", "InstanceId", "TemplateId", "EntityId", "ObjectType", "ObjectId",
			"CurrentStep", "Status", "TransitionHistoryJson", "StartedAt", "CompletedAt", "UpdatedAt",
		})
		if err != nil {
			if isNotFound(err) {
				return ProcessInstance{}, false, nil
			}
			return ProcessInstance{}, false, err
		}
		inst, err := scanProcessInstanceRow(row)
		return inst, err == nil, err
	}
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, InstanceId, TemplateId, EntityId, ObjectType, ObjectId,
		             CurrentStep, Status, TransitionHistoryJson, StartedAt, CompletedAt, UpdatedAt
		      FROM ProcessInstances
		      WHERE TenantId = @tenantId AND ObjectType = @objectType AND ObjectId = @objectId
		      ORDER BY StartedAt DESC LIMIT 1`,
		Params: map[string]any{"tenantId": tenantID, "objectType": objectType, "objectId": objectID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return ProcessInstance{}, false, nil
	}
	if err != nil {
		return ProcessInstance{}, false, err
	}
	inst, err := scanProcessInstanceRow(row)
	return inst, err == nil, err
}

func buildAdvanceInstanceMutation(inst ProcessInstance, tpl ProcessTemplate, nextStep, actorID, reason string, complete bool, now time.Time) (ProcessInstance, *spanner.Mutation, error) {
	nextUpper := strings.ToUpper(strings.TrimSpace(nextStep))
	if nextUpper == "" {
		return inst, nil, nil
	}
	changed := false
	if !strings.EqualFold(inst.CurrentStep, nextUpper) {
		inst.Transitions = append(inst.Transitions, ProcessTransition{
			FromStep:   inst.CurrentStep,
			ToStep:     nextUpper,
			ActorID:    actorID,
			Reason:     reason,
			OccurredAt: now,
		})
		inst.CurrentStep = nextUpper
		changed = true
	}
	if complete || (len(tpl.Steps) > 0 && strings.EqualFold(nextUpper, tpl.Steps[len(tpl.Steps)-1].Code)) {
		if inst.Status != InstanceStatusCompleted {
			inst.Status = InstanceStatusCompleted
			doneAt := now
			inst.CompletedAt = &doneAt
			changed = true
		}
	}
	if !changed {
		return inst, nil, nil
	}
	histRaw, err := json.Marshal(inst.Transitions)
	if err != nil {
		return ProcessInstance{}, nil, fmt.Errorf("marshal transition history: %w", err)
	}
	inst.UpdatedAt = now
	return inst, spanner.UpdateMap("ProcessInstances", map[string]any{
		"TenantId":              inst.TenantID,
		"InstanceId":            inst.InstanceID,
		"CurrentStep":           inst.CurrentStep,
		"Status":                inst.Status,
		"TransitionHistoryJson": string(histRaw),
		"CompletedAt":           nullTime(inst.CompletedAt),
		"UpdatedAt":             spanner.CommitTimestamp,
	}), nil
}

func findPOByIdempotencyInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tenantID, idemKey string) (PurchaseOrder, bool, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT PoId FROM PurchaseOrders WHERE TenantId = @tenantId AND IdempotencyKey = @ik LIMIT 1`,
		Params: map[string]any{"tenantId": tenantID, "ik": idemKey},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return PurchaseOrder{}, false, nil
	}
	if err != nil {
		return PurchaseOrder{}, false, err
	}
	var poID string
	if err := row.Column(0, &poID); err != nil {
		return PurchaseOrder{}, false, err
	}
	return loadPurchaseOrderInTxn(ctx, txn, tenantID, poID)
}

func findReceiptByIdempotencyInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tenantID, idemKey string) (GoodsReceipt, bool, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, ReceiptId, PoId, EntityId, WarehouseId, Status, LinesJson,
		             TotalReceivedMinor, JournalEntryId, IdempotencyKey, ReceivedBy, ReceivedAt, CreatedAt
		      FROM GoodsReceipts WHERE TenantId = @tenantId AND IdempotencyKey = @ik LIMIT 1`,
		Params: map[string]any{"tenantId": tenantID, "ik": idemKey},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return GoodsReceipt{}, false, nil
	}
	if err != nil {
		return GoodsReceipt{}, false, err
	}
	rec, err := scanGoodsReceiptRow(row)
	return rec, err == nil, err
}

func findVendorInvoiceByIdempotencyInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tenantID, idemKey string) (VendorInvoice, bool, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, VendorInvoiceId, PoId, ReceiptId, EsfDocId, EntityId, VendorId, Currency,
		             SubtotalMinor, VatMinor, TotalMinor, MatchStatus, VarianceMinor, VarianceReason,
		             JournalEntryId, IdempotencyKey, CreatedBy, CreatedAt, UpdatedAt
		      FROM VendorInvoices WHERE TenantId = @tenantId AND IdempotencyKey = @ik LIMIT 1`,
		Params: map[string]any{"tenantId": tenantID, "ik": idemKey},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return VendorInvoice{}, false, nil
	}
	if err != nil {
		return VendorInvoice{}, false, err
	}
	inv, err := scanVendorInvoiceRow(row)
	return inv, err == nil, err
}

func scanProcessInstanceRow(row *spanner.Row) (ProcessInstance, error) {
	var (
		inst     ProcessInstance
		entityID spanner.NullString
		histRaw  string
		compAt   spanner.NullTime
	)
	if err := row.Columns(
		&inst.TenantID, &inst.InstanceID, &inst.TemplateID, &entityID, &inst.ObjectType, &inst.ObjectID,
		&inst.CurrentStep, &inst.Status, &histRaw, &inst.StartedAt, &compAt, &inst.UpdatedAt,
	); err != nil {
		return ProcessInstance{}, fmt.Errorf("scan ProcessInstance: %w", err)
	}
	inst.EntityID = entityID.StringVal
	if compAt.Valid {
		t := compAt.Time
		inst.CompletedAt = &t
	}
	_ = json.Unmarshal([]byte(histRaw), &inst.Transitions)
	return inst, nil
}

func scanPurchaseOrderRow(row *spanner.Row) (PurchaseOrder, error) {
	var (
		po                                                 PurchaseOrder
		entityID, vendorName, vendorTin, appReqID, idemKey spanner.NullString
		expectedAt                                         spanner.NullTime
	)
	if err := row.Columns(
		&po.TenantID, &po.PoID, &entityID, &po.VendorID, &vendorName, &vendorTin, &po.Currency,
		&po.SubtotalMinor, &po.VatMinor, &po.TotalMinor, &po.Status, &appReqID, &idemKey,
		&expectedAt, &po.CreatedBy, &po.CreatedAt, &po.UpdatedAt,
	); err != nil {
		return PurchaseOrder{}, fmt.Errorf("scan PurchaseOrder: %w", err)
	}
	po.EntityID = entityID.StringVal
	po.VendorName = vendorName.StringVal
	po.VendorTin = vendorTin.StringVal
	po.ApprovalRequestID = appReqID.StringVal
	po.IdempotencyKey = idemKey.StringVal
	if expectedAt.Valid {
		t := expectedAt.Time
		po.ExpectedAt = &t
	}
	return po, nil
}

func scanGoodsReceiptRow(row *spanner.Row) (GoodsReceipt, error) {
	var (
		rec                               GoodsReceipt
		entityID, linesRaw, jeID, idemKey spanner.NullString
	)
	if err := row.Columns(
		&rec.TenantID, &rec.ReceiptID, &rec.PoID, &entityID, &rec.WarehouseID, &rec.Status, &linesRaw,
		&rec.TotalReceivedMinor, &jeID, &idemKey, &rec.ReceivedBy, &rec.ReceivedAt, &rec.CreatedAt,
	); err != nil {
		return GoodsReceipt{}, fmt.Errorf("scan GoodsReceipt: %w", err)
	}
	rec.EntityID = entityID.StringVal
	rec.JournalEntryID = jeID.StringVal
	rec.IdempotencyKey = idemKey.StringVal
	if linesRaw.Valid && linesRaw.StringVal != "" {
		_ = json.Unmarshal([]byte(linesRaw.StringVal), &rec.Lines)
	}
	return rec, nil
}

func scanVendorInvoiceRow(row *spanner.Row) (VendorInvoice, error) {
	var (
		inv                                              VendorInvoice
		recID, esfID, entityID, varReason, jeID, idemKey spanner.NullString
	)
	if err := row.Columns(
		&inv.TenantID, &inv.VendorInvoiceID, &inv.PoID, &recID, &esfID, &entityID, &inv.VendorID, &inv.Currency,
		&inv.SubtotalMinor, &inv.VatMinor, &inv.TotalMinor, &inv.MatchStatus, &inv.VarianceMinor, &varReason,
		&jeID, &idemKey, &inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
	); err != nil {
		return VendorInvoice{}, fmt.Errorf("scan VendorInvoice: %w", err)
	}
	inv.ReceiptID = recID.StringVal
	inv.EsfDocID = esfID.StringVal
	inv.EntityID = entityID.StringVal
	inv.VarianceReason = varReason.StringVal
	inv.JournalEntryID = jeID.StringVal
	inv.IdempotencyKey = idemKey.StringVal
	return inv, nil
}

func isNotFound(err error) bool {
	return status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound
}

func nullStr(s string) any {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return t
}

func nullTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC()
}
