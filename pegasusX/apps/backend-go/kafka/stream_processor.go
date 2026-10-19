package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	streamProcessorOrdersTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "kafka",
		Name:      "stream_processor_orders_total",
		Help:      "Total aggregated orders processed by stream processor",
	})
	streamProcessorRevenueTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "kafka",
		Name:      "stream_processor_revenue_minor_total",
		Help:      "Total aggregated revenue in minor units processed by stream processor",
	})
	streamProcessorCancelsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "kafka",
		Name:      "stream_processor_cancellations_total",
		Help:      "Total aggregated cancellations processed by stream processor",
	})
)

// StreamProcessor defines an interface for continuous stream analytics and transformations.
type StreamProcessor interface {
	Start(ctx context.Context, eventStream <-chan []byte) error
	Stop() error
}

// AnalyticsStreamProcessor aggregates system events in real-time.
// This is useful for dashboard metrics, fraud detection, or materializing views.
type AnalyticsStreamProcessor struct {
	mu           sync.RWMutex
	orderCount   int64
	revenueMinor int64
	cancelCount  int64

	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

// NewAnalyticsStreamProcessor creates a new stream processor.
func NewAnalyticsStreamProcessor() *AnalyticsStreamProcessor {
	return &AnalyticsStreamProcessor{}
}

// Start begins processing the incoming event stream in a separate goroutine.
func (s *AnalyticsStreamProcessor) Start(ctx context.Context, eventStream <-chan []byte) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// Emit windowed analytics every minute
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.InfoContext(ctx, "AnalyticsStreamProcessor stopping")
				return

			case <-ticker.C:
				s.flushMetrics(ctx)

			case payload, ok := <-eventStream:
				if !ok {
					return
				}
				s.processMessage(ctx, payload)
			}
		}
	}()

	slog.InfoContext(ctx, "AnalyticsStreamProcessor started")
	return nil
}

// Stop gracefully stops the stream processor.
func (s *AnalyticsStreamProcessor) Stop() error {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.wg.Wait()
	return nil
}

// processMessage dynamically determines the event type and applies the correct stream transformation.
func (s *AnalyticsStreamProcessor) processMessage(ctx context.Context, payload []byte) {
	// First extract the base event to determine routing
	var base events.BaseEvent
	if err := json.Unmarshal(payload, &base); err != nil {
		slog.ErrorContext(ctx, "stream_processor: failed to unmarshal base event", "error", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch base.Type {
	case events.EventOrderCreated:
		s.orderCount++
	case events.EventOrderStatusChanged:
		var orderEvt events.OrderEvent
		if err := json.Unmarshal(payload, &orderEvt); err == nil {
			if orderEvt.Status == "CANCELLED" {
				s.cancelCount++
			}
		}
	case events.EventPaymentCleared:
		var fin events.FinanceEvent
		if err := json.Unmarshal(payload, &fin); err == nil {
			s.revenueMinor += fin.AmountMinor
		}
	}
}

// flushMetrics logs or persists the aggregated metrics, then resets the window.
func (s *AnalyticsStreamProcessor) flushMetrics(ctx context.Context) {
	s.mu.Lock()
	orders := s.orderCount
	revenue := s.revenueMinor
	cancels := s.cancelCount

	// Reset the tumbling window
	s.orderCount = 0
	s.revenueMinor = 0
	s.cancelCount = 0
	s.mu.Unlock()

	// In a real enterprise system, you would flush this to Datadog metrics, a Time-Series DB, or Spanner.
	if orders > 0 {
		streamProcessorOrdersTotal.Add(float64(orders))
	}
	if revenue > 0 {
		streamProcessorRevenueTotal.Add(float64(revenue))
	}
	if cancels > 0 {
		streamProcessorCancelsTotal.Add(float64(cancels))
	}

	slog.InfoContext(ctx, "Stream Window Aggregation Complete",
		"new_orders", orders,
		"revenue_minor", revenue,
		"cancellations", cancels,
	)
}
