// Package infraroutes owns infrastructure-only compatibility endpoints.
package infraroutes

import (
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/cache"
)

// Deps bundles collaborators required for infrastructure endpoints.
type Deps struct {
	Spanner              *spanner.Client
	EnableDebugMintToken bool
}

// RegisterRoutes mounts infrastructure-only compatibility endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	r.HandleFunc("/v1/health", handleHealth(d.Spanner))

	if d.EnableDebugMintToken {
		r.HandleFunc("/debug/mint-token", handleDebugMintToken)
	}
}

func handleHealth(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		spannerOK := false
		if spannerClient != nil {
			row, err := spannerClient.Single().ReadRow(r.Context(), "Suppliers", spanner.Key{"health-check-probe"}, []string{"SupplierId"})
			spannerOK = err != nil || row != nil
		}

		redisOK := cache.Client != nil
		if redisOK {
			if err := cache.Client.Ping(r.Context()).Err(); err != nil {
				redisOK = false
			}
		}

		status := "healthy"
		code := http.StatusOK
		if !spannerOK {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  status,
			"spanner": spannerOK,
			"redis":   redisOK,
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func handleDebugMintToken(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if role == "" {
		role = "RETAILER"
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "TEST-USER-99"
	}

	tokenString, err := auth.GenerateTestToken(userID, role)
	if err != nil {
		http.Error(w, "Failed to forge token", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write([]byte(tokenString))
}
