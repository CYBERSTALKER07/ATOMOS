package supplier

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

func (r *SpannerRepository) ListOrgMembers(ctx context.Context, supplierID string) ([]SupplierOrgMember, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner supplier repository: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT UserId, SupplierId, Name, Email, Phone, SupplierRole, AssignedWarehouseId, AssignedFactoryId, IsActive, CreatedAt, UpdatedAt
              FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_BySupplierUpdated}
              WHERE SupplierId = @supplier_id
              ORDER BY UpdatedAt DESC`,
		Params: map[string]any{"supplier_id": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	items := make([]SupplierOrgMember, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return items, nil
		}
		if err != nil {
			return nil, fmt.Errorf("query supplier org members: %w", err)
		}
		var item SupplierOrgMember
		var email, phone, assignedWarehouseID, assignedFactoryID spanner.NullString
		if err := row.Columns(
			&item.UserID,
			&item.SupplierID,
			&item.Name,
			&email,
			&phone,
			&item.SupplierRole,
			&assignedWarehouseID,
			&assignedFactoryID,
			&item.IsActive,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan supplier org member: %w", err)
		}
		item.Email = nullString(email)
		item.Phone = nullString(phone)
		item.AssignedWarehouseID = nullString(assignedWarehouseID)
		item.AssignedFactoryID = nullString(assignedFactoryID)
		items = append(items, item)
	}
}

func (r *SpannerRepository) CreateOrgMember(ctx context.Context, member CreateOrgMemberParams, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner supplier repository: nil client")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		if err := ensureSupplierUserPhoneAvailable(ctx, txn, member.SupplierID, member.Phone); err != nil {
			return err
		}
		mutations := []*spanner.Mutation{spanner.InsertMap("SupplierUsers", map[string]any{
			"UserId":              member.UserID,
			"SupplierId":          member.SupplierID,
			"Email":               nullableString(member.Email),
			"Phone":               member.Phone,
			"Name":                member.Name,
			"PasswordHash":        member.PasswordHash,
			"SupplierRole":        string(member.SupplierRole),
			"AssignedWarehouseId": nullableString(member.AssignedWarehouseID),
			"AssignedFactoryId":   nullableString(member.AssignedFactoryID),
			"IsActive":            member.IsActive,
			"FirebaseUid":         "",
			"CreatedAt":           member.CreatedAt.UTC(),
			"UpdatedAt":           member.UpdatedAt.UTC(),
		})}
		for _, event := range buf.events {
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
				"EventId":       event.EventID,
				"AggregateType": event.AggregateType,
				"AggregateId":   event.AggregateID,
				"TopicName":     event.TopicName,
				"Payload":       event.Payload,
				"CreatedAt":     event.CreatedAt.UTC(),
				"PublishedAt":   nil,
			}))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("create supplier org member transaction: %w", err)
	}
	return nil
}

func (r *SpannerRepository) ListFleetDrivers(ctx context.Context, supplierID string) ([]SupplierFleetDriver, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner supplier repository: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT DriverId, SupplierId, Name, Phone, HomeNodeType, HomeNodeId, VehicleId, IsActive, CreatedAt, UpdatedAt
              FROM Drivers@{FORCE_INDEX=Idx_Drivers_BySupplierPhone}
              WHERE SupplierId = @supplier_id
              ORDER BY Phone`,
		Params: map[string]any{"supplier_id": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	items := make([]SupplierFleetDriver, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return items, nil
		}
		if err != nil {
			return nil, fmt.Errorf("query supplier fleet drivers: %w", err)
		}
		var item SupplierFleetDriver
		var vehicleID spanner.NullString
		if err := row.Columns(
			&item.DriverID,
			&item.SupplierID,
			&item.Name,
			&item.Phone,
			&item.HomeNodeType,
			&item.HomeNodeID,
			&vehicleID,
			&item.IsActive,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan supplier fleet driver: %w", err)
		}
		item.VehicleID = nullString(vehicleID)
		items = append(items, item)
	}
}

func (r *SpannerRepository) CreateFleetDriver(ctx context.Context, driver CreateFleetDriverParams, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner supplier repository: nil client")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		if err := ensureDriverPhoneAvailable(ctx, txn, driver.SupplierID, driver.Phone); err != nil {
			return err
		}
		if strings.TrimSpace(driver.VehicleID) != "" {
			if err := ensureFleetVehicleAssignment(ctx, txn, driver.SupplierID, driver.VehicleID, driver.HomeNodeType, driver.HomeNodeID); err != nil {
				return err
			}
		}
		mutations := []*spanner.Mutation{spanner.InsertMap("Drivers", map[string]any{
			"DriverId":     driver.DriverID,
			"Name":         driver.Name,
			"Phone":        driver.Phone,
			"PinHash":      driver.PinHash,
			"SupplierId":   driver.SupplierID,
			"HomeNodeType": string(driver.HomeNodeType),
			"HomeNodeId":   driver.HomeNodeID,
			"VehicleId":    nullableString(driver.VehicleID),
			"IsActive":     driver.IsActive,
			"CreatedAt":    driver.CreatedAt.UTC(),
			"UpdatedAt":    driver.UpdatedAt.UTC(),
		})}
		for _, event := range buf.events {
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
				"EventId":       event.EventID,
				"AggregateType": event.AggregateType,
				"AggregateId":   event.AggregateID,
				"TopicName":     event.TopicName,
				"Payload":       event.Payload,
				"CreatedAt":     event.CreatedAt.UTC(),
				"PublishedAt":   nil,
			}))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("create supplier fleet driver transaction: %w", err)
	}
	return nil
}

func (r *SpannerRepository) ListFleetVehicles(ctx context.Context, supplierID string) ([]SupplierFleetVehicle, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner supplier repository: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT VehicleId, SupplierId, Label, LicensePlate, HomeNodeType, HomeNodeId,
                     VehicleClass, MaxVolumeVU, IsActive, CreatedAt, UpdatedAt
              FROM Vehicles@{FORCE_INDEX=Idx_Vehicles_BySupplierPlate}
              WHERE SupplierId = @supplier_id
              ORDER BY LicensePlate`,
		Params: map[string]any{"supplier_id": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	items := make([]SupplierFleetVehicle, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return items, nil
		}
		if err != nil {
			return nil, fmt.Errorf("query supplier fleet vehicles: %w", err)
		}
		var item SupplierFleetVehicle
		var label spanner.NullString
		if err := row.Columns(
			&item.VehicleID,
			&item.SupplierID,
			&label,
			&item.LicensePlate,
			&item.HomeNodeType,
			&item.HomeNodeID,
			&item.VehicleClass,
			&item.MaxVolumeVU,
			&item.IsActive,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan supplier fleet vehicle: %w", err)
		}
		item.Label = nullString(label)
		items = append(items, item)
	}
}

func (r *SpannerRepository) CreateFleetVehicle(ctx context.Context, vehicle CreateFleetVehicleParams, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner supplier repository: nil client")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		if err := ensureVehiclePlateAvailable(ctx, txn, vehicle.SupplierID, vehicle.LicensePlate); err != nil {
			return err
		}
		mutations := []*spanner.Mutation{spanner.InsertMap("Vehicles", map[string]any{
			"VehicleId":    vehicle.VehicleID,
			"Label":        nullableString(vehicle.Label),
			"LicensePlate": vehicle.LicensePlate,
			"SupplierId":   vehicle.SupplierID,
			"HomeNodeType": string(vehicle.HomeNodeType),
			"HomeNodeId":   vehicle.HomeNodeID,
			"VehicleClass": vehicle.VehicleClass,
			"MaxVolumeVU":  vehicle.MaxVolumeVU,
			"IsActive":     vehicle.IsActive,
			"CreatedAt":    vehicle.CreatedAt.UTC(),
			"UpdatedAt":    vehicle.UpdatedAt.UTC(),
		})}
		for _, event := range buf.events {
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
				"EventId":       event.EventID,
				"AggregateType": event.AggregateType,
				"AggregateId":   event.AggregateID,
				"TopicName":     event.TopicName,
				"Payload":       event.Payload,
				"CreatedAt":     event.CreatedAt.UTC(),
				"PublishedAt":   nil,
			}))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("create supplier fleet vehicle transaction: %w", err)
	}
	return nil
}

func ensureSupplierUserPhoneAvailable(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string, phone string) error {
	row, err := txn.ReadRowUsingIndex(ctx, "SupplierUsers", "Idx_SupplierUsers_ByPhone", spanner.Key{phone}, []string{"SupplierId", "UserId", "IsActive"})
	if err == nil {
		var existingSupplierID, userID string
		var isActive bool
		if err := row.Columns(&existingSupplierID, &userID, &isActive); err != nil {
			return fmt.Errorf("scan existing supplier user by phone: %w", err)
		}
		if existingSupplierID == supplierID && isActive && userID != "" {
			return errOrgMemberPhoneExists
		}
		return nil
	}
	if err != spanner.ErrRowNotFound {
		return fmt.Errorf("read existing supplier user by phone: %w", err)
	}
	return nil
}

func ensureDriverPhoneAvailable(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string, phone string) error {
	stmt := spanner.Statement{
		SQL: `SELECT DriverId
              FROM Drivers@{FORCE_INDEX=Idx_Drivers_BySupplierPhone}
              WHERE SupplierId = @supplier_id AND Phone = @phone
              LIMIT 1`,
		Params: map[string]any{"supplier_id": supplierID, "phone": phone},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil
	}
	if err != nil {
		return fmt.Errorf("query existing supplier driver by phone: %w", err)
	}
	var driverID string
	if err := row.Columns(&driverID); err != nil {
		return fmt.Errorf("scan existing supplier driver by phone: %w", err)
	}
	if strings.TrimSpace(driverID) != "" {
		return errDriverPhoneExists
	}
	return nil
}

func ensureVehiclePlateAvailable(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string, licensePlate string) error {
	stmt := spanner.Statement{
		SQL: `SELECT VehicleId
              FROM Vehicles@{FORCE_INDEX=Idx_Vehicles_BySupplierPlate}
              WHERE SupplierId = @supplier_id AND LicensePlate = @license_plate
              LIMIT 1`,
		Params: map[string]any{"supplier_id": supplierID, "license_plate": licensePlate},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil
	}
	if err != nil {
		return fmt.Errorf("query existing supplier vehicle by license plate: %w", err)
	}
	var vehicleID string
	if err := row.Columns(&vehicleID); err != nil {
		return fmt.Errorf("scan existing supplier vehicle by license plate: %w", err)
	}
	if strings.TrimSpace(vehicleID) != "" {
		return errVehiclePlateExists
	}
	return nil
}

func ensureFleetVehicleAssignment(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string, vehicleID string, homeNodeType auth.HomeNodeType, homeNodeID string) error {
	row, err := txn.ReadRow(ctx, "Vehicles", spanner.Key{vehicleID}, []string{"SupplierId", "HomeNodeType", "HomeNodeId"})
	if err != nil {
		if err == spanner.ErrRowNotFound {
			return errFleetVehicleMismatch
		}
		return fmt.Errorf("read supplier fleet vehicle assignment: %w", err)
	}
	var existingSupplierID, existingHomeNodeType, existingHomeNodeID string
	if err := row.Columns(&existingSupplierID, &existingHomeNodeType, &existingHomeNodeID); err != nil {
		return fmt.Errorf("scan supplier fleet vehicle assignment: %w", err)
	}
	if existingSupplierID != supplierID || existingHomeNodeType != string(homeNodeType) || existingHomeNodeID != homeNodeID {
		return errFleetVehicleMismatch
	}
	return nil
}

func nullString(value spanner.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.StringVal)
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
