package twin

import (
	"context"
)

type Repository interface {
	GetRouteTwin(ctx context.Context, routeID string) (*RouteTwinView, error)
	ListActiveRouteTwins(ctx context.Context, zoneH3 string) ([]RouteTwinView, error)
	GetVehicleInventory(ctx context.Context, routeID string) ([]VehicleInventory, error)
	GetStopTwin(ctx context.Context, routeID, stopID string) (*StopTwin, error)

	SaveRouteTwin(ctx context.Context, rt RouteTwin) error
	SaveStopTwin(ctx context.Context, st StopTwin) error
	SaveVehicleInventory(ctx context.Context, inv VehicleInventory) error

	// Rebuild is meant for the recovery worker/endpoint.
	RebuildRouteTwin(ctx context.Context, routeID string, view RouteTwinView) error
}
