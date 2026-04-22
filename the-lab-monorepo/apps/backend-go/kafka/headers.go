package kafka

import goKafka "github.com/segmentio/kafka-go"

// HeaderValue returns the value of the first header matching key, or empty
// string if not found. Used by all consumers to extract event_type and
// trace_id from outbox-relayed messages.
func HeaderValue(headers []goKafka.Header, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

// EventType returns the message discriminator from outbox headers, falling
// back to the legacy key-based discriminator for pre-outbox producers.
func EventType(headers []goKafka.Header, key []byte) string {
	eventType := HeaderValue(headers, "event_type")
	if eventType != "" {
		return eventType
	}
	return string(key)
}

// TraceID returns the request trace correlation token from outbox headers.
func TraceID(headers []goKafka.Header) string {
	return HeaderValue(headers, "trace_id")
}
