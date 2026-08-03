package laborcapacity

import (
	"context"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Service provides labor capacity operations.
type Service struct {
	spanner *spanner.Client
}

// NewService creates a new labor capacity service.
func NewService(spannerClient *spanner.Client) *Service {
	return &Service{spanner: spannerClient}
}

// GetDriverScore reads the latest score for a driver.
func (s *Service) GetDriverScore(ctx context.Context, driverID string) (*DriverScore, error) {
	row, err := s.spanner.Single().ReadRow(ctx, "DriverScores", spanner.Key{driverID}, []string{
		"DriverId", "Score", "OnTimeRate", "CompletionRate", "DamageRate",
		"ShopClosedRate", "FeedbackScore", "StopsPerHour", "WindowStart", "WindowEnd", "ComputedAt",
	})
	if err != nil {
		return nil, err
	}
	var ds DriverScore
	if err := row.Columns(
		&ds.DriverId, &ds.Score, &ds.OnTimeRate, &ds.CompletionRate, &ds.DamageRate,
		&ds.ShopClosedRate, &ds.FeedbackScore, &ds.StopsPerHour, &ds.WindowStart, &ds.WindowEnd, &ds.ComputedAt,
	); err != nil {
		return nil, err
	}
	return &ds, nil
}

// GetZoneCapacity reads capacity for a zone on a given date.
func (s *Service) GetZoneCapacity(ctx context.Context, zoneH3 string, date civil.Date) (*ZoneCapacity, error) {
	row, err := s.spanner.Single().ReadRow(ctx, "ZoneCapacity", spanner.Key{zoneH3, date}, []string{
		"ZoneH3", "Date", "TotalCapacity", "UsedCapacity", "ComputedAt",
	})
	if err != nil {
		return nil, err
	}
	var zc ZoneCapacity
	if err := row.Columns(&zc.ZoneH3, &zc.Date, &zc.TotalCapacity, &zc.UsedCapacity, &zc.ComputedAt); err != nil {
		return nil, err
	}
	return &zc, nil
}

// SetDriverAvailability upserts a driver's availability for a day.
func (s *Service) SetDriverAvailability(ctx context.Context, req SetAvailabilityRequest) error {
	d, err := civil.ParseDate(req.Date)
	if err != nil {
		return err
	}
	_, err = s.spanner.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("DriverAvailability", map[string]interface{}{
			"DriverId":       req.DriverId,
			"Date":           d,
			"AvailableHours": req.AvailableHours,
			"ZoneH3":         spanner.NullString{StringVal: req.ZoneH3, Valid: req.ZoneH3 != ""},
			"Status":         req.Status,
			"UpdatedAt":      time.Now().UTC(),
		}),
	})
	return err
}

// GetDriverAvailability reads a driver's availability for a specific date.
func (s *Service) GetDriverAvailability(ctx context.Context, driverID string, date civil.Date) (*DriverAvailability, error) {
	row, err := s.spanner.Single().ReadRow(ctx, "DriverAvailability", spanner.Key{driverID, date}, []string{
		"DriverId", "Date", "AvailableHours", "ZoneH3", "Status", "UpdatedAt",
	})
	if err != nil {
		return nil, err
	}
	var da DriverAvailability
	var zone spanner.NullString
	if err := row.Columns(&da.DriverId, &da.Date, &da.AvailableHours, &zone, &da.Status, &da.UpdatedAt); err != nil {
		return nil, err
	}
	if zone.Valid {
		da.ZoneH3 = zone.StringVal
	}
	return &da, nil
}

// ListZoneCapacities returns all zone capacities for a given date.
func (s *Service) ListZoneCapacities(ctx context.Context, date civil.Date) ([]ZoneCapacity, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ZoneH3, Date, TotalCapacity, UsedCapacity, ComputedAt
		      FROM ZoneCapacity WHERE Date = @Date`,
		Params: map[string]interface{}{"Date": date},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var out []ZoneCapacity
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var zc ZoneCapacity
		if err := row.Columns(&zc.ZoneH3, &zc.Date, &zc.TotalCapacity, &zc.UsedCapacity, &zc.ComputedAt); err != nil {
			return nil, err
		}
		out = append(out, zc)
	}
	return out, nil
}
