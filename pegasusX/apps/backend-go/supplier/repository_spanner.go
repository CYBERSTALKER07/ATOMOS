package supplier

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// SpannerRepository persists supplier rows in Spanner and writes emitted outbox
// events in the same RW transaction.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository builds a Spanner-backed supplier repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

// GetProfile fetches one supplier profile by id.
func (r *SpannerRepository) GetProfile(ctx context.Context, supplierID string) (Profile, bool, error) {
	if r == nil || r.client == nil {
		return Profile{}, false, fmt.Errorf("spanner supplier repository: nil client")
	}

	row, err := r.client.Single().ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{
		"SupplierId",
		"Name",
		"CountryCode",
		"Currency",
		"IsConfigured",
		"CreatedAt",
		"UpdatedAt",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return Profile{}, false, nil
		}
		return Profile{}, false, fmt.Errorf("read supplier %s: %w", supplierID, err)
	}

	var p Profile
	if err := row.Columns(
		&p.SupplierID,
		&p.LegalName,
		&p.Country,
		&p.Currency,
		&p.IsConfigured,
		&p.RegisteredAt,
		&p.UpdatedAt,
	); err != nil {
		return Profile{}, false, fmt.Errorf("scan supplier %s: %w", supplierID, err)
	}

	profileRow, err := r.client.Single().ReadRow(ctx, "SupplierProfiles", spanner.Key{supplierID}, []string{
		"SupplierId",
		"ContactName",
		"Email",
		"Phone",
		"WarehouseName",
		"WarehouseAddress",
		"WarehouseLat",
		"WarehouseLng",
		"BillingSameAsWarehouse",
		"BillingAddress",
		"BillingLat",
		"BillingLng",
		"TaxId",
		"CompanyRegNumber",
		"FleetVehicleCount",
		"FleetMaxVU",
		"FactoryCount",
		"CategoriesJson",
		"IsRegistered",
		"BankName",
		"AccountHolder",
		"AccountNumber",
		"SwiftBic",
		"IBAN",
		"SelectedGatewaysJson",
		"RegisteredAt",
		"ConfiguredAt",
		"UpdatedAt",
	})
	if err != nil {
		if !errors.Is(err, spanner.ErrRowNotFound) {
			return Profile{}, false, fmt.Errorf("read supplier profile %s: %w", supplierID, err)
		}
		return p, true, nil
	}

	var warehouseLat, warehouseLng spanner.NullFloat64
	var billingLat, billingLng spanner.NullFloat64
	var fleetVehicleCount, fleetMaxVU, factoryCount spanner.NullInt64
	var categoriesJSON, selectedGatewaysJSON []byte
	var configuredAt spanner.NullTime

	if err := profileRow.Columns(
		new(string),
		&p.ContactName,
		&p.Email,
		&p.Phone,
		&p.WarehouseName,
		&p.WarehouseAddress,
		&warehouseLat,
		&warehouseLng,
		&p.BillingSameAsWh,
		&p.BillingAddress,
		&billingLat,
		&billingLng,
		&p.TaxID,
		&p.CompanyRegNumber,
		&fleetVehicleCount,
		&fleetMaxVU,
		&factoryCount,
		&categoriesJSON,
		&p.IsRegistered,
		&p.BankName,
		&p.AccountHolder,
		&p.AccountNumber,
		&p.SwiftBic,
		&p.IBAN,
		&selectedGatewaysJSON,
		&p.RegisteredAt,
		&configuredAt,
		&p.UpdatedAt,
	); err != nil {
		return Profile{}, false, fmt.Errorf("scan supplier profile %s: %w", supplierID, err)
	}

	if warehouseLat.Valid {
		p.WarehouseLat = warehouseLat.Float64
	}
	if warehouseLng.Valid {
		p.WarehouseLng = warehouseLng.Float64
	}
	if billingLat.Valid {
		p.BillingLat = billingLat.Float64
	}
	if billingLng.Valid {
		p.BillingLng = billingLng.Float64
	}
	if fleetVehicleCount.Valid {
		p.FleetVehicleCount = int(fleetVehicleCount.Int64)
	}
	if fleetMaxVU.Valid {
		p.FleetMaxVU = int(fleetMaxVU.Int64)
	}
	if factoryCount.Valid {
		p.FactoryCount = int(factoryCount.Int64)
	}
	if configuredAt.Valid {
		p.ConfiguredAt = configuredAt.Time
	}

	if categories, err := decodeStringSlice(categoriesJSON); err == nil {
		p.Categories = categories
	}
	if gateways, err := decodeStringSlice(selectedGatewaysJSON); err == nil {
		p.SelectedGateways = gateways
	}

	return p, true, nil
}

// GetTopology returns supplier warehouse and factory nodes.
func (r *SpannerRepository) GetTopology(ctx context.Context, supplierID string) (SupplierTopology, error) {
	if r == nil || r.client == nil {
		return SupplierTopology{}, fmt.Errorf("spanner supplier repository: nil client")
	}

	result := SupplierTopology{}

	warehouseStmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Name, Lat, Lng, CoverageRadiusKm, IsActive, IsOnShift, CreatedAt, UpdatedAt
		      FROM Warehouses
		      WHERE SupplierId = @supplierId
		      ORDER BY WarehouseId`,
		Params: map[string]any{"supplierId": supplierID},
	}
	warehouseIter := r.client.Single().Query(ctx, warehouseStmt)
	defer warehouseIter.Stop()
	for {
		row, err := warehouseIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return SupplierTopology{}, fmt.Errorf("query supplier warehouses: %w", err)
		}

		var node WarehouseNode
		var lat, lng, coverage spanner.NullFloat64
		if err := row.Columns(
			&node.WarehouseID,
			&node.Name,
			&lat,
			&lng,
			&coverage,
			&node.IsActive,
			&node.IsOnShift,
			&node.CreatedAt,
			&node.UpdatedAt,
		); err != nil {
			return SupplierTopology{}, fmt.Errorf("scan supplier warehouse: %w", err)
		}
		if lat.Valid {
			node.Lat = lat.Float64
		}
		if lng.Valid {
			node.Lng = lng.Float64
		}
		if coverage.Valid {
			node.CoverageRadiusKm = coverage.Float64
		}
		result.Warehouses = append(result.Warehouses, node)
	}

	factoryStmt := spanner.Statement{
		SQL: `SELECT FactoryId, Name, Lat, Lng, IsActive, CreatedAt, UpdatedAt
		      FROM Factories
		      WHERE SupplierId = @supplierId
		      ORDER BY FactoryId`,
		Params: map[string]any{"supplierId": supplierID},
	}
	factoryIter := r.client.Single().Query(ctx, factoryStmt)
	defer factoryIter.Stop()
	for {
		row, err := factoryIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return SupplierTopology{}, fmt.Errorf("query supplier factories: %w", err)
		}

		var node FactoryNode
		var lat, lng spanner.NullFloat64
		if err := row.Columns(
			&node.FactoryID,
			&node.Name,
			&lat,
			&lng,
			&node.IsActive,
			&node.CreatedAt,
			&node.UpdatedAt,
		); err != nil {
			return SupplierTopology{}, fmt.Errorf("scan supplier factory: %w", err)
		}
		if lat.Valid {
			node.Lat = lat.Float64
		}
		if lng.Valid {
			node.Lng = lng.Float64
		}
		result.Factories = append(result.Factories, node)
	}

	return result, nil
}

// GetAuthByPhone fetches one active supplier-user credential by phone.
func (r *SpannerRepository) GetAuthByPhone(ctx context.Context, phone string) (SupplierAuthRecord, bool, error) {
	if r == nil || r.client == nil {
		return SupplierAuthRecord{}, false, fmt.Errorf("spanner supplier repository: nil client")
	}
	trimmedPhone := strings.TrimSpace(phone)
	if trimmedPhone == "" {
		return SupplierAuthRecord{}, false, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT su.UserId, su.SupplierId, su.Phone, su.PasswordHash, COALESCE(s.IsConfigured, false)
              FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_ByPhone} su
              LEFT JOIN Suppliers s ON su.SupplierId = s.SupplierId
              WHERE su.Phone = @phone AND su.IsActive = true
              LIMIT 1`,
		Params: map[string]any{"phone": trimmedPhone},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return SupplierAuthRecord{}, false, nil
	}
	if err != nil {
		return SupplierAuthRecord{}, false, fmt.Errorf("query supplier auth by phone: %w", err)
	}

	var rec SupplierAuthRecord
	if err := row.Columns(&rec.UserID, &rec.SupplierID, &rec.Phone, &rec.PasswordHash, &rec.IsConfigured); err != nil {
		return SupplierAuthRecord{}, false, fmt.Errorf("scan supplier auth by phone: %w", err)
	}
	return rec, true, nil
}

// UpdateProfile updates supplier and profile rows atomically and creates a
// bootstrap topology when none exists.
func (r *SpannerRepository) UpdateProfile(ctx context.Context, p Profile, emit func(outbox.TxnBuffer) error) error {
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

		now := time.Now().UTC()
		registeredAt := p.RegisteredAt.UTC()
		if registeredAt.IsZero() {
			registeredAt = now
		}
		updatedAt := p.UpdatedAt.UTC()
		if updatedAt.IsZero() {
			updatedAt = now
		}

		categoriesJSON, err := encodeStringSlice(p.Categories)
		if err != nil {
			return fmt.Errorf("encode categories: %w", err)
		}
		selectedGatewaysJSON, err := encodeStringSlice(p.SelectedGateways)
		if err != nil {
			return fmt.Errorf("encode selected gateways: %w", err)
		}

		profileRow := map[string]any{
			"SupplierId":             p.SupplierID,
			"ContactName":            strings.TrimSpace(p.ContactName),
			"Email":                  strings.TrimSpace(p.Email),
			"Phone":                  strings.TrimSpace(p.Phone),
			"WarehouseName":          strings.TrimSpace(p.WarehouseName),
			"WarehouseAddress":       strings.TrimSpace(p.WarehouseAddress),
			"WarehouseLat":           nullableFloat(p.WarehouseLat),
			"WarehouseLng":           nullableFloat(p.WarehouseLng),
			"BillingSameAsWarehouse": p.BillingSameAsWh,
			"BillingAddress":         strings.TrimSpace(p.BillingAddress),
			"BillingLat":             nullableFloat(p.BillingLat),
			"BillingLng":             nullableFloat(p.BillingLng),
			"TaxId":                  strings.TrimSpace(p.TaxID),
			"CompanyRegNumber":       strings.TrimSpace(p.CompanyRegNumber),
			"FleetVehicleCount":      int64(p.FleetVehicleCount),
			"FleetMaxVU":             int64(p.FleetMaxVU),
			"FactoryCount":           int64(p.FactoryCount),
			"CategoriesJson":         categoriesJSON,
			"IsRegistered":           p.IsRegistered,
			"BankName":               strings.TrimSpace(p.BankName),
			"AccountHolder":          strings.TrimSpace(p.AccountHolder),
			"AccountNumber":          strings.TrimSpace(p.AccountNumber),
			"SwiftBic":               strings.TrimSpace(p.SwiftBic),
			"IBAN":                   strings.TrimSpace(p.IBAN),
			"SelectedGatewaysJson":   selectedGatewaysJSON,
			"RegisteredAt":           registeredAt,
			"ConfiguredAt":           nullableTime(p.ConfiguredAt),
			"UpdatedAt":              updatedAt,
		}

		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("Suppliers", map[string]any{
				"SupplierId":   p.SupplierID,
				"Name":         p.LegalName,
				"CountryCode":  p.Country,
				"Currency":     p.Currency,
				"IsConfigured": p.IsConfigured,
				"CreatedAt":    registeredAt,
				"UpdatedAt":    updatedAt,
			}),
			spanner.InsertOrUpdateMap("SupplierProfiles", profileRow),
		}

		if strings.TrimSpace(p.AuthPasswordHash) != "" && strings.TrimSpace(p.Phone) != "" {
			userName := strings.TrimSpace(p.ContactName)
			if userName == "" {
				userName = strings.TrimSpace(p.LegalName)
			}
			userID := strings.TrimSpace(p.AuthUserID)
			if userID == "" {
				userID = rootSupplierUserID(p.SupplierID)
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
				"UserId":              userID,
				"SupplierId":          p.SupplierID,
				"Email":               strings.TrimSpace(p.Email),
				"Phone":               strings.TrimSpace(p.Phone),
				"Name":                userName,
				"PasswordHash":        p.AuthPasswordHash,
				"SupplierRole":        "ADMIN",
				"AssignedWarehouseId": nil,
				"AssignedFactoryId":   nil,
				"IsActive":            true,
				"FirebaseUid":         "",
				"CreatedAt":           registeredAt,
				"UpdatedAt":           updatedAt,
			}))
		}

		warehouseCount, err := countTopologyRows(ctx, txn, "Warehouses", p.SupplierID)
		if err != nil {
			return err
		}
		if warehouseCount == 0 && (p.WarehouseLat != 0 || p.WarehouseLng != 0) {
			warehouseName := strings.TrimSpace(p.WarehouseName)
			if warehouseName == "" {
				warehouseName = "Primary Warehouse"
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("Warehouses", map[string]any{
				"WarehouseId":      stableTopologyID("warehouse", p.SupplierID, warehouseName, 1),
				"SupplierId":       p.SupplierID,
				"Name":             warehouseName,
				"Lat":              p.WarehouseLat,
				"Lng":              p.WarehouseLng,
				"CoverageRadiusKm": defaultCoverageRadiusKm,
				"IsActive":         true,
				"IsOnShift":        true,
				"CreatedAt":        registeredAt,
				"UpdatedAt":        updatedAt,
			}))
		}

		factoryCount, err := countTopologyRows(ctx, txn, "Factories", p.SupplierID)
		if err != nil {
			return err
		}
		if factoryCount == 0 && p.FactoryCount > 0 {
			for i := 0; i < p.FactoryCount; i++ {
				name := fmt.Sprintf("Factory %d", i+1)
				mutations = append(mutations, spanner.InsertOrUpdateMap("Factories", map[string]any{
					"FactoryId":  stableTopologyID("factory", p.SupplierID, name, i+1),
					"SupplierId": p.SupplierID,
					"Name":       name,
					"Lat":        nullableFloat(p.WarehouseLat),
					"Lng":        nullableFloat(p.WarehouseLng),
					"IsActive":   true,
					"CreatedAt":  registeredAt,
					"UpdatedAt":  updatedAt,
				}))
			}
		}

		for _, e := range buf.events {
			createdAt := e.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = now
			}

			row := map[string]any{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     createdAt,
				"PublishedAt":   nil,
			}
			if e.PublishedAt != nil {
				row["PublishedAt"] = e.PublishedAt.UTC()
			}

			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("update supplier transaction: %w", err)
	}

	return nil
}

// ReplaceTopology atomically replaces supplier warehouse/factory nodes.
func (r *SpannerRepository) ReplaceTopology(ctx context.Context, supplierID string, topology SupplierTopology, emit func(outbox.TxnBuffer) error) error {
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

		if _, err := txn.Update(ctx, spanner.Statement{
			SQL:    `DELETE FROM Warehouses WHERE SupplierId = @supplierId`,
			Params: map[string]any{"supplierId": supplierID},
		}); err != nil {
			return fmt.Errorf("delete supplier warehouses: %w", err)
		}
		if _, err := txn.Update(ctx, spanner.Statement{
			SQL:    `DELETE FROM Factories WHERE SupplierId = @supplierId`,
			Params: map[string]any{"supplierId": supplierID},
		}); err != nil {
			return fmt.Errorf("delete supplier factories: %w", err)
		}

		now := time.Now().UTC()
		mutations := make([]*spanner.Mutation, 0, len(topology.Warehouses)+len(topology.Factories)+len(buf.events))

		for i, wh := range topology.Warehouses {
			id := strings.TrimSpace(wh.WarehouseID)
			if id == "" {
				id = stableTopologyID("warehouse", supplierID, wh.Name, i+1)
			}
			name := strings.TrimSpace(wh.Name)
			if name == "" {
				name = fmt.Sprintf("Warehouse %d", i+1)
			}
			coverage := wh.CoverageRadiusKm
			if coverage <= 0 {
				coverage = defaultCoverageRadiusKm
			}
			createdAt := wh.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = now
			}

			mutations = append(mutations, spanner.InsertOrUpdateMap("Warehouses", map[string]any{
				"WarehouseId":      id,
				"SupplierId":       supplierID,
				"Name":             name,
				"Lat":              nullableFloat(wh.Lat),
				"Lng":              nullableFloat(wh.Lng),
				"CoverageRadiusKm": coverage,
				"IsActive":         wh.IsActive,
				"IsOnShift":        wh.IsOnShift,
				"CreatedAt":        createdAt,
				"UpdatedAt":        now,
			}))
		}

		for i, fc := range topology.Factories {
			id := strings.TrimSpace(fc.FactoryID)
			if id == "" {
				id = stableTopologyID("factory", supplierID, fc.Name, i+1)
			}
			name := strings.TrimSpace(fc.Name)
			if name == "" {
				name = fmt.Sprintf("Factory %d", i+1)
			}
			createdAt := fc.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = now
			}

			mutations = append(mutations, spanner.InsertOrUpdateMap("Factories", map[string]any{
				"FactoryId":  id,
				"SupplierId": supplierID,
				"Name":       name,
				"Lat":        nullableFloat(fc.Lat),
				"Lng":        nullableFloat(fc.Lng),
				"IsActive":   fc.IsActive,
				"CreatedAt":  createdAt,
				"UpdatedAt":  now,
			}))
		}

		for _, e := range buf.events {
			createdAt := e.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = now
			}
			row := map[string]any{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     createdAt,
				"PublishedAt":   nil,
			}
			if e.PublishedAt != nil {
				row["PublishedAt"] = e.PublishedAt.UTC()
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
		}

		if len(mutations) == 0 {
			return nil
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("replace supplier topology transaction: %w", err)
	}

	return nil
}

func countTopologyRows(ctx context.Context, txn *spanner.ReadWriteTransaction, table string, supplierID string) (int64, error) {
	stmt := spanner.Statement{
		SQL:    fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE SupplierId = @supplierId", table),
		Params: map[string]any{"supplierId": supplierID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("count %s for supplier %s: %w", table, supplierID, err)
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, fmt.Errorf("scan %s count for supplier %s: %w", table, supplierID, err)
	}
	return count, nil
}

func encodeStringSlice(values []string) ([]byte, error) {
	if len(values) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(values)
}

func decodeStringSlice(payload []byte) ([]string, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func stableTopologyID(kind string, supplierID string, name string, index int) string {
	seed := fmt.Sprintf("%s|%s|%s|%d", kind, supplierID, strings.ToLower(strings.TrimSpace(name)), index)
	sum := sha256.Sum256([]byte(seed))
	b := sum[:16]
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		uint64(b[10])<<40|uint64(b[11])<<32|uint64(b[12])<<24|uint64(b[13])<<16|uint64(b[14])<<8|uint64(b[15]),
	)
}

func nullableFloat(value float64) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
}
