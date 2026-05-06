package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"time"

	"backend-go/auth"
	internalKafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/spannerx"
	"backend-go/telemetry"
	factoryws "backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// FACTORY DISPATCH ENGINE — BIN-PACK, ROUTE-OPTIMIZE, LIFO-LOAD
//
// Three-phase pipeline: First-Fit-Decreasing bin-packing →
// Nearest-neighbor route sort from factory origin →
// LIFO loading sequence (last stop loaded first = back of truck).
//
// Mirrors the supplier auto-dispatch engine in dispatcher.go but operates on
// InternalTransferOrders instead of retail Orders.
// ═══════════════════════════════════════════════════════════════════════════════

// BatcherService holds dependencies for the factory dispatch engine.
type BatcherService struct {
	Spanner    *spanner.Client
	Producer   *kafka.Writer
	FactoryHub *factoryws.FactoryHub
}

// batchableTransfer is the working struct for a single APPROVED transfer.
type batchableTransfer struct {
	TransferId  string
	WarehouseId string
	VolumeVU    float64
	WhLat       float64
	WhLng       float64
}

// factoryVehicle represents an available factory-assigned vehicle.
type factoryVehicle struct {
	VehicleId   string
	DriverId    string
	MaxVolumeVU float64
}

// BatchDispatchResult is the response from the dispatch engine.
type BatchDispatchResult struct {
	SnapshotTimestamp string           `json:"snapshot_timestamp"`
	Manifests         []ManifestResult `json:"manifests"`
	Unassigned        []string         `json:"unassigned"`
}

// ManifestResult describes a created factory truck manifest.
type ManifestResult struct {
	ManifestId    string          `json:"manifest_id"`
	DriverId      string          `json:"driver_id"`
	VehicleId     string          `json:"vehicle_id"`
	TotalVolumeVU float64         `json:"total_volume_vu"`
	MaxVolumeVU   float64         `json:"max_volume_vu"`
	StopCount     int             `json:"stop_count"`
	RegionCode    string          `json:"region_code"`
	Transfers     []string        `json:"transfer_ids"`
	LoadingOrder  []LoadStopEntry `json:"loading_order"`
}

// LoadStopEntry describes LIFO loading sequence for a stop.
type LoadStopEntry struct {
	Sequence    int     `json:"sequence"`
	TransferId  string  `json:"transfer_id"`
	WarehouseId string  `json:"warehouse_id"`
	VolumeVU    float64 `json:"volume_vu"`
	Instruction string  `json:"instruction"`
}

// HandleFactoryDispatch — POST /v1/factory/dispatch
// Groups APPROVED transfers, bin-packs into manifests, optimizes delivery routes.
func (b *BatcherService) HandleFactoryDispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	factoryID, ok := auth.MustFactoryID(w, r.Context())
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	result, err := b.runBatch(ctx, factoryID)
	if err != nil {
		log.Printf("[FACTORY-DISPATCH] error: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (b *BatcherService) runBatch(ctx context.Context, factoryID string) (*BatchDispatchResult, error) {
	snapshotTS := time.Now().UTC().Format(time.RFC3339)

	// 1. Fetch APPROVED transfers for this factory (no manifest yet)
	transfers, err := b.fetchApprovedTransfers(ctx, factoryID)
	if err != nil {
		return nil, fmt.Errorf("fetch transfers: %w", err)
	}
	if len(transfers) == 0 {
		return &BatchDispatchResult{
			SnapshotTimestamp: snapshotTS,
			Manifests:         []ManifestResult{},
			Unassigned:        []string{},
		}, nil
	}

	// 2. Fetch factory origin coordinates + supplier scope
	facLat, facLng, err := b.getFactoryCoords(ctx, factoryID)
	if err != nil {
		return nil, fmt.Errorf("factory coords: %w", err)
	}
	supplierID, err := b.resolveFactorySupplier(ctx, factoryID)
	if err != nil {
		return nil, fmt.Errorf("resolve factory supplier: %w", err)
	}

	// 3. Fetch available vehicles for this factory's supplier
	vehicles, err := b.fetchFactoryVehicles(ctx, factoryID)
	if err != nil {
		return nil, fmt.Errorf("fetch vehicles: %w", err)
	}
	if len(vehicles) == 0 {
		ids := make([]string, len(transfers))
		for i, t := range transfers {
			ids[i] = t.TransferId
		}
		return &BatchDispatchResult{
			SnapshotTimestamp: snapshotTS,
			Manifests:         []ManifestResult{},
			Unassigned:        ids,
		}, nil
	}

	// Sort vehicles by capacity DESC — biggest trucks absorb the most
	sort.Slice(vehicles, func(i, j int) bool {
		return vehicles[i].MaxVolumeVU > vehicles[j].MaxVolumeVU
	})

	// Sort transfers by volume DESC — First-Fit Decreasing bin-packing
	sort.Slice(transfers, func(i, j int) bool {
		return transfers[i].VolumeVU > transfers[j].VolumeVU
	})

	// ── PHASE 1: First-Fit Decreasing Bin-Pack ──────────────────────────────
	type manifestBuild struct {
		vehicle   factoryVehicle
		transfers []batchableTransfer
		usedVU    float64
	}

	var builds []manifestBuild
	var unassigned []string

	for _, t := range transfers {
		placed := false
		for bi := range builds {
			if builds[bi].usedVU+t.VolumeVU <= builds[bi].vehicle.MaxVolumeVU {
				builds[bi].transfers = append(builds[bi].transfers, t)
				builds[bi].usedVU += t.VolumeVU
				placed = true
				break
			}
		}
		if !placed {
			// Open a new manifest with next available vehicle
			if len(builds) < len(vehicles) {
				vIdx := len(builds)
				builds = append(builds, manifestBuild{
					vehicle:   vehicles[vIdx],
					transfers: []batchableTransfer{t},
					usedVU:    t.VolumeVU,
				})
			} else {
				unassigned = append(unassigned, t.TransferId)
			}
		}
	}

	// ── PHASE 2: Nearest-Neighbor Route Sort + LIFO Loading ─────────────────
	var results []ManifestResult
	var mutations []*spanner.Mutation

	for _, m := range builds {
		if len(m.transfers) == 0 {
			continue
		}

		// Sort stops by nearest-neighbor from factory origin
		sorted := nearestNeighborSort(facLat, facLng, m.transfers)
		n := len(sorted)

		manifestID := uuid.New().String()
		transferIds := make([]string, n)
		loadingOrder := make([]LoadStopEntry, n)

		for i, t := range sorted {
			transferIds[i] = t.TransferId

			// LIFO: last delivery stop loaded first (deepest in truck)
			seq := n - i
			instruction := fmt.Sprintf("Load position %d of %d", seq, n)
			if seq == 1 {
				instruction = "Load first — Back of Truck"
			} else if seq == n {
				instruction = "Load last — By the Doors"
			}
			loadingOrder[seq-1] = LoadStopEntry{
				Sequence:    seq,
				TransferId:  t.TransferId,
				WarehouseId: t.WarehouseId,
				VolumeVU:    t.VolumeVU,
				Instruction: instruction,
			}

			// Link transfer → manifest, transition APPROVED → LOADING
			mutations = append(mutations,
				spanner.Update("InternalTransferOrders",
					[]string{"TransferId", "ManifestId", "State", "UpdatedAt"},
					[]interface{}{t.TransferId, manifestID, "LOADING", spanner.CommitTimestamp}),
			)
		}

		// Build route path JSON (ordered warehouse coordinates)
		routePoints := make([]map[string]float64, n)
		for i, t := range sorted {
			routePoints[i] = map[string]float64{"lat": t.WhLat, "lng": t.WhLng}
		}
		routeJSON, _ := json.Marshal(routePoints)

		// Derive region code from route centroid
		region := routeZoneLabel(sorted)

		// Insert FactoryTruckManifest
		mutations = append(mutations,
			spanner.Insert("FactoryTruckManifests",
				[]string{"ManifestId", "FactoryId", "DriverId", "VehicleId",
					"State", "TotalVolumeVU", "MaxVolumeVU", "StopCount",
					"RegionCode", "RoutePath", "CreatedAt"},
				[]interface{}{manifestID, factoryID, m.vehicle.DriverId, m.vehicle.VehicleId,
					"READY_FOR_LOADING", m.usedVU, m.vehicle.MaxVolumeVU, int64(n),
					region, string(routeJSON), spanner.CommitTimestamp}),
		)

		results = append(results, ManifestResult{
			ManifestId:    manifestID,
			DriverId:      m.vehicle.DriverId,
			VehicleId:     m.vehicle.VehicleId,
			TotalVolumeVU: m.usedVU,
			MaxVolumeVU:   m.vehicle.MaxVolumeVU,
			StopCount:     n,
			RegionCode:    region,
			Transfers:     transferIds,
			LoadingOrder:  loadingOrder,
		})
	}

	// Commit all mutations + outbox events atomically
	if len(mutations) > 0 {
		_, err := b.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}
			for _, m := range results {
				if err := outbox.EmitJSON(txn, "FactoryManifest", m.ManifestId, internalKafka.EventFactoryManifestCreated, internalKafka.TopicMain, internalKafka.RouteCreatedEvent{
					RouteID:    m.ManifestId,
					DriverID:   m.DriverId,
					TruckID:    m.VehicleId,
					SupplierID: supplierID,
					FactoryID:  factoryID,
					StopCount:  m.StopCount,
					VolumeVU:   m.TotalVolumeVU,
					Timestamp:  time.Now().UTC(),
				}, telemetry.TraceIDFromContext(ctx)); err != nil {
					return fmt.Errorf("emit factory manifest event %s: %w", m.ManifestId, err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("commit manifests: %w", err)
		}

		if b.FactoryHub != nil {
			for _, m := range results {
				b.FactoryHub.BroadcastManifestUpdate(factoryID, m.ManifestId, "READY_FOR_LOADING", "CREATE", "", supplierID, m.Transfers)
				for _, stop := range m.LoadingOrder {
					b.FactoryHub.BroadcastTransferUpdate(factoryID, stop.TransferId, stop.WarehouseId, m.ManifestId, "APPROVED", "LOADING", "BATCH_ASSIGN", supplierID)
				}
			}
		}
	}

	log.Printf("[FACTORY-DISPATCH] factory=%s | %d transfers → %d manifests, %d unassigned",
		factoryID, len(transfers), len(results), len(unassigned))

	if results == nil {
		results = []ManifestResult{}
	}
	if unassigned == nil {
		unassigned = []string{}
	}

	return &BatchDispatchResult{
		SnapshotTimestamp: snapshotTS,
		Manifests:         results,
		Unassigned:        unassigned,
	}, nil
}

// ── Data Fetchers ───────────────────────────────────────────────────────────

func (b *BatcherService) fetchApprovedTransfers(ctx context.Context, factoryID string) ([]batchableTransfer, error) {
	stmt := spanner.Statement{
		SQL: `SELECT t.TransferId, t.WarehouseId, t.TotalVolumeVU,
		             IFNULL(w.Lat, 0), IFNULL(w.Lng, 0)
		      FROM InternalTransferOrders t
		      LEFT JOIN Warehouses w ON t.WarehouseId = w.WarehouseId
		      WHERE t.FactoryId = @fid AND t.State = 'APPROVED' AND t.ManifestId IS NULL
		      ORDER BY t.TotalVolumeVU DESC`,
		Params: map[string]interface{}{"fid": factoryID},
	}
	iter := spannerx.StaleQuery(ctx, b.Spanner, stmt)
	defer iter.Stop()

	var result []batchableTransfer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var t batchableTransfer
		if err := row.Columns(&t.TransferId, &t.WarehouseId, &t.VolumeVU,
			&t.WhLat, &t.WhLng); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

func (b *BatcherService) getFactoryCoords(ctx context.Context, factoryID string) (float64, float64, error) {
	row, err := spannerx.StaleReadRow(ctx, b.Spanner, "Factories",
		spanner.Key{factoryID}, []string{"Lat", "Lng"})
	if err != nil {
		return 0, 0, err
	}
	var lat, lng spanner.NullFloat64
	if err := row.Columns(&lat, &lng); err != nil {
		return 0, 0, err
	}
	return lat.Float64, lng.Float64, nil
}

// resolveFactorySupplier returns the SupplierId that owns the factory.
func (b *BatcherService) resolveFactorySupplier(ctx context.Context, factoryID string) (string, error) {
	row, err := b.Spanner.Single().ReadRow(ctx, "Factories",
		spanner.Key{factoryID}, []string{"SupplierId"})
	if err != nil {
		return "", err
	}
	var sid string
	if err := row.Columns(&sid); err != nil {
		return "", err
	}
	return sid, nil
}

func (b *BatcherService) fetchFactoryVehicles(ctx context.Context, factoryID string) ([]factoryVehicle, error) {
	// Resolve supplier from factory, then get available vehicles
	fRow, err := b.Spanner.Single().ReadRow(ctx, "Factories",
		spanner.Key{factoryID}, []string{"SupplierId"})
	if err != nil {
		return nil, err
	}
	var supplierID string
	if err := fRow.Columns(&supplierID); err != nil {
		return nil, err
	}

	stmt := spanner.Statement{
		SQL: `SELECT v.VehicleId, d.DriverId, v.MaxVolumeVU
		      FROM Vehicles v
		      JOIN Drivers d ON d.VehicleId = v.VehicleId
		      WHERE d.SupplierId = @sid
		        AND d.IsActive = true
		        AND v.IsActive = true
		        AND COALESCE(d.TruckStatus, 'AVAILABLE') = 'AVAILABLE'
		      ORDER BY v.MaxVolumeVU DESC`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := spannerx.StaleQuery(ctx, b.Spanner, stmt)
	defer iter.Stop()

	var result []factoryVehicle
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var v factoryVehicle
		if err := row.Columns(&v.VehicleId, &v.DriverId, &v.MaxVolumeVU); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

// ── Geo & Route Utilities ───────────────────────────────────────────────────

// nearestNeighborSort orders transfers by nearest-neighbor heuristic from factory.
func nearestNeighborSort(startLat, startLng float64, transfers []batchableTransfer) []batchableTransfer {
	if len(transfers) <= 1 {
		return transfers
	}

	remaining := make([]batchableTransfer, len(transfers))
	copy(remaining, transfers)

	sorted := make([]batchableTransfer, 0, len(transfers))
	curLat, curLng := startLat, startLng

	for len(remaining) > 0 {
		bestIdx := 0
		bestDist := math.MaxFloat64
		for i, t := range remaining {
			d := haversineKm(curLat, curLng, t.WhLat, t.WhLng)
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		sorted = append(sorted, remaining[bestIdx])
		curLat = remaining[bestIdx].WhLat
		curLng = remaining[bestIdx].WhLng
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}
	return sorted
}

// routeZoneLabel generates a zone label from the centroid of delivery stops.
func routeZoneLabel(transfers []batchableTransfer) string {
	if len(transfers) == 0 {
		return "UNKNOWN"
	}
	var sumLat, sumLng float64
	for _, t := range transfers {
		sumLat += t.WhLat
		sumLng += t.WhLng
	}
	n := float64(len(transfers))
	return fmt.Sprintf("%.3f,%.3f", sumLat/n, sumLng/n)
}

// haversineKm returns great-circle distance in km between two coordinates.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
