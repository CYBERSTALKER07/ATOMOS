package approvals

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SpannerRepository persists approval policies, requests, decisions, and delegations in Cloud Spanner.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed approvals repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

// Client exposes the underlying *spanner.Client.
func (r *SpannerRepository) Client() *spanner.Client {
	if r == nil {
		return nil
	}
	return r.client
}

type entityStampedOutboxBuf struct {
	inner    outbox.TxnBuffer
	entityID string
}

func (b *entityStampedOutboxBuf) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.entityID) != "" {
		e.SupplierID = outbox.ResolveSupplierID(b.entityID, e.Payload)
	}
	return b.inner.BufferOutbox(ctx, e)
}

func (r *SpannerRepository) CreatePolicy(ctx context.Context, p Policy) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	stepsRaw, err := json.Marshal(p.Steps)
	if err != nil {
		return fmt.Errorf("%w: marshal steps: %v", ErrInvalidInput, err)
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("ApprovalPolicies", map[string]any{
				"TenantId":      p.TenantID,
				"PolicyId":      p.PolicyID,
				"Name":          p.Name,
				"ObjectType":    p.ObjectType,
				"Action":        p.Action,
				"ConditionExpr": p.ConditionExpr,
				"StepsJson":     string(stepsRaw),
				"Version":       p.Version,
				"Enabled":       p.Enabled,
				"CreatedBy":     p.CreatedBy,
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		})
	})
}

func (r *SpannerRepository) UpdatePolicy(ctx context.Context, p Policy) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	stepsRaw, err := json.Marshal(p.Steps)
	if err != nil {
		return fmt.Errorf("%w: marshal steps: %v", ErrInvalidInput, err)
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, readErr := txn.ReadRow(txCtx, "ApprovalPolicies", spanner.Key{p.TenantID, p.PolicyID}, []string{"PolicyId"})
		if readErr != nil {
			if status.Code(readErr) == codes.NotFound || spanner.ErrCode(readErr) == codes.NotFound {
				return ErrPolicyNotFound
			}
			return fmt.Errorf("read ApprovalPolicies: %w", readErr)
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ApprovalPolicies", map[string]any{
				"TenantId":      p.TenantID,
				"PolicyId":      p.PolicyID,
				"Name":          p.Name,
				"ObjectType":    p.ObjectType,
				"Action":        p.Action,
				"ConditionExpr": p.ConditionExpr,
				"StepsJson":     string(stepsRaw),
				"Version":       p.Version,
				"Enabled":       p.Enabled,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		})
	})
}

func (r *SpannerRepository) GetPolicy(ctx context.Context, tenantID, policyID string) (Policy, bool, error) {
	if r == nil || r.client == nil {
		return Policy{}, false, ErrRepoUnavailable
	}
	row, err := r.client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ApprovalPolicies",
		spanner.Key{strings.TrimSpace(tenantID), strings.TrimSpace(policyID)},
		[]string{"TenantId", "PolicyId", "Name", "ObjectType", "Action", "ConditionExpr", "StepsJson", "Version", "Enabled", "CreatedBy", "CreatedAt", "UpdatedAt"},
	)
	if err != nil {
		if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
			return Policy{}, false, nil
		}
		return Policy{}, false, fmt.Errorf("read ApprovalPolicy: %w", err)
	}
	pol, err := scanPolicyRow(row)
	if err != nil {
		return Policy{}, false, err
	}
	return pol, true, nil
}

func (r *SpannerRepository) ListPolicies(ctx context.Context, tenantID, objectType, action string, enabledOnly bool) ([]Policy, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	params := map[string]any{"tenantId": strings.TrimSpace(tenantID)}
	sql := `SELECT TenantId, PolicyId, Name, ObjectType, Action, ConditionExpr, StepsJson, Version, Enabled, CreatedBy, CreatedAt, UpdatedAt
	        FROM ApprovalPolicies
	        WHERE TenantId = @tenantId`
	if ot := strings.TrimSpace(objectType); ot != "" {
		sql += ` AND LOWER(ObjectType) = LOWER(@objectType)`
		params["objectType"] = ot
	}
	if act := strings.TrimSpace(action); act != "" {
		sql += ` AND LOWER(Action) = LOWER(@action)`
		params["action"] = act
	}
	if enabledOnly {
		sql += ` AND Enabled = TRUE`
	}
	sql += ` ORDER BY CreatedAt DESC, PolicyId ASC`

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	out := make([]Policy, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ApprovalPolicies: %w", err)
		}
		pol, err := scanPolicyRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, pol)
	}
	return out, nil
}

func (r *SpannerRepository) FindRequestByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (ApprovalRequest, bool, error) {
	if r == nil || r.client == nil {
		return ApprovalRequest{}, false, ErrRepoUnavailable
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ApprovalRequest{}, false, nil
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, RequestId, PolicyId, PolicyVersion, ObjectType, ObjectId, Action, AmountMinor,
		             RequestedBy, ActorType, PayloadJson, PayloadHash, Status, CurrentStep, RequiredApprovers,
		             BreakGlass, BreakGlassReason, ExpiresAt, IdempotencyKey, ResolvedAt, CreatedAt, UpdatedAt
		      FROM ApprovalRequests@{FORCE_INDEX=UQ_ApprovalRequests_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
		      WHERE TenantId = @tenantId AND IdempotencyKey = @idemKey AND IdempotencyKey IS NOT NULL
		      LIMIT 1`,
		Params: map[string]any{
			"tenantId": strings.TrimSpace(tenantID),
			"idemKey":  strings.TrimSpace(idempotencyKey),
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return ApprovalRequest{}, false, nil
	}
	if err != nil {
		return ApprovalRequest{}, false, fmt.Errorf("lookup approval request idempotency: %w", err)
	}
	req, err := scanRequestRow(row)
	if err != nil {
		return ApprovalRequest{}, false, err
	}
	return req, true, nil
}

func (r *SpannerRepository) CreateRequest(ctx context.Context, req ApprovalRequest, auditRec AuditRecorder) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if req.IdempotencyKey != "" {
			iter := txn.Query(txCtx, spanner.Statement{
				SQL: `SELECT RequestId
				      FROM ApprovalRequests@{FORCE_INDEX=UQ_ApprovalRequests_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
				      WHERE TenantId = @tenantId AND IdempotencyKey = @idemKey AND IdempotencyKey IS NOT NULL
				      LIMIT 1`,
				Params: map[string]any{
					"tenantId": req.TenantID,
					"idemKey":  req.IdempotencyKey,
				},
			})
			_, err := iter.Next()
			iter.Stop()
			if err == nil {
				return nil
			}
			if err != iterator.Done {
				return fmt.Errorf("check approval idempotency in txn: %w", err)
			}
		}

		var resolvedAt any
		if req.Status == StatusApproved || req.Status == StatusRejected {
			resolvedAt = spanner.CommitTimestamp
		}
		var expiresAt any
		if req.ExpiresAt != nil {
			expiresAt = req.ExpiresAt.UTC()
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("ApprovalRequests", map[string]any{
				"TenantId":          req.TenantID,
				"RequestId":         req.RequestID,
				"PolicyId":          nullIfEmpty(req.PolicyID),
				"PolicyVersion":     req.PolicyVersion,
				"ObjectType":        req.ObjectType,
				"ObjectId":          req.ObjectID,
				"Action":            req.Action,
				"AmountMinor":       req.AmountMinor,
				"RequestedBy":       req.RequestedBy,
				"ActorType":         req.ActorType,
				"PayloadJson":       req.PayloadJSON,
				"PayloadHash":       req.PayloadHash,
				"Status":            req.Status,
				"CurrentStep":       req.CurrentStep,
				"RequiredApprovers": req.RequiredApprovers,
				"BreakGlass":        req.BreakGlass,
				"BreakGlassReason":  nullIfEmpty(req.BreakGlassReason),
				"ExpiresAt":         expiresAt,
				"IdempotencyKey":    nullIfEmpty(req.IdempotencyKey),
				"ResolvedAt":        resolvedAt,
				"CreatedAt":         spanner.CommitTimestamp,
				"UpdatedAt":         spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("insert ApprovalRequests: %w", err)
		}

		expStr := ""
		if req.ExpiresAt != nil {
			expStr = req.ExpiresAt.UTC().Format(time.RFC3339)
		}
		outboxInner := outbox.NewSpannerTxnBuffer(txn)
		stampedBuf := &entityStampedOutboxBuf{inner: outboxInner, entityID: req.TenantID}
		ev := events.ApprovalRequestedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventApprovalRequested,
				Version:   1,
				Timestamp: req.CreatedAt.UTC().Format(time.RFC3339Nano),
			},
			TenantID:          req.TenantID,
			RequestID:         req.RequestID,
			PolicyID:          req.PolicyID,
			PolicyVersion:     req.PolicyVersion,
			ObjectType:        req.ObjectType,
			ObjectID:          req.ObjectID,
			Action:            req.Action,
			AmountMinor:       req.AmountMinor,
			RequestedBy:       req.RequestedBy,
			ActorType:         req.ActorType,
			PayloadHash:       req.PayloadHash,
			Status:            req.Status,
			RequiredApprovers: req.RequiredApprovers,
			BreakGlass:        req.BreakGlass,
			ExpiresAt:         expStr,
		}
		if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateApprovalRequest, req.RequestID, events.TopicMain, ev); err != nil {
			return fmt.Errorf("emit approval.requested outbox: %w", err)
		}
		if err := outboxInner.Flush(txCtx); err != nil {
			return fmt.Errorf("flush approval.requested outbox: %w", err)
		}

		if auditRec != nil {
			if err := auditRec.RecordEventInTxn(txCtx, txn, AuditRecordInput{
				TenantID:   req.TenantID,
				ActorID:    req.RequestedBy,
				ActorType:  req.ActorType,
				Action:     "approval.requested",
				ObjectType: "ApprovalRequest",
				ObjectID:   req.RequestID,
				AfterJSON:  req.PayloadJSON,
				ReasonCode: req.Action,
				ApprovalID: req.RequestID,
			}); err != nil {
				return fmt.Errorf("audit approval.requested: %w", err)
			}
		}
		return nil
	})
}

func (r *SpannerRepository) GetRequest(ctx context.Context, tenantID, requestID string) (ApprovalRequest, []ApprovalDecision, bool, error) {
	if r == nil || r.client == nil {
		return ApprovalRequest{}, nil, false, ErrRepoUnavailable
	}
	var req ApprovalRequest
	var decs []ApprovalDecision
	var found bool
	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(txCtx context.Context, ro *spanner.ReadOnlyTransaction) error {
		row, err := ro.ReadRow(txCtx, "ApprovalRequests", spanner.Key{strings.TrimSpace(tenantID), strings.TrimSpace(requestID)}, requestColumns)
		if err != nil {
			if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
				found = false
				return nil
			}
			return fmt.Errorf("read ApprovalRequests: %w", err)
		}
		req, err = scanRequestRow(row)
		if err != nil {
			return err
		}
		found = true
		decs, err = readDecisionsInTxn(txCtx, ro, req.TenantID, req.RequestID)
		return err
	})
	if err != nil {
		return ApprovalRequest{}, nil, false, err
	}
	if !found {
		return ApprovalRequest{}, nil, false, nil
	}
	return req, decs, true, nil
}

func (r *SpannerRepository) GetRequestByID(ctx context.Context, requestID string) (ApprovalRequest, []ApprovalDecision, bool, error) {
	if r == nil || r.client == nil {
		return ApprovalRequest{}, nil, false, ErrRepoUnavailable
	}
	var req ApprovalRequest
	var decs []ApprovalDecision
	var found bool
	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(txCtx context.Context, ro *spanner.ReadOnlyTransaction) error {
		iter := ro.Query(txCtx, spanner.Statement{
			SQL: `SELECT TenantId, RequestId, PolicyId, PolicyVersion, ObjectType, ObjectId, Action, AmountMinor,
			             RequestedBy, ActorType, PayloadJson, PayloadHash, Status, CurrentStep, RequiredApprovers,
			             BreakGlass, BreakGlassReason, ExpiresAt, IdempotencyKey, ResolvedAt, CreatedAt, UpdatedAt
			      FROM ApprovalRequests
			      WHERE RequestId = @requestId
			      LIMIT 1`,
			Params: map[string]any{"requestId": strings.TrimSpace(requestID)},
		})
		row, err := iter.Next()
		iter.Stop()
		if err == iterator.Done {
			found = false
			return nil
		}
		if err != nil {
			return fmt.Errorf("query ApprovalRequests by RequestId: %w", err)
		}
		req, err = scanRequestRow(row)
		if err != nil {
			return err
		}
		found = true
		decs, err = readDecisionsInTxn(txCtx, ro, req.TenantID, req.RequestID)
		return err
	})
	if err != nil {
		return ApprovalRequest{}, nil, false, err
	}
	if !found {
		return ApprovalRequest{}, nil, false, nil
	}
	return req, decs, true, nil
}

func (r *SpannerRepository) ListInbox(ctx context.Context, f InboxFilter) ([]ApprovalRequest, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	params := map[string]any{
		"tenantId": strings.TrimSpace(f.TenantID),
		"limit":    int64(limit),
	}
	sql := `SELECT TenantId, RequestId, PolicyId, PolicyVersion, ObjectType, ObjectId, Action, AmountMinor,
	               RequestedBy, ActorType, PayloadJson, PayloadHash, Status, CurrentStep, RequiredApprovers,
	               BreakGlass, BreakGlassReason, ExpiresAt, IdempotencyKey, ResolvedAt, CreatedAt, UpdatedAt
	        FROM ApprovalRequests
	        WHERE TenantId = @tenantId`
	if st := strings.ToUpper(strings.TrimSpace(f.Status)); st != "" {
		sql += ` AND Status = @status`
		params["status"] = st
	}
	if ot := strings.TrimSpace(f.ObjectType); ot != "" {
		sql += ` AND LOWER(ObjectType) = LOWER(@objectType)`
		params["objectType"] = ot
	}
	if act := strings.TrimSpace(f.Action); act != "" {
		sql += ` AND LOWER(Action) = LOWER(@action)`
		params["action"] = act
	}
	if f.BreakGlass != nil {
		sql += ` AND BreakGlass = @breakGlass`
		params["breakGlass"] = *f.BreakGlass
	}
	sql += ` ORDER BY CreatedAt DESC, RequestId DESC LIMIT @limit`

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	out := make([]ApprovalRequest, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list ApprovalRequests inbox: %w", err)
		}
		req, err := scanRequestRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, nil
}

func (r *SpannerRepository) ApplyDecision(
	ctx context.Context,
	tenantID, requestID string,
	mutateFn func(req ApprovalRequest, existingDecisions []ApprovalDecision, policy *Policy) (ApprovalRequest, ApprovalDecision, error),
	auditRec AuditRecorder,
	traceID string,
) (ApprovalRequest, ApprovalDecision, error) {
	if r == nil || r.client == nil {
		return ApprovalRequest{}, ApprovalDecision{}, ErrRepoUnavailable
	}
	var updatedReq ApprovalRequest
	var recordedDec ApprovalDecision
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(txCtx, "ApprovalRequests", spanner.Key{strings.TrimSpace(tenantID), strings.TrimSpace(requestID)}, requestColumns)
		if err != nil {
			if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
				return ErrRequestNotFound
			}
			return fmt.Errorf("read ApprovalRequests in txn: %w", err)
		}
		req, err := scanRequestRow(row)
		if err != nil {
			return err
		}
		existing, err := readDecisionsInTxn(txCtx, txn, req.TenantID, req.RequestID)
		if err != nil {
			return err
		}
		var polPtr *Policy
		if req.PolicyID != "" {
			pRow, pErr := txn.ReadRow(txCtx, "ApprovalPolicies", spanner.Key{req.TenantID, req.PolicyID},
				[]string{"TenantId", "PolicyId", "Name", "ObjectType", "Action", "ConditionExpr", "StepsJson", "Version", "Enabled", "CreatedBy", "CreatedAt", "UpdatedAt"})
			if pErr == nil {
				pol, sErr := scanPolicyRow(pRow)
				if sErr == nil {
					polPtr = &pol
				}
			}
		}

		updated, dec, err := mutateFn(req, existing, polPtr)
		if err != nil {
			return err
		}

		var resolvedAt any
		if updated.Status == StatusApproved || updated.Status == StatusRejected {
			resolvedAt = spanner.CommitTimestamp
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ApprovalRequests", map[string]any{
				"TenantId":          updated.TenantID,
				"RequestId":         updated.RequestID,
				"Status":            updated.Status,
				"CurrentStep":       updated.CurrentStep,
				"RequiredApprovers": updated.RequiredApprovers,
				"BreakGlass":        updated.BreakGlass,
				"BreakGlassReason":  nullIfEmpty(updated.BreakGlassReason),
				"ResolvedAt":        resolvedAt,
				"UpdatedAt":         spanner.CommitTimestamp,
			}),
			spanner.InsertMap("ApprovalDecisions", map[string]any{
				"TenantId":      dec.TenantID,
				"RequestId":     dec.RequestID,
				"DecisionId":    dec.DecisionID,
				"StepIndex":     dec.StepIndex,
				"ApproverId":    dec.ApproverID,
				"DelegatedFrom": nullIfEmpty(dec.DelegatedFrom),
				"Decision":      dec.Decision,
				"Comment":       nullIfEmpty(dec.Comment),
				"MfaVerified":   dec.MfaVerified,
				"CreatedAt":     spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("buffer approval decision mutations: %w", err)
		}

		outboxInner := outbox.NewSpannerTxnBuffer(txn)
		stampedBuf := &entityStampedOutboxBuf{inner: outboxInner, entityID: updated.TenantID}
		ev := events.ApprovalDecidedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventApprovalDecided,
				Version:   1,
				Timestamp: dec.CreatedAt.UTC().Format(time.RFC3339Nano),
			},
			TenantID:      updated.TenantID,
			RequestID:     updated.RequestID,
			DecisionID:    dec.DecisionID,
			StepIndex:     dec.StepIndex,
			ApproverID:    dec.ApproverID,
			DelegatedFrom: dec.DelegatedFrom,
			Decision:      dec.Decision,
			Status:        updated.Status,
			MfaVerified:   dec.MfaVerified,
			BreakGlass:    updated.BreakGlass,
			Comment:       dec.Comment,
		}
		if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateApprovalRequest, updated.RequestID, events.TopicMain, ev); err != nil {
			return fmt.Errorf("emit approval.decided outbox: %w", err)
		}
		if err := outboxInner.Flush(txCtx); err != nil {
			return fmt.Errorf("flush approval.decided outbox: %w", err)
		}

		if auditRec != nil {
			reasonCode := dec.Decision
			if updated.BreakGlass {
				reasonCode = "BREAK_GLASS"
			}
			if err := auditRec.RecordEventInTxn(txCtx, txn, AuditRecordInput{
				TenantID:   updated.TenantID,
				ActorID:    dec.ApproverID,
				ActorType:  "human",
				Action:     "approval." + strings.ToLower(dec.Decision),
				ObjectType: "ApprovalRequest",
				ObjectID:   updated.RequestID,
				AfterJSON:  updated.PayloadJSON,
				ReasonCode: reasonCode,
				ApprovalID: updated.RequestID,
				TraceID:    traceID,
			}); err != nil {
				return fmt.Errorf("audit approval.decided: %w", err)
			}
		}

		updatedReq = updated
		recordedDec = dec
		return nil
	})
	if err != nil {
		return ApprovalRequest{}, ApprovalDecision{}, err
	}
	return updatedReq, recordedDec, nil
}

func (r *SpannerRepository) CreateDelegation(ctx context.Context, d ApprovalDelegation) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("ApprovalDelegations", map[string]any{
				"TenantId":     d.TenantID,
				"DelegationId": d.DelegationID,
				"DelegatorId":  d.DelegatorID,
				"DelegateId":   d.DelegateID,
				"RoleScope":    nullIfEmpty(d.RoleScope),
				"StartsAt":     d.StartsAt.UTC(),
				"EndsAt":       d.EndsAt.UTC(),
				"Reason":       nullIfEmpty(d.Reason),
				"Enabled":      d.Enabled,
				"CreatedAt":    spanner.CommitTimestamp,
			}),
		})
	})
}

func (r *SpannerRepository) GetDelegation(ctx context.Context, tenantID, delegationID string) (ApprovalDelegation, bool, error) {
	if r == nil || r.client == nil {
		return ApprovalDelegation{}, false, ErrRepoUnavailable
	}
	row, err := r.client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ApprovalDelegations",
		spanner.Key{strings.TrimSpace(tenantID), strings.TrimSpace(delegationID)},
		[]string{"TenantId", "DelegationId", "DelegatorId", "DelegateId", "RoleScope", "StartsAt", "EndsAt", "Reason", "Enabled", "CreatedAt"},
	)
	if err != nil {
		if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
			return ApprovalDelegation{}, false, nil
		}
		return ApprovalDelegation{}, false, fmt.Errorf("read ApprovalDelegations: %w", err)
	}
	d, err := scanDelegationRow(row)
	if err != nil {
		return ApprovalDelegation{}, false, err
	}
	return d, true, nil
}

func (r *SpannerRepository) ListDelegations(ctx context.Context, tenantID, actorID string) ([]ApprovalDelegation, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	params := map[string]any{"tenantId": strings.TrimSpace(tenantID)}
	sql := `SELECT TenantId, DelegationId, DelegatorId, DelegateId, RoleScope, StartsAt, EndsAt, Reason, Enabled, CreatedAt
	        FROM ApprovalDelegations
	        WHERE TenantId = @tenantId`
	if aid := strings.TrimSpace(actorID); aid != "" {
		sql += ` AND (DelegatorId = @actorId OR DelegateId = @actorId)`
		params["actorId"] = aid
	}
	sql += ` ORDER BY CreatedAt DESC`

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	out := make([]ApprovalDelegation, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list ApprovalDelegations: %w", err)
		}
		d, err := scanDelegationRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *SpannerRepository) FindActiveDelegationsForDelegate(ctx context.Context, tenantID, delegateID string, at time.Time) ([]ApprovalDelegation, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, DelegationId, DelegatorId, DelegateId, RoleScope, StartsAt, EndsAt, Reason, Enabled, CreatedAt
		      FROM ApprovalDelegations@{FORCE_INDEX=Idx_ApprovalDelegations_ByDelegate}
		      WHERE TenantId = @tenantId AND DelegateId = @delegateId AND Enabled = TRUE AND EndsAt > @at
		      ORDER BY EndsAt DESC`,
		Params: map[string]any{
			"tenantId":   strings.TrimSpace(tenantID),
			"delegateId": strings.TrimSpace(delegateID),
			"at":         at.UTC(),
		},
	})
	defer iter.Stop()

	out := make([]ApprovalDelegation, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query active delegations: %w", err)
		}
		d, err := scanDelegationRow(row)
		if err != nil {
			return nil, err
		}
		if !d.StartsAt.IsZero() && at.Before(d.StartsAt) {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *SpannerRepository) SetDelegationEnabled(ctx context.Context, tenantID, delegationID string, enabled bool) (ApprovalDelegation, error) {
	if r == nil || r.client == nil {
		return ApprovalDelegation{}, ErrRepoUnavailable
	}
	var out ApprovalDelegation
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(txCtx, "ApprovalDelegations",
			spanner.Key{strings.TrimSpace(tenantID), strings.TrimSpace(delegationID)},
			[]string{"TenantId", "DelegationId", "DelegatorId", "DelegateId", "RoleScope", "StartsAt", "EndsAt", "Reason", "Enabled", "CreatedAt"},
		)
		if err != nil {
			if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
				return ErrDelegationNotFound
			}
			return fmt.Errorf("read ApprovalDelegations in txn: %w", err)
		}
		d, err := scanDelegationRow(row)
		if err != nil {
			return err
		}
		d.Enabled = enabled
		out = d
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ApprovalDelegations", map[string]any{
				"TenantId":     d.TenantID,
				"DelegationId": d.DelegationID,
				"Enabled":      enabled,
			}),
		})
	})
	if err != nil {
		return ApprovalDelegation{}, err
	}
	return out, nil
}

var requestColumns = []string{
	"TenantId", "RequestId", "PolicyId", "PolicyVersion", "ObjectType", "ObjectId", "Action", "AmountMinor",
	"RequestedBy", "ActorType", "PayloadJson", "PayloadHash", "Status", "CurrentStep", "RequiredApprovers",
	"BreakGlass", "BreakGlassReason", "ExpiresAt", "IdempotencyKey", "ResolvedAt", "CreatedAt", "UpdatedAt",
}

type spannerQueryer interface {
	Query(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator
}

func readDecisionsInTxn(ctx context.Context, q spannerQueryer, tenantID, requestID string) ([]ApprovalDecision, error) {
	iter := q.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, RequestId, DecisionId, StepIndex, ApproverId, DelegatedFrom, Decision, Comment, MfaVerified, CreatedAt
		      FROM ApprovalDecisions
		      WHERE TenantId = @tenantId AND RequestId = @requestId
		      ORDER BY CreatedAt ASC, DecisionId ASC`,
		Params: map[string]any{
			"tenantId":  tenantID,
			"requestId": requestID,
		},
	})
	defer iter.Stop()
	out := make([]ApprovalDecision, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ApprovalDecisions: %w", err)
		}
		var d ApprovalDecision
		var delFrom, comment spanner.NullString
		if err := row.Columns(&d.TenantID, &d.RequestID, &d.DecisionID, &d.StepIndex, &d.ApproverID, &delFrom, &d.Decision, &comment, &d.MfaVerified, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan ApprovalDecision: %w", err)
		}
		d.DelegatedFrom = nullStrVal(delFrom)
		d.Comment = nullStrVal(comment)
		d.CreatedAt = d.CreatedAt.UTC()
		out = append(out, d)
	}
	return out, nil
}

func scanPolicyRow(row *spanner.Row) (Policy, error) {
	var p Policy
	var condNull spanner.NullString
	var stepsJSON string
	if err := row.Columns(&p.TenantID, &p.PolicyID, &p.Name, &p.ObjectType, &p.Action, &condNull, &stepsJSON, &p.Version, &p.Enabled, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return Policy{}, fmt.Errorf("scan ApprovalPolicy: %w", err)
	}
	p.ConditionExpr = nullStrVal(condNull)
	if strings.TrimSpace(stepsJSON) != "" {
		if err := json.Unmarshal([]byte(stepsJSON), &p.Steps); err != nil {
			return Policy{}, fmt.Errorf("unmarshal StepsJson: %w", err)
		}
	}
	p.CreatedAt = p.CreatedAt.UTC()
	p.UpdatedAt = p.UpdatedAt.UTC()
	return p, nil
}

func scanRequestRow(row *spanner.Row) (ApprovalRequest, error) {
	var r ApprovalRequest
	var polID, bgReason, idem spanner.NullString
	var expAt, resAt spanner.NullTime
	if err := row.Columns(
		&r.TenantID, &r.RequestID, &polID, &r.PolicyVersion, &r.ObjectType, &r.ObjectID, &r.Action, &r.AmountMinor,
		&r.RequestedBy, &r.ActorType, &r.PayloadJSON, &r.PayloadHash, &r.Status, &r.CurrentStep, &r.RequiredApprovers,
		&r.BreakGlass, &bgReason, &expAt, &idem, &resAt, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return ApprovalRequest{}, fmt.Errorf("scan ApprovalRequest: %w", err)
	}
	r.PolicyID = nullStrVal(polID)
	r.BreakGlassReason = nullStrVal(bgReason)
	r.IdempotencyKey = nullStrVal(idem)
	if expAt.Valid {
		t := expAt.Time.UTC()
		r.ExpiresAt = &t
	}
	if resAt.Valid {
		t := resAt.Time.UTC()
		r.ResolvedAt = &t
	}
	r.CreatedAt = r.CreatedAt.UTC()
	r.UpdatedAt = r.UpdatedAt.UTC()
	return r, nil
}

func scanDelegationRow(row *spanner.Row) (ApprovalDelegation, error) {
	var d ApprovalDelegation
	var roleScope, reason spanner.NullString
	if err := row.Columns(&d.TenantID, &d.DelegationID, &d.DelegatorID, &d.DelegateID, &roleScope, &d.StartsAt, &d.EndsAt, &reason, &d.Enabled, &d.CreatedAt); err != nil {
		return ApprovalDelegation{}, fmt.Errorf("scan ApprovalDelegation: %w", err)
	}
	d.RoleScope = nullStrVal(roleScope)
	d.Reason = nullStrVal(reason)
	d.StartsAt = d.StartsAt.UTC()
	d.EndsAt = d.EndsAt.UTC()
	d.CreatedAt = d.CreatedAt.UTC()
	return d, nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}

func nullStrVal(ns spanner.NullString) string {
	if ns.Valid {
		return ns.StringVal
	}
	return ""
}
