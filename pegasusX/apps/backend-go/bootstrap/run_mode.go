package bootstrap

import "strings"

const (
	RunModeAll    = "all"
	RunModeAPI    = "api"
	RunModeWorker = "worker"
)

// NormalizeRunMode maps PEGASUSX_RUN_MODE to a supported runtime profile.
//   - all: HTTP API + background workers (default, local dev)
//   - api: HTTP + WS only; no outbox relay or Kafka consumers
//   - worker: background workers + health/metrics; no public API routes
func NormalizeRunMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case RunModeAPI:
		return RunModeAPI
	case RunModeWorker:
		return RunModeWorker
	default:
		return RunModeAll
	}
}

func (c *Config) RunsAPI() bool {
	mode := NormalizeRunMode(c.RunMode)
	return mode == RunModeAll || mode == RunModeAPI
}

func (c *Config) RunsWorkers() bool {
	mode := NormalizeRunMode(c.RunMode)
	return mode == RunModeAll || mode == RunModeWorker
}
