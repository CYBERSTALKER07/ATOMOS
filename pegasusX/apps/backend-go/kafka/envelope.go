package kafka

import (
	"context"
	"encoding/json"
	"fmt"

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
