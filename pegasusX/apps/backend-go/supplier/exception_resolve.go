package supplier

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cashrecon"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnote"
)

// ExceptionResolveDeps wires finance services for inline exception resolution.
type ExceptionResolveDeps struct {
	CashRecon   *cashrecon.Service
	CreditNote  *creditnote.Service
	Credit      *credit.Service
}

// HandleResolveException serves POST /v1/supplier/exceptions/{kind}/{id}/resolve.
func HandleResolveException(deps ExceptionResolveDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		claims, ok := auth.FromContext(r.Context())
		if !ok || claims.Role != auth.RoleAdmin {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		kind := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "kind")))
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if kind == "" || id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "kind_and_id_required"})
			return
		}
		var body struct {
			Note           string `json:"note"`
			CreditNoteID   string `json:"credit_note_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		switch kind {
		case "CASH_DISCREPANCY":
			if deps.CashRecon == nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "cash_recon_unavailable"})
				return
			}
			if err := deps.CashRecon.Accept(r.Context(), id, claims.Subject, body.Note); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		case "CREDIT_NOTE_DRAFT":
			if deps.CreditNote == nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credit_note_unavailable"})
				return
			}
			cnID := strings.TrimSpace(body.CreditNoteID)
			if cnID == "" {
				cnID = id
			}
			if err := deps.CreditNote.Issue(r.Context(), cnID, claims.Subject); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		case "CREDIT_FREEZE":
			if deps.Credit == nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credit_unavailable"})
				return
			}
			existing, found, err := deps.Credit.GetProfile(r.Context(), id, claims.SupplierID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_profile_failed"})
				return
			}
			if !found {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "credit_profile_not_found"})
				return
			}
			existing.Status = credit.StatusActive
			if err := deps.Credit.UpsertProfile(r.Context(), existing, claims.Subject, body.Note); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_exception_kind"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
	}
}
