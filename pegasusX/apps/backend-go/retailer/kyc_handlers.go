package retailer

import (
	"encoding/json"
	"net/http"
	"github.com/go-chi/chi/v5"

	"github.com/google/uuid"
)

func (s *Service) HandleSubmitKyc(w http.ResponseWriter, r *http.Request) {
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
		DocumentType string `json:"document_type"`
		DocumentURL  string `json:"document_url"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	doc := KycDocument{
		DocumentID:   uuid.New().String(),
		RetailerID:   retailerID,
		Status:       "PENDING",
		DocumentType: req.DocumentType,
		DocumentURL:  req.DocumentURL,
	}

	// Assuming the repo is a *SpannerRepository; if it's an interface, we should cast or extend it.
	// We'll type assert for simplicity in this slice.
	if sr, ok := s.repo.(*SpannerRepository); ok {
		if err := sr.InsertKycDocument(r.Context(), doc); err != nil {
			s.log.ErrorContext(r.Context(), "failed to insert KYC doc", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	} else {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "repo_not_supported"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "submitted", "document_id": doc.DocumentID})
}

func (s *Service) HandleListKyc(w http.ResponseWriter, r *http.Request) {
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
		docs, err := sr.ListKycDocumentsByRetailer(r.Context(), retailerID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "failed to list KYC docs", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"documents": docs})
	} else {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "repo_not_supported"})
	}
}

func (s *Service) HandleReviewKyc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	docID := chi.URLParam(r, "documentID")
	if docID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "document_id_required"})
		return
	}

	adminID := "admin_id" // Typically from context

	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}

	var req struct {
		Status          string `json:"status"` // APPROVED, REJECTED
		RejectionReason string `json:"rejection_reason"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	if sr, ok := s.repo.(*SpannerRepository); ok {
		if err := sr.UpdateKycDocumentStatus(r.Context(), docID, req.Status, adminID, req.RejectionReason); err != nil {
			s.log.ErrorContext(r.Context(), "failed to update KYC doc", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
	} else {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "repo_not_supported"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
