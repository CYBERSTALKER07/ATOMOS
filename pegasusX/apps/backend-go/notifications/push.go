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
	tokenList, err := p.tokens.ListTokens(ctx, actorID, actorRole)
	if err != nil {
		p.log.WarnContext(ctx, "list device tokens failed", "err", err, "actor_id", actorID)
		return
	}
	for _, token := range tokenList {
		if err := p.fcm.SendDataMessage(ctx, token, data); err != nil {
			p.log.DebugContext(ctx, "fcm notify skipped", "err", err, "actor_id", actorID)
		}
	}
}
