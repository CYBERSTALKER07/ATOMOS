package telemetry

import (
	"context"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/api/iterator"
)

// SLOCollector computes the platform SLO metrics declared in
// docs/PLATFORM_SLOS.md and referenced by infra/terraform/observability.tf,
// polling Spanner on an interval and exposing them as Prometheus gauges:
//
//	void_outbox_lag_seconds                 p99 claim→publish lag (seconds)
//	void_fiscal_success_ratio               fiscal submit success ratio over the window
//	void_capture_success_ratio              payment capture success ratio over the window
//	void_outbox_dlq_depth                   OutboxDeadLetters row count (Kafka/Spanner DLQ)
//	void_partner_webhook_success_ratio      webhook 2xx/SUCCESS ratio over 1h
//
// Relay restart rate uses void_outbox_relay_restarts_total (outbox package counter).
// These are the metrics the burn/latency alert policies filter on; without this
// collector the alert policies have no data source.
type SLOCollector struct {
	client *spanner.Client
	log    *slog.Logger

	outboxLag      prometheus.Gauge
	fiscalSuccess  prometheus.Gauge
	captureSucc    prometheus.Gauge
	dlqDepth       prometheus.Gauge
	webhookSuccess prometheus.Gauge
}

// NewSLOCollector registers the gauges and returns the collector. Call Start to
// begin polling.
func NewSLOCollector(client *spanner.Client, reg *prometheus.Registry, log *slog.Logger) *SLOCollector {
	if log == nil {
		log = slog.Default()
	}
	c := &SLOCollector{
		client: client,
		log:    log,
		outboxLag: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "void_outbox_lag_seconds",
			Help: "Outbox claim→publish lag p99 in seconds (SLO < 30s).",
		}),
		fiscalSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "void_fiscal_success_ratio",
			Help: "Fiscal submit success ratio over the trailing hour (SLO >= 0.99).",
		}),
		captureSucc: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "void_capture_success_ratio",
			Help: "Payment capture success ratio over the trailing hour (SLO >= 0.99).",
		}),
		dlqDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "void_outbox_dlq_depth",
			Help: "OutboxDeadLetters depth (SLO: = 0 sustained 5m).",
		}),
		webhookSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "void_partner_webhook_success_ratio",
			Help: "Partner webhook delivery SUCCESS ratio over 1h (SLO >= 0.99).",
		}),
	}
	if reg == nil {
		reg = prometheus.DefaultRegisterer.(*prometheus.Registry)
	}
	reg.MustRegister(c.outboxLag, c.fiscalSuccess, c.captureSucc, c.dlqDepth, c.webhookSuccess)
	return c
}

// Start polls every interval until ctx is done. Safe to call in a goroutine.
func (c *SLOCollector) Start(ctx context.Context, interval time.Duration) {
	if c == nil || c.client == nil {
		return
	}
	if interval <= 0 {
		interval = 60 * time.Second
	}
	c.collect(ctx) // prime once so the first scrape has data
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.collect(ctx)
		}
	}
}

func (c *SLOCollector) collect(ctx context.Context) {
	if lag, err := c.outboxLagP99(ctx); err == nil {
		c.outboxLag.Set(lag)
	} else {
		c.log.Warn("slo outbox lag", "err", err)
	}
	if r, err := c.ratio(ctx, `SELECT COUNTIF(Status='SUCCESS'), COUNT(*) FROM OrderFiscalReceipts
		WHERE CreatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)`); err == nil {
		c.fiscalSuccess.Set(r)
	} else {
		c.log.Warn("slo fiscal ratio", "err", err)
	}
	if r, err := c.ratio(ctx, `SELECT COUNTIF(Status='CAPTURED'), COUNT(*) FROM OrderPaymentLegs
		WHERE CreatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)`); err == nil {
		c.captureSucc.Set(r)
	} else {
		c.log.Warn("slo capture ratio", "err", err)
	}
	if n, err := c.count(ctx, `SELECT COUNT(*) FROM OutboxDeadLetters`); err == nil {
		c.dlqDepth.Set(float64(n))
	} else {
		c.log.Warn("slo dlq depth", "err", err)
	}
	if r, err := c.ratio(ctx, `SELECT COUNTIF(Status='SUCCESS'), COUNT(*) FROM WebhookDeliveryAttempts
		WHERE CreatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)`); err == nil {
		c.webhookSuccess.Set(r)
	} else {
		c.log.Warn("slo webhook ratio", "err", err)
	}
}

func (c *SLOCollector) count(ctx context.Context, sql string) (int64, error) {
	iter := c.client.Single().Query(ctx, spanner.Statement{SQL: sql})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var n int64
	if err := row.Columns(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// outboxLagP99 returns the p99 claim→publish lag in seconds over the trailing
// hour. Approximates p99 via the 100 most recent published events.
func (c *SLOCollector) outboxLagP99(ctx context.Context) (float64, error) {
	iter := c.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT TIMESTAMP_DIFF(PublishedAt, CreatedAt, MILLISECOND) AS lag_ms
			FROM OutboxEvents
			WHERE PublishedAt IS NOT NULL
			  AND PublishedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)
			ORDER BY lag_ms DESC
			LIMIT 100`,
	})
	defer iter.Stop()
	var lags []float64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		var ms int64
		if err := row.Columns(&ms); err != nil {
			return 0, err
		}
		lags = append(lags, float64(ms)/1000.0)
	}
	if len(lags) == 0 {
		return 0, nil
	}
	// p99 of the worst-100 sample.
	idx := int(float64(len(lags)) * 0.99)
	if idx >= len(lags) {
		idx = len(lags) - 1
	}
	return lags[idx], nil
}

// ratio returns numerator/denominator, or 1.0 when there is no traffic in the
// window (no traffic is not a breach).
func (c *SLOCollector) ratio(ctx context.Context, sql string) (float64, error) {
	iter := c.client.Single().Query(ctx, spanner.Statement{SQL: sql})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 1.0, err
	}
	var num, den int64
	if err := row.Columns(&num, &den); err != nil {
		return 1.0, err
	}
	if den == 0 {
		return 1.0, nil
	}
	return float64(num) / float64(den), nil
}
