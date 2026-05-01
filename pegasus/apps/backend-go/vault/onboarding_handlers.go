package vault

import (
	"encoding/json"
	"log"
	"net/http"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
)

// HandleGatewayOnboarding handles POST (create), GET (read), DELETE (cancel)
// for supplier gateway onboarding sessions.
func HandleGatewayOnboarding(client *spanner.Client) http.HandlerFunc {
	svc := &Service{Spanner: client}

	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		switch r.Method {
		case http.MethodPost:
			handleCreateOnboarding(w, r, svc, supplierID)
		case http.MethodGet:
			handleGetOnboarding(w, r, svc, supplierID)
		case http.MethodDelete:
			handleCancelOnboarding(w, r, svc, supplierID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleCreateOnboarding(w http.ResponseWriter, r *http.Request, svc *Service, supplierID string) {
	var req struct {
		Gateway       string `json:"gateway"`
		ReturnSurface string `json:"return_surface"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if req.Gateway == "" {
		http.Error(w, `{"error":"gateway is required"}`, http.StatusBadRequest)
		return
	}
	if req.ReturnSurface == "" {
		req.ReturnSurface = "web"
	}
	if req.ReturnSurface != "web" && req.ReturnSurface != "desktop" {
		http.Error(w, `{"error":"return_surface must be web or desktop"}`, http.StatusBadRequest)
		return
	}

	session, err := svc.CreateOnboardingSession(r.Context(), supplierID, req.Gateway, req.ReturnSurface)
	if err != nil {
		log.Printf("[VAULT] CreateOnboardingSession error for %s: %v", supplierID, err)
		if isUserError(err) {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

func handleGetOnboarding(w http.ResponseWriter, r *http.Request, svc *Service, supplierID string) {
	sessionID := r.URL.Query().Get("session_id")

	if sessionID != "" {
		// Single session lookup
		session, err := svc.GetOnboardingSession(r.Context(), supplierID, sessionID)
		if err != nil {
			log.Printf("[VAULT] GetOnboardingSession error: %v", err)
			http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)
		return
	}

	// List active sessions
	sessions, err := svc.ListActiveOnboardingSessions(r.Context(), supplierID)
	if err != nil {
		log.Printf("[VAULT] ListActiveOnboardingSessions error for %s: %v", supplierID, err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	if sessions == nil {
		sessions = []OnboardingSessionSummary{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func handleCancelOnboarding(w http.ResponseWriter, r *http.Request, svc *Service, supplierID string) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SessionID == "" {
		http.Error(w, `{"error":"session_id required"}`, http.StatusBadRequest)
		return
	}

	if err := svc.CancelOnboardingSession(r.Context(), supplierID, req.SessionID); err != nil {
		log.Printf("[VAULT] CancelOnboardingSession error: %v", err)
		if isUserError(err) {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "CANCELLED", "session_id": req.SessionID})
}

// isUserError checks if an error message indicates a client-side issue.
func isUserError(err error) bool {
	msg := err.Error()
	for _, prefix := range []string{"gateway", "session does not belong", "session cannot be cancelled", "session not found"} {
		if len(msg) >= len(prefix) && msg[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
