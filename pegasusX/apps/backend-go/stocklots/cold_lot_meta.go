package stocklots

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ColdLotWarningCode is emitted when a cold-chain lot sits in staging or an
// ambient bin without a confirmed cold-zone placement / live telemetry baseline.
const ColdLotWarningCode = "cold_lot_migrated_without_live_baseline"

// EnrichedStockLotView extends StockLotView with product cold-chain classification
// and bin zone compatibility signals so Web/Desktop/Mobile operators see migrated
// staging lots and cold-zone mismatches before payload seal.
type EnrichedStockLotView struct {
	StockLotView
	ProductName       string `json:"product_name,omitempty"`
	Barcode           string `json:"barcode,omitempty"`
	HandlingClass     string `json:"handling_class,omitempty"`
	RequiresColdChain bool   `json:"requires_cold_chain,omitempty"`
	IsPerishable      bool   `json:"is_perishable,omitempty"`
	RequiredZone      string `json:"required_zone,omitempty"`
	LocationZone      string `json:"location_zone,omitempty"`
	LocationType      string `json:"location_type,omitempty"`
	MigratedStaging   bool   `json:"migrated_staging,omitempty"`
	ColdLotWarning    string `json:"cold_lot_warning,omitempty"`
}

// EnrichedPickWaveView extends PickWaveView with pre-seal cold-chain and
// migrated-lot staging readiness flags.
type EnrichedPickWaveView struct {
	PickWaveView
	ColdLotMigratedWithoutLiveBaseline bool     `json:"cold_lot_migrated_without_live_baseline,omitempty"`
	ColdChainBaselineMissing           bool     `json:"cold_chain_baseline_missing,omitempty"`
	ColdLotWarnings                    []string `json:"cold_lot_warnings,omitempty"`
}

type productColdMeta struct {
	Name              string
	Barcode           string
	HandlingClass     string
	IsPerishable      bool
	RequiresColdChain bool
}

type binZoneMeta struct {
	Zone         string
	LocationType string
}

// ResolveRequiredZone returns "FREEZE" or "CHILL" when a product requires cold
// storage, or "" for ambient SKUs.
func ResolveRequiredZone(requiresCold, isPerishable bool, handlingClass string) string {
	hc := strings.ToUpper(strings.TrimSpace(handlingClass))
	if hc == "FROZEN" {
		return "FREEZE"
	}
	if requiresCold || isPerishable || hc == "CHILLED" || hc == "COLD" || hc == "PERISHABLE" {
		return "CHILL"
	}
	return ""
}

// IsColdZoneCompatible reports whether a bin's storage zone satisfies the
// required cold-chain temperature band.
func IsColdZoneCompatible(locationZone, requiredZone string) bool {
	req := strings.ToUpper(strings.TrimSpace(requiredZone))
	if req == "" {
		return true
	}
	zone := strings.ToUpper(strings.TrimSpace(locationZone))
	switch req {
	case "FREEZE":
		return zone == "FREEZE" || zone == "FROZEN" || zone == "FREEZER"
	case "CHILL":
		return zone == "CHILL" || zone == "CHILLED" || zone == "COLD" ||
			zone == "REEFER" || zone == "FREEZE" || zone == "FROZEN"
	default:
		return false
	}
}

// IsMigratedStagingLocation reports whether a lot resides in initial migration
// receiving staging (recv-default / recv-* / STAGE bin).
func IsMigratedStagingLocation(locationID, locationType string) bool {
	id := strings.ToLower(strings.TrimSpace(locationID))
	if id == DefaultRecvLocationID || strings.HasPrefix(id, "recv-") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(locationType), "STAGE")
}

// ClassifyColdLotState computes required cold zone, staging status, and warning
// code for a lot + bin pair.
func ClassifyColdLotState(requiresCold, isPerishable bool, handlingClass, locationID, locationZone, locationType string) (requiredZone string, migratedStaging bool, warning string) {
	requiredZone = ResolveRequiredZone(requiresCold, isPerishable, handlingClass)
	migratedStaging = IsMigratedStagingLocation(locationID, locationType)
	if requiredZone != "" && (migratedStaging || !IsColdZoneCompatible(locationZone, requiredZone)) {
		warning = ColdLotWarningCode
	}
	return requiredZone, migratedStaging, warning
}

// EnrichLotsWithColdMeta decorates lots with product cold-chain flags, bin zone
// metadata, and counts of cold-chain warnings and staging lots needing putaway.
func EnrichLotsWithColdMeta(ctx context.Context, client *spanner.Client, warehouseID string, lots []StockLotView) (enriched []EnrichedStockLotView, coldWarnCount int64, stagingCount int64) {
	if len(lots) == 0 {
		return nil, 0, 0
	}
	pids := make([]string, 0, len(lots))
	lids := make([]string, 0, len(lots))
	for _, l := range lots {
		pids = appendUniqueTrimmed(pids, l.ProductID)
		lids = appendUniqueTrimmed(lids, l.LocationID)
	}
	prodMeta := fetchProductsColdMeta(ctx, client, pids)
	locMeta := fetchBinsZoneMeta(ctx, client, warehouseID, lids)

	enriched = make([]EnrichedStockLotView, 0, len(lots))
	for _, lot := range lots {
		p := prodMeta[strings.TrimSpace(lot.ProductID)]
		b := locMeta[strings.TrimSpace(lot.LocationID)]
		reqZone, staging, warn := ClassifyColdLotState(
			p.RequiresColdChain, p.IsPerishable, p.HandlingClass,
			lot.LocationID, b.Zone, b.LocationType,
		)
		if staging && lot.QuantityOnHand > 0 {
			stagingCount++
		}
		if warn != "" && lot.QuantityOnHand > 0 {
			coldWarnCount++
		}
		enriched = append(enriched, EnrichedStockLotView{
			StockLotView:      lot,
			ProductName:       p.Name,
			Barcode:           p.Barcode,
			HandlingClass:     p.HandlingClass,
			RequiresColdChain: p.RequiresColdChain,
			IsPerishable:      p.IsPerishable,
			RequiredZone:      reqZone,
			LocationZone:      b.Zone,
			LocationType:      b.LocationType,
			MigratedStaging:   staging,
			ColdLotWarning:    warn,
		})
	}
	return enriched, coldWarnCount, stagingCount
}

// EnrichPickWavesWithColdMeta decorates a slice of PickWaveView entries for list APIs.
func EnrichPickWavesWithColdMeta(ctx context.Context, client *spanner.Client, waves []PickWaveView) []EnrichedPickWaveView {
	out := make([]EnrichedPickWaveView, 0, len(waves))
	for _, w := range waves {
		out = append(out, EnrichPickWaveWithColdMeta(ctx, client, w))
	}
	return out
}

// EnrichPickWaveWithColdMeta checks whether a pick wave's tasks or manifest carry
// cold-chain / migrated-staging SKUs lacking zone placement or a temperature baseline.
func EnrichPickWaveWithColdMeta(ctx context.Context, client *spanner.Client, wave PickWaveView) EnrichedPickWaveView {
	out := EnrichedPickWaveView{PickWaveView: wave}
	if client == nil {
		return out
	}

	pids := make([]string, 0, len(wave.Tasks))
	lids := make([]string, 0, len(wave.Tasks))
	for _, t := range wave.Tasks {
		pids = appendUniqueTrimmed(pids, t.ProductID)
		lids = appendUniqueTrimmed(lids, t.LocationID)
	}

	prodMeta := fetchProductsColdMeta(ctx, client, pids)
	locMeta := fetchBinsZoneMeta(ctx, client, wave.WarehouseID, lids)

	warnings := make([]string, 0)
	appendWarn := func(msg string) {
		warnings = appendUniqueTrimmed(warnings, msg)
	}

	hasColdSKU := false
	for _, t := range wave.Tasks {
		p := prodMeta[strings.TrimSpace(t.ProductID)]
		b := locMeta[strings.TrimSpace(t.LocationID)]
		reqZone, _, warn := ClassifyColdLotState(
			p.RequiresColdChain, p.IsPerishable, p.HandlingClass,
			t.LocationID, b.Zone, b.LocationType,
		)
		if reqZone != "" {
			hasColdSKU = true
		}
		if warn != "" {
			out.ColdLotMigratedWithoutLiveBaseline = true
			appendWarn(t.ProductID + ":" + reqZone + "@" + firstNonEmptyStr(t.LocationID, DefaultRecvLocationID))
		}
	}

	if wave.ManifestID != "" && (hasColdSKU || len(wave.Tasks) == 0) && EffectiveColdChain(ctx, wave.WarehouseID, wave.SupplierID) {
		if err := AssertColdChainBaselineReady(ctx, client, wave.ManifestID); errors.Is(err, ErrColdChainBaselineRequired) {
			out.ColdChainBaselineMissing = true
			out.ColdLotMigratedWithoutLiveBaseline = true
			appendWarn(ErrColdChainBaselineRequired.Error())
		}
	}
	if len(warnings) > 0 {
		out.ColdLotWarnings = warnings
	}
	return out
}

func appendUniqueTrimmed(dst []string, val string) []string {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return dst
	}
	for _, existing := range dst {
		if existing == trimmed {
			return dst
		}
	}
	return append(dst, trimmed)
}

func fetchProductsColdMeta(ctx context.Context, client *spanner.Client, pids []string) map[string]productColdMeta {
	out := make(map[string]productColdMeta, len(pids))
	if client == nil || len(pids) == 0 {
		return out
	}
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, Name, Barcode, HandlingClass, IsPerishable, RequiresColdChain
		      FROM Products WHERE ProductId IN UNNEST(@pids)`,
		Params: map[string]any{"pids": pids},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var pid, name, hc string
		var barcode spanner.NullString
		var perishable, cold bool
		if err := row.Columns(&pid, &name, &barcode, &hc, &perishable, &cold); err == nil {
			out[strings.TrimSpace(pid)] = productColdMeta{
				Name:              name,
				Barcode:           barcode.StringVal,
				HandlingClass:     hc,
				IsPerishable:      perishable,
				RequiresColdChain: cold,
			}
		}
	}
	return out
}

func fetchBinsZoneMeta(ctx context.Context, client *spanner.Client, warehouseID string, lids []string) map[string]binZoneMeta {
	out := make(map[string]binZoneMeta, len(lids))
	whID := strings.TrimSpace(warehouseID)
	if client == nil || whID == "" || len(lids) == 0 {
		return out
	}
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT LocationId, Zone, LocationType
		      FROM WarehouseLocations
		      WHERE WarehouseId = @wid AND LocationId IN UNNEST(@lids)`,
		Params: map[string]any{"wid": whID, "lids": lids},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err != nil && err != iterator.Done {
			break
		}
		if err == iterator.Done {
			break
		}
		var lid, locType string
		var zone spanner.NullString
		if err := row.Columns(&lid, &zone, &locType); err == nil {
			out[strings.TrimSpace(lid)] = binZoneMeta{
				Zone:         zone.StringVal,
				LocationType: locType,
			}
		}
	}
	return out
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
