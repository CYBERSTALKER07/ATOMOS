package returns

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

// ServiceConfig wires the returns gate service.
type ServiceConfig struct {
	Spanner      *spanner.Client
	Cache        *cache.Cache
	PayloadHub   *ws.Hub
	WarehouseHub *ws.Hub
	SupplierHub  *ws.Hub
	Log          *slog.Logger
	Now          func() time.Time
}

// Service handles inbound return scanning and physical disposition.
type Service struct {
	spanner      *spanner.Client
	cache        *cache.Cache
	payloadHub   *ws.Hub
	warehouseHub *ws.Hub
	supplierHub  *ws.Hub
	log          *slog.Logger
	now          func() time.Time

	mu               sync.Mutex
	approachDedup    map[string]time.Time
	approachDedupTTL time.Duration
}

// NewService constructs a returns gate service.
func NewService(cfg ServiceConfig) *Service {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	log := cfg.Log
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		spanner:          cfg.Spanner,
		cache:            cfg.Cache,
		payloadHub:       cfg.PayloadHub,
		warehouseHub:     cfg.WarehouseHub,
		supplierHub:      cfg.SupplierHub,
		log:              log,
		now:              now,
		approachDedup:    make(map[string]time.Time),
		approachDedupTTL: 5 * time.Minute,
	}
}

func (s *Service) newID() string {
	return uuid.NewString()
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
