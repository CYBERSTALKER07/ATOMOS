package simulation

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// HandleStart starts the stress engine. POST /v1/internal/sim/start.
// Query params: orders, drivers, rps, solve_timeout_ms.
func HandleStart(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		cfg := Config{
			Orders:       atoiDefault(r.URL.Query().Get("orders"), 100),
			Drivers:      atoiDefault(r.URL.Query().Get("drivers"), 20),
			RPS:          atoiDefault(r.URL.Query().Get("rps"), 5),
			SolveTimeout: time.Duration(atoiDefault(r.URL.Query().Get("solve_timeout_ms"), 1500)) * time.Millisecond,
		}
		if err := engine.Start(cfg); err != nil {
			if errors.Is(err, ErrAlreadyRunning) {
				http.Error(w, "already running", http.StatusConflict)
				return
			}
			http.Error(w, "start failed", http.StatusInternalServerError)
			return
		}
		slog.Info("simulation.start",
			"orders", cfg.Orders,
			"drivers", cfg.Drivers,
			"rps", cfg.RPS,
			"solve_timeout_ms", cfg.SolveTimeout.Milliseconds(),
		)
		writeJSON(w, http.StatusOK, engine.Status())
	}
}

// HandleStop cancels the stress engine. POST /v1/internal/sim/stop.
func HandleStop(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := engine.Stop(); err != nil {
			if errors.Is(err, ErrNotRunning) {
				http.Error(w, "not running", http.StatusConflict)
				return
			}
			http.Error(w, "stop failed", http.StatusInternalServerError)
			return
		}
		slog.Info("simulation.stop", "final", engine.Status())
		writeJSON(w, http.StatusOK, engine.Status())
	}
}

// HandleStatus returns the current snapshot. GET /v1/internal/sim/status.
func HandleStatus(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, engine.Status())
	}
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
