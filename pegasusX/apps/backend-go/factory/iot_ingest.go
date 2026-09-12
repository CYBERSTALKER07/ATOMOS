package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	kafka "github.com/segmentio/kafka-go"
)

type IoTTelemetryEvent struct {
	MachineID string `json:"machine_id"`
	FactoryID string `json:"factory_id"`
	Status    string `json:"status"` // RUNNING, JAMMED, IDLE
	Units     int64  `json:"units"`
	Timestamp int64  `json:"timestamp"`
}

type IotIngestor struct {
	svc        *Service
	log        *slog.Logger
	bufferChan chan IoTTelemetryEvent
	wg         sync.WaitGroup
	quit       chan struct{}
}

func NewIotIngestor(svc *Service, log *slog.Logger) *IotIngestor {
	ingestor := &IotIngestor{
		svc:        svc,
		log:        log,
		bufferChan: make(chan IoTTelemetryEvent, 10000), // Enterprise buffer size
		quit:       make(chan struct{}),
	}
	ingestor.startBatchProcessor()
	return ingestor
}

// HandleMessage ingests raw Kafka messages from the pegasusx.iot.telemetry topic.
func (i *IotIngestor) HandleMessage(ctx context.Context, msg kafka.Message) error {
	var event IoTTelemetryEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		i.log.Warn("Failed to unmarshal IoT event", "err", err)
		return nil // drop invalid
	}

	select {
	case i.bufferChan <- event:
		// buffered
	default:
		i.log.Error("IoT ingest buffer full, dropping telemetry ping", "machine_id", event.MachineID)
	}
	return nil
}

func (i *IotIngestor) startBatchProcessor() {
	i.wg.Add(1)
	go func() {
		defer i.wg.Done()
		batch := make(map[string]int64) // aggregated units by machine
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-i.quit:
				i.flushBatch(batch)
				return
			case event := <-i.bufferChan:
				if event.Units > 0 {
					batch[event.MachineID] += event.Units
				}
				if event.Status == "JAMMED" {
					i.broadcastMachineAlert(event)
				}
				if len(batch) >= 1000 {
					i.flushBatch(batch)
					batch = make(map[string]int64)
				}
			case <-ticker.C:
				if len(batch) > 0 {
					i.flushBatch(batch)
					batch = make(map[string]int64)
				}
			}
		}
	}()
}

func (i *IotIngestor) flushBatch(batch map[string]int64) {
	if len(batch) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Hot cache increment in Redis
	if i.svc != nil && i.svc.cache != nil {
		for machineID, units := range batch {
			key := fmt.Sprintf("factory:iot:%s:units", machineID)
			_, _ = i.svc.cache.IncrBy(ctx, key, units)
		}
	}

	// 2. Persistent storage in Spanner FactoryMachineTelemetry to survive Redis restarts
	if i.svc != nil && i.svc.spannerClient != nil {
		now := time.Now().UTC()
		mutations := make([]*spanner.Mutation, 0, len(batch))
		for machineID, units := range batch {
			m := spanner.InsertOrUpdate("FactoryMachineTelemetry",
				[]string{"MachineId", "RecordedAt", "UnitsProduced", "CreatedAt"},
				[]interface{}{machineID, now, units, spanner.CommitTimestamp},
			)
			mutations = append(mutations, m)
		}
		if len(mutations) > 0 {
			if _, err := i.svc.spannerClient.Apply(ctx, mutations); err != nil {
				i.log.Error("failed to persist IoT telemetry batch to Spanner", "count", len(mutations), "err", err)
			}
		}
	}
}

func (i *IotIngestor) broadcastMachineAlert(event IoTTelemetryEvent) {
	if i.svc.factoryHub == nil {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"type":       "MACHINE_JAM",
		"machine_id": event.MachineID,
		"factory_id": event.FactoryID,
		"timestamp":  event.Timestamp,
	})
	// Broadcast to the specific factory's portal/tablets
	i.svc.factoryHub.Broadcast(context.Background(), "factory:"+event.FactoryID, payload)
}

func (i *IotIngestor) Close() {
	close(i.quit)
	i.wg.Wait()
}
