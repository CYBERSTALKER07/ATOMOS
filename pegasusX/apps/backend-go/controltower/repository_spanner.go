package controltower

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SpannerRepository implements Repository on Spanner.
type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) ListActivePlaybooks(ctx context.Context, supplierID string) ([]Playbook, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT PlaybookId, SupplierId, Name, Description, IsActive, Priority,
		             MatchRulesJson, ActionsJson, AutoExecute, CreatedAt, UpdatedAt, CreatedBy
		      FROM ControlTowerPlaybooks
		      WHERE IsActive = true
		        AND (SupplierId IS NULL OR SupplierId = @sid)
		      ORDER BY Priority DESC`,
		Params: map[string]any{"sid": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	return scanPlaybooks(iter)
}

func scanPlaybooks(iter *spanner.RowIterator) ([]Playbook, error) {
	var rows []Playbook
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var pb Playbook
		var supplier spanner.NullString
		var desc spanner.NullString
		var matchRaw, actionsRaw spanner.NullJSON
		if err := row.Columns(&pb.PlaybookID, &supplier, &pb.Name, &desc, &pb.IsActive, &pb.Priority,
			&matchRaw, &actionsRaw, &pb.AutoExecute, &pb.CreatedAt, &pb.UpdatedAt, &pb.CreatedBy); err != nil {
			return nil, err
		}
		if supplier.Valid {
			pb.SupplierID = supplier.StringVal
		}
		if desc.Valid {
			pb.Description = desc.StringVal
		}
		if matchRaw.Valid {
			if b, err := json.Marshal(matchRaw.Value); err == nil {
				pb.MatchRulesRaw = b
				pb.MatchRules, _ = decodeMatchRules(b)
			}
		}
		if actionsRaw.Valid {
			if b, err := json.Marshal(actionsRaw.Value); err == nil {
				pb.ActionsRaw = b
				pb.Actions, _ = decodeActions(b)
			}
		}
		rows = append(rows, pb)
	}
	return rows, nil
}

func (r *SpannerRepository) GetPlaybook(ctx context.Context, playbookID string) (Playbook, error) {
	row, err := r.client.Single().ReadRow(ctx, "ControlTowerPlaybooks", spanner.Key{playbookID},
		[]string{"PlaybookId", "SupplierId", "Name", "Description", "IsActive", "Priority",
			"MatchRulesJson", "ActionsJson", "AutoExecute", "CreatedAt", "UpdatedAt", "CreatedBy"})
	if err != nil {
		return Playbook{}, err
	}
	var pb Playbook
	var supplier spanner.NullString
	var desc spanner.NullString
	var matchRaw, actionsRaw spanner.NullJSON
	if err := row.Columns(&pb.PlaybookID, &supplier, &pb.Name, &desc, &pb.IsActive, &pb.Priority,
		&matchRaw, &actionsRaw, &pb.AutoExecute, &pb.CreatedAt, &pb.UpdatedAt, &pb.CreatedBy); err != nil {
		return Playbook{}, err
	}
	if supplier.Valid {
		pb.SupplierID = supplier.StringVal
	}
	if desc.Valid {
		pb.Description = desc.StringVal
	}
	if matchRaw.Valid {
		if b, err := json.Marshal(matchRaw.Value); err == nil {
			pb.MatchRulesRaw = b
			pb.MatchRules, _ = decodeMatchRules(b)
		}
	}
	if actionsRaw.Valid {
		if b, err := json.Marshal(actionsRaw.Value); err == nil {
			pb.ActionsRaw = b
			pb.Actions, _ = decodeActions(b)
		}
	}
	return pb, nil
}

func (r *SpannerRepository) CreatePlaybook(ctx context.Context, pb Playbook) error {
	matchRaw := pb.MatchRulesRaw
	if len(matchRaw) == 0 {
		matchRaw, _ = json.Marshal(pb.MatchRules)
	}
	actionsRaw := pb.ActionsRaw
	if len(actionsRaw) == 0 {
		actionsRaw, _ = json.Marshal(pb.Actions)
	}
	row := map[string]any{
		"PlaybookId":     pb.PlaybookID,
		"Name":           pb.Name,
		"Description":    nullableString(pb.Description),
		"IsActive":       pb.IsActive,
		"Priority":       pb.Priority,
		"MatchRulesJson": spanner.NullJSON{Value: jsonRawValue(matchRaw), Valid: true},
		"ActionsJson":    spanner.NullJSON{Value: jsonRawValue(actionsRaw), Valid: true},
		"AutoExecute":    pb.AutoExecute,
		"CreatedAt":      spanner.CommitTimestamp,
		"UpdatedAt":      spanner.CommitTimestamp,
		"CreatedBy":      pb.CreatedBy,
	}
	if strings.TrimSpace(pb.SupplierID) != "" {
		row["SupplierId"] = pb.SupplierID
	}
	// B4 M-P1-4: playbook write + outbox in one RW txn.
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("ControlTowerPlaybooks", row)}); err != nil {
			return err
		}
		return emitControlTowerOutbox(ctx, txn, events.EventControlTowerPlaybookChanged, pb.SupplierID, pb.PlaybookID, "", "", "ACTIVE", "", "CREATE", pb.CreatedBy)
	})
	return err
}

func (r *SpannerRepository) UpdatePlaybook(ctx context.Context, playbookID string, fields map[string]any) error {
	fields["PlaybookId"] = playbookID
	fields["UpdatedAt"] = spanner.CommitTimestamp
	supplierID := ""
	if v, ok := fields["SupplierId"].(string); ok {
		supplierID = v
	}
	action := "UPDATE"
	if active, ok := fields["IsActive"].(bool); ok && !active {
		action = "DEACTIVATE"
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if supplierID == "" {
			if row, rerr := txn.ReadRow(ctx, "ControlTowerPlaybooks", spanner.Key{playbookID}, []string{"SupplierId"}); rerr == nil {
				var sid spanner.NullString
				_ = row.Columns(&sid)
				if sid.Valid {
					supplierID = sid.StringVal
				}
			}
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("ControlTowerPlaybooks", fields)}); err != nil {
			return err
		}
		return emitControlTowerOutbox(ctx, txn, events.EventControlTowerPlaybookChanged, supplierID, playbookID, "", "", "", "", action, "")
	})
	return err
}

func (r *SpannerRepository) DeactivatePlaybook(ctx context.Context, playbookID string) error {
	return r.UpdatePlaybook(ctx, playbookID, map[string]any{"IsActive": false})
}

func (r *SpannerRepository) ListOpenExceptions(ctx context.Context, supplierID string) ([]Exception, error) {
	exceptions, err := r.listTicketExceptions(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	settlement, err := r.listSettlementExceptions(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	fiscal, err := r.listFiscalFailedExceptions(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	shopClosed, err := r.listShopClosedResolvedExceptions(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	out := append(exceptions, settlement...)
	out = append(out, fiscal...)
	out = append(out, shopClosed...)
	return out, nil
}

func (r *SpannerRepository) listTicketExceptions(ctx context.Context, supplierID string) ([]Exception, error) {
	stmt := spanner.Statement{
		SQL: `SELECT et.TicketId, et.Type, et.OrderId, et.Severity, et.Status, et.AssignedRole, et.CreatedAt, et.Payload,
		             o.SupplierId, o.RetailerId, o.TotalMinor, o.RouteId, o.WarehouseId
		      FROM ExceptionTickets et
		      INNER JOIN Orders o ON et.OrderId = o.OrderId
		      WHERE o.SupplierId = @sid
		        AND et.Status IN ('OPEN', 'ACKNOWLEDGED', 'IN_PROGRESS')`,
		Params: map[string]any{"sid": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []Exception
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var ex Exception
		var assigned spanner.NullString
		var payload spanner.NullJSON
		var routeID, warehouseID spanner.NullString
		if err := row.Columns(&ex.ExceptionID, &ex.Type, &ex.OrderID, &ex.Severity, &ex.Status, &assigned, &ex.CreatedAt, &payload,
			&ex.SupplierID, &ex.RetailerID, &ex.AmountMinor, &routeID, &warehouseID); err != nil {
			return nil, err
		}
		if assigned.Valid {
			ex.AssignedRole = assigned.StringVal
		}
		if routeID.Valid {
			ex.RouteID = routeID.StringVal
		}
		if warehouseID.Valid {
			ex.WarehouseID = warehouseID.StringVal
		}
		if payload.Valid {
			if b, err := json.Marshal(payload.Value); err == nil {
				ex.Payload = b
			}
		}
		ex.EntityType = "order"
		ex.EntityID = ex.OrderID
		out = append(out, ex)
	}
	return out, nil
}

func (r *SpannerRepository) listSettlementExceptions(ctx context.Context, supplierID string) ([]Exception, error) {
	stmt := spanner.Statement{
		SQL: `SELECT e.ExceptionId, e.Type, e.OrderId, e.AmountMinor, e.Status, e.CreatedAt,
		             o.SupplierId, o.RetailerId, o.RouteId, o.WarehouseId
		      FROM OrderSettlementExceptions e
		      INNER JOIN Orders o ON e.OrderId = o.OrderId
		      WHERE o.SupplierId = @sid AND e.Status = 'OPEN'`,
		Params: map[string]any{"sid": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []Exception
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var ex Exception
		var routeID, warehouseID spanner.NullString
		if err := row.Columns(&ex.ExceptionID, &ex.Type, &ex.OrderID, &ex.AmountMinor, &ex.Status, &ex.CreatedAt,
			&ex.SupplierID, &ex.RetailerID, &routeID, &warehouseID); err != nil {
			return nil, err
		}
		if routeID.Valid {
			ex.RouteID = routeID.StringVal
		}
		if warehouseID.Valid {
			ex.WarehouseID = warehouseID.StringVal
		}
		ex.Severity = "HIGH"
		ex.EntityType = "order"
		ex.EntityID = ex.OrderID
		out = append(out, ex)
	}
	return out, nil
}

func (r *SpannerRepository) listFiscalFailedExceptions(ctx context.Context, supplierID string) ([]Exception, error) {
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId, o.TotalMinor, o.RetailerId, o.RouteId, o.WarehouseId, o.CreatedAt
		      FROM Orders o
		      WHERE o.SupplierId = @sid
		        AND o.FiscalStatus = 'FISCAL_FAILED'
		        AND o.Status NOT IN ('CANCELLED', 'COMPLETED')`,
		Params: map[string]any{"sid": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []Exception
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var ex Exception
		var routeID, warehouseID spanner.NullString
		if err := row.Columns(&ex.OrderID, &ex.AmountMinor, &ex.RetailerID, &routeID, &warehouseID, &ex.CreatedAt); err != nil {
			return nil, err
		}
		ex.ExceptionID = "fiscal:" + ex.OrderID
		ex.Type = "FISCAL_FAILED"
		ex.Severity = "HIGH"
		ex.Status = ExceptionStatusOpen
		ex.SupplierID = supplierID
		if routeID.Valid {
			ex.RouteID = routeID.StringVal
		}
		if warehouseID.Valid {
			ex.WarehouseID = warehouseID.StringVal
		}
		ex.EntityType = "order"
		ex.EntityID = ex.OrderID
		out = append(out, ex)
	}
	return out, nil
}

func (r *SpannerRepository) listShopClosedResolvedExceptions(ctx context.Context, supplierID string) ([]Exception, error) {
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId, o.TotalMinor, o.RetailerId, o.RouteId, o.WarehouseId, o.UpdatedAt
		      FROM Orders o
		      WHERE o.SupplierId = @sid
		        AND o.ShopClosedResolution = 'RETURNED'`,
		Params: map[string]any{"sid": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []Exception
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var ex Exception
		var routeID, warehouseID spanner.NullString
		if err := row.Columns(&ex.OrderID, &ex.AmountMinor, &ex.RetailerID, &routeID, &warehouseID, &ex.CreatedAt); err != nil {
			return nil, err
		}
		ex.ExceptionID = "shop-closed:" + ex.OrderID
		ex.Type = "SHOP_CLOSED_RETURNED"
		ex.Severity = "MEDIUM"
		ex.Status = ExceptionStatusOpen
		ex.SupplierID = supplierID
		if routeID.Valid {
			ex.RouteID = routeID.StringVal
		}
		if warehouseID.Valid {
			ex.WarehouseID = warehouseID.StringVal
		}
		ex.EntityType = "order"
		ex.EntityID = ex.OrderID
		out = append(out, ex)
	}
	return out, nil
}

func (r *SpannerRepository) ListSupplierIDsWithOpenExceptions(ctx context.Context) ([]string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT o.SupplierId FROM ExceptionTickets et
		      INNER JOIN Orders o ON et.OrderId = o.OrderId
		      WHERE et.Status IN ('OPEN', 'ACKNOWLEDGED', 'IN_PROGRESS')
		      UNION
		      SELECT DISTINCT o.SupplierId FROM OrderSettlementExceptions e
		      INNER JOIN Orders o ON e.OrderId = o.OrderId WHERE e.Status = 'OPEN'
		      UNION
		      SELECT DISTINCT SupplierId FROM Orders WHERE FiscalStatus = 'FISCAL_FAILED' AND Status NOT IN ('CANCELLED', 'COMPLETED')
		      UNION
		      SELECT DISTINCT SupplierId FROM Orders WHERE ShopClosedResolution = 'RETURNED'`,
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var ids []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var sid string
		if err := row.Columns(&sid); err != nil {
			return nil, err
		}
		if strings.TrimSpace(sid) != "" {
			ids = append(ids, sid)
		}
	}
	return ids, nil
}

func (r *SpannerRepository) HasBlockingRun(ctx context.Context, exceptionID string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RunId FROM ControlTowerPlaybookRuns
		      WHERE ExceptionId = @eid AND Status IN ('SUGGESTED', 'APPROVED', 'EXECUTED')
		      LIMIT 1`,
		Params: map[string]any{"eid": exceptionID},
	}
	iter := r.client.Single().Query(ctx, stmt)
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

func (r *SpannerRepository) CreateRun(ctx context.Context, run PlaybookRun) error {
	resultsRaw := run.ActionsResultRaw
	if len(resultsRaw) == 0 && len(run.ActionsResult) > 0 {
		resultsRaw, _ = json.Marshal(run.ActionsResult)
	}
	row := map[string]any{
		"RunId":        run.RunID,
		"PlaybookId":   run.PlaybookID,
		"ExceptionId":  run.ExceptionID,
		"SupplierId":   run.SupplierID,
		"Mode":         run.Mode,
		"Status":       run.Status,
		"CreatedAt":    spanner.CommitTimestamp,
	}
	if len(resultsRaw) > 0 {
		row["ActionsResultJson"] = spanner.NullJSON{Value: jsonRawValue(resultsRaw), Valid: true}
	}
	if run.ExecutedAt != nil {
		row["ExecutedAt"] = *run.ExecutedAt
	}
	if run.ExecutedBy != "" {
		row["ExecutedBy"] = run.ExecutedBy
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("ControlTowerPlaybookRuns", row)}); err != nil {
			return err
		}
		return emitControlTowerOutbox(ctx, txn, events.EventControlTowerRunCreated, run.SupplierID, run.PlaybookID, run.RunID, run.ExceptionID, run.Status, run.Mode, "CREATE", run.ExecutedBy)
	})
	return err
}

func (r *SpannerRepository) GetRun(ctx context.Context, runID string) (PlaybookRun, error) {
	row, err := r.client.Single().ReadRow(ctx, "ControlTowerPlaybookRuns", spanner.Key{runID},
		[]string{"RunId", "PlaybookId", "ExceptionId", "SupplierId", "Mode", "Status", "ActionsResultJson", "CreatedAt", "ExecutedAt", "ExecutedBy"})
	if err != nil {
		return PlaybookRun{}, err
	}
	return scanRunRow(row)
}

func scanRunRow(row *spanner.Row) (PlaybookRun, error) {
	var run PlaybookRun
	var results spanner.NullJSON
	var executedAt spanner.NullTime
	var executedBy spanner.NullString
	if err := row.Columns(&run.RunID, &run.PlaybookID, &run.ExceptionID, &run.SupplierID, &run.Mode, &run.Status, &results, &run.CreatedAt, &executedAt, &executedBy); err != nil {
		return PlaybookRun{}, err
	}
	if results.Valid {
		if b, err := json.Marshal(results.Value); err == nil {
			run.ActionsResultRaw = b
			_ = json.Unmarshal(b, &run.ActionsResult)
		}
	}
	if executedAt.Valid {
		t := executedAt.Time
		run.ExecutedAt = &t
	}
	if executedBy.Valid {
		run.ExecutedBy = executedBy.StringVal
	}
	return run, nil
}

func (r *SpannerRepository) UpdateRun(ctx context.Context, run PlaybookRun) error {
	resultsRaw := run.ActionsResultRaw
	if len(resultsRaw) == 0 && len(run.ActionsResult) > 0 {
		resultsRaw, _ = json.Marshal(run.ActionsResult)
	}
	row := map[string]any{
		"RunId":  run.RunID,
		"Status": run.Status,
	}
	if len(resultsRaw) > 0 {
		row["ActionsResultJson"] = spanner.NullJSON{Value: jsonRawValue(resultsRaw), Valid: true}
	}
	if run.ExecutedAt != nil {
		row["ExecutedAt"] = *run.ExecutedAt
	}
	if run.ExecutedBy != "" {
		row["ExecutedBy"] = run.ExecutedBy
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("ControlTowerPlaybookRuns", row)}); err != nil {
			return err
		}
		return emitControlTowerOutbox(ctx, txn, events.EventControlTowerRunUpdated, run.SupplierID, run.PlaybookID, run.RunID, run.ExceptionID, run.Status, run.Mode, "UPDATE", run.ExecutedBy)
	})
	return err
}

// emitControlTowerOutbox buffers a control-tower lifecycle event on the active txn (B4 M-P1-4).
func emitControlTowerOutbox(ctx context.Context, txn *spanner.ReadWriteTransaction, eventType, supplierID, playbookID, runID, exceptionID, status, mode, action, actorID string) error {
	if txn == nil || strings.TrimSpace(eventType) == "" {
		return nil
	}
	aggID := strings.TrimSpace(runID)
	if aggID == "" {
		aggID = strings.TrimSpace(playbookID)
	}
	if aggID == "" {
		aggID = strings.TrimSpace(supplierID)
	}
	if aggID == "" {
		aggID = "control-tower"
	}
	payload := events.ControlTowerEvent{
		BaseEvent: events.BaseEvent{
			Type:      eventType,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		},
		SupplierID:  strings.TrimSpace(supplierID),
		PlaybookID:  strings.TrimSpace(playbookID),
		RunID:       strings.TrimSpace(runID),
		ExceptionID: strings.TrimSpace(exceptionID),
		Status:      strings.TrimSpace(status),
		Mode:        strings.TrimSpace(mode),
		Action:      strings.TrimSpace(action),
		ActorID:     strings.TrimSpace(actorID),
	}
	buf := outbox.NewSpannerTxnBuffer(txn)
	if err := outbox.EmitJSON(ctx, buf, events.AggregateControlTower, aggID, events.TopicMain, payload); err != nil {
		return err
	}
	return buf.Flush(ctx)
}

func (r *SpannerRepository) ListRuns(ctx context.Context, supplierID, status string, limit int) ([]PlaybookRun, error) {
	if limit <= 0 {
		limit = 50
	}
	sql := `SELECT r.RunId, r.PlaybookId, r.ExceptionId, r.SupplierId, r.Mode, r.Status,
	               r.ActionsResultJson, r.CreatedAt, r.ExecutedAt, r.ExecutedBy,
	               p.Name
	        FROM ControlTowerPlaybookRuns r
	        LEFT JOIN ControlTowerPlaybooks p ON r.PlaybookId = p.PlaybookId
	        WHERE r.SupplierId = @sid`
	params := map[string]any{"sid": supplierID, "limit": limit}
	if status != "" {
		sql += " AND r.Status = @status"
		params["status"] = status
	}
	sql += " ORDER BY r.CreatedAt DESC LIMIT @limit"
	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var runs []PlaybookRun
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var run PlaybookRun
		var results spanner.NullJSON
		var executedAt spanner.NullTime
		var executedBy, name spanner.NullString
		if err := row.Columns(&run.RunID, &run.PlaybookID, &run.ExceptionID, &run.SupplierID, &run.Mode, &run.Status, &results, &run.CreatedAt, &executedAt, &executedBy, &name); err != nil {
			return nil, err
		}
		if results.Valid {
			if b, err := json.Marshal(results.Value); err == nil {
				run.ActionsResultRaw = b
				_ = json.Unmarshal(b, &run.ActionsResult)
			}
		}
		if executedAt.Valid {
			t := executedAt.Time
			run.ExecutedAt = &t
		}
		if executedBy.Valid {
			run.ExecutedBy = executedBy.StringVal
		}
		if name.Valid {
			run.PlaybookName = name.StringVal
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (r *SpannerRepository) ListRunsForException(ctx context.Context, exceptionID string) ([]PlaybookRun, error) {
	stmt := spanner.Statement{
		SQL: `SELECT r.RunId, r.PlaybookId, r.ExceptionId, r.SupplierId, r.Mode, r.Status,
		             r.ActionsResultJson, r.CreatedAt, r.ExecutedAt, r.ExecutedBy, p.Name
		      FROM ControlTowerPlaybookRuns r
		      LEFT JOIN ControlTowerPlaybooks p ON r.PlaybookId = p.PlaybookId
		      WHERE r.ExceptionId = @eid
		      ORDER BY r.CreatedAt DESC`,
		Params: map[string]any{"eid": exceptionID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var runs []PlaybookRun
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var run PlaybookRun
		var results spanner.NullJSON
		var executedAt spanner.NullTime
		var executedBy, name spanner.NullString
		if err := row.Columns(&run.RunID, &run.PlaybookID, &run.ExceptionID, &run.SupplierID, &run.Mode, &run.Status, &results, &run.CreatedAt, &executedAt, &executedBy, &name); err != nil {
			return nil, err
		}
		if results.Valid {
			if b, err := json.Marshal(results.Value); err == nil {
				run.ActionsResultRaw = b
				_ = json.Unmarshal(b, &run.ActionsResult)
			}
		}
		if executedAt.Valid {
			t := executedAt.Time
			run.ExecutedAt = &t
		}
		if executedBy.Valid {
			run.ExecutedBy = executedBy.StringVal
		}
		if name.Valid {
			run.PlaybookName = name.StringVal
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (r *SpannerRepository) UpdateExceptionStatus(ctx context.Context, exceptionID, status string) error {
	if strings.HasPrefix(exceptionID, "fiscal:") || strings.HasPrefix(exceptionID, "shop-closed:") {
		return nil
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    `UPDATE ExceptionTickets SET Status = @status WHERE TicketId = @id`,
			Params: map[string]any{"status": status, "id": exceptionID},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	return err
}

func (r *SpannerRepository) UpdateExceptionAssignee(ctx context.Context, exceptionID, role string) error {
	if strings.HasPrefix(exceptionID, "fiscal:") || strings.HasPrefix(exceptionID, "shop-closed:") {
		return nil
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    `UPDATE ExceptionTickets SET AssignedRole = @role WHERE TicketId = @id`,
			Params: map[string]any{"role": role, "id": exceptionID},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	return err
}

// SeedPlatformPlaybooks inserts default playbooks when none exist for platform scope.
func (r *SpannerRepository) SeedPlatformPlaybooks(ctx context.Context) error {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT PlaybookId FROM ControlTowerPlaybooks WHERE SupplierId IS NULL LIMIT 1`,
	})
	defer iter.Stop()
	if _, err := iter.Next(); err != iterator.Done {
		if err != nil {
			return err
		}
		return nil
	}
	for _, pb := range defaultSeedPlaybooks() {
		if err := r.CreatePlaybook(ctx, pb); err != nil {
			return fmt.Errorf("seed playbook %s: %w", pb.Name, err)
		}
	}
	return nil
}

func defaultSeedPlaybooks() []Playbook {
	now := time.Now()
	createdBy := "system:seed"
	makePB := func(name, desc string, priority int64, rules MatchRules, actions []ActionSpec) Playbook {
		matchRaw, _ := json.Marshal(rules)
		actionsRaw, _ := json.Marshal(actions)
		return Playbook{
			PlaybookID:    uuid.NewString(),
			Name:          name,
			Description:   desc,
			IsActive:      true,
			Priority:      priority,
			MatchRules:    rules,
			MatchRulesRaw: matchRaw,
			Actions:       actions,
			ActionsRaw:    actionsRaw,
			AutoExecute:   false,
			CreatedAt:     now,
			UpdatedAt:     now,
			CreatedBy:     createdBy,
		}
	}
	return []Playbook{
		makePB("High-value buyer reject", "Credit note suggestion + finance notify for buyer rejections",
			100,
			MatchRules{Types: []string{"BUYER_REJECTED"}, Severities: []string{"HIGH", "CRITICAL"}, MinAmountMinor: 500000},
			[]ActionSpec{
				{Type: "CREATE_CREDIT_NOTE", Params: mustJSON(map[string]string{"from": "buyer_reject"})},
				{Type: "NOTIFY", Params: mustJSON(map[string]string{"role": "SUPPLIER_FINANCE", "template": "buyer_reject_high_value"})},
				{Type: "ACKNOWLEDGE_EXCEPTION"},
			}),
		makePB("Fiscal failed", "Notify finance and assign fiscal exception",
			90,
			MatchRules{Types: []string{"FISCAL_FAILED"}},
			[]ActionSpec{
				{Type: "NOTIFY", Params: mustJSON(map[string]string{"role": "SUPPLIER_FINANCE", "template": "fiscal_failed"})},
				{Type: "ASSIGN_EXCEPTION", Params: mustJSON(map[string]string{"role": "SUPPLIER_FINANCE"})},
			}),
		makePB("Cash short", "Notify finance on open cash reconciliation shortfall",
			80,
			MatchRules{Types: []string{"CASH_SHORT"}},
			[]ActionSpec{
				{Type: "NOTIFY", Params: mustJSON(map[string]string{"role": "SUPPLIER_FINANCE", "template": "cash_short"})},
				{Type: "ACKNOWLEDGE_EXCEPTION"},
			}),
		makePB("Shop closed returned", "Notify ops and replan route when goods returned",
			70,
			MatchRules{Types: []string{"SHOP_CLOSED_RETURNED"}},
			[]ActionSpec{
				{Type: "NOTIFY", Params: mustJSON(map[string]string{"role": "SUPPLIER_OPS", "template": "shop_closed_returned"})},
				{Type: "REPLAN_ROUTE"},
			}),
		makePB("Strategic allocation short", "Notify ops for segment A allocation shortfalls",
			60,
			MatchRules{Types: []string{"ALLOCATION_SHORT"}, RetailerSegments: []string{"A"}},
			[]ActionSpec{
				{Type: "NOTIFY", Params: mustJSON(map[string]string{"role": "SUPPLIER_OPS", "template": "allocation_short"})},
				{Type: "PRIORITY_BOOST_ORDER"},
			}),
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func nullableString(s string) spanner.NullString {
	if strings.TrimSpace(s) == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: s, Valid: true}
}

func jsonRawValue(raw []byte) any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return map[string]any{}
	}
	return v
}
