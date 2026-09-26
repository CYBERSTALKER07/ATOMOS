package syncroutes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

// MaxSyncBatchSize caps the number of commands in a single batch to prevent
// resource exhaustion (OOM, Spanner transaction starvation).
const MaxSyncBatchSize = 50

type SyncCommand struct {
	CommandID    string          `json:"commandId"`
	CommandType  string          `json:"commandType"`
	EntityID     string          `json:"entityId"`
	KnownVersion int             `json:"knownVersion"`
	PayloadJSON  json.RawMessage `json:"payloadJson"`
	CreatedAt    int64           `json:"createdAt"`
}

type SyncBatchRequest struct {
	Commands []SyncCommand `json:"commands"`
}

type SyncBatchResponse struct {
	Results []CommandResult `json:"results"`
}

type CommandResult struct {
	CommandID string `json:"commandId"`
	Status    string `json:"status"` // SUCCESS, FAILED, DISPUTED, DUPLICATE
	Error     string `json:"error,omitempty"`
}

// Deps holds the dependencies for sync routes.
type Deps struct {
	Engine *SemanticEngine
	Cache  cache.Backend // Redis backend for idempotency checks
}

// RegisterRoutes mounts the offline sync endpoints.
func RegisterRoutes(r chi.Router, deps Deps) {
	r.Post("/v1/sync/commands", handleSyncCommands(deps))
}

func handleSyncCommands(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.FromContext(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		supplierID := claims.SupplierID

		var req SyncBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}

		// Finding 5: Enforce batch size limit.
		if len(req.Commands) > MaxSyncBatchSize {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "batch_too_large",
				"limit": "50",
			})
			return
		}
		if len(req.Commands) == 0 {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(SyncBatchResponse{Results: []CommandResult{}})
			return
		}

		results := make([]CommandResult, 0, len(req.Commands))

		for _, cmd := range req.Commands {
			// Finding 4: Idempotency guard — check if CommandID was already processed.
			if cmd.CommandID != "" && deps.Cache != nil {
				idempotencyKey := "sync:processed:" + cmd.CommandID
				existing, found, _ := deps.Cache.Get(r.Context(), idempotencyKey)
				if found && len(existing) > 0 {
					// Already processed; return cached status.
					results = append(results, CommandResult{
						CommandID: cmd.CommandID,
						Status:    "DUPLICATE",
					})
					continue
				}
			}

			var res CommandResult
			if deps.Engine != nil {
				res = deps.Engine.ProcessCommand(r.Context(), supplierID, cmd, claims)
			} else {
				res = CommandResult{
					CommandID: cmd.CommandID,
					Status:    "SUCCESS",
				}
			}

			// Mark as processed in Redis with 24h TTL (covers offline reconnect window).
			if cmd.CommandID != "" && deps.Cache != nil && (res.Status == "SUCCESS" || res.Status == "DISPUTED") {
				idempotencyKey := "sync:processed:" + cmd.CommandID
				_ = deps.Cache.Set(r.Context(), idempotencyKey, []byte(res.Status), 24*60*60*1e9) // 24h in nanoseconds
			}

			results = append(results, res)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SyncBatchResponse{Results: results})
	}
}
