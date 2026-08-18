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
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
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
	audits []outbox.AuditEntry
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (b *spannerTxnBuffer) BufferAudit(_ context.Context, e outbox.AuditEntry) error {
	b.audits = append(b.audits, e)
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
		"MarketCode",
		"HomeCell",
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
	var marketCode, homeCell spanner.NullString
	if err := row.Columns(
		&p.SupplierID,
		&p.LegalName,
		&p.Country,
		&p.Currency,
		&p.IsConfigured,
		&marketCode,
		&homeCell,
		&p.RegisteredAt,
		&p.UpdatedAt,
	); err != nil {
		return Profile{}, false, fmt.Errorf("scan supplier %s: %w", supplierID, err)
	}
	p.MarketCode = nullString(marketCode)
	p.HomeCell = nullString(homeCell)

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
		"PaymentAcceptor",
		"Gln",
		"Gs1CompanyPrefix",
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
	var configuredAt, registeredAt spanner.NullTime
	var contactName, email, phone spanner.NullString
	var warehouseName, warehouseAddress, billingAddress spanner.NullString
	var taxID, companyRegNumber spanner.NullString
	var bankName, accountHolder, accountNumber, swiftBic, iban spanner.NullString
	var paymentAcceptor, gln, gs1Prefix spanner.NullString

	if err := profileRow.Columns(
		new(string),
		&contactName,
		&email,
		&phone,
		&warehouseName,
		&warehouseAddress,
		&warehouseLat,
		&warehouseLng,
		&p.BillingSameAsWh,
		&billingAddress,
		&billingLat,
		&billingLng,
		&taxID,
		&companyRegNumber,
		&fleetVehicleCount,
		&fleetMaxVU,
		&factoryCount,
		&categoriesJSON,
		&p.IsRegistered,
		&bankName,
		&accountHolder,
		&accountNumber,
		&swiftBic,
		&iban,
		&selectedGatewaysJSON,
		&paymentAcceptor,
		&gln,
		&gs1Prefix,
		&registeredAt,
		&configuredAt,
		&p.UpdatedAt,
	); err != nil {
		return Profile{}, false, fmt.Errorf("scan supplier profile %s: %w", supplierID, err)
	}
	p.ContactName = contactName.StringVal
	p.Email = email.StringVal
	p.Phone = phone.StringVal
	p.WarehouseName = warehouseName.StringVal
	p.WarehouseAddress = warehouseAddress.StringVal
	p.BillingAddress = billingAddress.StringVal
	p.TaxID = taxID.StringVal
	p.CompanyRegNumber = companyRegNumber.StringVal
	p.BankName = bankName.StringVal
	p.AccountHolder = accountHolder.StringVal
	p.AccountNumber = accountNumber.StringVal
	p.SwiftBic = swiftBic.StringVal
	p.IBAN = iban.StringVal
	if paymentAcceptor.Valid {
		p.PaymentAcceptor = paymentAcceptor.StringVal
	}
	if gln.Valid {
		p.Gln = gln.StringVal
	}
	if gs1Prefix.Valid {
		p.Gs1CompanyPrefix = gs1Prefix.StringVal
	}
	if registeredAt.Valid {
		p.RegisteredAt = registeredAt.Time
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
	if paymentAcceptor.Valid {
		p.PaymentAcceptor = strings.TrimSpace(paymentAcceptor.StringVal)
	}

	return p, true, nil
}

// CountOrders returns the number of orders for a supplier.
func (r *SpannerRepository) CountOrders(ctx context.Context, supplierID string) (int, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("spanner supplier repository: nil client")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return 0, nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM Orders WHERE SupplierId = @supplierId`,
		Params: map[string]any{"supplierId": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("count supplier orders: %w", err)
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, fmt.Errorf("scan supplier order count: %w", err)
	}
	return int(count), nil
}

// ListOrders returns recent supplier-scoped orders with durable assignment fields.
func (r *SpannerRepository) ListOrders(ctx context.Context, supplierID string, limit, offset int) ([]SupplierOrder, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner supplier repository: nil client")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return []SupplierOrder{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, RetailerId,
		             COALESCE(WarehouseId, ''), COALESCE(DriverId, ''), COALESCE(VehicleId, ''),
		             COALESCE(RouteId, ''), COALESCE(ManifestId, ''), Status, ConfirmationStatus,
		             TotalMinor, Currency, CreatedAt, UpdatedAt
		      FROM Orders@{FORCE_INDEX=Idx_Orders_BySupplierUpdated}
		      WHERE SupplierId = @supplierId
		      ORDER BY UpdatedAt DESC
		      LIMIT @limit
		      OFFSET @offset`,
		Params: map[string]any{
			"supplierId": supplierID,
			"limit":      int64(limit),
			"offset":     int64(offset),
		},
	}

	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	orders := make([]SupplierOrder, 0, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return orders, nil
		}
		if err != nil {
			return nil, fmt.Errorf("query supplier orders: %w", err)
		}

		current, err := decodeSupplierOrder(row)
		if err != nil {
			return nil, err
		}
		orders = append(orders, current)
	}
}

func decodeSupplierOrder(row *spanner.Row) (SupplierOrder, error) {
	var (
		order              SupplierOrder
		confirmationStatus string
		createdAt          time.Time
		updatedAt          time.Time
	)
	if err := row.Columns(
		&order.OrderID,
		&order.SupplierID,
		&order.RetailerID,
		&order.WarehouseID,
		&order.DriverID,
		&order.VehicleID,
		&order.RouteID,
		&order.ManifestID,
		&order.Status,
		&confirmationStatus,
		&order.TotalMinor,
		&order.Currency,
		&createdAt,
		&updatedAt,
	); err != nil {
		return SupplierOrder{}, fmt.Errorf("scan supplier order: %w", err)
	}

	order.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	order.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	applySupplierOrderPresentation(&order, confirmationStatus, order.Status)
	order.LiveLocationAvailable = false
	return order, nil
}

// GetTopology returns supplier warehouse and factory nodes.
func (r *SpannerRepository) GetTopology(ctx context.Context, supplierID string) (SupplierTopology, error) {
	if r == nil || r.client == nil {
		return SupplierTopology{}, fmt.Errorf("spanner supplier repository: nil client")
	}

	result := SupplierTopology{}

	warehouseStmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Name, Lat, Lng, Address, PlaceId, CoverageRadiusKm, IsActive, IsOnShift,
		      TransferMode, CoLocateWithFactoryId, PrimaryFactoryId, SecondaryFactoryId,
		      COALESCE(DefaultOutOfStockPolicy, 'REJECT'), OperatingSchedule,
		      COALESCE(CountryCode, ''), CreatedAt, UpdatedAt
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
		var address, placeID spanner.NullString
		var transferMode, coLocate, primaryFactory, secondaryFactory spanner.NullString
		var policy spanner.NullString
		var schedule spanner.NullJSON
		var country string
		if err := row.Columns(
			&node.WarehouseID,
			&node.Name,
			&lat,
			&lng,
			&address,
			&placeID,
			&coverage,
			&node.IsActive,
			&node.IsOnShift,
			&transferMode,
			&coLocate,
			&primaryFactory,
			&secondaryFactory,
			&policy,
			&schedule,
			&country,
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
		if address.Valid {
			node.Address = address.StringVal
		}
		if placeID.Valid {
			node.PlaceID = placeID.StringVal
		}
		if coverage.Valid {
			node.CoverageRadiusKm = coverage.Float64
		}
		if transferMode.Valid {
			node.TransferMode = normalizeTransferMode(transferMode.StringVal)
		} else {
			node.TransferMode = TransferModeTruck
		}
		if coLocate.Valid {
			node.CoLocateWithFactoryID = coLocate.StringVal
		}
		if primaryFactory.Valid {
			node.PrimaryFactoryID = primaryFactory.StringVal
		}
		if secondaryFactory.Valid {
			node.SecondaryFactoryID = secondaryFactory.StringVal
		}
		node.CountryCode = strings.ToUpper(strings.TrimSpace(country))
		if policy.Valid {
			node.DefaultOutOfStockPolicy = normalizeOutOfStockPolicy(policy.StringVal)
		}
		if schedule.Valid {
			if raw, err := json.Marshal(schedule.Value); err == nil {
				node.OperatingSchedule = string(raw)
			}
		}
		result.Warehouses = append(result.Warehouses, node)
	}

	factoryStmt := spanner.Statement{
		SQL: `SELECT FactoryId, Name, Lat, Lng, Address, PlaceId, IsActive,
		      COALESCE(CountryCode, ''), CreatedAt, UpdatedAt
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
		var address, placeID spanner.NullString
		var country string
		if err := row.Columns(
			&node.FactoryID,
			&node.Name,
			&lat,
			&lng,
			&address,
			&placeID,
			&node.IsActive,
			&country,
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
		if address.Valid {
			node.Address = address.StringVal
		}
		if placeID.Valid {
			node.PlaceID = placeID.StringVal
		}
		node.CountryCode = strings.ToUpper(strings.TrimSpace(country))
		result.Factories = append(result.Factories, node)
	}

	if err := r.attachTopologyGraph(ctx, supplierID, &result); err != nil {
		return SupplierTopology{}, err
	}
	return result, nil
}

func (r *SpannerRepository) attachTopologyGraph(ctx context.Context, supplierID string, result *SupplierTopology) error {
	if r == nil || r.client == nil || result == nil {
		return nil
	}
	cityIter := r.client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT WarehouseId, CityName, Lat, Lng FROM WarehouseCoverageCities WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": supplierID},
	})
	defer cityIter.Stop()
	cities := map[string][]order.CoverageCity{}
	for {
		row, err := cityIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query warehouse coverage cities: %w", err)
		}
		var wid, name string
		var lat, lng float64
		if err := row.Columns(&wid, &name, &lat, &lng); err != nil {
			continue
		}
		cities[wid] = append(cities[wid], order.CoverageCity{Name: name, Lat: lat, Lng: lng})
	}

	laneIter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, FactoryId, COALESCE(Priority, 0)
		      FROM SupplyLanes WHERE SupplierId = @sid AND COALESCE(IsActive, true) = true
		      ORDER BY WarehouseId, Priority, FactoryId`,
		Params: map[string]any{"sid": supplierID},
	})
	defer laneIter.Stop()
	lanes := map[string][]string{}
	secondary := map[string]string{}
	for {
		row, err := laneIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query supply lanes: %w", err)
		}
		var wid, fid string
		var priority int64
		if err := row.Columns(&wid, &fid, &priority); err != nil {
			continue
		}
		lanes[wid] = append(lanes[wid], fid)
		if priority == 1 && secondary[wid] == "" {
			secondary[wid] = fid
		}
	}

	for i := range result.Warehouses {
		wid := result.Warehouses[i].WarehouseID
		result.Warehouses[i].CoverageCities = cities[wid]
		result.Warehouses[i].AssignedFactoryIDs = lanes[wid]
		if result.Warehouses[i].SecondaryFactoryID == "" {
			result.Warehouses[i].SecondaryFactoryID = secondary[wid]
		}
	}
	return nil
}

// GetPricingRule fetches one supplier pricing authority row.
func (r *SpannerRepository) GetPricingRule(ctx context.Context, supplierID string) (SupplierPricingRule, bool, error) {
	if r == nil || r.client == nil {
		return SupplierPricingRule{}, false, fmt.Errorf("spanner supplier repository: nil client")
	}

	row, err := r.client.Single().ReadRow(ctx, "SupplierPricingRules", spanner.Key{supplierID}, []string{
		"SupplierId",
		"BaseMarkupBps",
		"RetailerDiscountBps",
		"MinMarginBps",
		"Currency",
		"RuleVersion",
		"UpdatedBy",
		"UpdatedAt",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return SupplierPricingRule{}, false, nil
		}
		return SupplierPricingRule{}, false, fmt.Errorf("read supplier pricing rule %s: %w", supplierID, err)
	}

	var (
		rule      SupplierPricingRule
		updatedBy spanner.NullString
	)
	if err := row.Columns(
		&rule.SupplierID,
		&rule.BaseMarkupBps,
		&rule.RetailerDiscountBps,
		&rule.MinMarginBps,
		&rule.Currency,
		&rule.RuleVersion,
		&updatedBy,
		&rule.UpdatedAt,
	); err != nil {
		return SupplierPricingRule{}, false, fmt.Errorf("scan supplier pricing rule %s: %w", supplierID, err)
	}
	if updatedBy.Valid {
		rule.UpdatedBy = updatedBy.StringVal
	}

	return rule, true, nil
}

// UpsertPricingRule updates supplier pricing authority and emitted outbox
// events atomically in one RW transaction.
func (r *SpannerRepository) UpsertPricingRule(ctx context.Context, rule SupplierPricingRule, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner supplier repository: nil client")
	}
	if strings.TrimSpace(rule.SupplierID) == "" {
		return fmt.Errorf("supplier_id required")
	}
	if len(strings.TrimSpace(rule.Currency)) != 3 {
		return fmt.Errorf("currency must be ISO-4217")
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		currentVersion := int64(0)
		versionRow, err := txn.ReadRow(ctx, "SupplierPricingRules", spanner.Key{rule.SupplierID}, []string{"RuleVersion"})
		if err != nil {
			if !errors.Is(err, spanner.ErrRowNotFound) {
				return fmt.Errorf("read pricing rule version %s: %w", rule.SupplierID, err)
			}
		} else {
			if err := versionRow.Columns(&currentVersion); err != nil {
				return fmt.Errorf("scan pricing rule version %s: %w", rule.SupplierID, err)
			}
		}

		nextVersion := currentVersion + 1
		if nextVersion <= 0 {
			nextVersion = 1
		}
		if rule.RuleVersion > currentVersion {
			nextVersion = rule.RuleVersion
		}

		updatedAt := rule.UpdatedAt.UTC()
		if updatedAt.IsZero() {
			updatedAt = time.Now().UTC()
		}

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("SupplierPricingRules", map[string]any{
				"SupplierId":          rule.SupplierID,
				"BaseMarkupBps":       rule.BaseMarkupBps,
				"RetailerDiscountBps": rule.RetailerDiscountBps,
				"MinMarginBps":        rule.MinMarginBps,
				"Currency":            strings.ToUpper(strings.TrimSpace(rule.Currency)),
				"RuleVersion":         nextVersion,
				"UpdatedBy":           strings.TrimSpace(rule.UpdatedBy),
				"UpdatedAt":           updatedAt,
			}),
		}

		for _, e := range buf.events {
			mutations = append(mutations, portalOutboxMutation(e))
		}
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("upsert supplier pricing rule transaction: %w", err)
	}

	return nil
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
		SQL: `SELECT su.UserId, su.SupplierId, su.Phone, su.PasswordHash,
                     COALESCE(sp.IsRegistered, false), COALESCE(s.IsConfigured, false)
              FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_ByPhone} su
              LEFT JOIN SupplierProfiles sp ON su.SupplierId = sp.SupplierId
              LEFT JOIN Suppliers s ON su.SupplierId = s.SupplierId
              WHERE su.Phone = @phone AND su.IsActive = true AND su.SupplierRole = 'ADMIN'
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
	if err := row.Columns(&rec.UserID, &rec.SupplierID, &rec.Phone, &rec.PasswordHash, &rec.IsRegistered, &rec.IsConfigured); err != nil {
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
			"PaymentAcceptor":        normalizeSupplierPaymentAcceptor(p.PaymentAcceptor),
			"Gln":                    strings.TrimSpace(p.Gln),
			"Gs1CompanyPrefix":       strings.TrimSpace(p.Gs1CompanyPrefix),
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
				"MarketCode":   nullableString(strings.ToUpper(strings.TrimSpace(p.MarketCode))),
				"HomeCell":     nullableString(strings.ToLower(strings.TrimSpace(p.HomeCell))),
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
			mutations = append(mutations, portalOutboxMutation(e))
		}
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
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
			SQL:    `DELETE FROM WarehouseCoverageCells WHERE SupplierId = @supplierId`,
			Params: map[string]any{"supplierId": supplierID},
		}); err != nil {
			return fmt.Errorf("delete warehouse coverage cells: %w", err)
		}
		if _, err := txn.Update(ctx, spanner.Statement{
			SQL:    `DELETE FROM WarehouseCoverageCities WHERE SupplierId = @supplierId`,
			Params: map[string]any{"supplierId": supplierID},
		}); err != nil {
			return fmt.Errorf("delete warehouse coverage cities: %w", err)
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

		factoryIDs := make([]string, 0, len(topology.Factories))
		for i, fc := range topology.Factories {
			id := strings.TrimSpace(fc.FactoryID)
			if id == "" {
				name := strings.TrimSpace(fc.Name)
				if name == "" {
					name = fmt.Sprintf("Factory %d", i+1)
				}
				id = stableTopologyID("factory", supplierID, name, i+1)
			}
			factoryIDs = append(factoryIDs, id)
		}
		primaryFactoryID := ""
		if len(factoryIDs) == 1 {
			primaryFactoryID = factoryIDs[0]
		}

		for i, fc := range topology.Factories {
			id := factoryIDs[i]
			name := strings.TrimSpace(fc.Name)
			if name == "" {
				name = fmt.Sprintf("Factory %d", i+1)
			}
			createdAt := fc.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = now
			}

			factoryCountry := strings.ToUpper(strings.TrimSpace(fc.CountryCode))
			if factoryCountry == "" {
				return fmt.Errorf("factory country_code required")
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("Factories", map[string]any{
				"FactoryId":   id,
				"SupplierId":  supplierID,
				"Name":        name,
				"Lat":         nullableFloat(fc.Lat),
				"Lng":         nullableFloat(fc.Lng),
				"H3Cell":      topologyH3CellString(fc.Lat, fc.Lng),
				"Address":     nullableString(strings.TrimSpace(fc.Address)),
				"PlaceId":     nullableString(strings.TrimSpace(fc.PlaceID)),
				"CountryCode": factoryCountry,
				"IsActive":    fc.IsActive,
				"CreatedAt":   createdAt,
				"UpdatedAt":   now,
			}))
		}

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

			row := map[string]any{
				"WarehouseId":             id,
				"SupplierId":              supplierID,
				"Name":                    name,
				"Lat":                     nullableFloat(wh.Lat),
				"Lng":                     nullableFloat(wh.Lng),
				"H3Cell":                  topologyH3CellString(wh.Lat, wh.Lng),
				"Address":                 nullableString(strings.TrimSpace(wh.Address)),
				"PlaceId":                 nullableString(strings.TrimSpace(wh.PlaceID)),
				"CoverageRadiusKm":        coverage,
				"TransferMode":            normalizeTransferMode(wh.TransferMode),
				"IsActive":                wh.IsActive,
				"IsOnShift":               wh.IsOnShift,
				"DefaultOutOfStockPolicy": normalizeOutOfStockPolicy(wh.DefaultOutOfStockPolicy),
				"CreatedAt":               createdAt,
				"UpdatedAt":               now,
			}
			if coLocate := strings.TrimSpace(wh.CoLocateWithFactoryID); coLocate != "" {
				row["CoLocateWithFactoryId"] = coLocate
			}
			primary := strings.TrimSpace(wh.PrimaryFactoryID)
			if primary == "" {
				primary = primaryFactoryID
			}
			if primary != "" {
				row["PrimaryFactoryId"] = primary
			}
			if sec := strings.TrimSpace(wh.SecondaryFactoryID); sec != "" {
				row["SecondaryFactoryId"] = sec
			}
			if cc := strings.ToUpper(strings.TrimSpace(wh.CountryCode)); cc != "" {
				row["CountryCode"] = cc
			} else {
				return fmt.Errorf("warehouse country_code required")
			}
			if sched := strings.TrimSpace(wh.OperatingSchedule); sched != "" {
				row["OperatingSchedule"] = spanner.NullJSON{Value: json.RawMessage(sched), Valid: true}
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("Warehouses", row))
			laneIDs := append([]string(nil), wh.AssignedFactoryIDs...)
			if primary != "" {
				laneIDs = append([]string{primary}, laneIDs...)
			}
			if sec := strings.TrimSpace(wh.SecondaryFactoryID); sec != "" {
				laneIDs = append(laneIDs, sec)
			}
			mutations = append(mutations, order.SupplyLaneMutations(supplierID, id, laneIDs, now)...)
			mutations = append(mutations, order.CoverageMutations(supplierID, id, wh.CoverageCities, now)...)

			for _, seed := range wh.InitialInventory {
				if seed.Quantity <= 0 || strings.TrimSpace(seed.ProductID) == "" {
					continue
				}
				mutations = append(mutations, spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
					"SupplierId":       supplierID,
					"WarehouseId":      id,
					"ProductId":        strings.TrimSpace(seed.ProductID),
					"QuantityOnHand":   seed.Quantity,
					"QuantityReserved": int64(0),
					"UpdatedAt":        now,
				}))
			}
		}

		for _, e := range buf.events {
			mutations = append(mutations, portalOutboxMutation(e))
		}
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
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

// CountSuppliers returns the number of supplier tenant rows.
func (r *SpannerRepository) CountSuppliers(ctx context.Context) (int64, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("spanner supplier repository: nil client")
	}
	stmt := spanner.Statement{SQL: `SELECT COUNT(*) FROM Suppliers`}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("count suppliers: %w", err)
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, fmt.Errorf("scan supplier count: %w", err)
	}
	return count, nil
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

func normalizeSupplierPaymentAcceptor(acceptor string) string {
	switch strings.ToUpper(strings.TrimSpace(acceptor)) {
	case PaymentAcceptorWarehouse:
		return PaymentAcceptorWarehouse
	default:
		return PaymentAcceptorSupplier
	}
}

func topologyH3CellString(lat, lng float64) any {
	if lat == 0 && lng == 0 {
		return nil
	}
	cell := proximity.MatchingH3Cell(lat, lng)
	if cell == "" {
		return nil
	}
	return cell
}
