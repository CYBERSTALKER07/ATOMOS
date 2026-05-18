package entityresolution

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const defaultPerTypeLimit = 12

// SpannerRepository provides supplier-scoped read-side resolution queries.
type SpannerRepository struct {
	client *spanner.Client
}

// NewRepository builds a Spanner-backed entity-resolution repository.
func NewRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

// FindExactByID returns exact records by entity ID within supplier scope.
func (r *SpannerRepository) FindExactByID(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error) {
	supplierID = strings.TrimSpace(supplierID)
	entityID = strings.TrimSpace(entityID)
	if supplierID == "" || entityID == "" {
		return nil, nil
	}

	typeName := normalizeEntityType(entityType)
	if typeName == EntityTypeAny {
		out := make([]EntityRecord, 0, len(searchableEntityTypes))
		for _, candidateType := range searchableEntityTypes {
			records, err := r.findExactByType(ctx, supplierID, candidateType, entityID)
			if err != nil {
				return nil, err
			}
			out = append(out, records...)
		}
		return out, nil
	}
	return r.findExactByType(ctx, supplierID, typeName, entityID)
}

// ListScopedRecords returns bounded supplier-scoped rows for probabilistic ranking.
func (r *SpannerRepository) ListScopedRecords(ctx context.Context, supplierID, entityType string, perTypeLimit int) ([]EntityRecord, error) {
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, nil
	}
	if perTypeLimit <= 0 {
		perTypeLimit = defaultPerTypeLimit
	}

	typeName := normalizeEntityType(entityType)
	if typeName == EntityTypeAny {
		out := make([]EntityRecord, 0, perTypeLimit*len(searchableEntityTypes))
		for _, candidateType := range searchableEntityTypes {
			records, err := r.listByType(ctx, supplierID, candidateType, perTypeLimit)
			if err != nil {
				return nil, err
			}
			out = append(out, records...)
		}
		return out, nil
	}
	return r.listByType(ctx, supplierID, typeName, perTypeLimit)
}

// LoadLineage returns one-hop lineage links from a supplier-scoped source entity.
func (r *SpannerRepository) LoadLineage(ctx context.Context, supplierID, entityType, entityID string) ([]LineageLink, error) {
	supplierID = strings.TrimSpace(supplierID)
	entityID = strings.TrimSpace(entityID)
	if supplierID == "" || entityID == "" {
		return nil, ErrInvalidInput
	}

	switch normalizeEntityType(entityType) {
	case EntityTypeOrder:
		return r.loadOrderLineage(ctx, supplierID, entityID)
	case EntityTypeDriver:
		return r.loadDriverLineage(ctx, supplierID, entityID)
	case EntityTypeVehicle:
		return r.loadVehicleLineage(ctx, supplierID, entityID)
	case EntityTypeWarehouse:
		return r.loadWarehouseLineage(ctx, supplierID, entityID)
	case EntityTypeFactory:
		return r.loadFactoryLineage(ctx, supplierID, entityID)
	case EntityTypeRetailer:
		return r.loadRetailerLineage(ctx, supplierID, entityID)
	case EntityTypeSupplier:
		return r.loadSupplierLineage(ctx, supplierID, entityID)
	default:
		return nil, ErrInvalidInput
	}
}

func (r *SpannerRepository) findExactByType(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error) {
	switch entityType {
	case EntityTypeSupplier:
		return r.findSupplierExact(ctx, supplierID, entityID)
	case EntityTypeWarehouse:
		return r.findWarehouseExact(ctx, supplierID, entityID)
	case EntityTypeFactory:
		return r.findFactoryExact(ctx, supplierID, entityID)
	case EntityTypeDriver:
		return r.findDriverExact(ctx, supplierID, entityID)
	case EntityTypeVehicle:
		return r.findVehicleExact(ctx, supplierID, entityID)
	case EntityTypeRetailer:
		return r.findRetailerExact(ctx, supplierID, entityID)
	case EntityTypeOrder:
		return r.findOrderExact(ctx, supplierID, entityID)
	default:
		return nil, ErrInvalidInput
	}
}

func (r *SpannerRepository) listByType(ctx context.Context, supplierID, entityType string, limit int) ([]EntityRecord, error) {
	switch entityType {
	case EntityTypeSupplier:
		return r.listSupplierScoped(ctx, supplierID)
	case EntityTypeWarehouse:
		return r.listWarehouseScoped(ctx, supplierID, limit)
	case EntityTypeFactory:
		return r.listFactoryScoped(ctx, supplierID, limit)
	case EntityTypeDriver:
		return r.listDriverScoped(ctx, supplierID, limit)
	case EntityTypeVehicle:
		return r.listVehicleScoped(ctx, supplierID, limit)
	case EntityTypeRetailer:
		return r.listRetailerScoped(ctx, supplierID, limit)
	case EntityTypeOrder:
		return r.listOrderScoped(ctx, supplierID, limit)
	default:
		return nil, ErrInvalidInput
	}
}

func (r *SpannerRepository) findSupplierExact(ctx context.Context, supplierID, entityID string) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SupplierId, COALESCE(Name, ''), COALESCE(Email, ''), COALESCE(Phone, '')
              FROM Suppliers
              WHERE SupplierId = @sid AND SupplierId = @eid
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "eid": entityID},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, name, email, phone string
		if err := row.Columns(&id, &name, &email, &phone); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeSupplier,
			EntityID:   id,
			Label:      firstNonEmpty(name, id),
			SearchText: strings.TrimSpace(name + " " + email + " " + phone + " " + id),
			Metadata: map[string]string{
				"email": email,
				"phone": phone,
			},
		}, nil
	})
}

func (r *SpannerRepository) findWarehouseExact(ctx context.Context, supplierID, entityID string) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, COALESCE(Name, ''), COALESCE(Address, '')
              FROM Warehouses@{FORCE_INDEX=Idx_Warehouses_BySupplierId}
              WHERE SupplierId = @sid AND WarehouseId = @eid
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "eid": entityID},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, name, address string
		if err := row.Columns(&id, &name, &address); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeWarehouse,
			EntityID:   id,
			Label:      firstNonEmpty(name, id),
			SearchText: strings.TrimSpace(name + " " + address + " " + id),
			Metadata: map[string]string{
				"address": address,
			},
		}, nil
	})
}

func (r *SpannerRepository) findFactoryExact(ctx context.Context, supplierID, entityID string) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT FactoryId, COALESCE(Name, ''), COALESCE(Address, '')
              FROM Factories@{FORCE_INDEX=Idx_Factories_BySupplierId}
              WHERE SupplierId = @sid AND FactoryId = @eid
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "eid": entityID},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, name, address string
		if err := row.Columns(&id, &name, &address); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeFactory,
			EntityID:   id,
			Label:      firstNonEmpty(name, id),
			SearchText: strings.TrimSpace(name + " " + address + " " + id),
			Metadata: map[string]string{
				"address": address,
			},
		}, nil
	})
}

func (r *SpannerRepository) findDriverExact(ctx context.Context, supplierID, entityID string) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DriverId, COALESCE(Name, ''), COALESCE(Phone, ''), COALESCE(LicensePlate, ''),
                     COALESCE(VehicleId, ''), COALESCE(HomeNodeType, ''), COALESCE(HomeNodeId, ''), COALESCE(WarehouseId, '')
              FROM Drivers@{FORCE_INDEX=Idx_Drivers_BySupplierId}
              WHERE SupplierId = @sid AND DriverId = @eid
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "eid": entityID},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, name, phone, license, vehicleID, homeType, homeID, warehouseID string
		if err := row.Columns(&id, &name, &phone, &license, &vehicleID, &homeType, &homeID, &warehouseID); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeDriver,
			EntityID:   id,
			Label:      firstNonEmpty(name, id),
			SearchText: strings.TrimSpace(name + " " + phone + " " + license + " " + id + " " + vehicleID),
			Metadata: map[string]string{
				"phone":          phone,
				"license_plate":  license,
				"vehicle_id":     vehicleID,
				"home_node_type": homeType,
				"home_node_id":   homeID,
				"warehouse_id":   warehouseID,
			},
		}, nil
	})
}

func (r *SpannerRepository) findVehicleExact(ctx context.Context, supplierID, entityID string) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT VehicleId, COALESCE(Label, ''), COALESCE(LicensePlate, ''), COALESCE(VehicleClass, ''),
                     COALESCE(HomeNodeType, ''), COALESCE(HomeNodeId, ''), COALESCE(WarehouseId, '')
              FROM Vehicles@{FORCE_INDEX=Idx_Vehicles_BySupplier}
              WHERE SupplierId = @sid AND VehicleId = @eid
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "eid": entityID},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, label, license, className, homeType, homeID, warehouseID string
		if err := row.Columns(&id, &label, &license, &className, &homeType, &homeID, &warehouseID); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeVehicle,
			EntityID:   id,
			Label:      firstNonEmpty(label, license, id),
			SearchText: strings.TrimSpace(label + " " + license + " " + className + " " + id),
			Metadata: map[string]string{
				"license_plate":  license,
				"vehicle_class":  className,
				"home_node_type": homeType,
				"home_node_id":   homeID,
				"warehouse_id":   warehouseID,
			},
		}, nil
	})
}

func (r *SpannerRepository) findRetailerExact(ctx context.Context, supplierID, entityID string) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT r.RetailerId, COALESCE(r.ShopName, ''), COALESCE(r.Name, ''), COALESCE(r.Phone, '')
              FROM Retailers r
              WHERE r.RetailerId = @eid
                AND EXISTS (
                    SELECT 1
                    FROM Orders@{FORCE_INDEX=Idx_Orders_SupplierId} o
                    WHERE o.SupplierId = @sid AND o.RetailerId = r.RetailerId
                    LIMIT 1
                )
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "eid": entityID},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, shopName, name, phone string
		if err := row.Columns(&id, &shopName, &name, &phone); err != nil {
			return EntityRecord{}, err
		}
		label := firstNonEmpty(shopName, name, id)
		return EntityRecord{
			EntityType: EntityTypeRetailer,
			EntityID:   id,
			Label:      label,
			SearchText: strings.TrimSpace(shopName + " " + name + " " + phone + " " + id),
			Metadata: map[string]string{
				"phone": phone,
			},
		}, nil
	})
}

func (r *SpannerRepository) findOrderExact(ctx context.Context, supplierID, entityID string) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, COALESCE(RetailerId, ''), COALESCE(DriverId, ''), COALESCE(State, ''),
                     COALESCE(RouteId, ''), COALESCE(InvoiceId, '')
              FROM Orders@{FORCE_INDEX=Idx_Orders_SupplierId}
              WHERE SupplierId = @sid AND OrderId = @eid
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "eid": entityID},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, retailerID, driverID, state, routeID, invoiceID string
		if err := row.Columns(&id, &retailerID, &driverID, &state, &routeID, &invoiceID); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeOrder,
			EntityID:   id,
			Label:      id,
			SearchText: strings.TrimSpace(id + " " + retailerID + " " + driverID + " " + state + " " + routeID + " " + invoiceID),
			Metadata: map[string]string{
				"retailer_id": retailerID,
				"driver_id":   driverID,
				"state":       state,
				"route_id":    routeID,
				"invoice_id":  invoiceID,
			},
		}, nil
	})
}

func (r *SpannerRepository) listSupplierScoped(ctx context.Context, supplierID string) ([]EntityRecord, error) {
	return r.findSupplierExact(ctx, supplierID, supplierID)
}

func (r *SpannerRepository) listWarehouseScoped(ctx context.Context, supplierID string, limit int) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, COALESCE(Name, ''), COALESCE(Address, '')
              FROM Warehouses@{FORCE_INDEX=Idx_Warehouses_BySupplierId}
              WHERE SupplierId = @sid AND IsActive = true
              LIMIT @lim`,
		Params: map[string]interface{}{"sid": supplierID, "lim": int64(limit)},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, name, address string
		if err := row.Columns(&id, &name, &address); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeWarehouse,
			EntityID:   id,
			Label:      firstNonEmpty(name, id),
			SearchText: strings.TrimSpace(name + " " + address + " " + id),
			Metadata:   map[string]string{"address": address},
		}, nil
	})
}

func (r *SpannerRepository) listFactoryScoped(ctx context.Context, supplierID string, limit int) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT FactoryId, COALESCE(Name, ''), COALESCE(Address, '')
              FROM Factories@{FORCE_INDEX=Idx_Factories_BySupplierId}
              WHERE SupplierId = @sid AND IsActive = true
              LIMIT @lim`,
		Params: map[string]interface{}{"sid": supplierID, "lim": int64(limit)},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, name, address string
		if err := row.Columns(&id, &name, &address); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeFactory,
			EntityID:   id,
			Label:      firstNonEmpty(name, id),
			SearchText: strings.TrimSpace(name + " " + address + " " + id),
			Metadata:   map[string]string{"address": address},
		}, nil
	})
}

func (r *SpannerRepository) listDriverScoped(ctx context.Context, supplierID string, limit int) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DriverId, COALESCE(Name, ''), COALESCE(Phone, ''), COALESCE(LicensePlate, ''), COALESCE(VehicleId, '')
              FROM Drivers@{FORCE_INDEX=Idx_Drivers_BySupplierId}
              WHERE SupplierId = @sid
              LIMIT @lim`,
		Params: map[string]interface{}{"sid": supplierID, "lim": int64(limit)},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, name, phone, license, vehicleID string
		if err := row.Columns(&id, &name, &phone, &license, &vehicleID); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeDriver,
			EntityID:   id,
			Label:      firstNonEmpty(name, id),
			SearchText: strings.TrimSpace(name + " " + phone + " " + license + " " + id + " " + vehicleID),
			Metadata: map[string]string{
				"phone":         phone,
				"license_plate": license,
				"vehicle_id":    vehicleID,
			},
		}, nil
	})
}

func (r *SpannerRepository) listVehicleScoped(ctx context.Context, supplierID string, limit int) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT VehicleId, COALESCE(Label, ''), COALESCE(LicensePlate, ''), COALESCE(VehicleClass, '')
              FROM Vehicles@{FORCE_INDEX=Idx_Vehicles_BySupplier}
              WHERE SupplierId = @sid AND IsActive = true
              LIMIT @lim`,
		Params: map[string]interface{}{"sid": supplierID, "lim": int64(limit)},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, label, license, className string
		if err := row.Columns(&id, &label, &license, &className); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeVehicle,
			EntityID:   id,
			Label:      firstNonEmpty(label, license, id),
			SearchText: strings.TrimSpace(label + " " + license + " " + className + " " + id),
			Metadata: map[string]string{
				"license_plate": license,
				"vehicle_class": className,
			},
		}, nil
	})
}

func (r *SpannerRepository) listRetailerScoped(ctx context.Context, supplierID string, limit int) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT r.RetailerId, COALESCE(r.ShopName, ''), COALESCE(r.Name, ''), COALESCE(r.Phone, '')
              FROM Orders@{FORCE_INDEX=Idx_Orders_SupplierId} o
              JOIN Retailers r ON r.RetailerId = o.RetailerId
              WHERE o.SupplierId = @sid
              LIMIT @lim`,
		Params: map[string]interface{}{"sid": supplierID, "lim": int64(limit)},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, shopName, name, phone string
		if err := row.Columns(&id, &shopName, &name, &phone); err != nil {
			return EntityRecord{}, err
		}
		label := firstNonEmpty(shopName, name, id)
		return EntityRecord{
			EntityType: EntityTypeRetailer,
			EntityID:   id,
			Label:      label,
			SearchText: strings.TrimSpace(shopName + " " + name + " " + phone + " " + id),
			Metadata:   map[string]string{"phone": phone},
		}, nil
	})
}

func (r *SpannerRepository) listOrderScoped(ctx context.Context, supplierID string, limit int) ([]EntityRecord, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, COALESCE(RetailerId, ''), COALESCE(DriverId, ''), COALESCE(State, ''), COALESCE(RouteId, ''), COALESCE(InvoiceId, '')
              FROM Orders@{FORCE_INDEX=Idx_Orders_SupplierId}
              WHERE SupplierId = @sid
              LIMIT @lim`,
		Params: map[string]interface{}{"sid": supplierID, "lim": int64(limit)},
	}
	return r.collectRecords(ctx, stmt, func(row *spanner.Row) (EntityRecord, error) {
		var id, retailerID, driverID, state, routeID, invoiceID string
		if err := row.Columns(&id, &retailerID, &driverID, &state, &routeID, &invoiceID); err != nil {
			return EntityRecord{}, err
		}
		return EntityRecord{
			EntityType: EntityTypeOrder,
			EntityID:   id,
			Label:      id,
			SearchText: strings.TrimSpace(id + " " + retailerID + " " + driverID + " " + state + " " + routeID + " " + invoiceID),
			Metadata: map[string]string{
				"retailer_id": retailerID,
				"driver_id":   driverID,
				"state":       state,
				"route_id":    routeID,
				"invoice_id":  invoiceID,
			},
		}, nil
	})
}

func (r *SpannerRepository) loadOrderLineage(ctx context.Context, supplierID, orderID string) ([]LineageLink, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SupplierId, ''), COALESCE(RetailerId, ''), COALESCE(DriverId, ''),
                     COALESCE(WarehouseId, ''), COALESCE(InvoiceId, ''), COALESCE(RouteId, '')
              FROM Orders@{FORCE_INDEX=Idx_Orders_SupplierId}
              WHERE SupplierId = @sid AND OrderId = @oid
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "oid": orderID},
	}

	links := make([]LineageLink, 0, 6)
	found := false
	err := r.forEachRow(ctx, stmt, func(row *spanner.Row) error {
		found = true
		var supplierRef, retailerID, driverID, warehouseID, invoiceID, routeID string
		if err := row.Columns(&supplierRef, &retailerID, &driverID, &warehouseID, &invoiceID, &routeID); err != nil {
			return err
		}
		links = appendLineageLink(links, EntityTypeSupplier, supplierRef, supplierRef, "belongs_to_supplier", 0.99)
		links = appendLineageLink(links, EntityTypeRetailer, retailerID, retailerID, "delivers_to_retailer", 0.98)
		links = appendLineageLink(links, EntityTypeDriver, driverID, driverID, "assigned_to_driver", 0.97)
		links = appendLineageLink(links, EntityTypeWarehouse, warehouseID, warehouseID, "dispatched_from_warehouse", 0.95)
		links = appendLineageLink(links, EntityTypeInvoice, invoiceID, invoiceID, "billed_on_invoice", 0.90)
		links = appendLineageLink(links, EntityTypeRoute, routeID, routeID, "scheduled_on_route", 0.88)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return links, nil
}

func (r *SpannerRepository) loadDriverLineage(ctx context.Context, supplierID, driverID string) ([]LineageLink, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SupplierId, ''), COALESCE(VehicleId, ''), COALESCE(WarehouseId, ''),
                     COALESCE(HomeNodeType, ''), COALESCE(HomeNodeId, '')
              FROM Drivers@{FORCE_INDEX=Idx_Drivers_BySupplierId}
              WHERE SupplierId = @sid AND DriverId = @did
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "did": driverID},
	}
	links := make([]LineageLink, 0, 4)
	found := false
	err := r.forEachRow(ctx, stmt, func(row *spanner.Row) error {
		found = true
		var supplierRef, vehicleID, warehouseID, homeType, homeID string
		if err := row.Columns(&supplierRef, &vehicleID, &warehouseID, &homeType, &homeID); err != nil {
			return err
		}
		links = appendLineageLink(links, EntityTypeSupplier, supplierRef, supplierRef, "belongs_to_supplier", 0.99)
		links = appendLineageLink(links, EntityTypeVehicle, vehicleID, vehicleID, "assigned_vehicle", 0.97)
		links = appendLineageLink(links, EntityTypeWarehouse, warehouseID, warehouseID, "operates_from_warehouse", 0.95)
		if homeID != "" {
			links = appendLineageLink(links, normalizeEntityType(homeType), homeID, homeID, "home_node", 0.95)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return links, nil
}

func (r *SpannerRepository) loadVehicleLineage(ctx context.Context, supplierID, vehicleID string) ([]LineageLink, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SupplierId, ''), COALESCE(WarehouseId, ''), COALESCE(HomeNodeType, ''), COALESCE(HomeNodeId, '')
              FROM Vehicles@{FORCE_INDEX=Idx_Vehicles_BySupplier}
              WHERE SupplierId = @sid AND VehicleId = @vid
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "vid": vehicleID},
	}
	links := make([]LineageLink, 0, 3)
	found := false
	err := r.forEachRow(ctx, stmt, func(row *spanner.Row) error {
		found = true
		var supplierRef, warehouseID, homeType, homeID string
		if err := row.Columns(&supplierRef, &warehouseID, &homeType, &homeID); err != nil {
			return err
		}
		links = appendLineageLink(links, EntityTypeSupplier, supplierRef, supplierRef, "belongs_to_supplier", 0.99)
		links = appendLineageLink(links, EntityTypeWarehouse, warehouseID, warehouseID, "assigned_warehouse", 0.94)
		if homeID != "" {
			links = appendLineageLink(links, normalizeEntityType(homeType), homeID, homeID, "home_node", 0.95)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return links, nil
}

func (r *SpannerRepository) loadWarehouseLineage(ctx context.Context, supplierID, warehouseID string) ([]LineageLink, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SupplierId, ''), COALESCE(Name, '')
              FROM Warehouses@{FORCE_INDEX=Idx_Warehouses_BySupplierId}
              WHERE SupplierId = @sid AND WarehouseId = @wid
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "wid": warehouseID},
	}
	links := make([]LineageLink, 0, 1)
	found := false
	err := r.forEachRow(ctx, stmt, func(row *spanner.Row) error {
		found = true
		var supplierRef, warehouseName string
		if err := row.Columns(&supplierRef, &warehouseName); err != nil {
			return err
		}
		links = appendLineageLink(links, EntityTypeSupplier, supplierRef, supplierRef, "belongs_to_supplier", 0.99)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return links, nil
}

func (r *SpannerRepository) loadFactoryLineage(ctx context.Context, supplierID, factoryID string) ([]LineageLink, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SupplierId, ''), COALESCE(Name, '')
              FROM Factories@{FORCE_INDEX=Idx_Factories_BySupplierId}
              WHERE SupplierId = @sid AND FactoryId = @fid
              LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID, "fid": factoryID},
	}
	links := make([]LineageLink, 0, 1)
	found := false
	err := r.forEachRow(ctx, stmt, func(row *spanner.Row) error {
		found = true
		var supplierRef, factoryName string
		if err := row.Columns(&supplierRef, &factoryName); err != nil {
			return err
		}
		links = appendLineageLink(links, EntityTypeSupplier, supplierRef, supplierRef, "belongs_to_supplier", 0.99)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return links, nil
}

func (r *SpannerRepository) loadRetailerLineage(ctx context.Context, supplierID, retailerID string) ([]LineageLink, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId
              FROM Orders@{FORCE_INDEX=IDX_Orders_RetailerId}
              WHERE RetailerId = @rid AND SupplierId = @sid
              LIMIT 3`,
		Params: map[string]interface{}{"rid": retailerID, "sid": supplierID},
	}

	links := make([]LineageLink, 0, 4)
	found := false
	err := r.forEachRow(ctx, stmt, func(row *spanner.Row) error {
		found = true
		var orderID string
		if err := row.Columns(&orderID); err != nil {
			return err
		}
		links = appendLineageLink(links, EntityTypeOrder, orderID, orderID, "places_order", 0.86)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	links = appendLineageLink(links, EntityTypeSupplier, supplierID, supplierID, "served_by_supplier", 0.95)
	return links, nil
}

func (r *SpannerRepository) loadSupplierLineage(ctx context.Context, supplierID, entityID string) ([]LineageLink, error) {
	if supplierID != entityID {
		return nil, ErrNotFound
	}
	links := make([]LineageLink, 0, 6)

	warehouseStmt := spanner.Statement{
		SQL: `SELECT WarehouseId, COALESCE(Name, '')
              FROM Warehouses@{FORCE_INDEX=Idx_Warehouses_BySupplierId}
              WHERE SupplierId = @sid
              LIMIT 3`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	if err := r.forEachRow(ctx, warehouseStmt, func(row *spanner.Row) error {
		var warehouseID, warehouseName string
		if err := row.Columns(&warehouseID, &warehouseName); err != nil {
			return err
		}
		links = appendLineageLink(links, EntityTypeWarehouse, warehouseID, firstNonEmpty(warehouseName, warehouseID), "operates_warehouse", 0.84)
		return nil
	}); err != nil {
		return nil, err
	}

	factoryStmt := spanner.Statement{
		SQL: `SELECT FactoryId, COALESCE(Name, '')
              FROM Factories@{FORCE_INDEX=Idx_Factories_BySupplierId}
              WHERE SupplierId = @sid
              LIMIT 3`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	if err := r.forEachRow(ctx, factoryStmt, func(row *spanner.Row) error {
		var factoryID, factoryName string
		if err := row.Columns(&factoryID, &factoryName); err != nil {
			return err
		}
		links = appendLineageLink(links, EntityTypeFactory, factoryID, firstNonEmpty(factoryName, factoryID), "operates_factory", 0.84)
		return nil
	}); err != nil {
		return nil, err
	}

	return links, nil
}

func appendLineageLink(links []LineageLink, targetType, targetID, targetLabel, relation string, confidence float64) []LineageLink {
	tid := strings.TrimSpace(targetID)
	if tid == "" {
		return links
	}
	return append(links, LineageLink{
		TargetType:  normalizeEntityType(targetType),
		TargetID:    tid,
		TargetLabel: strings.TrimSpace(firstNonEmpty(targetLabel, tid)),
		Relation:    relation,
		Confidence:  confidence,
	})
}

func (r *SpannerRepository) collectRecords(ctx context.Context, stmt spanner.Statement, scan func(*spanner.Row) (EntityRecord, error)) ([]EntityRecord, error) {
	records := make([]EntityRecord, 0, 8)
	err := r.forEachRow(ctx, stmt, func(row *spanner.Row) error {
		record, err := scan(row)
		if err != nil {
			return err
		}
		records = append(records, record)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (r *SpannerRepository) forEachRow(ctx context.Context, stmt spanner.Statement, fn func(*spanner.Row) error) error {
	if r.client == nil {
		return fmt.Errorf("spanner repository is not initialized")
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		if err := fn(row); err != nil {
			return err
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
