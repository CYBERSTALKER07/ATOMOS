package supplier

import (
	"errors"
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func writeMarketLaw(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, auth.ErrCrossMarketDeferred) || errors.Is(err, auth.ErrGeographyIncomplete) ||
		errors.Is(err, auth.ErrMarketPackUnknown) || errors.Is(err, auth.ErrMarketPackNotShipped) {
		status, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, status, map[string]string{"error": code})
		return true
	}
	return false
}
