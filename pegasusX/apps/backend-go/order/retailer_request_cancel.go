package order

import "net/http"

// HandleRetailerRequestCancel is disabled: retailers cannot cancel orders on an active route.
func (s *Service) HandleRetailerRequestCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusForbidden, map[string]string{
		"error":   "cancel_not_allowed",
		"message": "retailers cannot cancel orders after dispatch; contact warehouse or supplier",
	})
}
