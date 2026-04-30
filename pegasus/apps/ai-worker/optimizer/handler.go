package optimizer

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	contract "optimizercontract"
)

// Handler returns the http.HandlerFunc that mounts at contract.SolvePath.
// It is the only optimiser-side wire surface; everything below this line is
// pure compute. The handler enforces:
//   - method must be POST,
//   - X-Internal-Api-Key must equal apiKey,
//   - request body must be valid JSON SolveRequest with V == contract.V,
//   - solver runs under softTimeout; on timeout the response carries
//     ErrCodeTimeout and the backend triggers Phase 1 fallback.
func Handler(apiKey string, log *slog.Logger, softTimeout time.Duration) http.HandlerFunc {
	if softTimeout <= 0 {
		softTimeout = 2 * time.Second
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, "", contract.ErrCodeBadRequest, http.StatusMethodNotAllowed,
				"POST required")
			return
		}
		if r.Header.Get(contract.AuthHeader) != apiKey {
			writeErr(w, "", contract.ErrCodeAuth, http.StatusUnauthorized,
				"missing or invalid internal api key")
			return
		}
		var req contract.SolveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, "", contract.ErrCodeBadRequest, http.StatusBadRequest,
				"invalid JSON body")
			return
		}
		if req.V != contract.V {
			writeErr(w, req.TraceID, contract.ErrCodeVersion, http.StatusBadRequest,
				"contract version mismatch: server expects "+contract.V)
			return
		}
		if len(req.Stops) == 0 {
			writeErr(w, req.TraceID, contract.ErrCodeBadRequest, http.StatusBadRequest,
				"stops slice is empty")
			return
		}

		// Run solver in a goroutine bounded by softTimeout. The solver itself
		// is single-threaded and CPU-bound; the timeout protects against
		// pathological inputs. On timeout we abandon the goroutine (it will
		// finish in the background and be GC'd when the response flushes).
		done := make(chan result, 1)
		started := time.Now()
		go func() {
			resp, err := Solve(req)
			done <- result{resp: resp, err: err}
		}()

		select {
		case res := <-done:
			if errors.Is(res.err, ErrEmptyFleet) {
				writeErr(w, req.TraceID, contract.ErrCodeEmptyFleet, http.StatusBadRequest,
					"fleet is empty")
				return
			}
			if res.err != nil {
				log.ErrorContext(r.Context(), "optimizer internal",
					"trace_id", req.TraceID,
					"err", res.err,
					"elapsed_ms", time.Since(started).Milliseconds(),
				)
				writeErr(w, req.TraceID, contract.ErrCodeInternal, http.StatusInternalServerError,
					"internal solver error")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(res.resp)
		case <-time.After(softTimeout):
			log.WarnContext(r.Context(), "optimizer timeout",
				"trace_id", req.TraceID,
				"timeout_ms", softTimeout.Milliseconds(),
			)
			writeErr(w, req.TraceID, contract.ErrCodeTimeout, http.StatusGatewayTimeout,
				"solver exceeded timeout budget")
		}
	}
}

type result struct {
	resp contract.SolveResponse
	err  error
}

func writeErr(w http.ResponseWriter, traceID string, code contract.ErrorCode, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(contract.ErrorResponse{
		V:       contract.V,
		TraceID: traceID,
		Code:    code,
		Message: msg,
	})
}
