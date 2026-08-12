package twin

import (
	"context"
	"testing"

	kafka "github.com/segmentio/kafka-go"
)

type twinRepoStub struct {
	lastRouteID string
	lastLat     float64
	lastLng     float64
}

func (s *twinRepoStub) GetRouteTwin(ctx context.Context, routeID string) (*RouteTwinView, error) {
	return nil, nil
}
func (s *twinRepoStub) ListActiveRouteTwins(ctx context.Context, zoneH3 string) ([]RouteTwinView, error) {
	return nil, nil
}
func (s *twinRepoStub) GetVehicleInventory(ctx context.Context, routeID string) ([]VehicleInventory, error) {
	return nil, nil
}
func (s *twinRepoStub) GetStopTwin(ctx context.Context, routeID, stopID string) (*StopTwin, error) {
	return nil, nil
}
func (s *twinRepoStub) SaveRouteTwin(ctx context.Context, rt RouteTwin) error {
	s.lastRouteID = rt.RouteID
	s.lastLat = rt.CurrentLat
	s.lastLng = rt.CurrentLng
	return nil
}
func (s *twinRepoStub) SaveStopTwin(ctx context.Context, st StopTwin) error { return nil }
func (s *twinRepoStub) SaveVehicleInventory(ctx context.Context, inv VehicleInventory) error {
	return nil
}
func (s *twinRepoStub) RebuildRouteTwin(ctx context.Context, routeID string, view RouteTwinView) error {
	return nil
}

func TestHandleEvent_DriverLocationTelemetryEnvelope(t *testing.T) {
	repo := &twinRepoStub{}
	svc := NewService(ServiceConfig{Repo: repo})
	c := NewEventConsumer(svc, nil)
	raw := []byte(`{
		"type":"DRIVER_LOCATION_UPDATED",
		"route_id":"route-9",
		"data":{"driver_id":"drv-1","lat":41.3,"lng":69.2,"latitude":41.3,"longitude":69.2}
	}`)
	if err := c.HandleEvent(context.Background(), kafka.Message{Value: raw}); err != nil {
		t.Fatal(err)
	}
	if repo.lastRouteID != "route-9" {
		t.Fatalf("route_id=%q want route-9", repo.lastRouteID)
	}
	if repo.lastLat != 41.3 || repo.lastLng != 69.2 {
		t.Fatalf("coords=(%v,%v)", repo.lastLat, repo.lastLng)
	}
}

func TestParseDriverLocationPayload_legacyOrderEvent(t *testing.T) {
	routeID, lat, lng, _ := parseDriverLocationPayload([]byte(
		`{"type":"DRIVER_LOCATION_UPDATED","route_id":"r1","gps_lat":1.5,"gps_lng":2.5}`,
	))
	if routeID != "r1" || lat != 1.5 || lng != 2.5 {
		t.Fatalf("got %s %v %v", routeID, lat, lng)
	}
}
