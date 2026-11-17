package stocklots

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

const (
	defaultChilledMinC = 0.0
	defaultChilledMaxC = 8.0
	defaultFrozenMinC  = -22.0
	defaultFrozenMaxC  = -15.0
)

// TemperatureReadingView is one cold-chain sample.
type TemperatureReadingView struct {
	ReadingID  string  `json:"reading_id"`
	ManifestID string  `json:"manifest_id"`
	SensorID   string  `json:"sensor_id,omitempty"`
	RecordedAt string  `json:"recorded_at"`
	TempC      float64 `json:"temp_c"`
	Lat        float64 `json:"lat,omitempty"`
	Lng        float64 `json:"lng,omitempty"`
	MinC       float64 `json:"min_c,omitempty"`
	MaxC       float64 `json:"max_c,omitempty"`
	Excursion  bool    `json:"excursion,omitempty"`
}

// TempBand is a closed Celsius interval for excursion detection.
type TempBand struct {
	MinC float64
	MaxC float64
}

// TemperatureBreachArgs is passed to the registered system breach raiser.
type TemperatureBreachArgs struct {
	ManifestID string
	ReadingID  string
	TempC      float64
	MinC       float64
	MaxC       float64
	OrderIDs   []string
}

// TemperatureBreachRaiser creates TEMPERATURE_BREACH condition reports in the same RW txn.
type TemperatureBreachRaiser func(ctx context.Context, txn *spanner.ReadWriteTransaction, args TemperatureBreachArgs) error

var (
	coldChainMemMu       sync.RWMutex
	coldChainMemReadings = map[string][]TemperatureReadingView{}
)

var (
	breachRaiserMu         sync.RWMutex
	temperatureBreachRaiser TemperatureBreachRaiser
)

// SetTemperatureBreachRaiser wires order-system auto-raise (bootstrap / main).
func SetTemperatureBreachRaiser(fn TemperatureBreachRaiser) {
	breachRaiserMu.Lock()
	temperatureBreachRaiser = fn
	breachRaiserMu.Unlock()
}

func currentTemperatureBreachRaiser() TemperatureBreachRaiser {
	breachRaiserMu.RLock()
	fn := temperatureBreachRaiser
	breachRaiserMu.RUnlock()
	return fn
}

// RecordTemperatureReadingMemory stores a reading in local/dev memory when Spanner is absent.
func RecordTemperatureReadingMemory(manifestID, sensorID string, tempC float64, bandOverride *TempBand) (*TemperatureReadingView, error) {
	if err := memoryBlocked(); err != nil {
		return nil, err
	}
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return nil, fmt.Errorf("manifest_id required")
	}
	band := resolveTempBandFromClasses(nil, bandOverride)
	v := TemperatureReadingView{
		ReadingID:  uuid.NewString(),
		ManifestID: manifestID,
		SensorID:   strings.TrimSpace(sensorID),
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
		TempC:      tempC,
		MinC:       band.MinC,
		MaxC:       band.MaxC,
		Excursion:  tempC < band.MinC || tempC > band.MaxC,
	}
	coldChainMemMu.Lock()
	coldChainMemReadings[manifestID] = append([]TemperatureReadingView{v}, coldChainMemReadings[manifestID]...)
	coldChainMemMu.Unlock()
	return &v, nil
}

func hasColdChainReadingMemory(manifestID string) bool {
	coldChainMemMu.RLock()
	defer coldChainMemMu.RUnlock()
	return len(coldChainMemReadings[strings.TrimSpace(manifestID)]) > 0
}

func resolveTempBandFromClasses(handlingClasses []string, override *TempBand) TempBand {
	if override != nil && override.MinC <= override.MaxC {
		return *override
	}
	for _, hc := range handlingClasses {
		if strings.EqualFold(strings.TrimSpace(hc), "FROZEN") {
			return TempBand{MinC: defaultFrozenMinC, MaxC: defaultFrozenMaxC}
		}
	}
	return TempBand{MinC: defaultChilledMinC, MaxC: defaultChilledMaxC}
}

func resolveTempBandInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID string, override *TempBand) TempBand {
	if override != nil && override.MinC <= override.MaxC {
		return *override
	}
	if txn == nil || strings.TrimSpace(manifestID) == "" {
		return TempBand{MinC: defaultChilledMinC, MaxC: defaultChilledMaxC}
	}
	seen := make(map[string]struct{})
	for _, stmt := range []spanner.Statement{
		{
			SQL: `SELECT o.LineItemsJson
			      FROM ManifestOrders mo
			      JOIN Orders o ON o.OrderId = mo.OrderId
			      WHERE mo.ManifestId = @mid
			      LIMIT 200`,
			Params: map[string]any{"mid": manifestID},
		},
		{
			SQL:    `SELECT LineItemsJson FROM Orders WHERE ManifestId = @mid LIMIT 200`,
			Params: map[string]any{"mid": manifestID},
		},
	} {
		iter := txn.Query(ctx, stmt)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var raw []byte
			if err := row.Columns(&raw); err != nil {
				continue
			}
			for _, id := range extractProductIDsFromLineItems(raw) {
				seen[id] = struct{}{}
			}
		}
		iter.Stop()
	}
	if len(seen) == 0 {
		return TempBand{MinC: defaultChilledMinC, MaxC: defaultChilledMaxC}
	}
	pids := make([]string, 0, len(seen))
	for id := range seen {
		pids = append(pids, id)
	}
	pIter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT HandlingClass FROM Products WHERE ProductId IN UNNEST(@pids)`,
		Params: map[string]any{"pids": pids},
	})
	defer pIter.Stop()
	var classes []string
	for {
		row, err := pIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var hc string
		if err := row.Columns(&hc); err == nil {
			classes = append(classes, hc)
		}
	}
	return resolveTempBandFromClasses(classes, override)
}

func loadColdChainOrderIDsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID string) ([]string, error) {
	orderIDs, err := loadManifestOrderIDs(ctx, txn, manifestID)
	if err == nil && len(orderIDs) > 0 {
		return orderIDs, nil
	}
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT OrderId FROM Orders WHERE ManifestId = @mid`,
		Params: map[string]any{"mid": manifestID},
	})
	defer iter.Stop()
	var out []string
	for {
		row, qErr := iter.Next()
		if qErr == iterator.Done {
			break
		}
		if qErr != nil {
			return nil, qErr
		}
		var oid string
		if cErr := row.Columns(&oid); cErr == nil && strings.TrimSpace(oid) != "" {
			out = append(out, strings.TrimSpace(oid))
		}
	}
	return out, nil
}

// IngestTemperatureInTxn stores a reading and auto-raises breaches on band excursion.
// bandOverride: when non-nil and valid, use caller min/max; otherwise hydrate from manifest SKUs ([0,8] chilled or [-22,-15] frozen).
func IngestTemperatureInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	manifestID, sensorID string,
	tempC, lat, lng float64,
	bandOverride *TempBand,
) (*TemperatureReadingView, error) {
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return nil, fmt.Errorf("manifest_id required")
	}
	if txn == nil {
		return nil, fmt.Errorf("spanner transaction required")
	}

	band := resolveTempBandInTxn(ctx, txn, manifestID, bandOverride)
	excursion := tempC < band.MinC || tempC > band.MaxC
	id := uuid.NewString()
	now := time.Now().UTC()
	cols := map[string]any{
		"ReadingId":  id,
		"ManifestId": manifestID,
		"RecordedAt": now,
		"TempC":      tempC,
		"CreatedAt":  spanner.CommitTimestamp,
	}
	if s := strings.TrimSpace(sensorID); s != "" {
		cols["SensorId"] = s
	}
	if lat != 0 || lng != 0 {
		cols["Lat"] = lat
		cols["Lng"] = lng
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("TemperatureReadings", cols)}); err != nil {
		return nil, err
	}
	if excursion {
		orderIDs, err := loadColdChainOrderIDsInTxn(ctx, txn, manifestID)
		if err != nil {
			return nil, err
		}
		if raiser := currentTemperatureBreachRaiser(); raiser != nil && len(orderIDs) > 0 {
			if err := raiser(ctx, txn, TemperatureBreachArgs{
				ManifestID: manifestID,
				ReadingID:  id,
				TempC:      tempC,
				MinC:       band.MinC,
				MaxC:       band.MaxC,
				OrderIDs:   orderIDs,
			}); err != nil {
				return nil, err
			}
		}
	}
	return &TemperatureReadingView{
		ReadingID: id, ManifestID: manifestID, SensorID: strings.TrimSpace(sensorID),
		RecordedAt: now.Format(time.RFC3339Nano), TempC: tempC, Lat: lat, Lng: lng,
		MinC: band.MinC, MaxC: band.MaxC, Excursion: excursion,
	}, nil
}

// LoadManifestDetails retrieves the supplier, warehouse, driver, and truck ID for a manifest.
func LoadManifestDetails(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID string) (supplierID, warehouseID, driverID, truckID string, err error) {
	manifestID = strings.TrimSpace(manifestID)
	if txn == nil || manifestID == "" {
		return "", "", "", "", fmt.Errorf("invalid manifest query")
	}
	row, err := txn.ReadRow(ctx, "SupplierTruckManifests", spanner.Key{manifestID},
		[]string{"SupplierId", "WarehouseId", "DriverId", "TruckId"})
	if err != nil {
		return "", "", "", "", err
	}
	var sup, drv, trk string
	var wh spanner.NullString
	if err := row.Columns(&sup, &wh, &drv, &trk); err != nil {
		return "", "", "", "", err
	}
	whID := ""
	if wh.Valid {
		whID = wh.StringVal
	}
	return strings.TrimSpace(sup), strings.TrimSpace(whID), strings.TrimSpace(drv), strings.TrimSpace(trk), nil
}

// ListTemperatureReadings lists recent readings for a manifest.
func ListTemperatureReadings(ctx context.Context, client *spanner.Client, manifestID string) ([]TemperatureReadingView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	iter := client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, spanner.Statement{
		SQL: `SELECT ReadingId, ManifestId, SensorId, RecordedAt, TempC, Lat, Lng
		      FROM TemperatureReadings WHERE ManifestId = @mid ORDER BY RecordedAt DESC LIMIT 100`,
		Params: map[string]any{"mid": strings.TrimSpace(manifestID)},
	})
	defer iter.Stop()
	var out []TemperatureReadingView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var v TemperatureReadingView
		var sensor spanner.NullString
		var recorded spanner.NullTime
		var lat, lng spanner.NullFloat64
		if err := row.Columns(&v.ReadingID, &v.ManifestID, &sensor, &recorded, &v.TempC, &lat, &lng); err != nil {
			return nil, err
		}
		if sensor.Valid {
			v.SensorID = sensor.StringVal
		}
		if recorded.Valid {
			v.RecordedAt = recorded.Time.UTC().Format(time.RFC3339Nano)
		}
		if lat.Valid {
			v.Lat = lat.Float64
		}
		if lng.Valid {
			v.Lng = lng.Float64
		}
		out = append(out, v)
	}
	if out == nil {
		out = []TemperatureReadingView{}
	}
	return out, nil
}
