package planning

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
)

// AgentExecutionResult is the synchronous outcome of a governed agent hook.
type AgentExecutionResult struct {
	Status         string `json:"status"`
	Action         string `json:"action"`
	IdempotencyKey string `json:"idempotency_key"`
	ResultID       string `json:"result_id,omitempty"`
}

// Executor runs allowlisted agent mutations.
type Executor struct {
	Spanner *spanner.Client
	Now     func() time.Time
}

func NewExecutor(client *spanner.Client) *Executor {
	return &Executor{
		Spanner: client,
		Now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// Execute validates and runs one governed agent action.
func (x *Executor) Execute(ctx context.Context, inv AgentInvocation) (AgentExecutionResult, error) {
	if err := ValidateAgentInvocation(inv); err != nil {
		return AgentExecutionResult{}, err
	}
	if x == nil || x.Spanner == nil {
		return AgentExecutionResult{}, ErrAgentInvocationInvalid
	}
	action := GovernedAgentAction(strings.TrimSpace(string(inv.Action)))
	base := AgentExecutionResult{
		Status:         "completed",
		Action:         string(action),
		IdempotencyKey: inv.IdempotencyKey,
	}
	switch action {
	case AgentApproveInsight:
		transferID, err := replenishment.FulfillApprovedInsight(ctx, x.Spanner, strings.TrimSpace(inv.TargetID), x.Now)
		if err != nil {
			return AgentExecutionResult{}, err
		}
		base.ResultID = transferID
		return base, nil
	case AgentOpenSupplyRequest:
		requestID, err := x.openSupplyRequestFromInsight(ctx, inv)
		if err != nil {
			return AgentExecutionResult{}, err
		}
		base.ResultID = requestID
		return base, nil
	case AgentBroadcastTemplate:
		eventID, err := x.broadcastTemplate(ctx, inv)
		if err != nil {
			return AgentExecutionResult{}, err
		}
		base.ResultID = eventID
		return base, nil
	default:
		return AgentExecutionResult{}, ErrAgentActionDenied
	}
}

func (x *Executor) openSupplyRequestFromInsight(ctx context.Context, inv AgentInvocation) (string, error) {
	row, err := loadInsightForAgent(ctx, x.Spanner, inv.SupplierID, strings.TrimSpace(inv.TargetID))
	if err != nil {
		return "", err
	}
	factoryID := strings.TrimSpace(row.TargetFactoryID)
	if factoryID == "" {
		factoryID = strings.TrimSpace(row.PrimaryFactoryID)
	}
	if factoryID == "" {
		return "", ErrAgentInvocationInvalid
	}
	requestID := uuid.NewString()
	itemID := uuid.NewString()
	now := x.Now().UTC()
	coverageStart := now.Format("2006-01-02")
	qty := row.SuggestedQuantity
	if qty <= 0 {
		qty = 1
	}
	notes := strings.TrimSpace(inv.Note)
	if notes == "" {
		notes = "governed_agent:open_supply_request"
	}
	_, err = x.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("WarehouseSupplyRequests", map[string]any{
				"RequestId":                requestID,
				"SupplierId":               inv.SupplierID,
				"WarehouseId":              row.WarehouseID,
				"State":                    "SUBMITTED",
				"FactoryId":                factoryID,
				"TransferMode":             nullableAgentString(row.TransferMode),
				"RequestedBy":              "governed_agent",
				"CoverageStartDate":        coverageStart,
				"CoverageDays":             int64(7),
				"ProjectedUnits":           qty,
				"CommittedUnits":           int64(0),
				"PendingConfirmationUnits": qty,
				"Priority":                 "NORMAL",
				"Notes":                    notes,
				"TotalVolumeVU":            float64(qty),
				"CreatedAt":                spanner.CommitTimestamp,
				"UpdatedAt":                spanner.CommitTimestamp,
			}),
			spanner.InsertOrUpdateMap("WarehouseSupplyRequestItems", map[string]any{
				"RequestId":           requestID,
				"ItemId":              itemID,
				"ProductId":           row.ProductID,
				"RequestedQuantity":   qty,
				"RecommendedQuantity": qty,
				"CreatedAt":           spanner.CommitTimestamp,
			}),
		}
		buf := &planningTxnBuffer{}
		payload := events.WarehouseEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventWarehouseSupplyRequestOpened,
				Timestamp: now.Format(time.RFC3339Nano),
			},
			RequestID:   requestID,
			SupplierID:  inv.SupplierID,
			WarehouseID: row.WarehouseID,
			FactoryID:   factoryID,
			Status:      "SUBMITTED",
			Projected:   qty,
			RequestedBy: "governed_agent",
		}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, requestID, events.TopicMain, payload); emitErr != nil {
			return emitErr
		}
		mutations = append(mutations, planningOutboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	return requestID, err
}

func (x *Executor) broadcastTemplate(ctx context.Context, inv AgentInvocation) (string, error) {
	eventID := uuid.NewString()
	templateID := strings.TrimSpace(inv.TargetID)
	if templateID == "" {
		templateID = strings.TrimSpace(inv.Note)
	}
	_, err := x.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &planningTxnBuffer{}
		payload := events.PlanningEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventPlanningAgentBroadcast,
				Timestamp: x.Now().Format(time.RFC3339Nano),
			},
			SupplierID: inv.SupplierID,
			Action:     templateID,
		}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, inv.SupplierID, events.TopicMain, payload); emitErr != nil {
			return emitErr
		}
		return txn.BufferWrite(planningOutboxMutations(buf.events))
	})
	return eventID, err
}

type agentInsightRow struct {
	WarehouseID      string
	ProductID        string
	SuggestedQuantity int64
	TargetFactoryID  string
	PrimaryFactoryID string
	TransferMode     string
}

func loadInsightForAgent(ctx context.Context, client *spanner.Client, supplierID, insightID string) (agentInsightRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ri.WarehouseId, ri.ProductId, ri.SuggestedQuantity,
		             COALESCE(ri.TargetFactoryId, ''), COALESCE(w.PrimaryFactoryId, ''),
		             COALESCE(w.TransferMode, 'TRUCK')
		      FROM ReplenishmentInsights ri
		      LEFT JOIN Warehouses w ON ri.WarehouseId = w.WarehouseId
		      WHERE ri.InsightId = @iid AND ri.SupplierId = @sid`,
		Params: map[string]any{"iid": insightID, "sid": supplierID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return agentInsightRow{}, err
	}
	var out agentInsightRow
	if err := row.Columns(
		&out.WarehouseID, &out.ProductID, &out.SuggestedQuantity,
		&out.TargetFactoryID, &out.PrimaryFactoryID, &out.TransferMode,
	); err != nil {
		return agentInsightRow{}, err
	}
	return out, nil
}

func nullableAgentString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}
