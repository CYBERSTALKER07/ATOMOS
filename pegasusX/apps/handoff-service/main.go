// handoff-service exposes stateless delivery QR token validation for internal callers.
// backend-go embeds the same packages/handoff engine in-process for local docker sims;
// run this service separately when you want an isolated validation boundary in production.
package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/packages/handoff"
)

type validateRequest struct {
	OrderID        string `json:"order_id"`
	StoredToken    string `json:"stored_token"`
	PresentedToken string `json:"presented_token"`
}

type publicRequest struct {
	OrderID     string `json:"order_id"`
	StoredToken string `json:"stored_token"`
	Status      string `json:"status"`
}

func buildRouter(engine *handoff.Engine, apiKey string) http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Route("/internal/v1/handoff", func(r chi.Router) {
		r.Use(internalAPIKeyMiddleware(apiKey))
		r.Post("/validate", func(w http.ResponseWriter, req *http.Request) {
			var body validateRequest
			req.Body = http.MaxBytesReader(w, req.Body, 1<<20)
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
			req.Body = http.MaxBytesReader(w, req.Body, 1<<20)
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
	return r
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	engine := handoff.FromEnv()
	if engine.LegacyOrderIDFallback() && handoffFallbackForbidden() {
		log.Fatal("HANDOFF_LEGACY_ORDER_ID_FALLBACK must be false when PEGASUSX_ENV is production or staging")
	}
	apiKey := strings.TrimSpace(os.Getenv("INTERNAL_API_KEY"))
	if apiKey == "" && handoffFallbackForbidden() {
		log.Fatal("INTERNAL_API_KEY must be configured when PEGASUSX_ENV is production or staging")
	}
	port := strings.TrimSpace(os.Getenv("HTTP_PORT"))
	if port == "" {
		port = "8082"
	}

	handler := buildRouter(engine, apiKey)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("handoff-service listening on :%s legacy_fallback=%v", port, engine.LegacyOrderIDFallback())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Fatalf("handoff-service failed to start: %v", err)
	case <-ctx.Done():
		log.Printf("handoff-service shutting down")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("handoff-service graceful shutdown failed: %v", err)
		}
		log.Printf("handoff-service shutdown complete")
	}
}

func handoffFallbackForbidden() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PEGASUSX_ENV"))) {
	case "production", "prod", "staging":
		return true
	default:
		return false
	}
}

func internalAPIKeyMiddleware(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headerKey := strings.TrimSpace(r.Header.Get("X-Internal-Api-Key"))
			if expected == "" || subtle.ConstantTimeCompare([]byte(headerKey), []byte(expected)) != 1 {
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
