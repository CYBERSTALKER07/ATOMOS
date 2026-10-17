package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/kafka/workerpool"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/segmentio/kafka-go"
)

// Envelope is the universal Kafka JSON wrapper for pegasusX state events.
type Envelope struct {
	TraceID   string `json:"trace_id"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Version   int64  `json:"v"`
}

// ParseEnvelope decodes the event type envelope. Malformed JSON returns an error
// so the consumer can route poison pills to the DLQ after retries.
func ParseEnvelope(value []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(value, &env); err != nil {
		return Envelope{}, fmt.Errorf("parse kafka envelope: %w", err)
	}
	return env, nil
}

// EnvelopeFromMessage decodes the event envelope, falling back to message headers
// when fields like type or trace_id are carried on the Kafka transport layer.
func EnvelopeFromMessage(msg kafka.Message) (Envelope, error) {
	env, err := ParseEnvelope(msg.Value)
	if err != nil {
		return env, err
	}
	if strings.TrimSpace(env.Type) == "" {
		for _, key := range []string{"type", "event_type", "eventType", "aggregate_type"} {
			if val := workerpool.HeaderValue(msg.Headers, key); val != "" {
				env.Type = strings.TrimSpace(val)
				break
			}
		}
	}
	if strings.TrimSpace(env.TraceID) == "" {
		env.TraceID = TraceIDFromMessage(msg)
	}
	if strings.TrimSpace(env.Timestamp) == "" {
		if ts := workerpool.HeaderValue(msg.Headers, "timestamp"); ts != "" {
			env.Timestamp = strings.TrimSpace(ts)
		} else if !msg.Time.IsZero() {
			env.Timestamp = msg.Time.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
		}
	}
	return env, nil
}


// ContextFromMessage attaches trace_id from headers/body to ctx.
func ContextFromMessage(parent context.Context, msg kafka.Message) context.Context {
	return workerpool.ContextWithTrace(parent, msg)
}

// TraceIDFromMessage returns the resolved trace id without mutating context.
func TraceIDFromMessage(msg kafka.Message) string {
	if id := workerpool.HeaderValue(msg.Headers, "trace_id"); id != "" {
		return id
	}
	return workerpool.TraceIDFromPayload(msg.Value)
}

// WithTraceFromMessage is a convenience for handlers that only need trace ctx.
func WithTraceFromMessage(parent context.Context, msg kafka.Message) context.Context {
	traceID := TraceIDFromMessage(msg)
	if traceID == "" {
		return parent
	}
	return outbox.WithTraceID(parent, traceID)
}
