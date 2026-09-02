package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type PickerLocation struct {
	StaffID string  `json:"staff_id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Zone    string  `json:"zone"`
}

// UpdatePickerLocation is called via WebSocket or REST periodically by the Android Scanner App
// to record the staff's location on the warehouse floor (e.g. inferred from scan bin IDs).
func (s *Service) UpdatePickerLocation(ctx context.Context, warehouseID string, loc PickerLocation) error {
	if s.redisClient == nil {
		return nil
	}

	key := fmt.Sprintf("warehouse:%s:heatmap", warehouseID)
	data, err := json.Marshal(loc)
	if err != nil {
		return err
	}

	// We use an HSET so we can quickly fetch all active pickers in one O(N) call
	err = s.redisClient.HSet(ctx, key, loc.StaffID, data).Err()
	if err != nil {
		return fmt.Errorf("failed to update picker location in redis: %w", err)
	}

	// Expire the heatmap key so inactive warehouses don't hold memory forever
	s.redisClient.Expire(ctx, key, 24*time.Hour)

	// Broadcast the live location to the Ops Dashboard
	if s.warehouseHub != nil {
		s.warehouseHub.Broadcast(ctx, "warehouse:"+warehouseID, data)
	}

	return nil
}

// GetLiveHeatmap retrieves all active pickers for a warehouse to render on the Ops Dashboard.
func (s *Service) GetLiveHeatmap(ctx context.Context, warehouseID string) ([]PickerLocation, error) {
	if s.redisClient == nil {
		return []PickerLocation{}, nil
	}

	key := fmt.Sprintf("warehouse:%s:heatmap", warehouseID)
	res, err := s.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get live heatmap: %w", err)
	}

	var heatmap []PickerLocation
	for _, val := range res {
		var loc PickerLocation
		if err := json.Unmarshal([]byte(val), &loc); err == nil {
			heatmap = append(heatmap, loc)
		}
	}
	
	if heatmap == nil {
		heatmap = []PickerLocation{}
	}
	return heatmap, nil
}
