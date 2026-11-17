package proposals

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SpannerRepository persists Proposals in Cloud Spanner and emits transactional
// outbox events alongside live Spanner reads for the 3 L1 agents.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed proposals repository.
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

func (r *SpannerRepository) FindByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (Proposal, bool, error) {
	if r == nil || r.client == nil {
		return Proposal{}, false, ErrRepoUnavailable
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return Proposal{}, false, nil
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT ` + proposalSelectCols + `
		      FROM Proposals@{FORCE_INDEX=UQ_Proposals_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
		      WHERE TenantId = @tid AND IdempotencyKey = @idem AND IdempotencyKey IS NOT NULL
		      LIMIT 1`,
		Params: map[string]any{
			"tid":  strings.TrimSpace(tenantID),
			"idem": strings.TrimSpace(idempotencyKey),
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return Proposal{}, false, nil
	}
	if err != nil {
		return Proposal{}, false, fmt.Errorf("lookup Proposals idempotency: %w", err)
	}
	p, err := scanProposalRow(row)
	if err != nil {
		return Proposal{}, false, err
	}
	return p, true, nil
}

func (r *SpannerRepository) Create(ctx context.Context, p Proposal) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if p.IdempotencyKey != "" {
			iter := txn.Query(txCtx, spanner.Statement{
				SQL: `SELECT ProposalId
				      FROM Proposals@{FORCE_INDEX=UQ_Proposals_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
				      WHERE TenantId = @tid AND IdempotencyKey = @idem AND IdempotencyKey IS NOT NULL
				      LIMIT 1`,
				Params: map[string]any{
					"tid":  p.TenantID,
					"idem": p.IdempotencyKey,
				},
			})
			_, err := iter.Next()
			iter.Stop()
			if err == nil {
				return nil
			}
			if err != iterator.Done {
				return fmt.Errorf("check Proposals idempotency in txn: %w", err)
			}
		}

		var expiresAt any
		if p.ExpiresAt != nil {
			expiresAt = p.ExpiresAt.UTC()
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("Proposals", map[string]any{
				"TenantId":             p.TenantID,
				"ProposalId":           p.ProposalID,
				"AgentId":              p.AgentID,
				"ProposalType":         p.ProposalType,
				"TargetObjectType":     p.TargetObjectType,
				"TargetObjectId":       p.TargetObjectID,
				"Title":                p.Title,
				"Summary":              nullIfEmpty(p.Summary),
				"Rationale":            p.Rationale,
				"PayloadJson":          p.PayloadJSON,
				"PayloadHash":          p.PayloadHash,
				"EvidenceRefsJson":     nonEmptyOr(p.EvidenceRefsJSON, "[]"),
				"ConfidenceBps":        p.ConfidenceBps,
				"EstimatedImpactMinor": p.EstimatedImpactMinor,
				"Currency":             nonEmptyOr(p.Currency, "UZS"),
				"RoiCategory":          nullIfEmpty(p.RoiCategory),
				"AutonomyLevel":        nonEmptyOr(p.AutonomyLevel, AutonomyLevelL1),
				"Status":               p.Status,
				"ApprovalRequestId":    nullIfEmpty(p.ApprovalRequestID),
				"CreatedBy":            p.CreatedBy,
				"ActorType":            p.ActorType,
				"DecidedBy":            nullIfEmpty(p.DecidedBy),
				"DecisionComment":      nullIfEmpty(p.DecisionComment),
				"ExecutedBy":           nullIfEmpty(p.ExecutedBy),
				"ExecutionResultJson":  nullIfEmpty(p.ExecutionResultJSON),
				"ExecutionError":       nullIfEmpty(p.ExecutionError),
				"RoiEventId":           nullIfEmpty(p.RoiEventID),
				"IdempotencyKey":       nullIfEmpty(p.IdempotencyKey),
				"ExpiresAt":            expiresAt,
				"DecidedAt":            nil,
				"ExecutedAt":           nil,
				"CreatedAt":            spanner.CommitTimestamp,
				"UpdatedAt":            spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("insert Proposals: %w", err)
		}

		ts := p.CreatedAt.UTC().Format(time.RFC3339Nano)
		outboxInner := outbox.NewSpannerTxnBuffer(txn)
		stampedBuf := &entityStampedOutboxBuf{inner: outboxInner, entityID: p.TenantID}
		ev := events.ProposalEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventProposalCreated,
				Version:   1,
				Timestamp: ts,
			},
			TenantID:            p.TenantID,
			ProposalID:          p.ProposalID,
			AgentID:             p.AgentID,
			ProposalType:        p.ProposalType,
			TargetObjectType:    p.TargetObjectType,
			TargetObjectID:      p.TargetObjectID,
			Status:              p.Status,
			PayloadHash:         p.PayloadHash,
			ApprovalRequestID:   p.ApprovalRequestID,
			ActorID:             p.CreatedBy,
			ActorType:           p.ActorType,
			ExpectedImpactMinor: p.EstimatedImpactMinor,
			Currency:            nonEmptyOr(p.Currency, "UZS"),
			ConfidenceBps:       p.ConfidenceBps,
			OccurredAt:          ts,
		}
		if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateProposal, p.ProposalID, events.TopicMain, ev); err != nil {
			return fmt.Errorf("emit proposal.created: %w", err)
		}
		if strings.EqualFold(p.ProposalType, TypeExpiryRebalance) {
			payloadMap := decodePayloadMap(p.PayloadJSON)
			rebEv := events.InventoryRebalanceProposedEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventInventoryRebalanceProposed,
					Version:   1,
					Timestamp: ts,
				},
				TenantID:          p.TenantID,
				ProposalID:        p.ProposalID,
				SourceWarehouseID: strField(payloadMap, "source_warehouse_id"),
				TargetWarehouseID: strField(payloadMap, "target_warehouse_id"),
				ProductID:         strField(payloadMap, "product_id"),
				LotID:             strField(payloadMap, "lot_id"),
				LayerID:           strField(payloadMap, "layer_id"),
				Qty:               int64Field(payloadMap, "qty_to_rebalance"),
				ValueAtRiskMinor:  p.EstimatedImpactMinor,
				Currency:          nonEmptyOr(p.Currency, "UZS"),
				ExpiryDate:        strField(payloadMap, "expiry_date"),
				ActionMode:        strField(payloadMap, "action"),
				ConfidenceBps:     p.ConfidenceBps,
				ApprovalRequestID: p.ApprovalRequestID,
				OccurredAt:        ts,
			}
			if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateProposal, p.ProposalID, events.TopicMain, rebEv); err != nil {
				return fmt.Errorf("emit inventory.rebalance.proposed: %w", err)
			}
		}
		return outboxInner.Flush(txCtx)
	})
}

func (r *SpannerRepository) Get(ctx context.Context, tenantID, proposalID string) (Proposal, bool, error) {
	if r == nil || r.client == nil {
		return Proposal{}, false, ErrRepoUnavailable
	}
	row, err := r.client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "Proposals",
		spanner.Key{strings.TrimSpace(tenantID), strings.TrimSpace(proposalID)},
		proposalColumns,
	)
	if err != nil {
		if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
			return Proposal{}, false, nil
		}
		return Proposal{}, false, fmt.Errorf("read Proposals: %w", err)
	}
	p, err := scanProposalRow(row)
	if err != nil {
		return Proposal{}, false, err
	}
	return p, true, nil
}

func (r *SpannerRepository) List(ctx context.Context, f ListFilter) ([]Proposal, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	params := map[string]any{"tid": strings.TrimSpace(f.TenantID)}
	sql := `SELECT ` + proposalSelectCols + ` FROM Proposals WHERE TenantId = @tid`
	if a := strings.TrimSpace(f.AgentID); a != "" {
		sql += ` AND UPPER(AgentId) = UPPER(@agentId)`
		params["agentId"] = a
	}
	if st := strings.TrimSpace(f.Status); st != "" {
		sql += ` AND UPPER(Status) = UPPER(@status)`
		params["status"] = st
	}
	if tt := strings.TrimSpace(f.TargetType); tt != "" {
		sql += ` AND LOWER(TargetObjectType) = LOWER(@targetType)`
		params["targetType"] = tt
	}
	if tid := strings.TrimSpace(f.TargetID); tid != "" {
		sql += ` AND TargetObjectId = @targetId`
		params["targetId"] = tid
	}
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	sql += ` ORDER BY CreatedAt DESC, ProposalId DESC LIMIT @lim`
	params["lim"] = int64(limit)

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []Proposal
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list Proposals: %w", err)
		}
		p, err := scanProposalRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *SpannerRepository) Update(ctx context.Context, p Proposal) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(txCtx, "Proposals", spanner.Key{p.TenantID, p.ProposalID}, []string{"Status", "CreatedAt"})
		if err != nil {
			if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
				return ErrProposalNotFound
			}
			return fmt.Errorf("read Proposals for update: %w", err)
		}
		var prevStatus string
		var createdAt time.Time
		if err := row.Columns(&prevStatus, &createdAt); err != nil {
			return err
		}

		// Atomic CAS guard when transitioning to EXECUTED in Spanner: prevent double or overlapping target execution.
		if p.Status == StatusExecuted {
			allowedL2 := prevStatus == StatusPending && strings.EqualFold(strings.TrimSpace(p.AutonomyLevel), AutonomyLevelL2)
			if prevStatus != StatusApproved && !allowedL2 {
				if prevStatus == StatusExecuted {
					return ErrAlreadyExecuted
				}
				return fmt.Errorf("%w: cannot execute from status %s", ErrInvalidTransition, prevStatus)
			}
			overlapIter := txn.Query(txCtx, spanner.Statement{
				SQL: `SELECT ProposalId
				      FROM Proposals
				      WHERE TenantId = @tid AND TargetObjectType = @tt AND TargetObjectId = @tobj
				        AND ProposalType = @ptype AND Status = 'EXECUTED'
				        AND ProposalId != @pid AND ExecutedAt >= @createdAt
				      LIMIT 1`,
				Params: map[string]any{
					"tid":       p.TenantID,
					"tt":        p.TargetObjectType,
					"tobj":      p.TargetObjectID,
					"ptype":     p.ProposalType,
					"pid":       p.ProposalID,
					"createdAt": createdAt.UTC(),
				},
			})
			_, oErr := overlapIter.Next()
			overlapIter.Stop()
			if oErr == nil {
				return ErrTargetExecutionConflict
			}
			if oErr != iterator.Done {
				return fmt.Errorf("check overlapping executed proposal: %w", oErr)
			}
		}

		var decidedAt any
		if p.DecidedAt != nil {
			decidedAt = p.DecidedAt.UTC()
		}
		var executedAt any
		if p.ExecutedAt != nil {
			executedAt = p.ExecutedAt.UTC()
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("Proposals", map[string]any{
				"TenantId":            p.TenantID,
				"ProposalId":          p.ProposalID,
				"PayloadJson":         p.PayloadJSON,
				"PayloadHash":         p.PayloadHash,
				"AutonomyLevel":       nonEmptyOr(p.AutonomyLevel, AutonomyLevelL1),
				"Status":              p.Status,
				"ApprovalRequestId":   nullIfEmpty(p.ApprovalRequestID),
				"DecidedBy":           nullIfEmpty(p.DecidedBy),
				"DecisionComment":     nullIfEmpty(p.DecisionComment),
				"ExecutedBy":          nullIfEmpty(p.ExecutedBy),
				"ExecutionResultJson": nullIfEmpty(p.ExecutionResultJSON),
				"ExecutionError":      nullIfEmpty(p.ExecutionError),
				"RoiEventId":          nullIfEmpty(p.RoiEventID),
				"DecidedAt":           decidedAt,
				"ExecutedAt":          executedAt,
				"UpdatedAt":           spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("update Proposals: %w", err)
		}

		if p.Status != prevStatus && (p.Status == StatusApproved || p.Status == StatusRejected || p.Status == StatusExecuted) {
			evType := events.EventProposalDecided
			actorID := p.DecidedBy
			decision := ""
			if p.Status == StatusApproved {
				decision = "APPROVE"
			} else if p.Status == StatusRejected {
				decision = "REJECT"
			} else if p.Status == StatusExecuted {
				evType = events.EventProposalExecuted
				actorID = p.ExecutedBy
			}
			ts := p.UpdatedAt.UTC().Format(time.RFC3339Nano)
			outboxInner := outbox.NewSpannerTxnBuffer(txn)
			stampedBuf := &entityStampedOutboxBuf{inner: outboxInner, entityID: p.TenantID}
			ev := events.ProposalEvent{
				BaseEvent: events.BaseEvent{
					Type:      evType,
					Version:   1,
					Timestamp: ts,
				},
				TenantID:            p.TenantID,
				ProposalID:          p.ProposalID,
				AgentID:             p.AgentID,
				ProposalType:        p.ProposalType,
				TargetObjectType:    p.TargetObjectType,
				TargetObjectID:      p.TargetObjectID,
				Status:              p.Status,
				PayloadHash:         p.PayloadHash,
				ApprovalRequestID:   p.ApprovalRequestID,
				ActorID:             actorID,
				ActorType:           "human",
				RoiEventID:          p.RoiEventID,
				ExpectedImpactMinor: p.EstimatedImpactMinor,
				Currency:            nonEmptyOr(p.Currency, "UZS"),
				ConfidenceBps:       p.ConfidenceBps,
				Decision:            decision,
				Reason:              p.DecisionComment,
				OccurredAt:          ts,
			}
			if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateProposal, p.ProposalID, events.TopicMain, ev); err != nil {
				return fmt.Errorf("emit %s: %w", evType, err)
			}
			if p.Status == StatusExecuted && strings.EqualFold(strings.TrimSpace(p.AutonomyLevel), AutonomyLevelL2) {
				var soakDays, soakExecCount, rollbackBps int64
				if p.L2SoakContext != nil {
					soakDays = p.L2SoakContext.SoakDays
					soakExecCount = p.L2SoakContext.ExecutedCount
					rollbackBps = p.L2SoakContext.RollbackRateBps
				}
				l2Ev := events.ProposalL2AutoExecutedEvent{
					BaseEvent: events.BaseEvent{
						Type:      events.EventProposalL2AutoExecuted,
						Version:   1,
						Timestamp: ts,
					},
					TenantID:             p.TenantID,
					ProposalID:           p.ProposalID,
					AgentID:              p.AgentID,
					ProposalType:         p.ProposalType,
					TargetObjectType:     p.TargetObjectType,
					TargetObjectID:       p.TargetObjectID,
					AutonomyLevel:        AutonomyLevelL2,
					Status:               p.Status,
					PayloadHash:          p.PayloadHash,
					EstimatedImpactMinor: p.EstimatedImpactMinor,
					Currency:             nonEmptyOr(p.Currency, "UZS"),
					ConfidenceBps:        p.ConfidenceBps,
					SoakDays:             soakDays,
					SoakExecutedCount:    soakExecCount,
					RollbackRateBps:      rollbackBps,
					RoiEventID:           p.RoiEventID,
					ActorID:              actorID,
					OccurredAt:           ts,
				}
				if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateProposal, p.ProposalID, events.TopicMain, l2Ev); err != nil {
					return fmt.Errorf("emit %s: %w", events.EventProposalL2AutoExecuted, err)
				}
			}
			return outboxInner.Flush(txCtx)
		}
		return nil
	})
}

func (r *SpannerRepository) HasOverlappingExecuted(ctx context.Context, tenantID, targetType, targetID, proposalType, excludeProposalID string, createdBeforeOrAt time.Time) (bool, error) {
	if r == nil || r.client == nil {
		return false, ErrRepoUnavailable
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT ProposalId
		      FROM Proposals
		      WHERE TenantId = @tid AND TargetObjectType = @tt AND TargetObjectId = @tobj
		        AND ProposalType = @ptype AND Status = 'EXECUTED'
		        AND ProposalId != @pid AND ExecutedAt >= @createdAt
		      LIMIT 1`,
		Params: map[string]any{
			"tid":       strings.TrimSpace(tenantID),
			"tt":        strings.TrimSpace(targetType),
			"tobj":      strings.TrimSpace(targetID),
			"ptype":     strings.TrimSpace(proposalType),
			"pid":       strings.TrimSpace(excludeProposalID),
			"createdAt": createdBeforeOrAt.UTC(),
		},
	})
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *SpannerRepository) HasActiveForTarget(ctx context.Context, tenantID, targetType, targetID, proposalType string) (bool, error) {
	if r == nil || r.client == nil {
		return false, ErrRepoUnavailable
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT ProposalId
		      FROM Proposals
		      WHERE TenantId = @tid AND TargetObjectType = @tt AND TargetObjectId = @tobj
		        AND ProposalType = @ptype AND Status IN ('PENDING', 'APPROVED')
		      LIMIT 1`,
		Params: map[string]any{
			"tid":   strings.TrimSpace(tenantID),
			"tt":    strings.TrimSpace(targetType),
			"tobj":  strings.TrimSpace(targetID),
			"ptype": strings.TrimSpace(proposalType),
		},
	})
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *SpannerRepository) GetL2SoakStats(ctx context.Context, tenantID string, now time.Time) (L2SoakMetrics, error) {
	if r == nil || r.client == nil {
		return L2SoakMetrics{}, ErrRepoUnavailable
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tid := strings.TrimSpace(tenantID)
	var out L2SoakMetrics
	var earliest *time.Time

	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(roCtx context.Context, ro *spanner.ReadOnlyTransaction) error {
		iter := ro.Query(roCtx, spanner.Statement{
			SQL: `SELECT Status, ExecutedAt, CreatedAt, COALESCE(ExecutionError, '')
			      FROM Proposals
			      WHERE TenantId = @tid AND Status IN ('EXECUTED', 'FAILED', 'ROLLED_BACK', 'REVERSED')`,
			Params: map[string]any{"tid": tid},
		})
		defer iter.Stop()
		for {
			row, qErr := iter.Next()
			if qErr == iterator.Done {
				break
			}
			if qErr != nil {
				return fmt.Errorf("query L2 soak metrics: %w", qErr)
			}
			var st, execErr string
			var execAt spanner.NullTime
			var createdAt time.Time
			if cErr := row.Columns(&st, &execAt, &createdAt, &execErr); cErr != nil {
				return cErr
			}
			normSt := strings.ToUpper(strings.TrimSpace(st))
			switch normSt {
			case StatusExecuted:
				out.ExecutedCount++
				ref := createdAt.UTC()
				if execAt.Valid && !execAt.Time.IsZero() {
					ref = execAt.Time.UTC()
				}
				if !ref.IsZero() && (earliest == nil || ref.Before(*earliest)) {
					t := ref
					earliest = &t
				}
			case "ROLLED_BACK", "REVERSED":
				out.RollbackCount++
			case StatusFailed:
				if (execAt.Valid && !execAt.Time.IsZero()) || strings.TrimSpace(execErr) != "" {
					out.RollbackCount++
				}
			}
		}
		return nil
	})
	if err != nil {
		return L2SoakMetrics{}, err
	}
	out.EarliestExecuteAt = earliest
	if earliest != nil && now.After(*earliest) {
		out.SoakDays = int64(now.Sub(*earliest) / (24 * time.Hour))
	}
	total := out.ExecutedCount + out.RollbackCount
	if total > 0 {
		out.RollbackRateBps = (out.RollbackCount * 10000) / total
	}
	return out, nil
}

// --- Live Spanner Readers & Fresh Precondition Verifiers ---

func (r *SpannerRepository) ListReorderCandidates(ctx context.Context, tenantID string, limit int) ([]ReorderCandidate, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	tid := strings.TrimSpace(tenantID)
	var out []ReorderCandidate
	seenKey := make(map[string]struct{})

	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(roCtx context.Context, ro *spanner.ReadOnlyTransaction) error {
		// 1. Pending ReplenishmentInsights enriched with Products price.
		insIter := ro.Query(roCtx, spanner.Statement{
			SQL: `SELECT ri.InsightId, ri.WarehouseId, ri.ProductId, ri.CurrentStock,
			             ri.SuggestedQuantity, ri.UrgencyLevel, ri.ReasonCode,
			             COALESCE(ri.TargetFactoryId, ''), COALESCE(p.Name, ''),
			             COALESCE(p.PriceMinor, 0), COALESCE(p.Currency, 'UZS')
			      FROM ReplenishmentInsights ri
			      LEFT JOIN Products p ON ri.ProductId = p.ProductId
			      WHERE ri.SupplierId = @tid AND ri.Status = 'PENDING'
			      ORDER BY ri.CreatedAt DESC
			      LIMIT @lim`,
			Params: map[string]any{"tid": tid, "lim": int64(limit)},
		})
		for {
			row, iErr := insIter.Next()
			if iErr == iterator.Done {
				break
			}
			if iErr != nil {
				insIter.Stop()
				return fmt.Errorf("query ReplenishmentInsights: %w", iErr)
			}
			var c ReorderCandidate
			if err := row.Columns(
				&c.InsightID, &c.WarehouseID, &c.ProductID, &c.CurrentStock,
				&c.SuggestedQty, &c.UrgencyLevel, &c.ReasonCode,
				&c.TargetFactoryID, &c.ProductName, &c.UnitPriceMinor, &c.Currency,
			); err != nil {
				insIter.Stop()
				return err
			}
			if c.SuggestedQty <= 0 {
				c.SuggestedQty = 1
			}
			c.ConfidenceBps = 8400
			seenKey[c.WarehouseID+":"+c.ProductID] = struct{}{}
			out = append(out, c)
		}
		insIter.Stop()

		// 2. Low-cover SupplierInventoryV2 SKUs without pending insights.
		if len(out) < limit {
			invIter := ro.Query(roCtx, spanner.Statement{
				SQL: `SELECT si.WarehouseId, si.ProductId, si.QuantityOnHand, si.QuantityReserved,
				             si.ReorderThreshold, COALESCE(p.Name, ''), COALESCE(p.PriceMinor, 0),
				             COALESCE(p.Currency, 'UZS'), COALESCE(w.PrimaryFactoryId, '')
				      FROM SupplierInventoryV2 si
				      LEFT JOIN Products p ON si.ProductId = p.ProductId
				      LEFT JOIN Warehouses w ON si.WarehouseId = w.WarehouseId
				      WHERE si.SupplierId = @tid AND si.ReorderThreshold > 0
				        AND (si.QuantityOnHand - si.QuantityReserved) <= si.ReorderThreshold
				      LIMIT @lim`,
				Params: map[string]any{"tid": tid, "lim": int64(limit - len(out))},
			})
			for {
				row, sErr := invIter.Next()
				if sErr == iterator.Done {
					break
				}
				if sErr != nil {
					invIter.Stop()
					return fmt.Errorf("query SupplierInventoryV2 low cover: %w", sErr)
				}
				var whID, prodID, pName, cur, facID string
				var qoh, qres, thresh, price int64
				if err := row.Columns(&whID, &prodID, &qoh, &qres, &thresh, &pName, &price, &cur, &facID); err != nil {
					invIter.Stop()
					return err
				}
				if _, dup := seenKey[whID+":"+prodID]; dup {
					continue
				}
				avail := qoh - qres
				if avail < 0 {
					avail = 0
				}
				sugQty := (thresh * 2) - avail
				if sugQty <= 0 {
					sugQty = thresh
				}
				if sugQty <= 0 {
					sugQty = 1
				}
				seenKey[whID+":"+prodID] = struct{}{}
				out = append(out, ReorderCandidate{
					WarehouseID:      whID,
					ProductID:        prodID,
					ProductName:      pName,
					CurrentStock:     avail,
					ReorderThreshold: thresh,
					SuggestedQty:     sugQty,
					UnitPriceMinor:   price,
					Currency:         cur,
					TargetFactoryID:  facID,
					ConfidenceBps:    7900,
					UrgencyLevel:     "HIGH",
					ReasonCode:       "LOW_COVER",
				})
			}
			invIter.Stop()
		}

		// 3. Enrich with latest ForecastValues P50Milli in a single batched query.
		if len(out) > 0 {
			locSet := make(map[string]struct{}, len(out))
			pidSet := make(map[string]struct{}, len(out))
			locs := make([]string, 0, len(out))
			pids := make([]string, 0, len(out))
			for _, cand := range out {
				if _, ok := locSet[cand.WarehouseID]; !ok && cand.WarehouseID != "" {
					locSet[cand.WarehouseID] = struct{}{}
					locs = append(locs, cand.WarehouseID)
				}
				if _, ok := pidSet[cand.ProductID]; !ok && cand.ProductID != "" {
					pidSet[cand.ProductID] = struct{}{}
					pids = append(pids, cand.ProductID)
				}
			}
			if len(locs) > 0 && len(pids) > 0 {
				type fcMatch struct {
					runID string
					p50   int64
					conf  int64
				}
				fcByKey := make(map[string]fcMatch, len(out))
				fvIter := ro.Query(roCtx, spanner.Statement{
					SQL: `SELECT LocationId, COALESCE(ProductId, ''), COALESCE(Sku, ''), RunId, P50Milli, ConfidenceBps
					      FROM ForecastValues
					      WHERE EntityId = @tid AND IsWinner = TRUE
					        AND LocationId IN UNNEST(@locs)
					        AND (ProductId IN UNNEST(@pids) OR Sku IN UNNEST(@pids))
					      ORDER BY ForecastDate DESC`,
					Params: map[string]any{"tid": tid, "locs": locs, "pids": pids},
				})
				for {
					row, fErr := fvIter.Next()
					if fErr == iterator.Done {
						break
					}
					if fErr != nil {
						break
					}
					var locID, fProdID, fSku, runID string
					var p50, conf int64
					if err := row.Columns(&locID, &fProdID, &fSku, &runID, &p50, &conf); err != nil {
						continue
					}
					m := fcMatch{runID: runID, p50: p50, conf: conf}
					if fProdID != "" {
						k := locID + "|" + fProdID
						if _, exists := fcByKey[k]; !exists {
							fcByKey[k] = m
						}
					}
					if fSku != "" {
						k := locID + "|" + fSku
						if _, exists := fcByKey[k]; !exists {
							fcByKey[k] = m
						}
					}
				}
				fvIter.Stop()
				for i := range out {
					if m, ok := fcByKey[out[i].WarehouseID+"|"+out[i].ProductID]; ok {
						out[i].ForecastRunID = m.runID
						out[i].ForecastP50Milli = m.p50
						if m.conf > 0 {
							out[i].ConfidenceBps = m.conf
						}
					}
				}
			}
		}

		// 4. Also check linked open ReorderSuggestions if still room.
		if len(out) < limit {
			rsIter := ro.Query(roCtx, spanner.Statement{
				SQL: `SELECT rs.RetailerId, rs.Sku, rs.CurrentStock, rs.SuggestedQty
				      FROM ReorderSuggestions rs
				      INNER JOIN SupplierRetailerLinks srl ON rs.RetailerId = srl.RetailerId
				      WHERE srl.SupplierId = @tid AND rs.Status = 'OPEN'
				      LIMIT @lim`,
				Params: map[string]any{"tid": tid, "lim": int64(limit - len(out))},
			})
			for {
				row, rErr := rsIter.Next()
				if rErr == iterator.Done {
					break
				}
				if rErr != nil {
					rsIter.Stop()
					return fmt.Errorf("query ReorderSuggestions: %w", rErr)
				}
				var retID, sku string
				var curStock, sugQty int64
				if err := row.Columns(&retID, &sku, &curStock, &sugQty); err != nil {
					rsIter.Stop()
					return err
				}
				if sugQty <= 0 {
					continue
				}
				out = append(out, ReorderCandidate{
					SuggestionKey: retID + ":" + sku,
					RetailerID:    retID,
					ProductID:     sku,
					CurrentStock:  curStock,
					SuggestedQty:  sugQty,
					Currency:      "UZS",
					ConfidenceBps: 7800,
					UrgencyLevel:  "MEDIUM",
					ReasonCode:    "RETAILER_REORDER_SUGGESTION",
				})
			}
			rsIter.Stop()
		}
		return nil
	})
	return out, err
}

func (r *SpannerRepository) ListOverdueInvoices(ctx context.Context, tenantID string, now time.Time, limit int) ([]OverdueInvoiceCandidate, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	tid := strings.TrimSpace(tenantID)
	var out []OverdueInvoiceCandidate

	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(roCtx context.Context, ro *spanner.ReadOnlyTransaction) error {
		iter := ro.Query(roCtx, spanner.Statement{
			SQL: `SELECT ai.InvoiceId, ai.OrderId, ai.RetailerId, COALESCE(ret.Name, ''),
			             COALESCE(ret.Phone, ''), COALESCE(ret.CountryCode, 'UZ'),
			             ai.BalanceMinor, ai.Currency, ai.DueAt, ai.DunningStep
			      FROM ArInvoices ai
			      LEFT JOIN Retailers ret ON ai.RetailerId = ret.RetailerId
			      WHERE ai.SupplierId = @tid AND ai.Status IN ('OPEN', 'PARTIAL')
			        AND ai.BalanceMinor > 0 AND ai.DueAt < @now
			      ORDER BY ai.DueAt ASC
			      LIMIT @lim`,
			Params: map[string]any{"tid": tid, "now": now.UTC(), "lim": int64(limit)},
		})
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("query overdue ArInvoices: %w", err)
			}
			var c OverdueInvoiceCandidate
			var country string
			if err := row.Columns(
				&c.InvoiceID, &c.OrderID, &c.RetailerID, &c.RetailerName,
				&c.Phone, &country, &c.BalanceMinor, &c.Currency, &c.DueAt, &c.DunningStep,
			); err != nil {
				return err
			}
			days := int64(now.UTC().Sub(c.DueAt.UTC()) / (24 * time.Hour))
			if days < 1 {
				days = 1
			}
			c.DaysOverdue = days
			c.Locale = "uz"
			if strings.EqualFold(country, "RU") || strings.EqualFold(country, "KZ") {
				c.Locale = "ru"
			}
			out = append(out, c)
		}
		return nil
	})
	return out, err
}

func (r *SpannerRepository) ListExpiryRiskCandidates(ctx context.Context, tenantID string, now time.Time, windowDays int, limit int) ([]ExpiryRiskCandidate, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	if windowDays <= 0 {
		windowDays = 30
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	tid := strings.TrimSpace(tenantID)
	today := civil.DateOf(now.UTC())
	cutoff := civil.DateOf(now.UTC().Add(time.Duration(windowDays) * 24 * time.Hour))
	var out []ExpiryRiskCandidate

	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(roCtx context.Context, ro *spanner.ReadOnlyTransaction) error {
		// Load active warehouses for rebalancing destination suggestions.
		var warehouses []string
		whIter := ro.Query(roCtx, spanner.Statement{
			SQL:    `SELECT WarehouseId FROM Warehouses WHERE SupplierId = @tid AND IsActive = TRUE ORDER BY WarehouseId ASC`,
			Params: map[string]any{"tid": tid},
		})
		for {
			row, wErr := whIter.Next()
			if wErr == iterator.Done {
				break
			}
			if wErr != nil {
				whIter.Stop()
				return fmt.Errorf("query Warehouses: %w", wErr)
			}
			var wid string
			if err := row.Columns(&wid); err == nil {
				warehouses = append(warehouses, wid)
			}
		}
		whIter.Stop()

		pickTargetWH := func(srcWH string) string {
			for _, w := range warehouses {
				if w != srcWH {
					return w
				}
			}
			return "MARKDOWN_CHANNEL"
		}

		seenLots := make(map[string]struct{})
		// 1. Near-expiry InventoryCostLayers (contains actual UnitCostMinor!).
		clIter := ro.Query(roCtx, spanner.Statement{
			SQL: `SELECT cl.WarehouseId, cl.ProductId, cl.LayerId, COALESCE(cl.LotId, ''),
			             cl.QtyRemaining, cl.UnitCostMinor, cl.Currency, cl.ExpiryDate,
			             COALESCE(p.Name, '')
			      FROM InventoryCostLayers cl
			      LEFT JOIN Products p ON cl.ProductId = p.ProductId
			      WHERE cl.EntityId = @tid AND cl.QtyRemaining > 0
			        AND cl.ExpiryDate IS NOT NULL AND cl.ExpiryDate <= @cutoff
			      ORDER BY cl.ExpiryDate ASC
			      LIMIT @lim`,
			Params: map[string]any{"tid": tid, "cutoff": cutoff, "lim": int64(limit)},
		})
		for {
			row, cErr := clIter.Next()
			if cErr == iterator.Done {
				break
			}
			if cErr != nil {
				clIter.Stop()
				return fmt.Errorf("query near-expiry InventoryCostLayers: %w", cErr)
			}
			var c ExpiryRiskCandidate
			var exp civil.Date
			if err := row.Columns(
				&c.SourceWarehouseID, &c.ProductID, &c.LayerID, &c.LotID,
				&c.QtyAvailable, &c.UnitCostMinor, &c.Currency, &exp, &c.ProductName,
			); err != nil {
				clIter.Stop()
				return err
			}
			c.ExpiryDate = exp.String()
			c.DaysToExpiry = int64(exp.DaysSince(today))
			if c.DaysToExpiry < 0 {
				c.DaysToExpiry = 0
			}
			c.ValueAtRiskMinor = c.QtyAvailable * c.UnitCostMinor
			c.TargetWarehouseID = pickTargetWH(c.SourceWarehouseID)
			if c.LotID != "" {
				seenLots[c.LotID] = struct{}{}
			}
			out = append(out, c)
		}
		clIter.Stop()

		// 2. Near-expiry StockLots not already represented via InventoryCostLayers.
		if len(out) < limit {
			slIter := ro.Query(roCtx, spanner.Statement{
				SQL: `SELECT sl.LotId, sl.WarehouseId, sl.ProductId, sl.QuantityOnHand, sl.QuantityReserved,
				             sl.ExpiryDate, COALESCE(p.Name, ''), COALESCE(p.PriceMinor, 0), COALESCE(p.Currency, 'UZS')
				      FROM StockLots sl
				      LEFT JOIN Products p ON sl.ProductId = p.ProductId
				      WHERE sl.SupplierId = @tid AND sl.Status = 'AVAILABLE'
				        AND sl.QuantityOnHand > sl.QuantityReserved
				        AND sl.ExpiryDate IS NOT NULL AND sl.ExpiryDate <= @cutoff
				      ORDER BY sl.ExpiryDate ASC
				      LIMIT @lim`,
				Params: map[string]any{"tid": tid, "cutoff": cutoff, "lim": int64(limit - len(out))},
			})
			for {
				row, sErr := slIter.Next()
				if sErr == iterator.Done {
					break
				}
				if sErr != nil {
					slIter.Stop()
					return fmt.Errorf("query near-expiry StockLots: %w", sErr)
				}
				var lotID, whID, prodID, pName, cur string
				var qoh, qres, price int64
				var exp civil.Date
				if err := row.Columns(&lotID, &whID, &prodID, &qoh, &qres, &exp, &pName, &price, &cur); err != nil {
					slIter.Stop()
					return err
				}
				if _, dup := seenLots[lotID]; dup {
					continue
				}
				avail := qoh - qres
				if avail <= 0 {
					continue
				}
				days := int64(exp.DaysSince(today))
				if days < 0 {
					days = 0
				}
				out = append(out, ExpiryRiskCandidate{
					LotID:             lotID,
					ProductID:         prodID,
					ProductName:       pName,
					SourceWarehouseID: whID,
					TargetWarehouseID: pickTargetWH(whID),
					QtyAvailable:      avail,
					UnitCostMinor:     price,
					ValueAtRiskMinor:  avail * price,
					Currency:          cur,
					ExpiryDate:        exp.String(),
					DaysToExpiry:      days,
				})
			}
			slIter.Stop()
		}
		return nil
	})
	return out, err
}

func (r *SpannerRepository) VerifyReorderPrecondition(ctx context.Context, tenantID string, p Proposal) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	payload := decodePayloadMap(p.PayloadJSON)
	insightID := strField(payload, "insight_id")
	if insightID != "" {
		row, err := r.client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ReplenishmentInsights", spanner.Key{insightID}, []string{"SupplierId", "Status"})
		if err != nil {
			if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
				return fmt.Errorf("%w: insight %s not found", ErrPreconditionFailed, insightID)
			}
			return err
		}
		var sid, st string
		if err := row.Columns(&sid, &st); err != nil {
			return err
		}
		if sid != tenantID || !strings.EqualFold(st, "PENDING") {
			return fmt.Errorf("%w: insight %s status is %s", ErrPreconditionFailed, insightID, st)
		}
		return nil
	}
	sugKey := strField(payload, "suggestion_key")
	if sugKey != "" {
		parts := strings.SplitN(sugKey, ":", 2)
		if len(parts) == 2 {
			row, err := r.client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ReorderSuggestions", spanner.Key{parts[0], parts[1]}, []string{"Status"})
			if err != nil {
				return fmt.Errorf("%w: suggestion %s not found", ErrPreconditionFailed, sugKey)
			}
			var st string
			if err := row.Columns(&st); err != nil {
				return err
			}
			if !strings.EqualFold(st, "OPEN") {
				return fmt.Errorf("%w: suggestion %s status is %s", ErrPreconditionFailed, sugKey, st)
			}
		}
	}
	return nil
}

func (r *SpannerRepository) VerifyCollectionsPrecondition(ctx context.Context, tenantID string, p Proposal) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	payload := decodePayloadMap(p.PayloadJSON)
	invID := strField(payload, "invoice_id")
	if invID == "" {
		invID = p.TargetObjectID
	}
	row, err := r.client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ArInvoices", spanner.Key{invID}, []string{"SupplierId", "Status", "BalanceMinor"})
	if err != nil {
		if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
			return fmt.Errorf("%w: invoice %s not found", ErrPreconditionFailed, invID)
		}
		return err
	}
	var sid, st string
	var bal int64
	if err := row.Columns(&sid, &st, &bal); err != nil {
		return err
	}
	if sid != tenantID {
		return fmt.Errorf("%w: invoice %s tenant mismatch", ErrPreconditionFailed, invID)
	}
	if (st != "OPEN" && st != "PARTIAL") || bal <= 0 {
		return fmt.Errorf("%w: invoice %s already settled (status=%s balance=%d)", ErrPreconditionFailed, invID, st, bal)
	}
	return nil
}

func (r *SpannerRepository) VerifyExpiryRebalancePrecondition(ctx context.Context, tenantID string, p Proposal) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	payload := decodePayloadMap(p.PayloadJSON)
	qty := int64Field(payload, "qty_to_rebalance")
	if lotID := strField(payload, "lot_id"); lotID != "" {
		row, err := r.client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "StockLots", spanner.Key{lotID}, []string{"SupplierId", "Status", "QuantityOnHand", "QuantityReserved"})
		if err != nil {
			if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
				return fmt.Errorf("%w: stock lot %s not found", ErrPreconditionFailed, lotID)
			}
			return err
		}
		var sid, st string
		var qoh, qres int64
		if err := row.Columns(&sid, &st, &qoh, &qres); err != nil {
			return err
		}
		if sid != tenantID || st != "AVAILABLE" || (qoh-qres) < qty || (qoh-qres) <= 0 {
			return fmt.Errorf("%w: stock lot %s insufficient or unavailable (status=%s avail=%d need=%d)", ErrPreconditionFailed, lotID, st, qoh-qres, qty)
		}
		return nil
	}
	if layerID := strField(payload, "layer_id"); layerID != "" {
		whID := strField(payload, "source_warehouse_id")
		prodID := strField(payload, "product_id")
		row, err := r.client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "InventoryCostLayers", spanner.Key{tenantID, whID, prodID, layerID}, []string{"QtyRemaining"})
		if err != nil {
			return fmt.Errorf("%w: cost layer %s not found", ErrPreconditionFailed, layerID)
		}
		var rem int64
		if err := row.Columns(&rem); err != nil {
			return err
		}
		if rem < qty || rem <= 0 {
			return fmt.Errorf("%w: cost layer %s insufficient qty (%d < %d)", ErrPreconditionFailed, layerID, rem, qty)
		}
	}
	return nil
}

var proposalColumns = []string{
	"TenantId", "ProposalId", "AgentId", "ProposalType", "TargetObjectType", "TargetObjectId",
	"Title", "Summary", "Rationale", "PayloadJson", "PayloadHash", "EvidenceRefsJson",
	"ConfidenceBps", "EstimatedImpactMinor", "Currency", "RoiCategory", "AutonomyLevel", "Status", "ApprovalRequestId",
	"CreatedBy", "ActorType", "DecidedBy", "DecisionComment", "ExecutedBy",
	"ExecutionResultJson", "ExecutionError", "RoiEventId", "IdempotencyKey",
	"ExpiresAt", "DecidedAt", "ExecutedAt", "CreatedAt", "UpdatedAt",
}

var proposalSelectCols = strings.Join(proposalColumns, ", ")

func scanProposalRow(row *spanner.Row) (Proposal, error) {
	var (
		p                                                                                  Proposal
		summary, roiCat, autonomyLvl, appReqID, decidedBy, decComment, execBy, execResJSON spanner.NullString
		execErr, roiEventID, idemKey                                                       spanner.NullString
		expiresAt, decidedAt, executedAt                                                   spanner.NullTime
	)
	if err := row.Columns(
		&p.TenantID, &p.ProposalID, &p.AgentID, &p.ProposalType, &p.TargetObjectType, &p.TargetObjectID,
		&p.Title, &summary, &p.Rationale, &p.PayloadJSON, &p.PayloadHash, &p.EvidenceRefsJSON,
		&p.ConfidenceBps, &p.EstimatedImpactMinor, &p.Currency, &roiCat, &autonomyLvl, &p.Status, &appReqID,
		&p.CreatedBy, &p.ActorType, &decidedBy, &decComment, &execBy,
		&execResJSON, &execErr, &roiEventID, &idemKey,
		&expiresAt, &decidedAt, &executedAt, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return Proposal{}, fmt.Errorf("scan Proposals row: %w", err)
	}
	p.Summary = summary.StringVal
	p.RoiCategory = roiCat.StringVal
	p.AutonomyLevel = nonEmptyOr(autonomyLvl.StringVal, AutonomyLevelL1)
	p.ApprovalRequestID = appReqID.StringVal
	p.DecidedBy = decidedBy.StringVal
	p.DecisionComment = decComment.StringVal
	p.ExecutedBy = execBy.StringVal
	p.ExecutionResultJSON = execResJSON.StringVal
	p.ExecutionError = execErr.StringVal
	p.RoiEventID = roiEventID.StringVal
	p.IdempotencyKey = idemKey.StringVal
	if expiresAt.Valid {
		t := expiresAt.Time.UTC()
		p.ExpiresAt = &t
	}
	if decidedAt.Valid {
		t := decidedAt.Time.UTC()
		p.DecidedAt = &t
	}
	if executedAt.Valid {
		t := executedAt.Time.UTC()
		p.ExecutedAt = &t
	}
	p.CreatedAt = p.CreatedAt.UTC()
	p.UpdatedAt = p.UpdatedAt.UTC()
	p.Payload = decodePayloadMap(p.PayloadJSON)
	if p.EvidenceRefsJSON != "" {
		_ = json.Unmarshal([]byte(p.EvidenceRefsJSON), &p.EvidenceRefs)
	}
	if p.EvidenceRefs == nil {
		p.EvidenceRefs = []EvidenceRef{}
	}
	if p.ExecutionResultJSON != "" {
		p.ExecutionResult = decodePayloadMap(p.ExecutionResultJSON)
	}
	return p, nil
}

func nullIfEmpty(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func nonEmptyOr(s, def string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	return s
}

func strField(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	v, _ := m[k].(string)
	return strings.TrimSpace(v)
}

func int64Field(m map[string]any, k string) int64 {
	if m == nil {
		return 0
	}
	switch tv := m[k].(type) {
	case int64:
		return tv
	case int:
		return int64(tv)
	case json.Number:
		if n, err := tv.Int64(); err == nil {
			return n
		}
	}
	return 0
}
