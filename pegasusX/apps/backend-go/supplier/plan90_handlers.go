package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
)

func (s *Service) planningService() *planning.Service {
	if s.portalSpanner == nil {
		return nil
	}
	svc := planning.NewService(s.portalSpanner).WithCache(s.cache)
	svc.TwinScenarioEnabled = strings.EqualFold(strings.TrimSpace(os.Getenv("TWIN_SCENARIO_ENABLED")), "true")
	return svc
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
	claims, _ := auth.FromContext(r.Context())
	result, err := svc.RunScenario(r.Context(), s.scopedSupplierID(r), claims.Subject, in)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scenario_run_failed"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HandlePlanningScenarioList serves GET /v1/supplier/planning/scenarios.
func (s *Service) HandlePlanningScenarioList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	rows, err := svc.ListScenarios(r.Context(), s.scopedSupplierID(r), 20)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scenario_list_failed"})
		return
	}
	if rows == nil {
		rows = []planning.ScenarioResult{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"scenarios": rows})
}

// HandlePlanningScenarioClone serves POST /v1/supplier/planning/scenarios/{scenarioID}/clone.
func (s *Service) HandlePlanningScenarioClone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	scenarioID := chi.URLParam(r, "scenarioID")
	body, _ := io.ReadAll(io.LimitReader(r.Body, 8*1024))
	var in planning.ScenarioCloneInput
	_ = json.Unmarshal(body, &in)
	claims, _ := auth.FromContext(r.Context())
	result, err := svc.CloneScenario(r.Context(), s.scopedSupplierID(r), scenarioID, claims.Subject, in)
	if err != nil {
		if errors.Is(err, planning.ErrScenarioNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scenario_not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scenario_clone_failed"})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// HandlePlanningScenarioCompare serves POST /v1/supplier/planning/scenarios/compare.
func (s *Service) HandlePlanningScenarioCompare(w http.ResponseWriter, r *http.Request) {
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
	var in planning.ScenarioCompareRequest
	if err := json.Unmarshal(body, &in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_body"})
		return
	}
	result, err := svc.CompareScenarios(r.Context(), s.scopedSupplierID(r), in.ScenarioIDs)
	if err != nil {
		if errors.Is(err, planning.ErrScenarioNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scenario_not_found"})
			return
		}
		if strings.Contains(err.Error(), "compare_requires_two") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "compare_requires_two_scenarios"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scenario_compare_failed"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HandlePlanningScenarioPublish serves POST /v1/supplier/planning/scenarios/{scenarioID}/publish.
func (s *Service) HandlePlanningScenarioPublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	scenarioID := chi.URLParam(r, "scenarioID")
	claims, _ := auth.FromContext(r.Context())
	result, err := svc.PublishScenario(r.Context(), s.scopedSupplierID(r), scenarioID, claims.Subject)
	if err != nil {
		if errors.Is(err, planning.ErrScenarioNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scenario_not_found"})
			return
		}
		if errors.Is(err, planning.ErrScenarioPublishConflict) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "scenario_publish_conflict"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scenario_publish_failed"})
		return
	}
	s.broadcastSupplierPlanningEvent(r.Context(), s.scopedSupplierID(r), "", map[string]any{
		"type":        "planning.scenario.published.v1",
		"scenario_id": result.ScenarioID,
		"version":     result.Version,
		"supplier_id": result.SupplierID,
	})
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

// HandleReplenishmentPolicies serves GET/PATCH /v1/supplier/replenishment/policies.
func (s *Service) HandleReplenishmentPolicies(w http.ResponseWriter, r *http.Request) {
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "policies_unavailable"})
		return
	}
	sid := s.scopedSupplierID(r)
	switch r.Method {
	case http.MethodGet:
		_ = replenishment.EnsurePolicy(r.Context(), s.portalSpanner, sid)
		policy, err := replenishment.LoadPolicy(r.Context(), s.portalSpanner, sid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "policy_load_failed"})
			return
		}
		writeJSON(w, http.StatusOK, policy)
	case http.MethodPatch:
		body, ok := readMutationBody(w, r, 8*1024)
		if !ok {
			return
		}
		key, handled := s.guardMutationReplay(w, r, body)
		if handled {
			return
		}
		cur, err := replenishment.LoadPolicy(r.Context(), s.portalSpanner, sid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "policy_load_failed"})
			return
		}
		var patch struct {
			AutoApproveStable         *bool    `json:"auto_approve_stable"`
			AutoApprovePredictivePush *bool    `json:"auto_approve_predictive_push"`
			MaxDailyTransferUnits     *int64   `json:"max_daily_transfer_units"`
			MinConfidenceScore        *float64 `json:"min_confidence_score"`
			TargetServiceLevel        *float64 `json:"target_service_level"`
			LeadTimeDays              *int64   `json:"lead_time_days"`
			LeadTimeSigmaDays         *float64 `json:"lead_time_sigma_days"`
		}
		if err := json.Unmarshal(body, &patch); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		if patch.AutoApproveStable != nil {
			cur.AutoApproveStable = *patch.AutoApproveStable
		}
		if patch.AutoApprovePredictivePush != nil {
			cur.AutoApprovePredictivePush = *patch.AutoApprovePredictivePush
		}
		if patch.MaxDailyTransferUnits != nil {
			cur.MaxDailyTransferUnits = *patch.MaxDailyTransferUnits
		}
		if patch.MinConfidenceScore != nil {
			cur.MinConfidenceScore = *patch.MinConfidenceScore
		}
		if patch.TargetServiceLevel != nil {
			sl := *patch.TargetServiceLevel
			if sl < 0.5 || sl > 0.999 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "target_service_level_out_of_range"})
				return
			}
			cur.TargetServiceLevel = sl
		}
		if patch.LeadTimeDays != nil {
			if *patch.LeadTimeDays < 1 || *patch.LeadTimeDays > 90 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lead_time_days_out_of_range"})
				return
			}
			cur.LeadTimeDays = *patch.LeadTimeDays
		}
		if patch.LeadTimeSigmaDays != nil {
			if *patch.LeadTimeSigmaDays < 0 || *patch.LeadTimeSigmaDays > 30 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lead_time_sigma_days_out_of_range"})
				return
			}
			cur.LeadTimeSigmaDays = *patch.LeadTimeSigmaDays
		}
		cur.SupplierID = sid
		if err := replenishment.UpsertPolicy(r.Context(), s.portalSpanner, cur); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "policy_upsert_failed"})
			return
		}
		respBytes, _ := json.Marshal(cur)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respBytes)
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
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

// HandlePlanningSeasonalEstimate serves POST /v1/supplier/planning/seasonal-estimate.
// Flag-gated by FORECAST_SEASONAL_ESTIMATE_ENABLED. Returns YoY/month suggestions;
// optional persist_drafts upserts inactive overrides for review (never auto-activates).
func (s *Service) HandlePlanningSeasonalEstimate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if !planning.SeasonalEstimateEnabled() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "seasonal_estimate_disabled"})
		return
	}
	svc := s.planningService()
	if svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	var body struct {
		PersistDrafts bool `json:"persist_drafts"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 4*1024)).Decode(&body)
	result, err := svc.EstimateCalendarMultipliers(r.Context(), s.scopedSupplierID(r), time.Time{}, body.PersistDrafts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
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

// HandlePlanningSignalStatus serves GET /v1/supplier/planning/signals/status.
func (s *Service) HandlePlanningSignalStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "signal_status_unavailable"})
		return
	}
	status, err := planning.LoadSignalIngestStatus(r.Context(), s.portalSpanner, s.scopedSupplierID(r), s.now())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "signal_status_failed"})
		return
	}
	writeJSON(w, http.StatusOK, status)
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
