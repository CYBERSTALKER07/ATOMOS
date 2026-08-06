package stocklots

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// BinLocation is a warehouse bin/slot.
type BinLocation struct {
	WarehouseID  string  `json:"warehouse_id"`
	LocationID   string  `json:"location_id"`
	Zone         string  `json:"zone,omitempty"`
	Aisle        string  `json:"aisle,omitempty"`
	Rack         string  `json:"rack,omitempty"`
	Level        string  `json:"level,omitempty"`
	Bin          string  `json:"bin,omitempty"`
	LocationType string  `json:"location_type"`
	PickSequence int64   `json:"pick_sequence"`
	MaxVolumeVU  float64 `json:"max_volume_vu,omitempty"`
	IsActive     bool    `json:"is_active"`
	UpdatedAt    string  `json:"updated_at,omitempty"`
}

// CreateBinRequest creates a bin location.
type CreateBinRequest struct {
	WarehouseID  string
	LocationID   string
	Zone         string
	Aisle        string
	Rack         string
	Level        string
	Bin          string
	LocationType string
	PickSequence int64
	MaxVolumeVU  float64
}

// UpsertBinInTxn inserts or updates a WarehouseLocations row.
func UpsertBinInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, req CreateBinRequest) (*BinLocation, error) {
	req.WarehouseID = strings.TrimSpace(req.WarehouseID)
	if req.WarehouseID == "" {
		return nil, fmt.Errorf("warehouse_id required")
	}
	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		locID = uuid.NewString()
	}
	locType := strings.ToUpper(strings.TrimSpace(req.LocationType))
	if locType == "" {
		locType = "PICK"
	}
	switch locType {
	case "PICK", "BULK", "STAGE", "QUARANTINE":
	default:
		return nil, fmt.Errorf("invalid location_type")
	}

	existing, err := txn.ReadRow(ctx, "WarehouseLocations", spanner.Key{req.WarehouseID, locID},
		[]string{"LocationId"})
	isUpdate := err == nil
	if err != nil && spanner.ErrCode(err) != 5 {
		return nil, err
	}
	_ = existing

	cols := map[string]any{
		"WarehouseId":  req.WarehouseID,
		"LocationId":   locID,
		"Zone":         strings.TrimSpace(req.Zone),
		"Aisle":        strings.TrimSpace(req.Aisle),
		"Rack":         strings.TrimSpace(req.Rack),
		"Level":        strings.TrimSpace(req.Level),
		"Bin":          strings.TrimSpace(req.Bin),
		"LocationType": locType,
		"PickSequence": req.PickSequence,
		"IsActive":     true,
		"UpdatedAt":    spanner.CommitTimestamp,
	}
	if req.MaxVolumeVU > 0 {
		cols["MaxVolumeVU"] = req.MaxVolumeVU
	}
	var mut *spanner.Mutation
	if isUpdate {
		mut = spanner.UpdateMap("WarehouseLocations", cols)
	} else {
		cols["CreatedAt"] = spanner.CommitTimestamp
		mut = spanner.InsertMap("WarehouseLocations", cols)
	}
	if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
		return nil, err
	}
	return &BinLocation{
		WarehouseID:  req.WarehouseID,
		LocationID:   locID,
		Zone:         strings.TrimSpace(req.Zone),
		Aisle:        strings.TrimSpace(req.Aisle),
		Rack:         strings.TrimSpace(req.Rack),
		Level:        strings.TrimSpace(req.Level),
		Bin:          strings.TrimSpace(req.Bin),
		LocationType: locType,
		PickSequence: req.PickSequence,
		MaxVolumeVU:  req.MaxVolumeVU,
		IsActive:     true,
	}, nil
}

// PatchBinInTxn updates mutable bin fields.
func PatchBinInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, locationID string, patch map[string]any) (*BinLocation, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	locationID = strings.TrimSpace(locationID)
	row, err := txn.ReadRow(ctx, "WarehouseLocations", spanner.Key{warehouseID, locationID},
		[]string{"Zone", "Aisle", "Rack", "Level", "Bin", "LocationType", "PickSequence", "MaxVolumeVU", "IsActive"})
	if err != nil {
		return nil, err
	}
	var zone, aisle, rack, level, bin, locType string
	var pickSeq int64
	var maxVU spanner.NullFloat64
	var active bool
	if err := row.Columns(&zone, &aisle, &rack, &level, &bin, &locType, &pickSeq, &maxVU, &active); err != nil {
		return nil, err
	}
	if v, ok := patch["zone"].(string); ok {
		zone = v
	}
	if v, ok := patch["aisle"].(string); ok {
		aisle = v
	}
	if v, ok := patch["rack"].(string); ok {
		rack = v
	}
	if v, ok := patch["level"].(string); ok {
		level = v
	}
	if v, ok := patch["bin"].(string); ok {
		bin = v
	}
	if v, ok := patch["location_type"].(string); ok && strings.TrimSpace(v) != "" {
		locType = strings.ToUpper(strings.TrimSpace(v))
	}
	if v, ok := patch["pick_sequence"].(float64); ok {
		pickSeq = int64(v)
	}
	if v, ok := patch["is_active"].(bool); ok {
		active = v
	}
	vu := 0.0
	if maxVU.Valid {
		vu = maxVU.Float64
	}
	if v, ok := patch["max_volume_vu"].(float64); ok {
		vu = v
	}
	cols := map[string]any{
		"WarehouseId":  warehouseID,
		"LocationId":   locationID,
		"Zone":         zone,
		"Aisle":        aisle,
		"Rack":         rack,
		"Level":        level,
		"Bin":          bin,
		"LocationType": locType,
		"PickSequence": pickSeq,
		"IsActive":     active,
		"UpdatedAt":    spanner.CommitTimestamp,
	}
	if vu > 0 {
		cols["MaxVolumeVU"] = vu
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("WarehouseLocations", cols)}); err != nil {
		return nil, err
	}
	return &BinLocation{
		WarehouseID: warehouseID, LocationID: locationID, Zone: zone, Aisle: aisle, Rack: rack,
		Level: level, Bin: bin, LocationType: locType, PickSequence: pickSeq, MaxVolumeVU: vu, IsActive: active,
	}, nil
}

// ListBins lists bin locations for a warehouse.
func ListBins(ctx context.Context, client *spanner.Client, warehouseID string) ([]BinLocation, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner required")
	}
	warehouseID = strings.TrimSpace(warehouseID)
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, LocationId, Zone, Aisle, Rack, Level, Bin, LocationType,
		             PickSequence, MaxVolumeVU, IsActive, UpdatedAt
		      FROM WarehouseLocations WHERE WarehouseId = @wid
		      ORDER BY Zone, PickSequence, LocationId`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []BinLocation
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		b, err := scanBin(row)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// GetBin loads one bin.
func GetBin(ctx context.Context, client *spanner.Client, warehouseID, locationID string) (*BinLocation, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner required")
	}
	row, err := client.Single().ReadRow(ctx, "WarehouseLocations",
		spanner.Key{strings.TrimSpace(warehouseID), strings.TrimSpace(locationID)},
		[]string{"WarehouseId", "LocationId", "Zone", "Aisle", "Rack", "Level", "Bin", "LocationType",
			"PickSequence", "MaxVolumeVU", "IsActive", "UpdatedAt"})
	if err != nil {
		return nil, err
	}
	b, err := scanBin(row)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func scanBin(row *spanner.Row) (BinLocation, error) {
	var b BinLocation
	var zone, aisle, rack, level, bin spanner.NullString
	var maxVU spanner.NullFloat64
	var updated time.Time
	if err := row.Columns(
		&b.WarehouseID, &b.LocationID, &zone, &aisle, &rack, &level, &bin,
		&b.LocationType, &b.PickSequence, &maxVU, &b.IsActive, &updated,
	); err != nil {
		return b, err
	}
	if zone.Valid {
		b.Zone = zone.StringVal
	}
	if aisle.Valid {
		b.Aisle = aisle.StringVal
	}
	if rack.Valid {
		b.Rack = rack.StringVal
	}
	if level.Valid {
		b.Level = level.StringVal
	}
	if bin.Valid {
		b.Bin = bin.StringVal
	}
	if maxVU.Valid {
		b.MaxVolumeVU = maxVU.Float64
	}
	b.UpdatedAt = updated.UTC().Format(time.RFC3339)
	return b, nil
}
