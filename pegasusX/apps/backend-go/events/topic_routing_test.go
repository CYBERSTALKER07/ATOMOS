package events

import (
	"os"
	"testing"
)

func TestDomainTopicForEventType(t *testing.T) {
	t.Parallel()
	if got := DomainTopicForEventType(EventOrderCreated); got != TopicOrders {
		t.Fatalf("ORDER_CREATED -> %q, want %q", got, TopicOrders)
	}
	if got := DomainTopicForEventType(EventManifestSealed); got != TopicDispatch {
		t.Fatalf("MANIFEST_SEALED -> %q, want %q", got, TopicDispatch)
	}
	if got := DomainTopicForEventType(EventDriverLocationUpdated); got != TopicRealtime {
		t.Fatalf("DRIVER_LOCATION -> %q, want %q", got, TopicRealtime)
	}
	if got := DomainTopicForEventType(EventSupplierCreated); got != "" {
		t.Fatalf("SUPPLIER_CREATED -> %q, want empty", got)
	}
	if got := DomainTopicForEventType(EventDemandSignal); got != TopicDemand {
		t.Fatalf("DEMAND_SIGNAL -> %q, want %q", got, TopicDemand)
	}
}

func TestRelayPublishTopics_demandSignalDualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	payload := []byte(`{"type":"DEMAND_SIGNAL","retailer_id":"r1","sku":"SKU-1","day":"2026-08-02","source":"STORE_POS"}`)
	topics := RelayPublishTopics(TopicMain, payload)
	if len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicDemand {
		t.Fatalf("topics=%v want [%s %s]", topics, TopicMain, TopicDemand)
	}
}

func TestRelayPublishTopics_dualWrite(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	payload := []byte(`{"type":"ORDER_CREATED","order_id":"o1"}`)
	topics := RelayPublishTopics(TopicMain, payload)
	if len(topics) != 2 || topics[0] != TopicMain || topics[1] != TopicOrders {
		t.Fatalf("topics=%v want [%s %s]", topics, TopicMain, TopicOrders)
	}
}

func TestRelayPublishTopics_disabled(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "false")
	payload := []byte(`{"type":"ORDER_CREATED"}`)
	topics := RelayPublishTopics(TopicMain, payload)
	if len(topics) != 1 || topics[0] != TopicMain {
		t.Fatalf("topics=%v want [%s]", topics, TopicMain)
	}
}

func TestDualWriteDomainTopics_env(t *testing.T) {
	os.Unsetenv("KAFKA_TOPIC_DUAL_WRITE")
	if DualWriteDomainTopics() {
		t.Fatal("expected false by default")
	}
	t.Setenv("KAFKA_TOPIC_DUAL_WRITE", "true")
	if !DualWriteDomainTopics() {
		t.Fatal("expected true")
	}
}

func TestOrderConsumerTopic_cutover(t *testing.T) {
	os.Unsetenv("KAFKA_TOPIC_CONSUME_DOMAIN")
	if OrderConsumerTopic() != TopicMain {
		t.Fatalf("default topic = %q want %q", OrderConsumerTopic(), TopicMain)
	}
	t.Setenv("KAFKA_TOPIC_CONSUME_DOMAIN", "true")
	if OrderConsumerTopic() != TopicOrders {
		t.Fatalf("domain topic = %q want %q", OrderConsumerTopic(), TopicOrders)
	}
}

func TestDispatchConsumerTopic_cutover(t *testing.T) {
	os.Unsetenv("KAFKA_TOPIC_CONSUME_DOMAIN")
	if DispatchConsumerTopic() != TopicMain {
		t.Fatalf("default topic = %q want %q", DispatchConsumerTopic(), TopicMain)
	}
	t.Setenv("KAFKA_TOPIC_CONSUME_DOMAIN", "true")
	if DispatchConsumerTopic() != TopicDispatch {
		t.Fatalf("domain topic = %q want %q", DispatchConsumerTopic(), TopicDispatch)
	}
}

func TestDispatcherConsumerTopics_cutover(t *testing.T) {
	os.Unsetenv("KAFKA_TOPIC_CONSUME_DOMAIN")
	topics := DispatcherConsumerTopics()
	if len(topics) != 1 || topics[0] != TopicMain {
		t.Fatalf("default topics = %v", topics)
	}
	t.Setenv("KAFKA_TOPIC_CONSUME_DOMAIN", "true")
	topics = DispatcherConsumerTopics()
	if len(topics) < 4 {
		t.Fatalf("domain fan-in topics = %v", topics)
	}
}
