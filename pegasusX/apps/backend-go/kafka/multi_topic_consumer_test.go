package kafka

import "testing"

func TestNormalizeConsumerTopicsDedupes(t *testing.T) {
	t.Parallel()
	got := normalizeConsumerTopics([]string{"pegasusx-main", "pegasusx-orders", "pegasusx-main", ""})
	if len(got) != 2 || got[0] != "pegasusx-main" || got[1] != "pegasusx-orders" {
		t.Fatalf("normalizeConsumerTopics() = %v", got)
	}
}

func TestNewMultiTopicConsumerSingleTopic(t *testing.T) {
	t.Parallel()
	c := NewMultiTopicConsumer(ConsumerDeps{
		Brokers: []string{"localhost:9092"},
		GroupID: "test-group",
		Topics:  []string{"pegasusx-main"},
	})
	if c == nil || c.deps.Topic != "pegasusx-main" {
		t.Fatalf("consumer topic = %q", c.deps.Topic)
	}
}
