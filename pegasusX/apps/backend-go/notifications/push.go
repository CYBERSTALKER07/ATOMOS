package notifications

import (
	"context"
	"log/slog"

	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
)

// PushBridge fans out Kafka notification payloads to FCM when tokens exist.
type PushBridge struct {
	fcm    *FCMClient
	tokens platform.DeviceTokenRepository
	log    *slog.Logger
}

// NewPushBridge wires FCM fallback for mobile actors.
func NewPushBridge(fcm *FCMClient, tokens platform.DeviceTokenRepository, log *slog.Logger) *PushBridge {
	if log == nil {
		log = slog.Default()
	}
	return &PushBridge{fcm: fcm, tokens: tokens, log: log}
}

// NotifyActor attempts FCM delivery for each registered token (best-effort).
func (p *PushBridge) NotifyActor(ctx context.Context, actorID, actorRole string, data map[string]string) {
	if p == nil || p.fcm == nil || p.tokens == nil || actorID == "" {
		return
	}
	// G1.D: never look like push succeeded when FCM is no-op — log degraded loudly.
	if p.fcm.IsNoOp() {
		p.log.WarnContext(ctx, "fcm notify skipped: push_degraded no-op client",
			"push_degraded", true, "alert", "fcm_noop", "actor_id", actorID, "actor_role", actorRole)
		return
	}
	tokenList, err := p.tokens.ListTokens(ctx, actorID, actorRole)
	if err != nil {
		p.log.WarnContext(ctx, "list device tokens failed", "err", err, "actor_id", actorID)
		return
	}
	for _, token := range tokenList {
		if err := p.fcm.SendDataMessage(ctx, token, data); err != nil {
			p.log.WarnContext(ctx, "fcm notify failed", "err", err, "actor_id", actorID, "push_degraded", false)
		}
	}
}
