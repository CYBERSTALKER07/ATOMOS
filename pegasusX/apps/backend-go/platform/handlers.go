package platform

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Handler exposes platform HTTP endpoints.
type Handler struct {
	svc        *Service
	tokens     DeviceTokenRepository
	log        *slog.Logger
	writeJSON  func(http.ResponseWriter, int, any)
	writeError func(http.ResponseWriter, int, string)
}

// HandlerConfig wires handler dependencies.
type HandlerConfig struct {
	Service      *Service
	DeviceTokens DeviceTokenRepository
	Log          *slog.Logger
	WriteJSON    func(http.ResponseWriter, int, any)
	WriteError   func(http.ResponseWriter, int, string)
}

// NewHandler creates platform HTTP handlers.
func NewHandler(cfg HandlerConfig) *Handler {
	if cfg.WriteJSON == nil {
		cfg.WriteJSON = defaultWriteJSON
	}
	if cfg.WriteError == nil {
		cfg.WriteError = defaultWriteError
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	return &Handler{
		svc:        cfg.Service,
		tokens:     cfg.DeviceTokens,
		log:        cfg.Log,
		writeJSON:  cfg.WriteJSON,
		writeError: cfg.WriteError,
	}
}

// HandleClientPolicy serves GET /v1/platform/client-policy.
func (h *Handler) HandleClientPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	role := r.URL.Query().Get("role")
	platform := r.URL.Query().Get("platform")
	channel := r.URL.Query().Get("channel")
	version := r.URL.Query().Get("version")
	if version == "" {
		version = r.Header.Get("X-App-Version")
	}
	actorID := r.URL.Query().Get("actor_id")
	if actorID == "" {
		if claims, ok := auth.FromContext(r.Context()); ok {
			actorID = ClaimsActorID(claims)
			if role == "" {
				role = ClaimsRoleForPolicy(claims)
			}
		}
	}
	if role == "" || platform == "" {
		h.writeError(w, http.StatusBadRequest, "role and platform are required")
		return
	}
	resp, err := h.svc.Evaluate(r.Context(), role, platform, channel, version, actorID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "client policy evaluate failed", "err", err)
		h.writeError(w, http.StatusInternalServerError, "policy_evaluate_failed")
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// HandleClientConfig serves GET /v1/platform/client-config.
func (h *Handler) HandleClientConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	role := r.URL.Query().Get("role")
	platform := r.URL.Query().Get("platform")
	version := r.URL.Query().Get("version")
	if version == "" {
		version = r.Header.Get("X-App-Version")
	}

	if claims, ok := auth.FromContext(r.Context()); ok {
		if role == "" {
			role = ClaimsRoleForPolicy(claims)
		}
	}

	if role == "" || platform == "" {
		h.writeError(w, http.StatusBadRequest, "role and platform are required")
		return
	}

	resp, err := h.svc.GetClientConfig(r.Context(), role, platform, version)
	if err != nil {
		h.log.ErrorContext(r.Context(), "client config evaluate failed", "err", err)
		h.writeError(w, http.StatusInternalServerError, "client_config_failed")
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

type upsertPolicyRequest struct {
	Role               string `json:"role"`
	Platform           string `json:"platform"`
	Channel            string `json:"channel"`
	MinimumVersion     string `json:"minimum_version"`
	RecommendedVersion string `json:"recommended_version"`
	UpdateURL          string `json:"update_url"`
	ForceUpdate        bool   `json:"force_update"`
}

// HandleUpsertPolicy serves PUT /v1/platform/client-policy (supplier ADMIN only).
func (h *Handler) HandleUpsertPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		h.writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 16*1024))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "read_body_failed")
		return
	}
	defer r.Body.Close()
	var req upsertPolicyRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if req.Role == "" || req.Platform == "" {
		h.writeError(w, http.StatusBadRequest, "role and platform required")
		return
	}
	if req.Channel == "" {
		req.Channel = "production"
	}
	row := PolicyRow{
		Role:               normalizeRole(req.Role),
		Platform:           normalizePlatform(req.Platform),
		Channel:            req.Channel,
		MinimumVersion:     req.MinimumVersion,
		RecommendedVersion: req.RecommendedVersion,
		UpdateURL:          req.UpdateURL,
		ForceUpdate:        req.ForceUpdate,
	}
	if err := h.svc.policies.UpsertPolicy(r.Context(), row); err != nil {
		h.log.ErrorContext(r.Context(), "upsert client policy failed", "err", err)
		h.writeError(w, http.StatusInternalServerError, "policy_upsert_failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type deviceTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// HandleDeviceToken serves POST /v1/user/device-token with durable registration.
func (h *Handler) HandleDeviceToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8*1024))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "read_body_failed")
		return
	}
	defer r.Body.Close()
	var req deviceTokenRequest
	if err := json.Unmarshal(body, &req); err != nil || req.Token == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if !IsFCMRegistrationToken(req.Token, req.Platform) {
		h.writeError(w, http.StatusUnprocessableEntity, "not_fcm_registration_token")
		return
	}
	if h.tokens == nil {
		h.writeError(w, http.StatusServiceUnavailable, "device_token_unavailable")
		return
	}
	row := DeviceTokenRow{
		ActorID:   ClaimsActorID(claims),
		ActorRole: ClaimsRoleForPolicy(claims),
		Platform:  normalizePlatform(req.Platform),
		Token:     req.Token,
	}
	if err := h.tokens.UpsertToken(r.Context(), row); err != nil {
		h.log.ErrorContext(r.Context(), "device token upsert failed", "err", err)
		h.writeError(w, http.StatusInternalServerError, "device_token_failed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func defaultWriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func defaultWriteError(w http.ResponseWriter, status int, msg string) {
	defaultWriteJSON(w, status, map[string]string{"error": msg})
}
