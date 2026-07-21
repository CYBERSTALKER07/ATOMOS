package kafka

import (
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/segmentio/kafka-go"
)

// NewMultiTopicConsumer builds a consumer group reader across multiple topics.
// When Topics is empty, falls back to single Topic from deps.
func NewMultiTopicConsumer(deps ConsumerDeps) *Consumer {
	topics := normalizeConsumerTopics(deps.Topics)
	if len(topics) == 0 && strings.TrimSpace(deps.Topic) != "" {
		topics = []string{strings.TrimSpace(deps.Topic)}
	}
	if len(topics) == 0 {
		return NewConsumer(deps)
	}
	if deps.MaxAttempts <= 0 {
		deps.MaxAttempts = 3
	}
	dialer, err := kafkautil.Dialer(deps.Auth)
	if err != nil {
		dialer = &kafka.Dialer{Timeout: 15 * time.Second, DualStack: true}
	}
	cfg := kafka.ReaderConfig{
		Brokers:        deps.Brokers,
		GroupID:        deps.GroupID,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		Dialer:         dialer,
	}
	if len(topics) == 1 {
		cfg.Topic = topics[0]
		deps.Topic = topics[0]
	} else {
		cfg.GroupTopics = topics
		deps.Topic = strings.Join(topics, ",")
	}
	return &Consumer{
		reader: kafka.NewReader(cfg),
		deps:   deps,
	}
}

func normalizeConsumerTopics(topics []string) []string {
	if len(topics) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(topics))
	out := make([]string, 0, len(topics))
	for _, t := range topics {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
