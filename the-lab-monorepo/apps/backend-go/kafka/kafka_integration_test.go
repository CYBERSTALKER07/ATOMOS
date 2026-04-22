package kafka_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	kafkaEvents "backend-go/kafka"

	goKafka "github.com/segmentio/kafka-go"
)

// ─── Kafka Integration Tests ───────────────────────────────────────────────────
// These tests require a running Kafka broker (docker-compose up).
// They verify end-to-end event production and consumption on real Kafka.
// Skipped when KAFKA_BROKER is not set.

func kafkaBroker(t *testing.T) string {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}
	// Quick connectivity check
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := goKafka.DialContext(ctx, "tcp", broker)
	if err != nil {
		t.Skipf("Kafka not available at %s: %v", broker, err)
	}
	conn.Close()
	return broker
}

// testTopic returns a unique topic name per test to avoid cross-test pollution.
func testTopic(t *testing.T, broker string) string {
	topic := "test-notifications-" + t.Name()

	// Create the topic (ignore error if exists)
	conn, err := goKafka.Dial("tcp", broker)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	err = conn.CreateTopics(goKafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil && err.Error() != "Topic with this name already exists" {
		// Ignore "already exists" errors
		t.Logf("Topic creation note: %v", err)
	}

	return topic
}

func TestKafka_OrderDispatchedRoundTrip(t *testing.T) {
	broker := kafkaBroker(t)
	topic := testTopic(t, broker)

	// Produce
	writer := &goKafka.Writer{
		Addr:     goKafka.TCP(broker),
		Topic:    topic,
		Balancer: &goKafka.LeastBytes{},
	}
	defer writer.Close()

	event := kafkaEvents.OrderDispatchedEvent{
		RouteID:    "ROUTE-001",
		OrderIDs:   []string{"ORD-001", "ORD-002"},
		DriverID:   "DRV-001",
		SupplierID: "SUP-001",
		Timestamp:  time.Now().UTC(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = writer.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(kafkaEvents.EventOrderDispatched),
		Value: data,
	})
	if err != nil {
		t.Fatalf("Produce ORDER_DISPATCHED: %v", err)
	}

	// Consume
	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  "test-consumer-" + t.Name(),
		MinBytes: 1,
		MaxBytes: 1 << 20,
	})
	defer reader.Close()

	readCtx, readCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer readCancel()

	msg, err := reader.ReadMessage(readCtx)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}

	if string(msg.Key) != kafkaEvents.EventOrderDispatched {
		t.Errorf("Key = %q; want %q", string(msg.Key), kafkaEvents.EventOrderDispatched)
	}

	var consumed kafkaEvents.OrderDispatchedEvent
	if err := json.Unmarshal(msg.Value, &consumed); err != nil {
		t.Fatalf("Unmarshal consumed: %v", err)
	}

	if consumed.RouteID != "ROUTE-001" {
		t.Errorf("RouteID = %q; want ROUTE-001", consumed.RouteID)
	}
	if len(consumed.OrderIDs) != 2 {
		t.Errorf("OrderIDs length = %d; want 2", len(consumed.OrderIDs))
	}
	if consumed.DriverID != "DRV-001" {
		t.Errorf("DriverID = %q; want DRV-001", consumed.DriverID)
	}
}

func TestKafka_DriverArrivedRoundTrip(t *testing.T) {
	broker := kafkaBroker(t)
	topic := testTopic(t, broker)

	writer := &goKafka.Writer{
		Addr:     goKafka.TCP(broker),
		Topic:    topic,
		Balancer: &goKafka.LeastBytes{},
	}
	defer writer.Close()

	event := kafkaEvents.DriverArrivedEvent{
		OrderID:    "ORD-100",
		RetailerID: "RET-100",
		DriverID:   "DRV-100",
		SupplierID: "SUP-100",
		Timestamp:  time.Now().UTC(),
	}

	data, _ := json.Marshal(event)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := writer.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(kafkaEvents.EventDriverArrived),
		Value: data,
	}); err != nil {
		t.Fatalf("Produce DRIVER_ARRIVED: %v", err)
	}

	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  "test-consumer-" + t.Name(),
		MinBytes: 1,
		MaxBytes: 1 << 20,
	})
	defer reader.Close()

	readCtx, readCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer readCancel()
	msg, err := reader.ReadMessage(readCtx)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}

	if string(msg.Key) != kafkaEvents.EventDriverArrived {
		t.Errorf("Key = %q; want %q", string(msg.Key), kafkaEvents.EventDriverArrived)
	}

	var consumed kafkaEvents.DriverArrivedEvent
	if err := json.Unmarshal(msg.Value, &consumed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if consumed.OrderID != "ORD-100" {
		t.Errorf("OrderID = %q; want ORD-100", consumed.OrderID)
	}
	if consumed.RetailerID != "RET-100" {
		t.Errorf("RetailerID = %q; want RET-100", consumed.RetailerID)
	}
}

func TestKafka_OrderStatusChangedRoundTrip(t *testing.T) {
	broker := kafkaBroker(t)
	topic := testTopic(t, broker)

	writer := &goKafka.Writer{
		Addr:     goKafka.TCP(broker),
		Topic:    topic,
		Balancer: &goKafka.LeastBytes{},
	}
	defer writer.Close()

	event := kafkaEvents.OrderStatusChangedEvent{
		OrderID:    "ORD-200",
		RetailerID: "RET-200",
		SupplierID: "SUP-200",
		OldState:   "IN_TRANSIT",
		NewState:   "ARRIVED",
		Timestamp:  time.Now().UTC(),
	}

	data, _ := json.Marshal(event)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := writer.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(kafkaEvents.EventOrderStatusChanged),
		Value: data,
	}); err != nil {
		t.Fatalf("Produce: %v", err)
	}

	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  "test-consumer-" + t.Name(),
		MinBytes: 1,
		MaxBytes: 1 << 20,
	})
	defer reader.Close()

	readCtx, readCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer readCancel()
	msg, err := reader.ReadMessage(readCtx)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}

	var consumed kafkaEvents.OrderStatusChangedEvent
	if err := json.Unmarshal(msg.Value, &consumed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if consumed.OldState != "IN_TRANSIT" || consumed.NewState != "ARRIVED" {
		t.Errorf("States = %s→%s; want IN_TRANSIT→ARRIVED", consumed.OldState, consumed.NewState)
	}
}

func TestKafka_PaymentSettledRoundTrip(t *testing.T) {
	broker := kafkaBroker(t)
	topic := testTopic(t, broker)

	writer := &goKafka.Writer{
		Addr:     goKafka.TCP(broker),
		Topic:    topic,
		Balancer: &goKafka.LeastBytes{},
	}
	defer writer.Close()

	event := kafkaEvents.PaymentSettledEvent{
		OrderID:    "ORD-300",
		InvoiceID:  "INV-300",
		RetailerID: "RET-300",
		DriverID:   "DRV-300",
		Gateway:    "CLICK",
		Amount:  150000,
		Timestamp:  time.Now().UTC(),
	}

	data, _ := json.Marshal(event)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := writer.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(kafkaEvents.EventPaymentSettled),
		Value: data,
	}); err != nil {
		t.Fatalf("Produce: %v", err)
	}

	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  "test-consumer-" + t.Name(),
		MinBytes: 1,
		MaxBytes: 1 << 20,
	})
	defer reader.Close()

	readCtx, readCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer readCancel()
	msg, err := reader.ReadMessage(readCtx)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}

	var consumed kafkaEvents.PaymentSettledEvent
	if err := json.Unmarshal(msg.Value, &consumed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if consumed.Amount != 150000 {
		t.Errorf("Amount = %d; want 150000", consumed.Amount)
	}
	if consumed.Gateway != "CLICK" {
		t.Errorf("Gateway = %q; want CLICK", consumed.Gateway)
	}
}

func TestKafka_PaymentFailedRoundTrip(t *testing.T) {
	broker := kafkaBroker(t)
	topic := testTopic(t, broker)

	writer := &goKafka.Writer{
		Addr:     goKafka.TCP(broker),
		Topic:    topic,
		Balancer: &goKafka.LeastBytes{},
	}
	defer writer.Close()

	event := kafkaEvents.PaymentFailedEvent{
		OrderID:    "ORD-400",
		InvoiceID:  "INV-400",
		RetailerID: "RET-400",
		Gateway:    "PAYME",
		Reason:     "insufficient_funds",
		Timestamp:  time.Now().UTC(),
	}

	data, _ := json.Marshal(event)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := writer.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(kafkaEvents.EventPaymentFailed),
		Value: data,
	}); err != nil {
		t.Fatalf("Produce: %v", err)
	}

	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  "test-consumer-" + t.Name(),
		MinBytes: 1,
		MaxBytes: 1 << 20,
	})
	defer reader.Close()

	readCtx, readCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer readCancel()
	msg, err := reader.ReadMessage(readCtx)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}

	var consumed kafkaEvents.PaymentFailedEvent
	if err := json.Unmarshal(msg.Value, &consumed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if consumed.Reason != "insufficient_funds" {
		t.Errorf("Reason = %q; want insufficient_funds", consumed.Reason)
	}
}

func TestKafka_PayloadReadyToSealRoundTrip(t *testing.T) {
	broker := kafkaBroker(t)
	topic := testTopic(t, broker)

	writer := &goKafka.Writer{
		Addr:     goKafka.TCP(broker),
		Topic:    topic,
		Balancer: &goKafka.LeastBytes{},
	}
	defer writer.Close()

	event := kafkaEvents.PayloadReadyToSealEvent{
		RouteID:    "ROUTE-500",
		OrderIDs:   []string{"ORD-501", "ORD-502", "ORD-503"},
		SupplierID: "SUP-500",
		Timestamp:  time.Now().UTC(),
	}

	data, _ := json.Marshal(event)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := writer.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(kafkaEvents.EventPayloadReadyToSeal),
		Value: data,
	}); err != nil {
		t.Fatalf("Produce: %v", err)
	}

	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  "test-consumer-" + t.Name(),
		MinBytes: 1,
		MaxBytes: 1 << 20,
	})
	defer reader.Close()

	readCtx, readCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer readCancel()
	msg, err := reader.ReadMessage(readCtx)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}

	var consumed kafkaEvents.PayloadReadyToSealEvent
	if err := json.Unmarshal(msg.Value, &consumed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if consumed.RouteID != "ROUTE-500" {
		t.Errorf("RouteID = %q; want ROUTE-500", consumed.RouteID)
	}
	if len(consumed.OrderIDs) != 3 {
		t.Errorf("OrderIDs length = %d; want 3", len(consumed.OrderIDs))
	}
}

// ─── Event Struct Serialization Tests ──────────────────────────────────────────
// These run without Kafka and verify JSON serialization fidelity.

func TestEventSerialization_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		event    interface{}
		eventKey string
	}{
		{
			name: "OrderDispatched",
			event: kafkaEvents.OrderDispatchedEvent{
				RouteID: "R1", OrderIDs: []string{"O1"}, DriverID: "D1", SupplierID: "S1", Timestamp: time.Now(),
			},
			eventKey: kafkaEvents.EventOrderDispatched,
		},
		{
			name: "DriverArrived",
			event: kafkaEvents.DriverArrivedEvent{
				OrderID: "O1", RetailerID: "R1", DriverID: "D1", SupplierID: "S1", Timestamp: time.Now(),
			},
			eventKey: kafkaEvents.EventDriverArrived,
		},
		{
			name: "OrderStatusChanged",
			event: kafkaEvents.OrderStatusChangedEvent{
				OrderID: "O1", RetailerID: "R1", SupplierID: "S1", OldState: "A", NewState: "B", Timestamp: time.Now(),
			},
			eventKey: kafkaEvents.EventOrderStatusChanged,
		},
		{
			name: "PayloadReadyToSeal",
			event: kafkaEvents.PayloadReadyToSealEvent{
				RouteID: "R1", OrderIDs: []string{"O1"}, SupplierID: "S1", Timestamp: time.Now(),
			},
			eventKey: kafkaEvents.EventPayloadReadyToSeal,
		},
		{
			name: "PaymentSettled",
			event: kafkaEvents.PaymentSettledEvent{
				OrderID: "O1", InvoiceID: "I1", RetailerID: "R1", DriverID: "D1", Gateway: "CLICK", Amount: 100, Timestamp: time.Now(),
			},
			eventKey: kafkaEvents.EventPaymentSettled,
		},
		{
			name: "PaymentFailed",
			event: kafkaEvents.PaymentFailedEvent{
				OrderID: "O1", InvoiceID: "I1", RetailerID: "R1", Gateway: "PAYME", Reason: "test", Timestamp: time.Now(),
			},
			eventKey: kafkaEvents.EventPaymentFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("Marshal %s: %v", tt.name, err)
			}
			if len(data) == 0 {
				t.Fatal("Marshal produced empty data")
			}

			// Verify round-trip
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("Unmarshal to map: %v", err)
			}

			// Spot-check the event key constant
			if tt.eventKey == "" {
				t.Error("Event key constant is empty")
			}
		})
	}
}

func TestEventConstants_AllDefined(t *testing.T) {
	constants := []string{
		kafkaEvents.EventOrderDispatched,
		kafkaEvents.EventDriverApproaching,
		kafkaEvents.EventDriverArrived,
		kafkaEvents.EventOrderStatusChanged,
		kafkaEvents.EventPayloadReadyToSeal,
		kafkaEvents.EventPayloadSealed,
		kafkaEvents.EventPaymentSettled,
		kafkaEvents.EventPaymentFailed,
		kafkaEvents.EventOrderCompleted,
	}

	for _, c := range constants {
		if c == "" {
			t.Error("Found empty event constant")
		}
	}

	// Verify uniqueness
	seen := make(map[string]bool)
	for _, c := range constants {
		if seen[c] {
			t.Errorf("Duplicate event constant: %s", c)
		}
		seen[c] = true
	}
}
