package controltower

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnote"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/returns"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
)

// ExecutorDeps wires existing domain services for playbook actions.
type ExecutorDeps struct {
	CreditNotes   *creditnote.Service
	Credit        *credit.Service
	Planning      *planning.Service
	Routing       *routing.Service
	Notifications *notifications.Service
	Returns       *returns.Service
	Segment       *segment.Service
	Log           *slog.Logger
}

// ActionExecutor runs catalogue actions against domain services.
type ActionExecutor struct {
	deps ExecutorDeps
	repo Repository
}

func NewActionExecutor(repo Repository, deps ExecutorDeps) *ActionExecutor {
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	return &ActionExecutor{deps: deps, repo: repo}
}

func (e *ActionExecutor) ExecuteAction(ctx context.Context, runID string, idx int, spec ActionSpec, ex Exception, actor string) ActionResult {
	result := ActionResult{Index: idx, Type: spec.Type, Status: "ok"}
	if e == nil {
		result.Status = "failed"
		result.Message = "executor_unavailable"
		return result
	}
	idemKey := fmt.Sprintf("%s:%d", runID, idx)
	switch strings.ToUpper(strings.TrimSpace(spec.Type)) {
	case "CREATE_CREDIT_NOTE":
		if err := e.createCreditNote(ctx, spec, ex, actor, idemKey); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
		}
	case "FREEZE_CREDIT":
		if err := e.freezeCredit(ctx, ex, actor); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
		}
	case "APPLY_ZONE_OVERRIDE":
		if err := e.applyZoneOverride(ctx, spec, ex, actor); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
		}
	case "REPLAN_ROUTE":
		if err := e.replanRoute(ctx, ex, actor); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
		}
	case "NOTIFY":
		if err := e.notify(ctx, spec, ex); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
		}
	case "ACKNOWLEDGE_EXCEPTION":
		if err := e.repo.UpdateExceptionStatus(ctx, ex.ExceptionID, "ACKNOWLEDGED"); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
		}
	case "OPEN_REVERSE_TASK":
		if err := e.openReverseTask(ctx, spec, ex); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
		}
	case "PRIORITY_BOOST_ORDER":
		if err := e.priorityBoost(ctx, ex, actor); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
		}
	case "ASSIGN_EXCEPTION":
		if err := e.assignException(ctx, spec, ex); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
		}
	default:
		result.Status = "failed"
		result.Message = "unknown_action_type"
	}
	return result
}

func (e *ActionExecutor) createCreditNote(ctx context.Context, spec ActionSpec, ex Exception, actor, idemKey string) error {
	if e.deps.CreditNotes == nil {
		return fmt.Errorf("credit_note_service_unavailable")
	}
	params := decodeStringMap(spec.Params)
	from := strings.TrimSpace(params["from"])
	switch from {
	case "buyer_reject":
		_, err := e.deps.CreditNotes.CreateFromBuyerReject(ctx, ex.OrderID, actor)
		if err != nil && strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return err
	case "claim":
		claimID := strings.TrimSpace(params["claim_id"])
		if claimID == "" {
			claimID = strings.TrimSpace(ex.ClaimID)
		}
		if claimID == "" {
			return fmt.Errorf("claim_id_required")
		}
		_, err := e.deps.CreditNotes.CreateFromClaim(ctx, claimID, actor)
		return err
	default:
		_, err := e.deps.CreditNotes.CreateFromBuyerReject(ctx, ex.OrderID, actor)
		if err != nil && strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return err
	}
}

func (e *ActionExecutor) freezeCredit(ctx context.Context, ex Exception, actor string) error {
	if e.deps.Credit == nil || ex.RetailerID == "" || ex.SupplierID == "" {
		return fmt.Errorf("credit_service_unavailable")
	}
	profile, found, err := e.deps.Credit.GetProfile(ctx, ex.RetailerID, ex.SupplierID)
	if err != nil {
		return err
	}
	if !found {
		profile = credit.Profile{
			RetailerID: ex.RetailerID,
			SupplierID: ex.SupplierID,
		}
	}
	profile.Status = credit.StatusFrozen
	return e.deps.Credit.UpsertProfile(ctx, profile, actor, "playbook:freeze_credit")
}

// unfreezeCredit compensates FREEZE_CREDIT when run finalize fails after freeze.
func (e *ActionExecutor) unfreezeCredit(ctx context.Context, retailerID, supplierID, actor string) error {
	if e == nil || e.deps.Credit == nil || retailerID == "" || supplierID == "" {
		return nil
	}
	profile, found, err := e.deps.Credit.GetProfile(ctx, retailerID, supplierID)
	if err != nil || !found {
		return err
	}
	if profile.Status != credit.StatusFrozen {
		return nil
	}
	profile.Status = credit.StatusActive
	return e.deps.Credit.UpsertProfile(ctx, profile, actor, "playbook:freeze_credit_compensate")
}

func (e *ActionExecutor) applyZoneOverride(ctx context.Context, spec ActionSpec, ex Exception, actor string) error {
	if e.deps.Planning == nil {
		return fmt.Errorf("planning_unavailable")
	}
	params := decodeStringMap(spec.Params)
	action := strings.TrimSpace(params["action"])
	if action == "" {
		action = "FREEZE_DISPATCH"
	}
	warehouseID := strings.TrimSpace(params["warehouse_id"])
	if warehouseID == "" {
		warehouseID = ex.WarehouseID
	}
	ttl := int64(3600)
	if v, ok := params["ttl_seconds"]; ok {
		if n, err := parseInt64(v); err == nil && n > 0 {
			ttl = n
		}
	}
	polygon := params["polygon_geojson"]
	if polygon == "" {
		polygon = "{}"
	}
	_, err := e.deps.Planning.CreateZoneOverride(ctx, ex.SupplierID, actor, planning.ZoneOverrideInput{
		WarehouseID:    warehouseID,
		Action:         action,
		PolygonGeoJSON: json.RawMessage(polygon),
		TTLSeconds:     ttl,
	})
	return err
}

func (e *ActionExecutor) replanRoute(ctx context.Context, ex Exception, actor string) error {
	if e.deps.Routing == nil || strings.TrimSpace(ex.RouteID) == "" {
		return fmt.Errorf("route_id_required")
	}
	return e.deps.Routing.ReplanRoute(ctx, ex.RouteID, "playbook_replan", actor)
}

func (e *ActionExecutor) notify(ctx context.Context, spec ActionSpec, ex Exception) error {
	if e.deps.Notifications == nil {
		return fmt.Errorf("notifications_unavailable")
	}
	params := decodeStringMap(spec.Params)
	role := strings.TrimSpace(params["role"])
	if role == "" {
		role = "SUPPLIER_OPS"
	}
	template := strings.TrimSpace(params["template"])
	title := template
	if title == "" {
		title = ex.Type
	}
	body := fmt.Sprintf("Exception %s on order %s", ex.Type, ex.OrderID)
	recipientID := ex.SupplierID
	if strings.TrimSpace(params["recipient_id"]) != "" {
		recipientID = strings.TrimSpace(params["recipient_id"])
	}
	return e.deps.Notifications.CreateNotification(ctx, recipientID, role, template, title, body, "/exceptions")
}

func (e *ActionExecutor) openReverseTask(ctx context.Context, spec ActionSpec, ex Exception) error {
	if e.deps.Returns == nil {
		return fmt.Errorf("returns_unavailable")
	}
	params := decodeStringMap(spec.Params)
	_, err := e.deps.Returns.OpenTickets(ctx, returns.OpenTicketsInput{
		OrderID:     ex.OrderID,
		WarehouseID: ex.WarehouseID,
		SupplierID:  ex.SupplierID,
		ClaimID:     strings.TrimSpace(params["claim_id"]),
		Source:      "CLAIM",
		Note:        "playbook reverse logistics",
	})
	return err
}

func (e *ActionExecutor) priorityBoost(ctx context.Context, ex Exception, actor string) error {
	return e.applyZoneOverride(ctx, ActionSpec{
		Type:   "APPLY_ZONE_OVERRIDE",
		Params: mustJSON(map[string]string{"action": "PRIORITY_BOOST", "warehouse_id": ex.WarehouseID}),
	}, ex, actor)
}

func (e *ActionExecutor) assignException(ctx context.Context, spec ActionSpec, ex Exception) error {
	params := decodeStringMap(spec.Params)
	role := strings.TrimSpace(params["role"])
	if role == "" {
		return fmt.Errorf("role_required")
	}
	return e.repo.UpdateExceptionAssignee(ctx, ex.ExceptionID, role)
}

func decodeStringMap(raw json.RawMessage) map[string]string {
	out := map[string]string{}
	if len(raw) == 0 {
		return out
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return out
	}
	for k, v := range generic {
		out[k] = fmt.Sprint(v)
	}
	return out
}

func parseInt64(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscan(s, &n)
	return n, err
}

// IsAutoSafeAction returns true for actions that may run without human approval when flags allow.
func IsAutoSafeAction(actionType string) bool {
	switch strings.ToUpper(strings.TrimSpace(actionType)) {
	case "NOTIFY", "ACKNOWLEDGE_EXCEPTION", "ASSIGN_EXCEPTION", "REPLAN_ROUTE":
		return true
	default:
		return false
	}
}
