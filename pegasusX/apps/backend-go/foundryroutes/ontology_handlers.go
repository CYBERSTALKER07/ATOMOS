package foundryroutes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

type ontologyHandler struct {
	eng *foundry.Engine
}

func newOntologyHandler(eng *foundry.Engine) *ontologyHandler {
	return &ontologyHandler{eng: eng}
}

func (h *ontologyHandler) mountRoutes(r chi.Router) {
	r.Post("/v1/foundry/ontology/objects", h.handleUpsertObject)
	r.Get("/v1/foundry/ontology/objects/{cellID}/{objectType}/{objectID}", h.handleGetObject)
	r.Post("/v1/foundry/ontology/links", h.handleLinkObjects)
	r.Post("/v1/foundry/ontology/search-around", h.handleSearchAround)
	r.Post("/v1/foundry/ontology/lot-recall", h.handleLotRecall)
	r.Post("/v1/foundry/kinetic/actions/execute", h.handleExecuteAction)
	r.Post("/v1/foundry/scenarios/{scenarioID}/patches", h.handleApplyScenarioPatch)
	r.Get("/v1/foundry/scenarios/{scenarioID}/resolve/{cellID}/{objectType}/{objectID}", h.handleResolveScenarioObject)
	r.Post("/v1/foundry/scenarios/{scenarioID}/commit", h.handleCommitScenario)
}

func (h *ontologyHandler) handleUpsertObject(w http.ResponseWriter, r *http.Request) {
	var req foundry.OntologyObject
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	obj, err := h.eng.UpsertObject(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, obj)
}

func (h *ontologyHandler) handleGetObject(w http.ResponseWriter, r *http.Request) {
	cellID := chi.URLParam(r, "cellID")
	objectType := chi.URLParam(r, "objectType")
	objectID := chi.URLParam(r, "objectID")
	obj, err := h.eng.GetObject(r.Context(), cellID, objectType, objectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (h *ontologyHandler) handleLinkObjects(w http.ResponseWriter, r *http.Request) {
	var req foundry.OntologyGraphEdge
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	edge, err := h.eng.LinkObjects(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, edge)
}

func (h *ontologyHandler) handleSearchAround(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CellID           string   `json:"cell_id"`
		StartObjectType  string   `json:"start_object_type"`
		StartObjectIDs   []string `json:"start_object_ids"`
		AllowedLinkTypes []string `json:"allowed_link_types"`
		MaxHops          int      `json:"max_hops"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	objects, err := h.eng.SearchAround(r.Context(), req.CellID, req.StartObjectType, req.StartObjectIDs, req.AllowedLinkTypes, req.MaxHops)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"objects": objects})
}

func (h *ontologyHandler) handleLotRecall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CellID           string `json:"cell_id"`
		SupplierID       string `json:"supplier_id"`
		RawMaterialLotID string `json:"raw_material_lot_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	report, err := h.eng.TriggerContaminatedLotRecall(r.Context(), req.CellID, req.SupplierID, req.RawMaterialLotID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *ontologyHandler) handleExecuteAction(w http.ResponseWriter, r *http.Request) {
	var req foundry.ActionExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	rec, err := h.eng.ExecuteKineticAction(r.Context(), req, func(ctx context.Context) (map[string]any, error) {
		return req.Parameters, nil
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (h *ontologyHandler) handleApplyScenarioPatch(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "scenarioID")
	var req struct {
		CellID          string         `json:"cell_id"`
		SupplierID      string         `json:"supplier_id"`
		ObjectType      string         `json:"object_type"`
		ObjectID        string         `json:"object_id"`
		Operation       string         `json:"operation"`
		PatchProperties map[string]any `json:"patch_properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	patch, err := h.eng.ApplyScenarioPatch(r.Context(), req.CellID, scenarioID, req.SupplierID, req.ObjectType, req.ObjectID, req.Operation, req.PatchProperties)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, patch)
}

func (h *ontologyHandler) handleResolveScenarioObject(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "scenarioID")
	cellID := chi.URLParam(r, "cellID")
	objectType := chi.URLParam(r, "objectType")
	objectID := chi.URLParam(r, "objectID")
	obj, err := h.eng.ResolveObjectState(r.Context(), cellID, scenarioID, objectType, objectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (h *ontologyHandler) handleCommitScenario(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "scenarioID")
	res, err := h.eng.CommitScenario(r.Context(), scenarioID)
	if err != nil {
		if errors.Is(err, foundry.ErrScenarioWatermarkDrift) {
			writeJSON(w, http.StatusConflict, res)
			return
		}
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
