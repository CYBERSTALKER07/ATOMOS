package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type kycDocumentRepository interface {
	InsertKycDocument(ctx context.Context, doc KycDocument) error
	ListKycDocumentsByRetailer(ctx context.Context, retailerID string) ([]KycDocument, error)
}

type kycReviewRepository interface {
	UpdateKycDocumentStatus(ctx context.Context, docID string, status string, reviewedBy string, rejectionReason string) error
}

type kycDocumentLookupRepository interface {
	GetKycDocument(ctx context.Context, docID string) (KycDocument, bool, error)
}

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

	if kr, ok := s.repo.(kycDocumentRepository); ok {
		if err := kr.InsertKycDocument(r.Context(), doc); err != nil {
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

	if kr, ok := s.repo.(kycDocumentRepository); ok {
		docs, err := kr.ListKycDocumentsByRetailer(r.Context(), retailerID)
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

	claims, ok := auth.FromContext(r.Context())
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleAdmin && claims.Role != auth.RolePlatformAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	adminID := claims.Subject

	docID := chi.URLParam(r, "documentID")
	if docID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "document_id_required"})
		return
	}

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

	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
	if req.Status != "APPROVED" && req.Status != "REJECTED" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_status"})
		return
	}

	sr, ok := s.repo.(kycReviewRepository)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "repo_not_supported"})
		return
	}

	if claims.Role == auth.RoleAdmin {
		supplierID := strings.TrimSpace(claims.SupplierID)
		if supplierID == "" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		var retailerID string
		if lkp, ok := s.repo.(kycDocumentLookupRepository); ok {
			doc, found, err := lkp.GetKycDocument(r.Context(), docID)
			if err != nil {
				s.log.ErrorContext(r.Context(), "failed to look up KYC doc", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
				return
			}
			if !found {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "document_not_found"})
				return
			}
			retailerID = doc.RetailerID
		}
		authorized, err := s.HasSupplierRetailerLink(r.Context(), supplierID, retailerID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "failed to verify supplier-retailer authorization", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		if !authorized {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
	}

	if err := sr.UpdateKycDocumentStatus(r.Context(), docID, req.Status, adminID, req.RejectionReason); err != nil {
		s.log.ErrorContext(r.Context(), "failed to update KYC doc", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
