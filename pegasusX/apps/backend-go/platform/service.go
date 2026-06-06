package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// ClientPolicyResponse is the wire DTO for GET /v1/platform/client-policy.
type ClientPolicyResponse struct {
	Role               string     `json:"role"`
	Platform           string     `json:"platform"`
	Channel            string     `json:"channel"`
	ClientVersion      string     `json:"client_version"`
	MinimumVersion     string     `json:"minimum_version"`
	RecommendedVersion string     `json:"recommended_version"`
	ForceUpdate        bool       `json:"force_update"`
	UpdateURL          string     `json:"update_url,omitempty"`
	UpdateDeferred     bool       `json:"update_deferred"`
	DeferReason        string     `json:"defer_reason,omitempty"`
	GraceUntil         *time.Time `json:"grace_until,omitempty"`
	Outdated           bool       `json:"outdated"`
}

// Service evaluates version policy and builds outdated WS payloads.
type Service struct {
	policies PolicyRepository
	sessions SessionChecker
	log      *slog.Logger
}

// NewService constructs a platform policy service.
func NewService(policies PolicyRepository, sessions SessionChecker, log *slog.Logger) *Service {
	if sessions == nil {
		sessions = NoopSessionChecker{}
	}
	if log == nil {
		log = slog.Default()
	}
	return &Service{policies: policies, sessions: sessions, log: log}
}

// Evaluate returns the client policy for the given tuple and actor (optional deferral).
func (s *Service) Evaluate(ctx context.Context, role, platform, channel, clientVersion, actorID string) (ClientPolicyResponse, error) {
	if channel == "" {
		channel = "production"
	}
	platform = normalizePlatform(platform)
	role = normalizeRole(role)

	policy, ok, err := s.policies.GetPolicy(ctx, role, platform, channel)
	if err != nil {
		return ClientPolicyResponse{}, err
	}
	if !ok {
		policy = PolicyRow{
			Role: role, Platform: platform, Channel: channel,
			MinimumVersion: "0.0.0", RecommendedVersion: "0.0.0",
		}
	}

	resp := ClientPolicyResponse{
		Role:               role,
		Platform:           platform,
		Channel:            channel,
		ClientVersion:      clientVersion,
		MinimumVersion:     policy.MinimumVersion,
		RecommendedVersion: policy.RecommendedVersion,
		UpdateURL:          policy.UpdateURL,
		ForceUpdate:        policy.ForceUpdate,
	}
	if clientVersion != "" && CompareSemver(clientVersion, policy.MinimumVersion) < 0 {
		resp.Outdated = true
	}
	if policy.ForceUpdate && resp.Outdated && actorID != "" {
		active, reason, checkErr := s.sessions.HasActiveCriticalSession(ctx, actorID, role)
		if checkErr != nil {
			s.log.WarnContext(ctx, "update deferral check failed", "err", checkErr, "actor_id", actorID)
		} else if active {
			resp.UpdateDeferred = true
			resp.DeferReason = reason
			resp.ForceUpdate = false
		}
	}
	return resp, nil
}

// OutdatedWSPayload builds the SYSTEM_APP_OUTDATED envelope bytes for a connection.
func (s *Service) OutdatedWSPayload(ctx context.Context, role, platform, channel, clientVersion, actorID, traceID string) ([]byte, bool, error) {
	eval, err := s.Evaluate(ctx, role, platform, channel, clientVersion, actorID)
	if err != nil {
		return nil, false, err
	}
	if !eval.Outdated || eval.UpdateDeferred {
		return nil, false, nil
	}
	body := map[string]any{
		"type":             events.EventSystemAppOutdated,
		"trace_id":         traceID,
		"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
		"minimum_version":  eval.MinimumVersion,
		"client_version":   eval.ClientVersion,
		"recommended_version": eval.RecommendedVersion,
		"update_url":       eval.UpdateURL,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, false, fmt.Errorf("marshal outdated payload: %w", err)
	}
	return raw, true, nil
}

// ClaimsActorID resolves the actor id used for deferral checks.
func ClaimsActorID(claims auth.Claims) string {
	switch claims.Role {
	case auth.RoleDriver, auth.RoleRetailer, auth.RolePayload:
		return claims.Subject
	case auth.RoleAdmin:
		return claims.SupplierID
	default:
		if claims.HomeNodeID != "" {
			return claims.HomeNodeID
		}
		return claims.SupplierID
	}
}

// ClaimsRoleForPolicy maps JWT role to policy role string.
func ClaimsRoleForPolicy(claims auth.Claims) string {
	switch claims.Role {
	case auth.RoleAdmin:
		return "ADMIN"
	case auth.RoleDriver:
		return "DRIVER"
	case auth.RoleRetailer:
		return "RETAILER"
	case auth.RolePayload:
		return "PAYLOAD"
	case auth.RoleWarehouse, auth.RoleWarehouseAdmin:
		return "WAREHOUSE"
	case auth.RoleFactory, auth.RoleFactoryAdmin:
		return "FACTORY"
	default:
		return string(claims.Role)
	}
}

func normalizePlatform(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch p {
	case "iphone", "ipad":
		return "ios"
	case "android", "ios", "web", "desktop", "expo":
		return p
	default:
		if p == "" {
			return "unknown"
		}
		return p
	}
}

func normalizeRole(r string) string {
	r = strings.ToUpper(strings.TrimSpace(r))
	if r == "SUPPLIER" {
		return "ADMIN"
	}
	return r
}
