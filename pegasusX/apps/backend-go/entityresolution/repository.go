package entityresolution

import (
	"context"
	"strings"

	"time"
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
		return r.loadRetailerLineage(ctx, supplierID, entityID)
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
		return r.findRetailersExact(ctx, supplierID, entityID)
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
		return r.listRetailers(ctx, supplierID, limit)
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
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
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
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
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
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
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

func (r *SpannerRepository) findRetailersExact(ctx context.Context, supplierID, retailerID string) ([]EntityRecord, error) {
	links, err := querySpannerRetailerLinks(ctx, r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)), spanner.Statement{
		SQL: `SELECT ` + retailerLinkSelectCols + `
			FROM SupplierRetailerLinks
			WHERE SupplierId = @sid AND RetailerId = @eid
			LIMIT 1`,
		Params: map[string]any{"sid": supplierID, "eid": retailerID},
	})
	if err != nil {
		return nil, err
	}
	fallback, err := r.collect3(ctx, spanner.Statement{
		SQL: `SELECT r.RetailerId, COALESCE(r.Name, r.RetailerId), COALESCE(r.Phone, '')
		      FROM Retailers r
		      WHERE r.RetailerId = @eid
		        AND EXISTS (SELECT 1 FROM Orders o WHERE o.RetailerId = r.RetailerId AND o.SupplierId = @sid)
		      LIMIT 1`,
		Params: map[string]any{"sid": supplierID, "eid": retailerID},
	}, EntityTypeRetailer, "phone")
	if err != nil {
		return nil, err
	}
	return mergeRetailerRecords(links, fallback, 1), nil
}

func (r *SpannerRepository) listRetailers(ctx context.Context, supplierID string, limit int) ([]EntityRecord, error) {
	links, err := querySpannerRetailerLinks(ctx, r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)), spanner.Statement{
		SQL: `SELECT ` + retailerLinkSelectCols + `
			FROM SupplierRetailerLinks
			WHERE SupplierId = @sid
			ORDER BY UpdatedAt DESC, RetailerId ASC
			LIMIT @lim`,
		Params: map[string]any{"sid": supplierID, "lim": int64(limit)},
	})
	if err != nil {
		return nil, err
	}
	fallback, err := r.collect3(ctx, spanner.Statement{
		SQL: `SELECT r.RetailerId, COALESCE(r.Name, r.RetailerId), COALESCE(r.Phone, '')
		      FROM Retailers r
		      WHERE EXISTS (SELECT 1 FROM Orders o WHERE o.RetailerId = r.RetailerId AND o.SupplierId = @sid)
		      LIMIT @lim`,
		Params: map[string]any{"sid": supplierID, "lim": int64(limit)},
	}, EntityTypeRetailer, "phone")
	if err != nil {
		return nil, err
	}
	return mergeRetailerRecords(links, fallback, limit), nil
}

// FindRetailersByIdentifier queries SupplierRetailerLinks strictly within SupplierId
// using the normalized exact TIN and UZ phone indexes.
func (r *SpannerRepository) FindRetailersByIdentifier(ctx context.Context, supplierID, normalizedTIN, normalizedPhone string, limit int) ([]EntityRecord, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	sid := strings.TrimSpace(supplierID)
	tin := NormalizeTIN(normalizedTIN)
	phone := NormalizeUZPhone(normalizedPhone)
	if sid == "" || (tin == "" && phone == "") {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultPerTypeLimit
	}
	links, err := querySpannerRetailerLinks(ctx, r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)), spanner.Statement{
		SQL: `SELECT ` + retailerLinkSelectCols + `
			FROM SupplierRetailerLinks
			WHERE SupplierId = @sid
			  AND (
			    (@tin != '' AND TaxIdTIN = @tin)
			    OR (@phone != '' AND NormalizedPhone = @phone)
			  )
			ORDER BY UpdatedAt DESC, RetailerId ASC
			LIMIT @lim`,
		Params: map[string]any{
			"sid":   sid,
			"tin":   tin,
			"phone": phone,
			"lim":   int64(limit),
		},
	})
	if err != nil {
		return nil, err
	}
	return mergeRetailerRecords(links, nil, limit), nil
}

func (r *SpannerRepository) loadRetailerLineage(ctx context.Context, supplierID, entityID string) ([]LineageLink, error) {
	links, err := r.queryLinks(ctx, `SELECT 'ORDER', OrderId, OrderId, 'PLACED', 0.85
		FROM Orders WHERE SupplierId = @sid AND RetailerId = @eid ORDER BY CreatedAt DESC LIMIT 8`, supplierID, entityID)
	if err != nil {
		return nil, err
	}
	mergedLinks, err := r.queryLinks(ctx, `SELECT 'RETAILER', COALESCE(MergedIntoId, ''), COALESCE(MergedIntoId, ''), 'MERGED_INTO', 0.99
		FROM SupplierRetailerLinks WHERE SupplierId = @sid AND RetailerId = @eid AND MergedIntoId IS NOT NULL AND MergedIntoId != '' LIMIT 1`, supplierID, entityID)
	if err != nil {
		return nil, err
	}
	return append(links, mergedLinks...), nil
}

func retailerLinkToEntityRecord(link SupplierRetailerLink) EntityRecord {
	label := strings.TrimSpace(link.LegalName)
	if label == "" {
		label = link.RetailerID
	}
	meta := make(map[string]string, 6)
	if link.TaxIDTIN != "" {
		meta["tax_id_tin"] = link.TaxIDTIN
	}
	if link.NormalizedPhone != "" {
		meta["normalized_phone"] = link.NormalizedPhone
		meta["phone"] = link.NormalizedPhone
	}
	if link.DedupStatus != "" {
		meta["dedup_status"] = link.DedupStatus
	}
	if link.DedupClusterID != "" {
		meta["dedup_cluster_id"] = link.DedupClusterID
	}
	if link.MergedIntoID != "" {
		meta["merged_into_id"] = link.MergedIntoID
	}
	search := strings.TrimSpace(strings.Join([]string{
		label,
		link.TaxIDTIN,
		link.NormalizedPhone,
		link.RetailerID,
	}, " "))
	return rec(EntityTypeRetailer, link.RetailerID, label, search, meta)
}

func mergeRetailerRecords(links []SupplierRetailerLink, fallback []EntityRecord, limit int) []EntityRecord {
	byID := make(map[string]EntityRecord, len(links)+len(fallback))
	order := make([]string, 0, len(links)+len(fallback))
	for _, link := range links {
		record := retailerLinkToEntityRecord(link)
		if _, exists := byID[record.EntityID]; !exists {
			order = append(order, record.EntityID)
		}
		byID[record.EntityID] = record
	}
	for _, fb := range fallback {
		existing, exists := byID[fb.EntityID]
		if !exists {
			if fb.Metadata != nil && fb.Metadata["phone"] != "" {
				fb.Metadata["normalized_phone"] = NormalizeUZPhone(fb.Metadata["phone"])
			}
			byID[fb.EntityID] = fb
			order = append(order, fb.EntityID)
			continue
		}
		if (existing.Label == "" || existing.Label == existing.EntityID) && fb.Label != "" {
			existing.Label = fb.Label
		}
		if existing.Metadata["phone"] == "" && fb.Metadata["phone"] != "" {
			if existing.Metadata == nil {
				existing.Metadata = map[string]string{}
			}
			existing.Metadata["phone"] = fb.Metadata["phone"]
			existing.Metadata["normalized_phone"] = NormalizeUZPhone(fb.Metadata["phone"])
		}
		byID[fb.EntityID] = existing
	}
	out := make([]EntityRecord, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
