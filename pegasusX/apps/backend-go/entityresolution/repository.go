package entityresolution

import (
	"context"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const defaultPerTypeLimit = 12

type SpannerRepository struct {
	client *spanner.Client
}

func NewRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) FindExactByID(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
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

func (r *SpannerRepository) ListScopedRecords(ctx context.Context, supplierID, entityType string, perTypeLimit int) ([]EntityRecord, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
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

func (r *SpannerRepository) LoadLineage(ctx context.Context, supplierID, entityType, entityID string) ([]LineageLink, error) {
	if r == nil || r.client == nil {
		return nil, ErrInvalidInput
	}
	supplierID = strings.TrimSpace(supplierID)
	entityID = strings.TrimSpace(entityID)
	if supplierID == "" || entityID == "" {
		return nil, ErrInvalidInput
	}
	switch normalizeEntityType(entityType) {
	case EntityTypeOrder:
		return r.queryLinks(ctx, `SELECT 'WAREHOUSE', COALESCE(WarehouseId, ''), COALESCE(WarehouseId, ''), 'FULFILLED_FROM', 0.9
			FROM Orders WHERE SupplierId = @sid AND OrderId = @eid LIMIT 1`, supplierID, entityID)
	case EntityTypeWarehouse:
		return r.queryLinks(ctx, `SELECT 'FACTORY', COALESCE(PrimaryFactoryId, ''), COALESCE(PrimaryFactoryId, ''), 'PRIMARY_FACTORY', 0.8
			FROM Warehouses WHERE SupplierId = @sid AND WarehouseId = @eid LIMIT 1`, supplierID, entityID)
	case EntityTypeRetailer:
		return r.queryLinks(ctx, `SELECT 'ORDER', OrderId, OrderId, 'PLACED', 0.85
			FROM Orders WHERE SupplierId = @sid AND RetailerId = @eid ORDER BY CreatedAt DESC LIMIT 8`, supplierID, entityID)
	case EntityTypeDriver:
		return r.queryLinks(ctx, `SELECT 'VEHICLE', COALESCE(VehicleId, ''), COALESCE(VehicleId, ''), 'ASSIGNED_VEHICLE', 0.8
			FROM Drivers WHERE SupplierId = @sid AND DriverId = @eid LIMIT 1`, supplierID, entityID)
	case EntityTypeFactory:
		return r.queryLinks(ctx, `SELECT 'WAREHOUSE', WarehouseId, Name, 'SERVES', 0.7
			FROM Warehouses WHERE SupplierId = @sid AND (PrimaryFactoryId = @eid OR SecondaryFactoryId = @eid) LIMIT 8`, supplierID, entityID)
	case EntityTypeVehicle:
		return r.queryLinks(ctx, `SELECT 'DRIVER', DriverId, Name, 'DRIVEN_BY', 0.8
			FROM Drivers WHERE SupplierId = @sid AND VehicleId = @eid LIMIT 8`, supplierID, entityID)
	case EntityTypeSupplier:
		return r.queryLinks(ctx, `SELECT 'WAREHOUSE', WarehouseId, Name, 'OWNS', 0.99
			FROM Warehouses WHERE SupplierId = @sid LIMIT 8`, supplierID, entityID)
	default:
		return nil, ErrInvalidInput
	}
}

func (r *SpannerRepository) findExactByType(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error) {
	switch entityType {
	case EntityTypeSupplier:
		return r.collect4(ctx, spanner.Statement{
			SQL: `SELECT s.SupplierId, s.Name, COALESCE(p.Email, ''), COALESCE(p.Phone, '')
			      FROM Suppliers s
			      LEFT JOIN SupplierProfiles p ON p.SupplierId = s.SupplierId
			      WHERE s.SupplierId = @sid AND s.SupplierId = @eid LIMIT 1`,
			Params: map[string]any{"sid": supplierID, "eid": entityID},
		}, EntityTypeSupplier, "email", "phone")
	case EntityTypeWarehouse:
		return r.collect3(ctx, spanner.Statement{
			SQL:    `SELECT WarehouseId, Name, COALESCE(Address, '') FROM Warehouses WHERE SupplierId = @sid AND WarehouseId = @eid LIMIT 1`,
			Params: map[string]any{"sid": supplierID, "eid": entityID},
		}, EntityTypeWarehouse, "address")
	case EntityTypeFactory:
		return r.collect3(ctx, spanner.Statement{
			SQL:    `SELECT FactoryId, Name, COALESCE(Address, '') FROM Factories WHERE SupplierId = @sid AND FactoryId = @eid LIMIT 1`,
			Params: map[string]any{"sid": supplierID, "eid": entityID},
		}, EntityTypeFactory, "address")
	case EntityTypeDriver:
		return r.collect3(ctx, spanner.Statement{
			SQL:    `SELECT DriverId, Name, COALESCE(Phone, '') FROM Drivers WHERE SupplierId = @sid AND DriverId = @eid LIMIT 1`,
			Params: map[string]any{"sid": supplierID, "eid": entityID},
		}, EntityTypeDriver, "phone")
	case EntityTypeVehicle:
		return r.collect3(ctx, spanner.Statement{
			SQL:    `SELECT VehicleId, COALESCE(Label, LicensePlate), LicensePlate FROM Vehicles WHERE SupplierId = @sid AND VehicleId = @eid LIMIT 1`,
			Params: map[string]any{"sid": supplierID, "eid": entityID},
		}, EntityTypeVehicle, "plate")
	case EntityTypeRetailer:
		return r.collect3(ctx, spanner.Statement{
			SQL: `SELECT r.RetailerId, COALESCE(r.Name, r.RetailerId), COALESCE(r.Phone, '')
			      FROM Retailers r
			      WHERE r.RetailerId = @eid
			        AND EXISTS (SELECT 1 FROM Orders o WHERE o.RetailerId = r.RetailerId AND o.SupplierId = @sid)
			      LIMIT 1`,
			Params: map[string]any{"sid": supplierID, "eid": entityID},
		}, EntityTypeRetailer, "phone")
	case EntityTypeOrder:
		return r.collect3(ctx, spanner.Statement{
			SQL:    `SELECT OrderId, OrderId, Status FROM Orders WHERE SupplierId = @sid AND OrderId = @eid LIMIT 1`,
			Params: map[string]any{"sid": supplierID, "eid": entityID},
		}, EntityTypeOrder, "status")
	default:
		return nil, ErrInvalidInput
	}
}

func (r *SpannerRepository) listByType(ctx context.Context, supplierID, entityType string, limit int) ([]EntityRecord, error) {
	switch entityType {
	case EntityTypeSupplier:
		return r.findExactByType(ctx, supplierID, EntityTypeSupplier, supplierID)
	case EntityTypeWarehouse:
		return r.collect3(ctx, spanner.Statement{
			SQL:    `SELECT WarehouseId, Name, COALESCE(Address, '') FROM Warehouses WHERE SupplierId = @sid LIMIT @lim`,
			Params: map[string]any{"sid": supplierID, "lim": int64(limit)},
		}, EntityTypeWarehouse, "address")
	case EntityTypeFactory:
		return r.collect3(ctx, spanner.Statement{
			SQL:    `SELECT FactoryId, Name, COALESCE(Address, '') FROM Factories WHERE SupplierId = @sid LIMIT @lim`,
			Params: map[string]any{"sid": supplierID, "lim": int64(limit)},
		}, EntityTypeFactory, "address")
	case EntityTypeDriver:
		return r.collect3(ctx, spanner.Statement{
			SQL:    `SELECT DriverId, Name, COALESCE(Phone, '') FROM Drivers WHERE SupplierId = @sid LIMIT @lim`,
			Params: map[string]any{"sid": supplierID, "lim": int64(limit)},
		}, EntityTypeDriver, "phone")
	case EntityTypeVehicle:
		return r.collect3(ctx, spanner.Statement{
			SQL:    `SELECT VehicleId, COALESCE(Label, LicensePlate), LicensePlate FROM Vehicles WHERE SupplierId = @sid LIMIT @lim`,
			Params: map[string]any{"sid": supplierID, "lim": int64(limit)},
		}, EntityTypeVehicle, "plate")
	case EntityTypeRetailer:
		return r.collect3(ctx, spanner.Statement{
			SQL: `SELECT r.RetailerId, COALESCE(r.Name, r.RetailerId), COALESCE(r.Phone, '')
			      FROM Retailers r
			      WHERE EXISTS (SELECT 1 FROM Orders o WHERE o.RetailerId = r.RetailerId AND o.SupplierId = @sid)
			      LIMIT @lim`,
			Params: map[string]any{"sid": supplierID, "lim": int64(limit)},
		}, EntityTypeRetailer, "phone")
	case EntityTypeOrder:
		return r.collect3(ctx, spanner.Statement{
			SQL:    `SELECT OrderId, OrderId, Status FROM Orders WHERE SupplierId = @sid ORDER BY CreatedAt DESC LIMIT @lim`,
			Params: map[string]any{"sid": supplierID, "lim": int64(limit)},
		}, EntityTypeOrder, "status")
	default:
		return nil, ErrInvalidInput
	}
}

func rec(typ, id, label, search string, meta map[string]string) EntityRecord {
	if strings.TrimSpace(label) == "" {
		label = id
	}
	return EntityRecord{EntityType: typ, EntityID: id, Label: label, SearchText: strings.TrimSpace(search), Metadata: meta}
}

func (r *SpannerRepository) collect3(ctx context.Context, stmt spanner.Statement, typ, extraKey string) ([]EntityRecord, error) {
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []EntityRecord
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var id, name, extra string
		if err := row.Columns(&id, &name, &extra); err != nil {
			continue
		}
		out = append(out, rec(typ, id, name, name+" "+extra+" "+id, map[string]string{extraKey: extra}))
	}
}

func (r *SpannerRepository) collect4(ctx context.Context, stmt spanner.Statement, typ, k1, k2 string) ([]EntityRecord, error) {
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []EntityRecord
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var id, name, extra, extra2 string
		if err := row.Columns(&id, &name, &extra, &extra2); err != nil {
			continue
		}
		out = append(out, rec(typ, id, name, name+" "+extra+" "+extra2+" "+id, map[string]string{k1: extra, k2: extra2}))
	}
}

func (r *SpannerRepository) queryLinks(ctx context.Context, sql, supplierID, entityID string) ([]LineageLink, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: map[string]any{"sid": supplierID, "eid": entityID},
	})
	defer iter.Stop()
	var out []LineageLink
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var typ, id, label, rel string
		var conf float64
		if err := row.Columns(&typ, &id, &label, &rel, &conf); err != nil {
			continue
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if strings.TrimSpace(label) == "" {
			label = id
		}
		out = append(out, LineageLink{
			TargetType:  typ,
			TargetID:    id,
			TargetLabel: label,
			Relation:    rel,
			Confidence:  conf,
		})
	}
}
