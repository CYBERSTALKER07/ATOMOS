// Package simulation provides a stealth background stress harness for the
// Phase 2 dispatch optimiser. It synthesises in-memory workloads (no Spanner
// writes — zero risk of polluting production data) and feeds them through
// plan.OptimizeAndValidate, which increments the same *plan.SourceCounters
// that power the shadow telemetry surface.
//
// Guardrails:
//   - Gated at route-registration time by SIMULATION_ENABLED env flag.
//   - ADMIN role required on every mutation endpoint.
//   - Single-flight: Start returns ErrAlreadyRunning on a second concurrent call.
//   - Per-solve context timeout (default 1500ms) isolates failures from the
//     backing goroutine; a hung optimiser cannot wedge the engine.
//   - Cancel-on-Stop: the engine's context is cancelled; the goroutine exits
//     within one tick.
package simulation

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
"log"

	"time"

	"backend-go/dispatch"
	"backend-go/dispatch/optimizerclient"
	"backend-go/dispatch/plan"
)

// ErrAlreadyRunning is returned by Start when an engine is already active.
var ErrAlreadyRunning = errors.New("simulation: already running")

// ErrNotRunning is returned by Stop when no engine is active.
var ErrNotRunning = errors.New("simulation: not running")

// Config parameterises a single simulation run.
type Config struct {
	Orders       int           // orders per tick
	Drivers      int           // drivers per tick
	RPS          int           // ticks per second (capped at 50)
	SolveTimeout time.Duration // per-solve ctx deadline
}

func (c *Config) normalise() {
	if c.Orders <= 0 {
		c.Orders = 100
	}
	if c.Orders > 5000 {
		c.Orders = 5000
	}
	if c.Drivers <= 0 {
		c.Drivers = 20
	}
	if c.Drivers > 500 {
		c.Drivers = 500
	}
	if c.RPS <= 0 {
		c.RPS = 5
	}
	if c.RPS > 50 {
		c.RPS = 50
	}
	if c.SolveTimeout <= 0 {
		c.SolveTimeout = 1500 * time.Millisecond
	}
}

// Snapshot is the public state exposed to /v1/internal/sim/status.
type Snapshot struct {
	Running           bool      `json:"running"`
	StartedAt         time.Time `json:"started_at,omitempty"`
	ElapsedSec        int64     `json:"elapsed_sec"`
	Config            Config    `json:"config"`
	Ticks             int64     `json:"ticks"`
	Solves            int64     `json:"solves"`
	Errors            int64     `json:"errors"`
	SourceOptimizer   int64     `json:"source_optimizer"`
	SourceFallback1   int64     `json:"source_fallback_phase1"`
	SourceFallbackVal int64     `json:"source_fallback_validation"`
	LastLatencyMS     int64     `json:"last_latency_ms"`
	P50LatencyMS      int64     `json:"p50_latency_ms"`
	P95LatencyMS      int64     `json:"p95_latency_ms"`
	P99LatencyMS      int64     `json:"p99_latency_ms"`
	LastOrphanCount   int64     `json:"last_orphan_count"`
	CumulativeOrphans int64     `json:"cumulative_orphans"`
	CumulativeRoutes  int64     `json:"cumulative_routes"`
}

// Engine is a single-flight background stress harness.
//
// The engine owns its cancel func and all counter fields; routes access it
// through the methods below so no caller can reach the goroutine directly.
type Engine struct {
	optimizer *optimizerclient.Client
	counters  *plan.SourceCounters

	mu      sync.Mutex
	cancel  context.CancelFunc
	cfg     Config
	running bool
	started time.Time

	// atomics for lock-free reads from Status()
	ticks       atomic.Int64
	solves      atomic.Int64
	errs        atomic.Int64
	srcOpt      atomic.Int64
	srcFb1      atomic.Int64
	srcFbV      atomic.Int64
	orphanLast  atomic.Int64
	orphanTotal atomic.Int64
	routeTotal  atomic.Int64
	latencyLast atomic.Int64

	// latencies is a ring of the last 256 solve durations (ms).
	latMu      sync.Mutex
	latencies  [256]int64
	latencyIdx int
	latencyLen int
}

// NewEngine constructs an idle engine bound to the supplied optimiser client
// and shared counter pointer (typically app.OptimizerClient + app.DispatchOptimizer).
func NewEngine(optimizer *optimizerclient.Client, counters *plan.SourceCounters) *Engine {
	return &Engine{optimizer: optimizer, counters: counters}
}

// Start launches the stress goroutine. Returns ErrAlreadyRunning if one is
// already active. The caller's context is irrelevant — the engine owns its
// own lifetime via the internal cancel func installed here.
func (e *Engine) Start(cfg Config) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return ErrAlreadyRunning
	}
	cfg.normalise()
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.cfg = cfg
	e.running = true
	e.started = time.Now().UTC()
	// Reset counters so a fresh run does not inherit stale values.
	e.ticks.Store(0)
	e.solves.Store(0)
	e.errs.Store(0)
	e.srcOpt.Store(0)
	e.srcFb1.Store(0)
	e.srcFbV.Store(0)
	e.orphanLast.Store(0)
	e.orphanTotal.Store(0)
	e.routeTotal.Store(0)
	e.latencyLast.Store(0)
	e.latMu.Lock()
	e.latencyIdx = 0
	e.latencyLen = 0
	e.latMu.Unlock()

	go e.run(ctx, cfg)
	return nil
}

// Stop cancels the goroutine. Returns ErrNotRunning if idle.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return ErrNotRunning
	}
	e.cancel()
	e.running = false
	return nil
}

// Status returns a point-in-time snapshot of engine state.
func (e *Engine) Status() Snapshot {
	e.mu.Lock()
	running := e.running
	started := e.started
	cfg := e.cfg
	e.mu.Unlock()

	var elapsed int64
	if running {
		elapsed = int64(time.Since(started).Seconds())
	}
	p50, p95, p99 := e.percentiles()
	return Snapshot{
		Running:           running,
		StartedAt:         started,
		ElapsedSec:        elapsed,
		Config:            cfg,
		Ticks:             e.ticks.Load(),
		Solves:            e.solves.Load(),
		Errors:            e.errs.Load(),
		SourceOptimizer:   e.srcOpt.Load(),
		SourceFallback1:   e.srcFb1.Load(),
		SourceFallbackVal: e.srcFbV.Load(),
		LastLatencyMS:     e.latencyLast.Load(),
		P50LatencyMS:      p50,
		P95LatencyMS:      p95,
		P99LatencyMS:      p99,
		LastOrphanCount:   e.orphanLast.Load(),
		CumulativeOrphans: e.orphanTotal.Load(),
		CumulativeRoutes:  e.routeTotal.Load(),
	}
}

func (e *Engine) run(ctx context.Context, cfg Config) {
	interval := time.Second / time.Duration(cfg.RPS)
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Deterministic-ish seed — tied to start moment, so two concurrent
	// engines (not allowed, but defensive) would diverge.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.tick(ctx, cfg, rng)
		}
	}
}

func (e *Engine) tick(parent context.Context, cfg Config, rng *rand.Rand) {
	e.ticks.Add(1)
	orders, fleet := synthesiseWorkload(rng, cfg.Orders, cfg.Drivers)
	solveCtx, cancel := context.WithTimeout(parent, cfg.SolveTimeout)
	defer cancel()
	t0 := time.Now()
	res, source, err := plan.OptimizeAndValidate(solveCtx, e.optimizer, plan.Job{
		TraceID:    "sim-" + time.Now().UTC().Format(time.RFC3339Nano),
		SupplierID: "sim-supplier",
		HomeNodeID: "sim-warehouse",
		Orders:     orders,
		Fleet:      fleet,
	})
	elapsed := time.Since(t0)
	e.latencyLast.Store(elapsed.Milliseconds())
	e.recordLatency(elapsed.Milliseconds())
	e.solves.Add(1)
	if err != nil {
		log.Printf("[SIM EVENT] source=%s err=%v msg=%v", source, err, res.Warnings)
	e.errs.Add(1)
		if e.counters != nil {
			e.counters.RecordError()
		}
		return
	}
	switch source {
	case plan.SourceOptimizer:
		e.srcOpt.Add(1)
	case plan.SourceFallbackPhase1:
			e.srcFb1.Add(1)
	case plan.SourceFallbackValidation:
		e.srcFbV.Add(1)
	}
	if e.counters != nil {
		e.counters.Record(source)
	}
	if res != nil {
		e.orphanLast.Store(int64(len(res.Orphans)))
		e.orphanTotal.Add(int64(len(res.Orphans)))
		e.routeTotal.Add(int64(len(res.Routes)))
	}
}

func (e *Engine) recordLatency(ms int64) {
	e.latMu.Lock()
	defer e.latMu.Unlock()
	e.latencies[e.latencyIdx] = ms
	e.latencyIdx = (e.latencyIdx + 1) % len(e.latencies)
	if e.latencyLen < len(e.latencies) {
		e.latencyLen++
	}
}

func (e *Engine) percentiles() (p50, p95, p99 int64) {
	e.latMu.Lock()
	n := e.latencyLen
	if n == 0 {
		e.latMu.Unlock()
		return 0, 0, 0
	}
	buf := make([]int64, n)
	copy(buf, e.latencies[:n])
	e.latMu.Unlock()
	sort.Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
	pick := func(p float64) int64 {
		idx := int(float64(n-1) * p)
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		return buf[idx]
	}
	return pick(0.50), pick(0.95), pick(0.99)
}

// synthesiseWorkload generates an in-memory order + fleet set inside the
// Tashkent metro bounding box. Fleet capacity is sized 1.5× total order
// volume so the workload is solvable but non-trivial.
func synthesiseWorkload(rng *rand.Rand, orderCount, driverCount int) ([]dispatch.DispatchableOrder, []dispatch.AvailableDriver) {
	const (
		latMin, latMax = 41.20, 41.50
		lngMin, lngMax = 69.10, 69.40
	)
	orders := make([]dispatch.DispatchableOrder, orderCount)
	var totalVU float64
	for i := 0; i < orderCount; i++ {
		vu := 0.5 + rng.Float64()*4.0
		totalVU += vu
		orders[i] = dispatch.DispatchableOrder{
			OrderID:      randID(rng, "o-"),
			RetailerID:   randID(rng, "r-"),
			RetailerName: "sim-retailer",
			Amount:       int64(10_000 + rng.Intn(490_000)),
			Lat:          latMin + rng.Float64()*(latMax-latMin),
			Lng:          lngMin + rng.Float64()*(lngMax-lngMin),
			VolumeVU:     vu,
		}
	}
	fleet := make([]dispatch.AvailableDriver, driverCount)
	perTruck := (totalVU * 1.5) / float64(driverCount)
	if perTruck < 2 {
		perTruck = 2
	}
	for i := 0; i < driverCount; i++ {
		fleet[i] = dispatch.AvailableDriver{
			DriverID:     randID(rng, "d-"),
			DriverName:   "sim-driver",
			VehicleID:    randID(rng, "v-"),
			VehicleClass: "sim",
			MaxVolumeVU:  perTruck,
		}
	}
	return orders, fleet
}

func randID(rng *rand.Rand, prefix string) string {
	const alphabet = "0123456789abcdef"
	buf := []byte(prefix)
	for i := 0; i < 12; i++ {
		buf = append(buf, alphabet[rng.Intn(len(alphabet))])
	}
	return string(buf)
}
