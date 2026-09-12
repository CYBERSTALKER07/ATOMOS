package retailer

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func (s *Service) HandleSubmitReturn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}

	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}

	var req struct {
		OrderID string            `json:"order_id"`
		Reason  string            `json:"reason"`
		Lines   []json.RawMessage `json:"lines"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	linesBytes, _ := json.Marshal(req.Lines)

	retReq := ReturnRequest{
		RequestID:  uuid.New().String(),
		RetailerID: retailerID,
		OrderID:    req.OrderID,
		Status:     "PENDING",
		Reason:     req.Reason,
		LinesJSON:  string(linesBytes),
	}

	if sr, ok := s.repo.(*SpannerRepository); ok {
		if err := sr.InsertReturnRequest(r.Context(), retReq); err != nil {
			s.log.ErrorContext(r.Context(), "failed to insert return request", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	} else {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "repo_not_supported"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "submitted", "request_id": retReq.RequestID})
}

func (s *Service) HandleListReturns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}

	if sr, ok := s.repo.(*SpannerRepository); ok {
		reqs, err := sr.ListReturnRequestsByRetailer(r.Context(), retailerID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "failed to list return requests", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"requests": reqs})
	} else {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "repo_not_supported"})
	}
}
