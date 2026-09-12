package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
	"github.com/pegasusx/pegasusx/apps/backend-go/payload"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"google.golang.org/api/iterator"
)

// driverHistoryListQuery returns completed-window driver history from Orders.
func driverHistoryListQuery(client *spanner.Client) driver.DriverHistoryQuery {
	if client == nil {
		return nil
	}
	return func(ctx context.Context, driverID string, since time.Time, limit int) ([]driver.HistoryRow, error) {
		driverID = strings.TrimSpace(driverID)
		if driverID == "" {
			return nil, nil
		}
		if limit <= 0 || limit > 100 {
			limit = 50
		}
		stmt := spanner.Statement{
			SQL: `SELECT o.OrderId, o.Status, o.TotalMinor, o.Currency, o.UpdatedAt
			      FROM Orders@{FORCE_INDEX=Idx_Orders_ByDriverCreated} o
			      WHERE o.DriverId = @did AND o.CreatedAt >= @since
			        AND o.Status IN ('COMPLETED', 'FISCALIZING', 'FISCAL_FAILED')
			      ORDER BY o.CreatedAt DESC
			      LIMIT @lim`,
			Params: map[string]any{
				"did":   driverID,
				"since": since,
				"lim":   int64(limit),
			},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		rows := make([]driver.HistoryRow, 0)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("driver history list: %w", err)
			}
			var rec driver.HistoryRow
			var updatedAt time.Time
			if err := row.Columns(&rec.OrderID, &rec.Status, &rec.TotalMinor, &rec.Currency, &updatedAt); err != nil {
				return nil, fmt.Errorf("driver history scan: %w", err)
			}
			rec.CompletedAt = updatedAt.UTC().Format(time.RFC3339Nano)
			rows = append(rows, rec)
		}
		return rows, nil
	}
}

// driverOrderListQuery returns a DriverOrderQuery backed by stale Spanner reads.
func driverOrderListQuery(client *spanner.Client) driver.DriverOrderQuery {
	return func(ctx context.Context, driverID string) ([]driver.DriverOrderView, error) {
		stmt := spanner.Statement{
			SQL: `SELECT o.OrderId, o.RetailerId, COALESCE(r.Name, o.RetailerId), o.Status,
			             o.TotalMinor, o.DeliveryFeeMinor, o.Lat, o.Lng, COALESCE(o.RouteId, ''),
			             o.LineItemsJson, o.CreatedAt, o.UpdatedAt,
			             COALESCE(mo.SequenceIndex, 0),
			             CASE WHEN (SELECT COUNT(DISTINCT ManifestId) FROM ManifestOrders WHERE OrderId = o.OrderId) > 1 THEN o.OrderId ELSE "" END
			      FROM Orders o
			      LEFT JOIN Retailers r ON r.RetailerId = o.RetailerId
			      LEFT JOIN ManifestOrders mo ON mo.ManifestId = o.ManifestId AND mo.OrderId = o.OrderId
			      WHERE o.DriverId = @did AND o.Status NOT IN ('COMPLETED', 'CANCELLED')
			      ORDER BY COALESCE(mo.SequenceIndex, 999999) ASC, o.CreatedAt ASC`,
			Params: map[string]interface{}{"did": driverID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		var orders []driver.DriverOrderView
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("driver order list: %w", err)
			}
			var o driver.DriverOrderView
			var lat, lng spanner.NullFloat64
			var lineItems []byte
			var createdAt, updatedAt time.Time
			if err := row.Columns(
				&o.OrderID, &o.RetailerID, &o.RetailerName, &o.Status, &o.TotalMinor, &o.DeliveryFeeMinor,
				&lat, &lng, &o.RouteID, &lineItems, &createdAt, &updatedAt, &o.SequenceIndex, &o.SplitGroupID,
			); err != nil {
				return nil, fmt.Errorf("driver order scan: %w", err)
			}
			if lat.Valid {
				o.Lat = lat.Float64
			}
			if lng.Valid {
				o.Lng = lng.Float64
			}
			o.Items = decodeDriverOrderLineItems(lineItems)
			o.CreatedAt = createdAt.Format(time.RFC3339Nano)
			o.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
			orders = append(orders, o)
		}
		if orders == nil {
			orders = []driver.DriverOrderView{}
		}
		return orders, nil
	}
}

func driverProfileLookupQuery(client *spanner.Client) driver.DriverProfileLookup {
	return func(ctx context.Context, driverID string) (driver.DriverProfileSnapshot, bool, error) {
		row, err := client.Single().ReadRow(ctx, "Drivers", spanner.Key{driverID},
			[]string{"VehicleId"})
		if err != nil {
			return driver.DriverProfileSnapshot{}, false, nil
		}
		var vehicleID spanner.NullString
		if err := row.Columns(&vehicleID); err != nil {
			return driver.DriverProfileSnapshot{}, false, err
		}
		return driver.DriverProfileSnapshot{
			VehicleID: strings.TrimSpace(vehicleID.StringVal),
		}, true, nil
	}
}

func decodeDriverOrderLineItems(raw []byte) []driver.DriverOrderLineView {
	if len(raw) == 0 {
		return nil
	}
	var source []struct {
		SKU       string `json:"sku_id"`
		Name      string `json:"name"`
		Quantity  int64  `json:"quantity"`
		UnitPrice int64  `json:"unit_price"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil
	}
	items := make([]driver.DriverOrderLineView, 0, len(source))
	for _, item := range source {
		items = append(items, driver.DriverOrderLineView{
			ProductID:   item.SKU,
			ProductName: item.Name,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
		})
	}
	return items
}

// driverOrderGetQuery returns a DriverOrderGetQuery backed by Spanner.
func driverOrderGetQuery(client *spanner.Client) driver.DriverOrderGetQuery {
	return func(ctx context.Context, orderID string) (driver.DriverOrderView, bool, error) {
		stmt := spanner.Statement{
			SQL: `SELECT o.OrderId, o.RetailerId, COALESCE(r.Name, o.RetailerId), o.Status,
			             o.TotalMinor, o.DeliveryFeeMinor, o.Lat, o.Lng, COALESCE(o.RouteId, ''),
			             o.LineItemsJson, o.CreatedAt, o.UpdatedAt,
			             COALESCE(mo.SequenceIndex, 0),
			             CASE WHEN (SELECT COUNT(DISTINCT ManifestId) FROM ManifestOrders WHERE OrderId = o.OrderId) > 1 THEN o.OrderId ELSE "" END,
			             COALESCE(o.DriverId, '')
			      FROM Orders o
			      LEFT JOIN Retailers r ON r.RetailerId = o.RetailerId
			      LEFT JOIN ManifestOrders mo ON mo.ManifestId = o.ManifestId AND mo.OrderId = o.OrderId
			      WHERE o.OrderId = @oid`,
			Params: map[string]interface{}{"oid": orderID},
		}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return driver.DriverOrderView{}, false, nil
		}
		if err != nil {
			return driver.DriverOrderView{}, false, fmt.Errorf("driver order get: %w", err)
		}
		var o driver.DriverOrderView
		var lat, lng spanner.NullFloat64
		var lineItems []byte
		var createdAt, updatedAt time.Time
		if err := row.Columns(
			&o.OrderID, &o.RetailerID, &o.RetailerName, &o.Status, &o.TotalMinor, &o.DeliveryFeeMinor,
			&lat, &lng, &o.RouteID, &lineItems, &createdAt, &updatedAt, &o.SequenceIndex, &o.SplitGroupID,
			&o.AssignedDriverID,
		); err != nil {
			return driver.DriverOrderView{}, false, fmt.Errorf("driver order get scan: %w", err)
		}
		if lat.Valid {
			o.Lat = lat.Float64
		}
		if lng.Valid {
			o.Lng = lng.Float64
		}
		o.Items = decodeDriverOrderLineItems(lineItems)
		o.CreatedAt = createdAt.Format(time.RFC3339Nano)
		o.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		return o, true, nil
	}
}

func driverRouteGeometryQuery(client *spanner.Client, builder *routing.GeometryBuilder) driver.RouteGeometryLookup {
	return func(ctx context.Context, driverID, routeID string, opts driver.RouteGeometryOptions) (routing.RouteGeometry, bool, error) {
		driverID = strings.TrimSpace(driverID)
		routeID = strings.TrimSpace(routeID)
		if driverID == "" || routeID == "" {
			return routing.RouteGeometry{}, false, nil
		}

		ownStmt := spanner.Statement{
			SQL: `SELECT COUNT(*) FROM Orders
			      WHERE DriverId = @did AND RouteId = @rid
			        AND Status NOT IN ('COMPLETED', 'CANCELLED')`,
			Params: map[string]interface{}{"did": driverID, "rid": routeID},
		}
		ownIter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, ownStmt)
		defer ownIter.Stop()
		ownRow, err := ownIter.Next()
		if err != nil {
			return routing.RouteGeometry{}, false, fmt.Errorf("route ownership check: %w", err)
		}
		var owned int64
		if err := ownRow.Columns(&owned); err != nil {
			return routing.RouteGeometry{}, false, fmt.Errorf("route ownership scan: %w", err)
		}
		if owned == 0 {
			return routing.RouteGeometry{}, false, nil
		}

		waypoints, waypointErr := routing.WaypointsForDriverRoute(ctx, client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)), driverID, routeID)
		if waypointErr != nil {
			return routing.RouteGeometry{}, false, waypointErr
		}

		if opts.RerouteFrom != nil {
			waypoints = routing.WaypointsAhead(*opts.RerouteFrom, waypoints, 0)
			var geometry routing.RouteGeometry
			if builder != nil {
				geometry = builder.BuildDetail(ctx, routeID, waypoints, opts.IncludeSteps)
			} else {
				geometry = routing.BuildDenseRouteGeometry(routeID, waypoints)
			}
			geometry.Source = "reroute_" + geometry.Source
			return geometry, true, nil
		}

		storedStmt := spanner.Statement{
			SQL: `SELECT EncodedRoutePolyline, RouteGeometrySource, StopCount
			      FROM SupplierTruckManifests
			      WHERE DriverId = @did AND RouteId = @rid
			        AND EncodedRoutePolyline IS NOT NULL
			        AND EncodedRoutePolyline != ''
			      ORDER BY UpdatedAt DESC
			      LIMIT 1`,
			Params: map[string]interface{}{"did": driverID, "rid": routeID},
		}
		storedIter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, storedStmt)
		storedRow, storedErr := storedIter.Next()
		storedIter.Stop()
		if storedErr == nil {
			var encoded string
			var source spanner.NullString
			var stopCount int64
			if err := storedRow.Columns(&encoded, &source, &stopCount); err == nil && encoded != "" {
				geometry, decodeErr := routing.GeometryFromStoredPolyline(
					routeID,
					encoded,
					source.StringVal,
					int(stopCount),
				)
				if decodeErr == nil {
					if opts.IncludeSteps && builder != nil && len(waypoints) >= 2 {
						detail := builder.BuildDetail(ctx, routeID, waypoints, true)
						geometry.Steps = detail.Steps
					}
					return geometry, true, nil
				}
			}
		}

		var geometry routing.RouteGeometry
		if builder != nil {
			geometry = builder.BuildDetail(ctx, routeID, waypoints, opts.IncludeSteps)
		} else {
			geometry = routing.BuildDenseRouteGeometry(routeID, waypoints)
		}
		return geometry, true, nil
	}
}

// warehouseAnalyticsCountQuery returns a WarehouseAnalyticsQuery backed by
// stale Spanner reads.
func warehouseAnalyticsCountQuery(client *spanner.Client) warehouse.WarehouseAnalyticsQuery {
	return func(ctx context.Context, warehouseID string) (warehouse.WarehouseAnalyticsCounts, error) {
		var counts warehouse.WarehouseAnalyticsCounts
		stmt := spanner.Statement{
			SQL: `SELECT COUNT(*) AS total,
			             COUNTIF(Status = 'COMPLETED') AS completed,
			             COUNTIF(Status = 'CANCELLED') AS cancelled,
			             IFNULL(SUM(CASE WHEN Status = 'COMPLETED' THEN TotalMinor ELSE 0 END), 0) AS revenue
			      FROM Orders WHERE WarehouseId = @wid`,
			Params: map[string]interface{}{"wid": warehouseID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err != nil {
			return counts, fmt.Errorf("warehouse analytics query: %w", err)
		}
		if err := row.Columns(&counts.TotalOrders, &counts.CompletedOrders, &counts.CancelledOrders, &counts.TotalRevenue); err != nil {
			return counts, fmt.Errorf("warehouse analytics scan: %w", err)
		}
		return counts, nil
	}
}

// warehouseOpsOrdersQuery returns warehouse-scoped order rows from Spanner.
func warehouseOpsOrdersQuery(client *spanner.Client) warehouse.WarehouseOpsOrdersQuery {
	return func(ctx context.Context, warehouseID string, limit int) ([]warehouse.OrderRow, error) {
		stmt := spanner.Statement{
			SQL: `SELECT OrderID, RetailerID, Status, TotalMinor, Currency, UpdatedAt
			      FROM Orders WHERE WarehouseId = @wid
			        AND NOT (OrderSource = 'MANUAL_PREORDER' AND Status = 'SCHEDULED')
			      ORDER BY UpdatedAt DESC LIMIT @lim`,
			Params: map[string]interface{}{"wid": warehouseID, "lim": int64(limit)},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		var orders []warehouse.OrderRow
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("warehouse ops orders: %w", err)
			}
			var o warehouse.OrderRow
			var updatedAt time.Time
			if err := row.Columns(&o.OrderID, &o.RetailerID, &o.Status, &o.TotalMinor, &o.Currency, &updatedAt); err != nil {
				return nil, fmt.Errorf("warehouse ops orders scan: %w", err)
			}
			o.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
			orders = append(orders, o)
		}
		if orders == nil {
			orders = []warehouse.OrderRow{}
		}
		return orders, nil
	}
}

func driverLoginLookup(client *spanner.Client) driver.DriverLoginLookup {
	return func(ctx context.Context, phone string) (driver.DriverLoginRecord, bool, error) {
		phone = strings.TrimSpace(phone)
		if phone == "" || client == nil {
			return driver.DriverLoginRecord{}, false, nil
		}
		stmt := spanner.Statement{
			SQL: `SELECT DriverId, Name, Phone, COALESCE(PinHash, ''), SupplierId,
			             HomeNodeType, HomeNodeId, COALESCE(VehicleId, '')
			      FROM Drivers
			      WHERE Phone = @phone AND IsActive = true
			      LIMIT 1`,
			Params: map[string]any{"phone": phone},
		}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return driver.DriverLoginRecord{}, false, nil
		}
		if err != nil {
			return driver.DriverLoginRecord{}, false, fmt.Errorf("driver login lookup: %w", err)
		}
		var rec driver.DriverLoginRecord
		if err := row.Columns(
			&rec.DriverID, &rec.Name, &rec.Phone, &rec.PinHash, &rec.SupplierID,
			&rec.HomeNodeType, &rec.HomeNodeID, &rec.VehicleID,
		); err != nil {
			return driver.DriverLoginRecord{}, false, fmt.Errorf("scan driver login: %w", err)
		}
		return rec, true, nil
	}
}

func payloadStaffLoginLookup(client *spanner.Client) payload.PayloadStaffLookup {
	return func(ctx context.Context, phone string) (payload.PayloadStaffRecord, bool, error) {
		phone = strings.TrimSpace(phone)
		if phone == "" || client == nil {
			return payload.PayloadStaffRecord{}, false, nil
		}
		stmt := spanner.Statement{
			SQL: `SELECT UserId, Name, Phone, PasswordHash, SupplierId, COALESCE(AssignedWarehouseId, '')
			      FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_ByPhone}
			      WHERE Phone = @phone
			        AND IsActive = true
			        AND SupplierRole IN ('PAYLOADER', 'WAREHOUSE', 'WAREHOUSE_ADMIN', 'WAREHOUSE_STAFF')
			      LIMIT 1`,
			Params: map[string]any{"phone": phone},
		}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return payload.PayloadStaffRecord{}, false, nil
		}
		if err != nil {
			return payload.PayloadStaffRecord{}, false, fmt.Errorf("payload staff lookup: %w", err)
		}
		var rec payload.PayloadStaffRecord
		if err := row.Columns(
			&rec.UserID, &rec.Name, &rec.Phone, &rec.PasswordHash, &rec.SupplierID, &rec.WarehouseID,
		); err != nil {
			return payload.PayloadStaffRecord{}, false, fmt.Errorf("scan payload staff: %w", err)
		}
		return rec, true, nil
	}
}

// warehouseOpsDriversQuery returns drivers home-noded to a warehouse.
func warehouseOpsDriversQuery(client *spanner.Client) warehouse.WarehouseOpsDriversQuery {
	return func(ctx context.Context, warehouseID string) ([]warehouse.PortalDriver, error) {
		stmt := spanner.Statement{
			SQL: `SELECT d.DriverId, d.Name, COALESCE(d.Phone, ''), d.IsActive, COALESCE(d.OnShift, true),
			             COALESCE(d.UnavailableReason, ''), COALESCE(d.UnavailableNote, ''),
			             COALESCE(d.VehicleId, ''), COALESCE(v.VehicleClass, 'CLASS_B'),
			             COALESCE(v.MaxVolumeVU, 150.0), COALESCE(v.IsActive, FALSE),
			             COALESCE(v.UnavailableReason, ''), COALESCE(v.UnavailableNote, ''),
			             COALESCE(v.Label, v.LicensePlate, ''), COALESCE(v.LicensePlate, '')
			      FROM Drivers@{FORCE_INDEX=Idx_Drivers_ByHomeNode} d
			      LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
			      WHERE d.HomeNodeType = 'WAREHOUSE' AND d.HomeNodeId = @wid
			      ORDER BY d.Name`,
			Params: map[string]interface{}{"wid": warehouseID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		var drivers []warehouse.PortalDriver
		driverIDs := make([]string, 0, 8)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("warehouse ops drivers: %w", err)
			}
			var d warehouse.PortalDriver
			var vehicleID spanner.NullString
			var unavailableReason, unavailableNote, vehicleUnavailableReason, vehicleUnavailableNote string
			if err := row.Columns(
				&d.DriverID,
				&d.Name,
				&d.Phone,
				&d.IsActive,
				&d.OnShift,
				&unavailableReason,
				&unavailableNote,
				&vehicleID,
				&d.VehicleClass,
				&d.MaxVolumeVU,
				&d.VehicleIsActive,
				&vehicleUnavailableReason,
				&vehicleUnavailableNote,
				&d.VehicleLabel,
				&d.LicensePlate,
			); err != nil {
				return nil, fmt.Errorf("warehouse ops drivers scan: %w", err)
			}
			d.UnavailableReason = strings.TrimSpace(unavailableReason)
			d.UnavailableNote = strings.TrimSpace(unavailableNote)
			d.VehicleUnavailableReason = strings.TrimSpace(vehicleUnavailableReason)
			d.VehicleUnavailableNote = strings.TrimSpace(vehicleUnavailableNote)
			if vehicleID.Valid {
				d.VehicleID = vehicleID.StringVal
			}
			switch {
			case !d.IsActive:
				d.TruckStatus = "INACTIVE"
			case !d.OnShift:
				if strings.EqualFold(d.UnavailableReason, "RETURNING_TO_WAREHOUSE") {
					d.TruckStatus = "RETURNING_TO_WAREHOUSE"
				} else {
					d.TruckStatus = "OFF_SHIFT"
				}
			case d.VehicleID == "":
				d.TruckStatus = "UNASSIGNED"
			case !d.VehicleIsActive:
				d.TruckStatus = "VEHICLE_INACTIVE"
			default:
				d.TruckStatus = "AVAILABLE"
			}
			drivers = append(drivers, d)
			driverIDs = append(driverIDs, d.DriverID)
		}
		if len(driverIDs) > 0 {
			busy, err := warehouseDriversOnActiveManifests(ctx, client, warehouseID, driverIDs)
			if err != nil {
				return nil, err
			}
			for i := range drivers {
				if busy[drivers[i].DriverID] {
					drivers[i].TruckStatus = "IN_TRANSIT"
				}
			}
		}
		if drivers == nil {
			drivers = []warehouse.PortalDriver{}
		}
		return drivers, nil
	}
}

func warehouseDriversOnActiveManifests(ctx context.Context, client *spanner.Client, warehouseID string, driverIDs []string) (map[string]bool, error) {
	out := make(map[string]bool, len(driverIDs))
	if client == nil || warehouseID == "" || len(driverIDs) == 0 {
		return out, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT DriverId
		      FROM SupplierTruckManifests@{FORCE_INDEX=Idx_SupplierManifests_ByWarehouse}
		      WHERE WarehouseId = @wid
		        AND DriverId IN UNNEST(@driverIds)
		        AND State IN ('DRAFT', 'LOADING', 'SEALED', 'DISPATCHED')`,
		Params: map[string]any{
			"wid":       warehouseID,
			"driverIds": driverIDs,
		},
	}
	iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("warehouse active manifest drivers: %w", err)
		}
		var driverID string
		if err := row.Columns(&driverID); err != nil {
			return nil, fmt.Errorf("warehouse active manifest drivers scan: %w", err)
		}
		if driverID != "" {
			out[driverID] = true
		}
	}
}

// warehouseOpsVehiclesQuery returns vehicles home-noded to a warehouse.
func warehouseOpsVehiclesQuery(client *spanner.Client) warehouse.WarehouseOpsVehiclesQuery {
	return func(ctx context.Context, warehouseID string) ([]warehouse.PortalVehicle, error) {
		stmt := spanner.Statement{
			SQL: `SELECT v.VehicleId, COALESCE(v.Label, ''), v.LicensePlate,
			             COALESCE(v.VehicleClass, 'CLASS_B'), COALESCE(v.MaxVolumeVU, 150.0), v.IsActive,
			             COALESCE(v.UnavailableReason, ''), COALESCE(v.UnavailableNote, ''),
			             COALESCE(d.DriverId, ''), COALESCE(d.Name, '')
			      FROM Vehicles@{FORCE_INDEX=Idx_Vehicles_ByHomeNode} v
			      LEFT JOIN Drivers@{FORCE_INDEX=Idx_Drivers_ByHomeNode} d
			        ON d.VehicleId = v.VehicleId
			       AND d.HomeNodeType = 'WAREHOUSE'
			       AND d.HomeNodeId = @wid
			      WHERE v.HomeNodeType = 'WAREHOUSE' AND v.HomeNodeId = @wid
			      ORDER BY v.LicensePlate`,
			Params: map[string]interface{}{"wid": warehouseID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		var vehicles []warehouse.PortalVehicle
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("warehouse ops vehicles: %w", err)
			}
			var v warehouse.PortalVehicle
			if err := row.Columns(
				&v.VehicleID,
				&v.Label,
				&v.LicensePlate,
				&v.VehicleClass,
				&v.MaxVolumeVU,
				&v.IsActive,
				&v.UnavailableReason,
				&v.UnavailableNote,
				&v.AssignedDriverID,
				&v.AssignedDriverName,
			); err != nil {
				return nil, fmt.Errorf("warehouse ops vehicles scan: %w", err)
			}
			vehicles = append(vehicles, v)
		}
		if vehicles == nil {
			vehicles = []warehouse.PortalVehicle{}
		}
		return vehicles, nil
	}
}

func driverAvailabilityReader(client *spanner.Client) driver.AvailabilityReader {
	return func(ctx context.Context, driverID string) (bool, string, string, bool, error) {
		if client == nil || strings.TrimSpace(driverID) == "" {
			return true, "", "", false, nil
		}
		stmt := spanner.Statement{
			SQL: `SELECT COALESCE(OnShift, true), COALESCE(UnavailableReason, ''), COALESCE(UnavailableNote, '')
			      FROM Drivers WHERE DriverId = @id`,
			Params: map[string]any{"id": driverID},
		}
		row, err := client.Single().Query(ctx, stmt).Next()
		if err == iterator.Done {
			return true, "", "", false, nil
		}
		if err != nil {
			return true, "", "", false, err
		}
		var onShift bool
		var reason, note string
		if err := row.Columns(&onShift, &reason, &note); err != nil {
			return true, "", "", false, err
		}
		return onShift, strings.TrimSpace(reason), strings.TrimSpace(note), true, nil
	}
}

// supplierDashboardCountQuery returns a DashboardCountQuery backed by stale
// Spanner reads for aggregate dashboard KPIs.
func supplierDashboardCountQuery(client *spanner.Client) supplier.DashboardCountQuery {
	return func(ctx context.Context, supplierID string) (supplier.DashboardCounts, error) {
		var counts supplier.DashboardCounts
		stmt := spanner.Statement{
			SQL: `SELECT COUNTIF(Status IN ('PENDING', 'AWAITING_REVIEW')) AS pending_orders,
			             COUNTIF(Status = 'IN_TRANSIT') AS active_deliveries
			      FROM Orders WHERE SupplierID = @sid`,
			Params: map[string]interface{}{"sid": supplierID},
		}
		iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err != nil {
			return counts, fmt.Errorf("dashboard count query: %w", err)
		}
		var pendingOrders, activeDeliveries int64
		if err := row.Columns(&pendingOrders, &activeDeliveries); err != nil {
			return counts, fmt.Errorf("dashboard count scan: %w", err)
		}
		counts.PendingOrders = int(pendingOrders)
		counts.ActiveDrivers = int(activeDeliveries)
		return counts, nil
	}
}

func loadSupplierEarningsAuthority(ctx context.Context, repo payment.Repository, supplierID, currency string, now time.Time) (supplier.SupplierEarningsResponse, error) {
	if repo == nil {
		return supplier.SupplierEarningsResponse{}, errors.New("payment repository unavailable")
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := dayStart.AddDate(0, 0, -int(dayStart.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	today, todayAuthoritative, err := sumSupplierEarningsWindow(ctx, repo, supplierID, currency, dayStart, now)
	if err != nil {
		return supplier.SupplierEarningsResponse{}, err
	}
	week, weekAuthoritative, err := sumSupplierEarningsWindow(ctx, repo, supplierID, currency, weekStart, now)
	if err != nil {
		return supplier.SupplierEarningsResponse{}, err
	}
	month, monthAuthoritative, err := sumSupplierEarningsWindow(ctx, repo, supplierID, currency, monthStart, now)
	if err != nil {
		return supplier.SupplierEarningsResponse{}, err
	}

	return supplier.SupplierEarningsResponse{
		Currency:        currency,
		TodayMinor:      today,
		WeekMinor:       week,
		MonthMinor:      month,
		AuthoritySource: "payment_ledger",
		Authoritative:   todayAuthoritative && weekAuthoritative && monthAuthoritative,
		UpdatedAt:       now.Format(time.RFC3339Nano),
	}, nil
}

func sumSupplierEarningsWindow(ctx context.Context, repo payment.Repository, supplierID, currency string, from, to time.Time) (int64, bool, error) {
	rows, err := repo.SummarizeLedgerEntries(ctx, payment.SettlementAuthorityQuery{
		SupplierID:   supplierID,
		OccurredFrom: &from,
		OccurredTo:   &to,
		GroupLimit:   1000,
	})
	if err != nil {
		return 0, false, fmt.Errorf("summarize supplier earnings window: %w", err)
	}
	total := int64(0)
	authoritative := true
	for _, row := range rows {
		if strings.TrimSpace(currency) != "" && !strings.EqualFold(row.Currency, currency) {
			authoritative = false
			continue
		}
		total += payment.SignedSettlementEntryAmount(row.EntryType, row.AmountMinorTotal)
	}
	return total, authoritative, nil
}
