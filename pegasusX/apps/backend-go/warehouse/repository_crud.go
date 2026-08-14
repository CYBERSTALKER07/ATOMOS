package warehouse

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// Warehouse represents the warehouse entity.
type Warehouse struct {
	WarehouseID                string    `json:"warehouse_id" spanner:"WarehouseId"`
	SupplierID                 string    `json:"supplier_id" spanner:"SupplierId"`
	Name                       string    `json:"name" spanner:"Name"`
	Lat                        *float64  `json:"lat,omitempty" spanner:"Lat"`
	Lng                        *float64  `json:"lng,omitempty" spanner:"Lng"`
	Address                    *string   `json:"address,omitempty" spanner:"Address"`
	PlaceID                    *string   `json:"place_id,omitempty" spanner:"PlaceId"`
	CoverageRadiusKm           float64   `json:"coverage_radius_km" spanner:"CoverageRadiusKm"`
	PrimaryFactoryID           *string   `json:"primary_factory_id,omitempty" spanner:"PrimaryFactoryId"`
	SecondaryFactoryID         *string   `json:"secondary_factory_id,omitempty" spanner:"SecondaryFactoryId"`
	TransferMode               string    `json:"transfer_mode" spanner:"TransferMode"`
	CoLocateWithFactoryID      *string   `json:"co_locate_with_factory_id,omitempty" spanner:"CoLocateWithFactoryId"`
	IsActive                   bool      `json:"is_active" spanner:"IsActive"`
	IsOnShift                  bool      `json:"is_on_shift" spanner:"IsOnShift"`
	RegionID                   *string   `json:"region_id,omitempty" spanner:"RegionId"`
	PaymentConfigID            *string   `json:"payment_config_id,omitempty" spanner:"PaymentConfigId"`
	AutoDispatchEnabled        bool      `json:"auto_dispatch_enabled" spanner:"AutoDispatchEnabled"`
	DefaultOutOfStockPolicy    string    `json:"default_out_of_stock_policy" spanner:"DefaultOutOfStockPolicy"`
	ShowStockCountsToRetailers bool      `json:"show_stock_counts_to_retailers" spanner:"ShowStockCountsToRetailers"`
	PreorderMinLeadDays        int64     `json:"preorder_min_lead_days" spanner:"PreorderMinLeadDays"`
	PreorderMaxLeadDays        int64     `json:"preorder_max_lead_days" spanner:"PreorderMaxLeadDays"`
	OrderLineMinQuantity       *int64    `json:"order_line_min_quantity,omitempty" spanner:"OrderLineMinQuantity"`
	OrderLineMaxQuantity       *int64    `json:"order_line_max_quantity,omitempty" spanner:"OrderLineMaxQuantity"`
	DeliveryFeeRules           *string   `json:"delivery_fee_rules,omitempty" spanner:"DeliveryFeeRules"`
	OperatingSchedule          *string   `json:"operating_schedule,omitempty" spanner:"OperatingSchedule"`
	CreatedAt                  time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt                  time.Time `json:"updated_at" spanner:"UpdatedAt"`
	H3Cell                     *string   `json:"h3_cell,omitempty" spanner:"H3Cell"`
	Gln                        *string   `json:"gln,omitempty" spanner:"Gln"`
	CountryCode                string    `json:"country_code,omitempty" spanner:"CountryCode"`
	CoverageCities             []order.CoverageCity `json:"coverage_cities,omitempty" spanner:"-"`
	AssignedFactoryIDs         []string  `json:"assigned_factory_ids,omitempty" spanner:"-"`
}

// CreateWarehouse inserts a new warehouse record and emits a WAREHOUSE_CREATED event atomically.
func (r *SpannerRepository) CreateWarehouse(ctx context.Context, w Warehouse, emit func(outbox.TxnBuffer) error) error {
	w.CreatedAt = spanner.CommitTimestamp
	w.UpdatedAt = spanner.CommitTimestamp
	m, err := spanner.InsertStruct("Warehouses", w)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{m}
		muts = append(muts, warehouseGraphMutations(w)...)
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

func warehouseGraphMutations(w Warehouse) []*spanner.Mutation {
	laneIDs := append([]string(nil), w.AssignedFactoryIDs...)
	if w.PrimaryFactoryID != nil {
		laneIDs = append(laneIDs, strings.TrimSpace(*w.PrimaryFactoryID))
	}
	if w.SecondaryFactoryID != nil {
		laneIDs = append(laneIDs, strings.TrimSpace(*w.SecondaryFactoryID))
	}
	muts := order.CoverageMutations(w.SupplierID, w.WarehouseID, w.CoverageCities, time.Time{})
	return append(muts, order.SupplyLaneMutations(w.SupplierID, w.WarehouseID, laneIDs, time.Time{})...)
}
func (r *SpannerRepository) GetWarehouse(ctx context.Context, warehouseID string) (Warehouse, error) {
	row, err := r.client.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{
		"WarehouseId", "SupplierId", "Name", "Lat", "Lng", "Address", "PlaceId",
		"CoverageRadiusKm", "PrimaryFactoryId", "SecondaryFactoryId", "TransferMode",
		"CoLocateWithFactoryId", "IsActive", "IsOnShift", "RegionId", "PaymentConfigId",
		"AutoDispatchEnabled", "DefaultOutOfStockPolicy", "ShowStockCountsToRetailers",
		"PreorderMinLeadDays", "PreorderMaxLeadDays", "OrderLineMinQuantity", "OrderLineMaxQuantity",
		"DeliveryFeeRules", "OperatingSchedule", "CreatedAt", "UpdatedAt", "H3Cell", "Gln", "CountryCode",
	})
	if err != nil {
		return Warehouse{}, err
	}
	var w Warehouse
	if err := row.ToStruct(&w); err != nil {
		return Warehouse{}, err
	}
	w.CoverageCities = r.loadCoverageCities(ctx, w.WarehouseID)
	return w, nil
}

func (r *SpannerRepository) loadCoverageCities(ctx context.Context, warehouseID string) []order.CoverageCity {
	if r == nil || r.client == nil || strings.TrimSpace(warehouseID) == "" {
		return nil
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT CityName, Lat, Lng FROM WarehouseCoverageCities WHERE WarehouseId = @wid ORDER BY CityName`,
		Params: map[string]any{"wid": warehouseID},
	})
	defer iter.Stop()
	var out []order.CoverageCity
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out
		}
		var name string
		var lat, lng float64
		if err := row.Columns(&name, &lat, &lng); err != nil {
			continue
		}
		out = append(out, order.CoverageCity{Name: name, Lat: lat, Lng: lng})
	}
	return out
}

// UpdateWarehouse updates an existing warehouse record and emits a WAREHOUSE_LOCATION_UPDATED event atomically.
func (r *SpannerRepository) UpdateWarehouse(ctx context.Context, w Warehouse, emit func(outbox.TxnBuffer) error) error {
	w.UpdatedAt = spanner.CommitTimestamp
	m, err := spanner.UpdateStruct("Warehouses", w)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{m}
		muts = append(muts, warehouseGraphMutations(w)...)
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

// ListWarehouses returns a list of warehouses filtered by supplier ID.
func (r *SpannerRepository) ListWarehouses(ctx context.Context, supplierID string, limit, offset int) ([]Warehouse, error) {
	stmt := spanner.Statement{
		SQL: `SELECT * FROM Warehouses WHERE SupplierId = @supplierId LIMIT @limit OFFSET @offset`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"limit":      limit,
			"offset":     offset,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var warehouses []Warehouse
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var w Warehouse
		if err := row.ToStruct(&w); err != nil {
			return nil, err
		}
		warehouses = append(warehouses, w)
	}
	return warehouses, nil
}
