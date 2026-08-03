package platform

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
)

// HandleMediaUploadTicket serves GET /v1/media/upload-ticket?purpose=claim_evidence&ext=jpg
//
// Returns short-lived GCS signed PUT URL + public image_url for claim/driver evidence photos.
// Auth required: retailer, driver, payload, admin, warehouse admin.
func (h *Handler) HandleMediaUploadTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	switch claims.Role {
	case auth.RoleRetailer, auth.RoleDriver, auth.RolePayload, auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse:
		// allowed
	default:
		h.writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	purpose := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("purpose")))
	if purpose == "" {
		purpose = "claim_evidence"
	}
	ext := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("ext")))
	if ext == "" {
		ext = "jpg"
	}
	allowedExt := map[string]bool{"jpg": true, "jpeg": true, "png": true, "webp": true}
	if !allowedExt[ext] {
		h.writeError(w, http.StatusBadRequest, "unsupported_extension")
		return
	}

	scope := strings.TrimSpace(claims.Subject)
	if scope == "" {
		scope = "anon"
	}
	// Optional order id for path organization only.
	orderID := strings.TrimSpace(r.URL.Query().Get("order_id"))
	prefix, err := evidenceObjectPrefix(purpose, scope, orderID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	uploadURL, publicURL, err := storage.GenerateUploadTicketFor(prefix, ext)
	if err != nil {
		h.log.ErrorContext(r.Context(), "media upload ticket failed", "err", err, "purpose", purpose)
		if errors.Is(err, storage.ErrMediaStorageUnavailable) {
			h.writeError(w, http.StatusServiceUnavailable, "media_storage_unavailable")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "internal")
		return
	}
	if storage.IsPlaceholderMediaURL(uploadURL) || storage.IsPlaceholderMediaURL(publicURL) {
		h.log.ErrorContext(r.Context(), "media upload ticket returned placeholder", "purpose", purpose)
		h.writeError(w, http.StatusServiceUnavailable, "media_storage_unavailable")
		return
	}
	contentType := "image/jpeg"
	switch ext {
	case "png":
		contentType = "image/png"
	case "webp":
		contentType = "image/webp"
	}
	h.writeJSON(w, http.StatusOK, map[string]string{
		"upload_url":   uploadURL,
		"public_url":   publicURL,
		"image_url":    publicURL, // alias for catalog clients
		"content_type": contentType,
		"purpose":      purpose,
	})
}

func evidenceObjectPrefix(purpose, subjectID, orderID string) (string, error) {
	subjectID = strings.ReplaceAll(subjectID, "/", "_")
	orderID = strings.ReplaceAll(orderID, "/", "_")
	switch purpose {
	case "claim_evidence", "claim":
		if orderID != "" {
			return fmt.Sprintf("evidence/claims/%s/%s", subjectID, orderID), nil
		}
		return fmt.Sprintf("evidence/claims/%s", subjectID), nil
	case "driver_exception", "exception", "os_and_d":
		if orderID != "" {
			return fmt.Sprintf("evidence/driver/%s/%s", subjectID, orderID), nil
		}
		return fmt.Sprintf("evidence/driver/%s", subjectID), nil
	case "credit_proof":
		return fmt.Sprintf("evidence/credit/%s", subjectID), nil
	default:
		return "", fmt.Errorf("unsupported_purpose")
	}
}
