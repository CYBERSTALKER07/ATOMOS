package partner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

const maxMasterDataBody = 2 << 20 // 2 MiB for batch upserts

func readMasterDataBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(io.LimitReader(r.Body, maxMasterDataBody))
}

func (h *Handlers) withPartnerIdempotency(
	w http.ResponseWriter,
	r *http.Request,
	p Principal,
	route string,
	body []byte,
	run func() (int, any, error),
) {
	store := h.Svc.IdempotencyStore()
	rawKey := r.Header.Get("Idempotency-Key")
	guardKey := ""
	bodyHash := ""
	if rawKey != "" && store != nil {
		sum := sha256.Sum256(body)
		bodyHash = hex.EncodeToString(sum[:])
		guardKey = idempotency.ScopeKey(string(p.TenantType)+":"+p.TenantID, route, rawKey)
		rec, hit, gErr := idempotency.Guard(r.Context(), store, guardKey, bodyHash)
		if gErr != nil {
			switch {
			case errors.Is(gErr, idempotency.ErrConflict):
				writePartnerError(w, http.StatusConflict, "idempotency_key_payload_mismatch")
			case errors.Is(gErr, idempotency.ErrInProgress):
				writePartnerError(w, http.StatusConflict, "request_in_progress")
			default:
				writePartnerError(w, http.StatusInternalServerError, "idempotency_guard_failed")
			}
			return
		}
		if hit {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Idempotent-Replay", "true")
			w.WriteHeader(rec.StatusCode)
			_, _ = w.Write(rec.Response)
			return
		}
	}
	status, resp, err := run()
	if err != nil {
		if guardKey != "" {
			_ = store.Release(r.Context(), guardKey)
		}
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respBytes, mErr := json.Marshal(resp)
	if mErr != nil {
		if guardKey != "" {
			_ = store.Release(r.Context(), guardKey)
		}
		writePartnerError(w, http.StatusInternalServerError, "encode_error")
		return
	}
	if guardKey != "" {
		if saveErr := store.Save(r.Context(), guardKey, idempotency.Record{
			BodyHash:   bodyHash,
			StatusCode: status,
			Response:   respBytes,
			StoredAt:   time.Now().UTC(),
		}, 24*time.Hour); saveErr != nil {
			slog.Warn("partner masterdata idempotency save failed", "err", saveErr)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(respBytes)
}

// HandleUpsertProducts PUT /partner/v1/catalog/products
func (h *Handlers) HandleUpsertProducts(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := readMasterDataBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		Items []ProductUpsertItem `json:"items"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	h.withPartnerIdempotency(w, r, p, "PUT /partner/v1/catalog/products", body, func() (int, any, error) {
		results, err := h.Svc.UpsertProducts(r.Context(), p, req.Items)
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, map[string]any{"results": results}, nil
	})
}

// HandleUpsertPrices PUT /partner/v1/catalog/prices
func (h *Handlers) HandleUpsertPrices(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := readMasterDataBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		Items []PriceUpsertItem `json:"items"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	h.withPartnerIdempotency(w, r, p, "PUT /partner/v1/catalog/prices", body, func() (int, any, error) {
		results, err := h.Svc.UpsertPrices(r.Context(), p, req.Items)
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, map[string]any{"results": results}, nil
	})
}

// HandleUpsertStock PUT /partner/v1/inventory/stock
func (h *Handlers) HandleUpsertStock(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := readMasterDataBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		Items []StockUpsertItem `json:"items"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	h.withPartnerIdempotency(w, r, p, "PUT /partner/v1/inventory/stock", body, func() (int, any, error) {
		results, err := h.Svc.UpsertStock(r.Context(), p, req.Items)
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, map[string]any{"results": results}, nil
	})
}

// HandleRotateWebhookSecret POST /partner/v1/webhooks/{subscriptionID}/rotate-secret
func (h *Handlers) HandleRotateWebhookSecret(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "subscriptionID")
	secret, err := h.Svc.RotateWebhookSecret(r.Context(), p, id)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subscription_id": id,
		"signing_secret":  secret,
	})
}
