// Package airoutes owns the /v1/ai/* surface: the Empathy Engine preorder
// scheduling endpoint plus the prediction-feedback surface. Handlers in
// backend-go/order implement the prediction CRUD; the preorder trigger
// adapts OrderService.GeneratePreorder directly.
package airoutes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/order"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// PreorderGenerator is the narrow interface /v1/ai/preorder needs — kept
// small so this package can be tested with a stub instead of a full
// order.OrderService.
type PreorderGenerator interface {
	GeneratePreorder(ctx context.Context, retailerID string) (*order.AIPredictionResult, error)
}

// Deps bundles the collaborators required to register /v1/ai routes.
type Deps struct {
	Spanner  *spanner.Client
	Preorder PreorderGenerator
	Log      Middleware
}

// RegisterRoutes mounts the retailer-facing AI surface:
//
//	POST /v1/ai/preorder             — trigger an Empathy Engine prediction
//	GET  /v1/ai/predictions          — list predictions for the caller
//	POST /v1/ai/predictions/correct  — RLHF correction submission
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	retailer := []string{"RETAILER"}

	r.HandleFunc("/v1/ai/preorder",
		auth.RequireRole(retailer, log(handlePreorder(d.Preorder))))
	r.HandleFunc("/v1/ai/predictions",
		auth.RequireRole(retailer, log(order.HandleListPredictions(d.Spanner))))
	r.HandleFunc("/v1/ai/predictions/correct",
		auth.RequireRole(retailer, log(order.HandlePatchPrediction(d.Spanner))))
}

// handlePreorder adapts PreorderGenerator into an http.HandlerFunc.
// Behaviour preserved verbatim from the inline closure it replaced.
func handlePreorder(gen PreorderGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			RetailerID string `json:"retailer_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.RetailerID == "" {
			if claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims); ok && claims.Role == "RETAILER" {
				req.RetailerID = claims.UserID
			}
		}
		if req.RetailerID == "" {
			http.Error(w, `{"error":"retailer_id required"}`, http.StatusBadRequest)
			return
		}

		prediction, err := gen.GeneratePreorder(r.Context(), req.RetailerID)
		if err != nil {
			log.Printf("[EMPATHY ENGINE] error for %s: %v", req.RetailerID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":           "PREDICTION_SCHEDULED",
			"retailer_id":      prediction.RetailerID,
			"predicted_amount": prediction.PredictedAmount,
			"trigger_date":     prediction.TriggerDate,
			"reasoning":        prediction.ReasoningSummary,
		})
	}
}
