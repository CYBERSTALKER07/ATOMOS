package twin

import (
	"context"
	"log/slog"
	"time"
)

type Service struct {
	repo Repository
	log  *slog.Logger
	now  func() time.Time
}

type ServiceConfig struct {
	Repo Repository
	Log  *slog.Logger
	Now  func() time.Time
}

func NewService(cfg ServiceConfig) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	return &Service{
		repo: cfg.Repo,
		log:  cfg.Log,
		now:  cfg.Now,
	}
}

func (s *Service) GetRouteTwin(ctx context.Context, routeID string) (*RouteTwinView, error) {
	return s.repo.GetRouteTwin(ctx, routeID)
}

func (s *Service) ListActiveRouteTwins(ctx context.Context, zoneH3 string) ([]RouteTwinView, error) {
	return s.repo.ListActiveRouteTwins(ctx, zoneH3)
}

func (s *Service) GetVehicleInventory(ctx context.Context, routeID string) ([]VehicleInventory, error) {
	return s.repo.GetVehicleInventory(ctx, routeID)
}

func (s *Service) GetStopTwin(ctx context.Context, routeID, stopID string) (*StopTwin, error) {
	return s.repo.GetStopTwin(ctx, routeID, stopID)
}

// Write API triggered by the outbox consumer

func (s *Service) HandleRouteStarted(ctx context.Context, routeID, driverID string, stopsCount int64) error {
	rt := RouteTwin{
		RouteID:        routeID,
		DriverID:       driverID,
		Status:         "ACTIVE",
		RemainingStops: stopsCount,
		LastEventAt:    s.now(),
		UpdatedAt:      s.now(),
	}
	return s.repo.SaveRouteTwin(ctx, rt)
}

func (s *Service) HandleLocationUpdate(ctx context.Context, routeID string, lat, lng float64, h3 string) error {
	rt := RouteTwin{
		RouteID:     routeID,
		CurrentLat:  lat,
		CurrentLng:  lng,
		CurrentH3:   h3,
		LocationAt:  s.now(),
		LastEventAt: s.now(),
		UpdatedAt:   s.now(),
	}
	return s.repo.SaveRouteTwin(ctx, rt)
}

func (s *Service) HandleStopStatusChanged(ctx context.Context, routeID, stopID, status string) error {
	st := StopTwin{
		RouteID:   routeID,
		StopID:    stopID,
		Status:    status,
		UpdatedAt: s.now(),
	}
	return s.repo.SaveStopTwin(ctx, st)
}

type StopETAUpdate struct {
	StopID           string
	PredictedArrival *time.Time
	WindowStart      *time.Time
	WindowEnd        *time.Time
}

func (s *Service) HandleETAUpdate(ctx context.Context, routeID string, stops []StopETAUpdate) error {
	for _, u := range stops {
		st, err := s.repo.GetStopTwin(ctx, routeID, u.StopID)
		if err != nil {
			s.log.ErrorContext(ctx, "failed to get stop twin for ETA update", "route_id", routeID, "stop_id", u.StopID, "err", err)
			continue
		}
		if st == nil {
			// Stub or default if StopTwin doesn't exist yet, but in reality we shouldn't insert without knowing order sequence etc.
			continue
		}

		st.PredictedArrival = u.PredictedArrival
		st.WindowStart = u.WindowStart
		st.WindowEnd = u.WindowEnd
		st.UpdatedAt = s.now()

		if err := s.repo.SaveStopTwin(ctx, *st); err != nil {
			s.log.ErrorContext(ctx, "failed to save stop twin during ETA update", "route_id", routeID, "stop_id", u.StopID, "err", err)
		}
	}
	return nil
}

func (s *Service) RebuildRouteTwin(ctx context.Context, routeID string) error {
	s.log.InfoContext(ctx, "rebuilding route twin", "route_id", routeID)
	// Query all events for routeID, replay them and build state
	// s.repo.RebuildRouteTwin(ctx, routeID, view)
	return nil
}
