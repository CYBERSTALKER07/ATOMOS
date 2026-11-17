package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
	"github.com/pegasusx/pegasusx/apps/backend-go/workerlease"
)

// OntologyConsumerGroupID is the Kafka consumer group for the real-time M0.5 ontology projector.
const OntologyConsumerGroupID = "void-ontology-projector"

func setupOntologyService(spannerClient *spanner.Client, cfg *Config, log *slog.Logger) *ontology.Service {
	var repo ontology.Repository
	if spannerClient != nil {
		repo = ontology.NewSpannerRepository(spannerClient)
	} else if cfg != nil && cfg.allowsRepoMemoryFallback() {
		repo = ontology.NewMemoryRepository()
		if log != nil {
			log.Warn("ontology repository fallback enabled", "backend", "in-memory")
		}
	}
	if repo == nil {
		return nil
	}
	svc := ontology.NewService(repo, log)
	cellID := strings.TrimSpace(os.Getenv("CELL_ID"))
	if cellID == "" {
		cellID = auth.DefaultHomeCellFromEnv()
	}
	svc.SetDefaultCell(cellID)
	return svc
}

// OntologyConsumerTopics returns the domain Kafka topics consumed by the ontology projector.
// Intentionally excludes TopicIntel to prevent self-consumption of EventOntologyObjectChanged.
func OntologyConsumerTopics() []string {
	if !events.ConsumeDomainTopics() {
		return []string{events.TopicMain}
	}
	candidates := []string{
		events.TopicMain,
		events.TopicOrders,
		events.TopicDispatch,
		events.TopicExceptions,
		events.TopicLedger,
		events.TopicConnectors,
	}
	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, topic := range candidates {
		t := strings.TrimSpace(topic)
		if t == "" {
			continue
		}
		if _, exists := seen[t]; exists {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// startOntologyRebuildNightlyWorker launches the lease-gated 24h drift rebuild sweeper.
func startOntologyRebuildNightlyWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	leaseMgr workerlease.Manager,
	holderID string,
	log *slog.Logger,
	ontSvc *ontology.Service,
) {
	if ontSvc == nil || ontSvc.Repository() == nil {
		return
	}
	if _, ok := ontSvc.Repository().(ontology.RebuildSource); !ok {
		return
	}
	startSpawningSweeper(ctx, wg, leaseMgr, "sweeper:ontology_rebuild", holderID, log, func(leaseCtx context.Context, localWg *sync.WaitGroup) {
		localWg.Add(1)
		go func() {
			defer localWg.Done()
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-leaseCtx.Done():
					return
				case <-ticker.C:
					res, err := ontSvc.RebuildAllFromSpanner(leaseCtx)
					if err != nil {
						if log != nil {
							log.Warn("nightly ontology rebuild failed", "err", err)
						}
						continue
					}
					if log != nil {
						log.Info("nightly ontology rebuild completed",
							"objects_upserted", res.ObjectsUpserted,
							"edges_upserted", res.EdgesUpserted,
							"batches", res.BatchesCommitted,
							"duration_ms", res.DurationMs,
						)
					}
				}
			}
		}()
	})
}
