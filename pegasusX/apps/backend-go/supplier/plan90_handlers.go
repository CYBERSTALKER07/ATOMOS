package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
)

func (s *Service) planningService() *planning.Service {
	if s.portalSpanner == nil {
		return nil
	}
	return planning.NewService(s.portalSpanner).WithCache(s.cache)
}

// HandleMEIONetworkSummary serves GET /v1/supplier/meio/network-summary.
func (s *Service) HandleMEIONetworkSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.replenishmentEngine == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "mei_unavailable"})
		return
	}
	sid := s.scopedSupplierID(r)
	summary, err := s.replenishmentEngine.RunMEIONetwork(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mei_network_failed"})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// HandleControlTowerZoneOverrides serves GET/POST /v1/supplier/control-tower/zone-overrides.
func (s *Service) HandleControlTowerZoneOverrides(w http.ResponseWriter, r *http.Request) {
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "control_tower_unavailable"})
		return
	}
	sid := s.scopedSupplierID(r)
	switch r.Method {
	case http.MethodGet:
		rows, err := svc.ListActiveZoneOverrides(r.Context(), sid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "zone_override_list_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"overrides": rows})
	case http.MethodPost:
		body, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		var in planning.ZoneOverrideInput
		if err := json.Unmarshal(body, &in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_body"})
			return
		}
		claims, _ := auth.FromContext(r.Context())
		createdBy := claims.Subject
		row, err := svc.CreateZoneOverride(r.Context(), sid, createdBy, in)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "zone_override_create_failed"})
			return
		}
		s.broadcastSupplierPlanningEvent(r.Context(), sid, row.WarehouseID, map[string]any{
			"type":        "DISPATCH_ZONE_OVERRIDE",
			"override_id": row.OverrideID,
			"action":      row.Action,
			"supplier_id": sid,
		})
		writeJSON(w, http.StatusCreated, row)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandlePlanningScenarioRun serves POST /v1/supplier/planning/scenarios/run.
func (s *Service) HandlePlanningScenarioRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 8*1024))
	var in planning.ScenarioInput
	_ = json.Unmarshal(body, &in)
	result, err := svc.RunScenario(r.Context(), s.scopedSupplierID(r), in)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scenario_run_failed"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HandlePlanningSAndOP serves GET /v1/supplier/planning/s-and-op.
func (s *Service) HandlePlanningSAndOP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	out, err := svc.GetSAndOP(r.Context(), s.scopedSupplierID(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sandop_failed"})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// HandleKnowledgeGraph serves GET /v1/supplier/knowledge-graph.
func (s *Service) HandleKnowledgeGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "knowledge_graph_unavailable"})
		return
	}
	kg, err := svc.GetKnowledgeGraph(r.Context(), s.scopedSupplierID(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "knowledge_graph_failed"})
		return
	}
	writeJSON(w, http.StatusOK, kg)
}

// HandleGovernedAgentHook serves POST /v1/supplier/planning/agent/invoke.
func (s *Service) HandleGovernedAgentHook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 8*1024))
	var inv planning.AgentInvocation
	if err := json.Unmarshal(body, &inv); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_body"})
		return
	}
	inv.SupplierID = s.scopedSupplierID(r)
	ex := planning.NewExecutor(s.portalSpanner)
	if ex == nil || ex.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent_executor_unavailable"})
		return
	}
	result, err := ex.Execute(r.Context(), inv)
	if err != nil {
		switch {
		case errors.Is(err, planning.ErrAgentActionDenied), errors.Is(err, planning.ErrAgentInvocationInvalid):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HandleReplenishmentPolicies serves GET /v1/supplier/replenishment/policies.
func (s *Service) HandleReplenishmentPolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "policies_unavailable"})
		return
	}
	sid := s.scopedSupplierID(r)
	_ = replenishment.EnsurePolicy(r.Context(), s.portalSpanner, sid)
	policy, err := replenishment.LoadPolicy(r.Context(), s.portalSpanner, sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "policy_load_failed"})
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (s *Service) broadcastSupplierPlanningEvent(ctx context.Context, supplierID, warehouseID string, payload map[string]any) {
	raw, _ := json.Marshal(payload)
	if s.portalSupplierHub != nil {
		s.portalSupplierHub.Broadcast(ctx, "supplier:"+strings.TrimSpace(supplierID), raw)
	}
	if warehouseID != "" && s.portalWarehouseHub != nil {
		s.portalWarehouseHub.Broadcast(ctx, "warehouse:"+strings.TrimSpace(warehouseID), raw)
	}
}

// HandlePlanningSeasonalOverrides serves GET/POST /v1/supplier/planning/seasonal-overrides.
func (s *Service) HandlePlanningSeasonalOverrides(w http.ResponseWriter, r *http.Request) {
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	sid := s.scopedSupplierID(r)
	switch r.Method {
	case http.MethodGet:
		overrides, err := svc.ListSeasonalOverrides(r.Context(), sid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "seasonal_list_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"builtin_templates": planning.BuiltinSeasonalTemplates(),
			"overrides":         overrides,
		})
	case http.MethodPost:
		in, err := planning.ReadSeasonalOverrideBody(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		row, err := svc.CreateSeasonalOverride(r.Context(), sid, in)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, row)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandlePlanningSignalIngest serves POST /v1/supplier/planning/signals/ingest.
func (s *Service) HandlePlanningSignalIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	brokers := strings.Split(strings.TrimSpace(os.Getenv("KAFKA_BROKERS")), ",")
	pub := planning.NewKafkaSignalPublisher(brokers, planning.IngestTopic())
	if pub == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ingest_unavailable"})
		return
	}
	in, err := planning.ReadSignalIngestBody(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	signalID, err := planning.IngestSignal(r.Context(), pub, s.scopedSupplierID(r), in)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"signal_id": signalID, "status": "queued"})
}

// HandlePlanningPromoSimulate serves POST /v1/supplier/planning/promotions/simulate.
func (s *Service) HandlePlanningPromoSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 8*1024))
	var in planning.PromoSimulateInput
	if err := json.Unmarshal(body, &in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_body"})
		return
	}
	result, err := svc.SimulatePromotionPandL(r.Context(), s.scopedSupplierID(r), in)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "promo_simulate_failed"})
		return
	}
	s.broadcastSupplierPlanningEvent(r.Context(), s.scopedSupplierID(r), "", map[string]any{
		"type":          "PLANNING_PROMO_SIMULATION_READY",
		"simulation_id": result.SimulationID,
		"supplier_id":   s.scopedSupplierID(r),
	})
	writeJSON(w, http.StatusOK, result)
}

// HandlePlanningPromoPerformance serves GET /v1/supplier/planning/promotions/{id}/performance.
func (s *Service) HandlePlanningPromoPerformance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	promotionID := chi.URLParam(r, "promotionID")
	if promotionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "promotion_id_required"})
		return
	}
	result, err := svc.GetPromotionPerformance(r.Context(), s.scopedSupplierID(r), promotionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "promo_performance_failed"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HandlePlanningSparsityCheck serves GET /v1/supplier/planning/sparsity/{retailerId}.
func (s *Service) HandlePlanningSparsityCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	retailerID := chi.URLParam(r, "retailerID")
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	result, err := planning.CanForecast(r.Context(), s.portalSpanner, retailerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sparsity_check_failed"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}
