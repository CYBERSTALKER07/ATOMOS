package driver

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// Driver represents the driver entity.
type Driver struct {
	DriverID          string    `json:"driver_id" spanner:"DriverId"`
	Name              string    `json:"name" spanner:"Name"`
	Phone             string    `json:"phone" spanner:"Phone"`
	PinHash           *string   `json:"-" spanner:"PinHash"`
	SupplierID        string    `json:"supplier_id" spanner:"SupplierId"`
	HomeNodeType      string    `json:"home_node_type" spanner:"HomeNodeType"`
	HomeNodeID        string    `json:"home_node_id" spanner:"HomeNodeId"`
	VehicleID         *string   `json:"vehicle_id,omitempty" spanner:"VehicleId"`
	IsActive          bool      `json:"is_active" spanner:"IsActive"`
	OnShift           bool      `json:"on_shift" spanner:"OnShift"`
	UnavailableReason *string   `json:"unavailable_reason,omitempty" spanner:"UnavailableReason"`
	UnavailableNote   *string   `json:"unavailable_note,omitempty" spanner:"UnavailableNote"`
	CreatedAt         time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt         time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// Vehicle represents the vehicle entity.
type Vehicle struct {
	VehicleID         string    `json:"vehicle_id" spanner:"VehicleId"`
	Label             *string   `json:"label,omitempty" spanner:"Label"`
	LicensePlate      string    `json:"license_plate" spanner:"LicensePlate"`
	SupplierID        string    `json:"supplier_id" spanner:"SupplierId"`
	HomeNodeType      string    `json:"home_node_type" spanner:"HomeNodeType"`
	HomeNodeID        string    `json:"home_node_id" spanner:"HomeNodeId"`
	VehicleClass      string    `json:"vehicle_class" spanner:"VehicleClass"`
	MaxVolumeVU       float64   `json:"max_volume_vu" spanner:"MaxVolumeVU"`
	IsActive          bool      `json:"is_active" spanner:"IsActive"`
	UnavailableReason *string   `json:"unavailable_reason,omitempty" spanner:"UnavailableReason"`
	UnavailableNote   *string   `json:"unavailable_note,omitempty" spanner:"UnavailableNote"`
	CreatedAt         time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt         time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// CreateDriver inserts a new driver record and emits a DRIVER_CREATED event atomically.
func (r *SpannerRepository) CreateDriver(ctx context.Context, d Driver, emit func(outbox.TxnBuffer) error) error {
	d.CreatedAt = spanner.CommitTimestamp
	d.UpdatedAt = spanner.CommitTimestamp
	m, err := spanner.InsertStruct("Drivers", d)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{m}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// GetDriver retrieves a driver by ID.
func (r *SpannerRepository) GetDriver(ctx context.Context, driverID string) (Driver, error) {
	row, err := r.client.Single().ReadRow(ctx, "Drivers", spanner.Key{driverID}, []string{
		"DriverId", "Name", "Phone", "PinHash", "SupplierId", "HomeNodeType",
		"HomeNodeId", "VehicleId", "IsActive", "OnShift", "UnavailableReason",
		"UnavailableNote", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return Driver{}, err
	}
	var d Driver
	if err := row.ToStruct(&d); err != nil {
		return Driver{}, err
	}
	return d, nil
}

// UpdateDriver updates an existing driver record and emits a DRIVER_AVAILABILITY_CHANGED event atomically.
// Uses UpdateMap with only supplied fields to prevent PinHash wipe (Finding 5.1).
func (r *SpannerRepository) UpdateDriver(ctx context.Context, d Driver, emit func(outbox.TxnBuffer) error) error {
	cols := map[string]interface{}{
		"DriverId":  d.DriverID,
		"UpdatedAt": spanner.CommitTimestamp,
	}
	if d.Name != "" {
		cols["Name"] = d.Name
	}
	if d.Phone != "" {
		cols["Phone"] = d.Phone
	}
	if d.SupplierID != "" {
		cols["SupplierId"] = d.SupplierID
	}
	if d.HomeNodeType != "" {
		cols["HomeNodeType"] = d.HomeNodeType
	}
	if d.HomeNodeID != "" {
		cols["HomeNodeId"] = d.HomeNodeID
	}
	if d.VehicleID != nil {
		cols["VehicleId"] = *d.VehicleID
	}
	// Boolean flags: always include since they have meaningful zero values
	cols["IsActive"] = d.IsActive
	cols["OnShift"] = d.OnShift
	if d.UnavailableReason != nil {
		cols["UnavailableReason"] = *d.UnavailableReason
	}
	if d.UnavailableNote != nil {
		cols["UnavailableNote"] = *d.UnavailableNote
	}
	// PinHash is NEVER set from REST updates (json:"-")

	m := spanner.UpdateMap("Drivers", cols)
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{m}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// ListDrivers lists drivers by supplier ID.
func (r *SpannerRepository) ListDrivers(ctx context.Context, supplierID string, limit, offset int) ([]Driver, error) {
	stmt := spanner.Statement{
		SQL: `SELECT * FROM Drivers WHERE SupplierId = @supplierId LIMIT @limit OFFSET @offset`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"limit":      limit,
			"offset":     offset,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var drivers []Driver
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var d Driver
		if err := row.ToStruct(&d); err != nil {
			return nil, err
		}
		drivers = append(drivers, d)
	}
	return drivers, nil
}

// CreateVehicle inserts a new vehicle record and emits a VEHICLE_CREATED event atomically.
func (r *SpannerRepository) CreateVehicle(ctx context.Context, v Vehicle, emit func(outbox.TxnBuffer) error) error {
	v.CreatedAt = spanner.CommitTimestamp
	v.UpdatedAt = spanner.CommitTimestamp
	m, err := spanner.InsertStruct("Vehicles", v)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{m}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// GetVehicle retrieves a vehicle by ID.
func (r *SpannerRepository) GetVehicle(ctx context.Context, vehicleID string) (Vehicle, error) {
	row, err := r.client.Single().ReadRow(ctx, "Vehicles", spanner.Key{vehicleID}, []string{
		"VehicleId", "Label", "LicensePlate", "SupplierId", "HomeNodeType",
		"HomeNodeId", "VehicleClass", "MaxVolumeVU", "IsActive", "UnavailableReason",
		"UnavailableNote", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return Vehicle{}, err
	}
	var v Vehicle
	if err := row.ToStruct(&v); err != nil {
		return Vehicle{}, err
	}
	return v, nil
}

// UpdateVehicle updates an existing vehicle record and emits a VEHICLE_AVAILABILITY_CHANGED event atomically.
func (r *SpannerRepository) UpdateVehicle(ctx context.Context, vehicleID string, updates map[string]any, emit func(outbox.TxnBuffer) error) error {
	updates["VehicleId"] = vehicleID
	updates["UpdatedAt"] = spanner.CommitTimestamp
	m := spanner.UpdateMap("Vehicles", updates)
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{m}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// ListVehicles lists vehicles by supplier ID.
func (r *SpannerRepository) ListVehicles(ctx context.Context, supplierID string, limit, offset int) ([]Vehicle, error) {
	stmt := spanner.Statement{
		SQL: `SELECT * FROM Vehicles WHERE SupplierId = @supplierId LIMIT @limit OFFSET @offset`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"limit":      limit,
			"offset":     offset,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var vehicles []Vehicle
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var v Vehicle
		if err := row.ToStruct(&v); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, nil
}
