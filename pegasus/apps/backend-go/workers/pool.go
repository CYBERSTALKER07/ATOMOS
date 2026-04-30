package workers

import (
	"log"
	"sync"
)

// Pool is a bounded goroutine pool that processes submitted jobs through a
// fixed number of workers. If the job channel is full, Submit drops the job
// and logs a warning (back-pressure: callers are never blocked).
type Pool struct {
	name string
	ch   chan func()
	wg   sync.WaitGroup
}

// NewPool starts n persistent goroutine workers backed by a buffered channel
// of the given capacity. Name is used in log lines.
func NewPool(name string, workers, queueSize int) *Pool {
	p := &Pool{
		name: name,
		ch:   make(chan func(), queueSize),
	}
	p.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go p.worker()
	}
	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for fn := range p.ch {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[WORKER_POOL:%s] panic recovered: %v", p.name, r)
				}
			}()
			fn()
		}()
	}
}

// Submit enqueues a job. Non-blocking: if the queue is full the job is dropped
// and a warning is logged. This prevents request-serving goroutines from ever
// blocking on background work.
func (p *Pool) Submit(fn func()) {
	select {
	case p.ch <- fn:
		// queued
	default:
		log.Printf("[WORKER_POOL:%s] queue full (cap=%d) — job dropped", p.name, cap(p.ch))
	}
}

// Stop drains the channel and waits for all workers to finish.
// Call during graceful shutdown.
func (p *Pool) Stop() {
	close(p.ch)
	p.wg.Wait()
}

// ── Global Pools ────────────────────────────────────────────────────────────

// EventPool handles fire-and-forget background work: audit writes, WebSocket
// broadcasts, Kafka publishes, and notification fan-outs.
// 64 workers × 4096 queue = sustains ~4k concurrent background ops.
var EventPool = NewPool("event", 64, 4096)

// ETAPool is a smaller, dedicated pool for heavy ETA computation jobs that
// call the Google Maps API (up to 30s per call). Keeping this separate
// prevents long-running ETA jobs from starving fast event deliveries.
var ETAPool = NewPool("eta", 8, 128)
