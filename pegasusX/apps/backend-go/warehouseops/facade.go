package warehouseops

import (
	"cloud.google.com/go/spanner"

	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
)

// Service is the ops-facing warehouse application service.
// HTTP handlers remain on warehouse.Service; this alias is the stable
// dependency for ops-only consumers (dashboards, dispatch, inbound).
type Service = warehouse.Service

// Repository is the warehouse persistence contract used by ops flows.
type Repository = warehouse.Repository

// ServiceConfig configures warehouse ops services.
type ServiceConfig = warehouse.ServiceConfig

// NewService constructs a warehouse ops service.
func NewService(cfg ServiceConfig) *Service {
	return warehouse.NewService(cfg)
}

// NewSpannerRepository constructs the Spanner-backed warehouse repository.
func NewSpannerRepository(client *spanner.Client) Repository {
	return warehouse.NewSpannerRepository(client)
}

// NewInMemoryRepository constructs the local/SSMR in-memory warehouse repository.
// Production bootstrap refuses this path unless ALLOW_MEMORY_FALLBACK is set.
func NewInMemoryRepository() Repository {
	return warehouse.NewInMemoryRepository()
}
