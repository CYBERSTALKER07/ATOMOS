package notifications

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// PreferenceHandlers serves notification preference CRUD.
type PreferenceHandlers struct {
	Repo Repository
}

func writePrefJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// HandleGetPreferences serves GET /v1/user/notification-preferences.
func (h *PreferenceHandlers) HandleGetPreferences(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writePrefJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	principalID := RecipientIDFromClaims(claims)
	if principalID == "" {
		writePrefJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if h == nil || h.Repo == nil {
		writePrefJSON(w, http.StatusOK, map[string]any{"preferences": []NotificationPreference{}})
		return
	}
	sp, ok := h.Repo.(*SpannerRepository)
	if !ok {
		writePrefJSON(w, http.StatusOK, map[string]any{"preferences": []NotificationPreference{}})
		return
	}
	prefs, err := sp.ListPreferencesForPrincipal(r.Context(), principalID)
	if err != nil {
		writePrefJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed"})
		return
	}
	writePrefJSON(w, http.StatusOK, map[string]any{"preferences": prefs})
}

// HandlePatchPreferences serves PATCH /v1/user/notification-preferences.
func (h *PreferenceHandlers) HandlePatchPreferences(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writePrefJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	principalID := RecipientIDFromClaims(claims)
	if principalID == "" {
		writePrefJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req struct {
		Preferences []NotificationPreference `json:"preferences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writePrefJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if h == nil || h.Repo == nil {
		writePrefJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	principalType := strings.TrimSpace(string(claims.Role))
	for _, pref := range req.Preferences {
		pref.PrincipalID = principalID
		if pref.PrincipalType == "" {
			pref.PrincipalType = principalType
		}
		if strings.TrimSpace(pref.EventType) == "" || strings.TrimSpace(pref.Channel) == "" {
			continue
		}
		if err := h.Repo.UpsertPreference(r.Context(), pref); err != nil {
			writePrefJSON(w, http.StatusInternalServerError, map[string]string{"error": "upsert_failed"})
			return
		}
	}
	writePrefJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
