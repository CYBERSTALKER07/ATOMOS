package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"backend-go/cache"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	CommandStateInitiated  = "INITIATED"
	CommandStateDispatched = "DISPATCHED"
	CommandStateReceived   = "RECEIVED"
	CommandStateSettled    = "SETTLED"
)

var ErrCommandNotFound = errors.New("ws command not found")

type commandEntry struct {
	Record    CommandRecord
	ExpiresAt time.Time
}

// CommandRecord captures the verified command lifecycle state persisted in Redis.
type CommandRecord struct {
	CommandID    string                 `json:"command_id"`
	TraceID      string                 `json:"trace_id"`
	EventType    string                 `json:"event_type"`
	TargetRole   string                 `json:"target_role"`
	TargetID     string                 `json:"target_id"`
	SupplierID   string                 `json:"supplier_id,omitempty"`
	State        string                 `json:"state"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	AckByUserID  string                 `json:"ack_by_user_id,omitempty"`
	AckByRole    string                 `json:"ack_by_role,omitempty"`
	InitiatedAt  time.Time              `json:"initiated_at"`
	DispatchedAt time.Time              `json:"dispatched_at"`
	ReceivedAt   *time.Time             `json:"received_at,omitempty"`
	SettledAt    *time.Time             `json:"settled_at,omitempty"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// CommandRegistry keeps websocket command lifecycle records in Redis with a
// process-local fallback when Redis is unavailable.
type CommandRegistry struct {
	mu    sync.RWMutex
	local map[string]commandEntry
	redis redis.UniversalClient
	ttl   time.Duration
	now   func() time.Time
}

func NewCommandRegistry(redisClient redis.UniversalClient) *CommandRegistry {
	return &CommandRegistry{
		local: make(map[string]commandEntry),
		redis: redisClient,
		ttl:   cache.TTLWSCommandRegistry,
		now:   time.Now,
	}
}

func (r *CommandRegistry) RegisterDispatch(
	ctx context.Context,
	targetRole string,
	targetID string,
	supplierID string,
	eventType string,
	traceID string,
	payload map[string]interface{},
) (CommandRecord, error) {
	now := r.now().UTC()
	if traceID == "" {
		traceID = uuid.NewString()
	}
	cmd := CommandRecord{
		CommandID:    uuid.NewString(),
		TraceID:      traceID,
		EventType:    eventType,
		TargetRole:   targetRole,
		TargetID:     targetID,
		SupplierID:   supplierID,
		State:        CommandStateInitiated,
		Payload:      clonePayload(payload),
		InitiatedAt:  now,
		DispatchedAt: now,
		UpdatedAt:    now,
	}
	if err := r.upsert(ctx, cmd); err != nil {
		return CommandRecord{}, err
	}
	return cmd, nil
}

func (r *CommandRegistry) MarkDispatched(
	ctx context.Context,
	commandID string,
	traceID string,
) (CommandRecord, error) {
	cmd, err := r.Get(ctx, commandID)
	if err != nil {
		return CommandRecord{}, err
	}
	if cmd.State == CommandStateDispatched || cmd.State == CommandStateReceived || cmd.State == CommandStateSettled {
		return cmd, nil
	}
	now := r.now().UTC()
	cmd.State = CommandStateDispatched
	cmd.DispatchedAt = now
	cmd.UpdatedAt = now
	if traceID != "" {
		cmd.TraceID = traceID
	}
	if err := r.upsert(ctx, cmd); err != nil {
		return CommandRecord{}, err
	}
	return cmd, nil
}

func (r *CommandRegistry) MarkReceived(
	ctx context.Context,
	commandID string,
	ackByUserID string,
	ackByRole string,
	traceID string,
) (CommandRecord, error) {
	cmd, err := r.Get(ctx, commandID)
	if err != nil {
		return CommandRecord{}, err
	}
	if cmd.State == CommandStateSettled || cmd.State == CommandStateReceived {
		return cmd, nil
	}
	now := r.now().UTC()
	if cmd.State == CommandStateInitiated {
		cmd.State = CommandStateDispatched
		cmd.DispatchedAt = now
	}
	cmd.State = CommandStateReceived
	cmd.ReceivedAt = &now
	cmd.UpdatedAt = now
	if ackByUserID != "" {
		cmd.AckByUserID = ackByUserID
	}
	if ackByRole != "" {
		cmd.AckByRole = ackByRole
	}
	if traceID != "" {
		cmd.TraceID = traceID
	}
	if err := r.upsert(ctx, cmd); err != nil {
		return CommandRecord{}, err
	}
	return cmd, nil
}

func (r *CommandRegistry) MarkSettled(
	ctx context.Context,
	commandID string,
	ackByUserID string,
	ackByRole string,
	traceID string,
) (CommandRecord, error) {
	cmd, err := r.Get(ctx, commandID)
	if err != nil {
		return CommandRecord{}, err
	}
	if cmd.State == CommandStateSettled {
		return cmd, nil
	}
	now := r.now().UTC()
	if cmd.State == CommandStateInitiated {
		cmd.State = CommandStateDispatched
		cmd.DispatchedAt = now
	}
	if cmd.ReceivedAt == nil {
		cmd.ReceivedAt = &now
	}
	cmd.State = CommandStateSettled
	cmd.SettledAt = &now
	cmd.UpdatedAt = now
	if ackByUserID != "" {
		cmd.AckByUserID = ackByUserID
	}
	if ackByRole != "" {
		cmd.AckByRole = ackByRole
	}
	if traceID != "" {
		cmd.TraceID = traceID
	}
	if err := r.upsert(ctx, cmd); err != nil {
		return CommandRecord{}, err
	}
	return cmd, nil
}

func (r *CommandRegistry) Get(ctx context.Context, commandID string) (CommandRecord, error) {
	if commandID == "" {
		return CommandRecord{}, ErrCommandNotFound
	}
	if cmd, ok := r.getLocal(commandID); ok {
		return cmd, nil
	}
	if r.redis == nil {
		return CommandRecord{}, ErrCommandNotFound
	}
	val, err := r.redis.Get(ctx, r.redisKey(commandID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return CommandRecord{}, ErrCommandNotFound
		}
		slog.Warn("ws command registry redis read failed; using local fallback",
			"command_id", commandID,
			"error", err,
		)
		if cmd, ok := r.getLocal(commandID); ok {
			return cmd, nil
		}
		return CommandRecord{}, ErrCommandNotFound
	}
	var cmd CommandRecord
	if err := json.Unmarshal(val, &cmd); err != nil {
		return CommandRecord{}, fmt.Errorf("unmarshal command %s: %w", commandID, err)
	}
	r.setLocal(cmd)
	return cmd, nil
}

func (r *CommandRegistry) upsert(ctx context.Context, cmd CommandRecord) error {
	if cmd.CommandID == "" {
		return errors.New("command_id is required")
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command %s: %w", cmd.CommandID, err)
	}
	if r.redis != nil {
		if err := r.redis.Set(ctx, r.redisKey(cmd.CommandID), data, r.ttl).Err(); err != nil {
			slog.Warn("ws command registry redis write failed; keeping local fallback",
				"command_id", cmd.CommandID,
				"error", err,
			)
		}
	}
	r.setLocal(cmd)
	return nil
}

func (r *CommandRegistry) redisKey(commandID string) string {
	return cache.PrefixWSCommandRegistry + commandID
}

func (r *CommandRegistry) getLocal(commandID string) (CommandRecord, bool) {
	r.mu.RLock()
	entry, ok := r.local[commandID]
	r.mu.RUnlock()
	if !ok {
		return CommandRecord{}, false
	}
	if !entry.ExpiresAt.IsZero() && r.now().After(entry.ExpiresAt) {
		r.mu.Lock()
		delete(r.local, commandID)
		r.mu.Unlock()
		return CommandRecord{}, false
	}
	return entry.Record, true
}

func (r *CommandRegistry) setLocal(cmd CommandRecord) {
	r.mu.Lock()
	r.local[cmd.CommandID] = commandEntry{Record: cmd, ExpiresAt: r.now().Add(r.ttl)}
	r.mu.Unlock()
}

func clonePayload(payload map[string]interface{}) map[string]interface{} {
	if len(payload) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(payload))
	for k, v := range payload {
		out[k] = v
	}
	return out
}
