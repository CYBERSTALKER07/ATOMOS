// Package infraroutes mounts infrastructure-only routes (health, readiness, metrics).
package infraroutes

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// HealthChecker verifies an infrastructure component is reachable.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthCheckFunc adapts a function to HealthChecker.
type HealthCheckFunc func(ctx context.Context) error

// HealthCheck implements HealthChecker.
func (f HealthCheckFunc) HealthCheck(ctx context.Context) error {
	return f(ctx)
}

// Deps supplies component probes for the readiness endpoint.
type Deps struct {
	Checks map[string]HealthChecker
}

// RegisterRoutes mounts /healthz, /ready, /metrics, and legacy /v1/health.
func RegisterRoutes(r chi.Router, d Deps) {
	r.Get("/healthz", handleHealthz)
	r.Get("/ready", handleReady(d))
	r.Get("/metrics", handleMetricsStub)
	r.Get("/v1/health", handleHealthz)
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "pegasusx-backend",
	})
}

func handleReady(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		results := make(map[string]string, len(d.Checks))
		allHealthy := true
		for name, checker := range d.Checks {
			if err := checker.HealthCheck(ctx); err != nil {
				results[name] = err.Error()
				allHealthy = false
			} else {
				results[name] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if allHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     readinessStatus(allHealthy),
			"service":    "pegasusx-backend",
			"components": results,
		})
	}
}

func handleMetricsStub(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("# pegasusx metrics stub\n"))
}

func readinessStatus(ok bool) string {
	if ok {
		return "ok"
	}
	return "degraded"
}
