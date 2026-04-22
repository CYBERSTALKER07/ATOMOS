package authroutes

import (
	"encoding/json"
	"log"
	"net/http"

	"backend-go/auth"
)

// handleLegacyRetailerLogin serves POST /v1/auth/login — the original web
// retailer login that predates the supplier.HandleRetailerLogin mobile
// handler. Behaviour preserved verbatim from the inline closure it replaced.
func handleLegacyRetailerLogin(rsp RetailerStatusProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			UserId   string `json:"user_id"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		status, err := rsp.GetRetailerStatus(r.Context(), req.UserId)
		if err != nil {
			log.Printf("Login failed for %s: %v", req.UserId, err)
			http.Error(w, "Invalid Credentials", http.StatusUnauthorized)
			return
		}

		if status != "VERIFIED" {
			http.Error(w, "Clearance Denied. Check KYC Status.", http.StatusForbidden)
			return
		}

		tokenString, err := auth.GenerateTestToken(req.UserId, "RETAILER")
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"token": tokenString,
		})
	}
}
