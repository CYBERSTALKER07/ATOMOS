// handoff-service exposes stateless delivery QR token validation for internal callers.
// backend-go embeds the same packages/handoff engine in-process for local docker sims;
// run this service separately when you want an isolated validation boundary in production.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/packages/handoff"
)

type validateRequest struct {
	OrderID       string `json:"order_id"`
	StoredToken   string `json:"stored_token"`
	PresentedToken string `json:"presented_token"`
}

type publicRequest struct {
	OrderID     string `json:"order_id"`
	StoredToken string `json:"stored_token"`
	Status      string `json:"status"`
}

func main() {
	engine := handoff.FromEnv()
	apiKey := strings.TrimSpace(os.Getenv("INTERNAL_API_KEY"))
	port := strings.TrimSpace(os.Getenv("HTTP_PORT"))
	if port == "" {
		port = "8082"
	}

	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Route("/internal/v1/handoff", func(r chi.Router) {
		r.Use(internalAPIKeyMiddleware(apiKey))
		r.Post("/validate", func(w http.ResponseWriter, req *http.Request) {
			var body validateRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
				return
			}
			if err := engine.Validate(body.OrderID, body.StoredToken, body.PresentedToken); err != nil {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"valid": true})
		})
		r.Post("/public-token", func(w http.ResponseWriter, req *http.Request) {
			var body publicRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
				return
			}
			token := engine.PublicToken(body.OrderID, body.StoredToken, body.Status)
			if strings.TrimSpace(token) == "" {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "delivery token not active"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{
				"order_id": body.OrderID,
				"token":    token,
			})
		})
	})

	log.Printf("handoff-service listening on :%s legacy_fallback=%v", port, engine != nil)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func internalAPIKeyMiddleware(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expected == "" {
				next.ServeHTTP(w, r)
				return
			}
			if strings.TrimSpace(r.Header.Get("X-Internal-Api-Key")) != expected {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
