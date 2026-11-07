package supplier

import "net/http"

// FactoryPlanner is the P5 planning surface. Interface avoids factory↔supplier import cycle.
type FactoryPlanner interface {
	HandleNetworkMode(w http.ResponseWriter, r *http.Request, supplierID string)
	HandlePullMatrix(w http.ResponseWriter, r *http.Request, supplierID string)
	HandlePredictivePush(w http.ResponseWriter, r *http.Request, supplierID string)
	HandleKillSwitch(w http.ResponseWriter, r *http.Request, supplierID string)
}

func (s *Service) HandleNetworkMode(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.factoryPlanning == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	s.factoryPlanning.HandleNetworkMode(w, r, s.scopedSupplierID(r))
}

func (s *Service) HandlePlanningPullMatrix(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.factoryPlanning == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	s.factoryPlanning.HandlePullMatrix(w, r, s.scopedSupplierID(r))
}

func (s *Service) HandlePlanningPredictivePush(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.factoryPlanning == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	s.factoryPlanning.HandlePredictivePush(w, r, s.scopedSupplierID(r))
}

func (s *Service) HandlePlanningKillSwitch(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.factoryPlanning == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	s.factoryPlanning.HandleKillSwitch(w, r, s.scopedSupplierID(r))
}
