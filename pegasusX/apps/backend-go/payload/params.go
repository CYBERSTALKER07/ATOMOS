package payload

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// manifestIDParam reads {manifestID} (payloader routes) or {id} (supplier manifest routes).
func manifestIDParam(r *http.Request) string {
	if v := strings.TrimSpace(chi.URLParam(r, "manifestID")); v != "" {
		return v
	}
	return strings.TrimSpace(chi.URLParam(r, "id"))
}
