package warehouse

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/spanner"
)

func readExpressOps(sched any) (bool, int64) {
	if sched == nil {
		return true, 0
	}
	raw, err := json.Marshal(sched)
	if err != nil {
		return true, 0
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return true, 0
	}
	express, ok := m["express"].(map[string]any)
	if !ok {
		return true, 0
	}
	enabled := true
	if v, ok := express["enabled"].(bool); ok {
		enabled = v
	}
	var floor int64
	switch v := express["stock_floor"].(type) {
	case float64:
		floor = int64(v)
	case int64:
		floor = v
	}
	return enabled, floor
}

func loadOperatingScheduleJSON(ctx context.Context, client *spanner.Client, warehouseID string) map[string]any {
	out := map[string]any{}
	if client == nil {
		return out
	}
	row, err := client.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"OperatingSchedule"})
	if err != nil {
		return out
	}
	var schedule spanner.NullJSON
	if err := row.Columns(&schedule); err != nil || !schedule.Valid {
		return out
	}
	_ = json.Unmarshal([]byte(schedule.String()), &out)
	return out
}
