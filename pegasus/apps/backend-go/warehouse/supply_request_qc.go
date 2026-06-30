package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
)

type SupplyRequestQCResponse struct {
	RequestID   string `json:"request_id"`
	Result      string `json:"result"`
	Notes       string `json:"notes,omitempty"`
	InspectedBy string `json:"inspected_by,omitempty"`
	InspectedAt string `json:"inspected_at,omitempty"`
}

type upsertSupplyRequestQCRequest struct {
	Result string `json:"result"`
	Notes  string `json:"notes"`
}

// HandleSupplyRequestQC reads or records factory QC for a supply request.
// GET  /v1/factory/supply-requests/{id}/qc
// POST /v1/factory/supply-requests/{id}/qc
func (s *SupplyRequestService) HandleSupplyRequestQC(w http.ResponseWriter, r *http.Request) {
	requestID := extractSupplyRequestQCID(r.URL.Path)
	if requestID == "" {
		http.Error(w, `{"error":"request_id required"}`, http.StatusBadRequest)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.readSupplyRequestQC(w, r, requestID)
	case http.MethodPost:
		s.upsertSupplyRequestQC(w, r, requestID, claims.UserID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func extractSupplyRequestQCID(path string) string {
	path = strings.TrimPrefix(path, "/v1/factory/supply-requests/")
	path = strings.TrimSuffix(path, "/qc")
	return strings.Trim(path, "/")
}

func (s *SupplyRequestService) readSupplyRequestQC(w http.ResponseWriter, r *http.Request, requestID string) {
	row, err := s.Spanner.Single().ReadRow(r.Context(), "FactorySupplyRequestQC",
		spanner.Key{requestID},
		[]string{"RequestId", "Result", "Notes", "InspectedBy", "InspectedAt"})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SupplyRequestQCResponse{RequestID: requestID})
		return
	}

	var resp SupplyRequestQCResponse
	var notes spanner.NullString
	var inspectedBy spanner.NullString
	var inspectedAt spanner.NullTime
	if err := row.Columns(&resp.RequestID, &resp.Result, &notes, &inspectedBy, &inspectedAt); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if notes.Valid {
		resp.Notes = notes.StringVal
	}
	if inspectedBy.Valid {
		resp.InspectedBy = inspectedBy.StringVal
	}
	if inspectedAt.Valid {
		resp.InspectedAt = inspectedAt.Time.Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *SupplyRequestService) upsertSupplyRequestQC(w http.ResponseWriter, r *http.Request, requestID, inspectorID string) {
	var req upsertSupplyRequestQCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	result := strings.ToUpper(strings.TrimSpace(req.Result))
	if result != "PASS" && result != "FAIL" {
		http.Error(w, `{"error":"result must be PASS or FAIL"}`, http.StatusBadRequest)
		return
	}

	_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		exists, err := txn.ReadRow(ctx, "SupplyRequests", spanner.Key{requestID}, []string{"RequestId"})
		if err != nil {
			return fmt.Errorf("supply request not found")
		}
		_ = exists

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdate("FactorySupplyRequestQC",
				[]string{"RequestId", "Result", "Notes", "InspectedBy", "InspectedAt"},
				[]interface{}{requestID, result, strings.TrimSpace(req.Notes), inspectorID, spanner.CommitTimestamp}),
		})
	})
	if err != nil {
		log.Printf("[SUPPLY QC] upsert error: %v", err)
		http.Error(w, `{"error":"failed to record QC"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SupplyRequestQCResponse{
		RequestID:   requestID,
		Result:      result,
		Notes:       strings.TrimSpace(req.Notes),
		InspectedBy: inspectorID,
		InspectedAt: time.Now().UTC().Format(time.RFC3339),
	})
}
