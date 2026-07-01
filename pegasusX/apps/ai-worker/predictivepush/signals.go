package predictivepush

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
)

// DemandSignal describes an external or internal driver for demand sensing.
type DemandSignal struct {
	SupplierID  string
	WarehouseID string
	ProductID   string
	Qty         int64
	Confidence  float64
	Source      string
	TargetDate  time.Time
}

// DemandSignalProvider aggregates pluggable demand drivers (order history, preorders, stubs).
type DemandSignalProvider interface {
	Collect(ctx context.Context, supplierID string, targetDay time.Time) ([]DemandSignal, error)
}

// CompositeSignalProvider merges built-in and stub providers.
type CompositeSignalProvider struct {
	History *Analyzer
}

// NewCompositeSignalProvider wires the default PX90 signal stack.
func NewCompositeSignalProvider(client *spanner.Client) *CompositeSignalProvider {
	if client == nil {
		return &CompositeSignalProvider{}
	}
	return &CompositeSignalProvider{History: NewAnalyzer(client)}
}

// Collect merges order-history patterns with stub adapters for future pegasus drivers.
func (p *CompositeSignalProvider) Collect(ctx context.Context, supplierID string, targetDay time.Time) ([]DemandSignal, error) {
	var out []DemandSignal
	if p != nil && p.History != nil {
		events, err := p.History.Analyze(ctx, targetDay)
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			if supplierID != "" && ev.SupplierId != supplierID {
				continue
			}
			out = append(out, DemandSignal{
				SupplierID: ev.SupplierId,
				ProductID:  ev.ProductId,
				Qty:        ev.Quantity,
				Confidence: ev.Confidence,
				Source:     "order_history",
				TargetDate: ev.TargetDate,
			})
		}
	}
	out = append(out, stubPreorderCalendarSignals(supplierID, targetDay)...)
	out = append(out, stubSeasonalitySignals(supplierID, targetDay)...)
	out = append(out, externalWeatherSignals(supplierID, targetDay)...)
	out = append(out, externalPOSSignals(supplierID, targetDay)...)
	if out == nil {
		out = []DemandSignal{}
	}
	return out, nil
}

func stubPreorderCalendarSignals(supplierID string, targetDay time.Time) []DemandSignal {
	_ = supplierID
	_ = targetDay
	return nil
}

// stubSeasonalitySignals applies a lightweight day-of-week uplift until external POS/weather feeds land.
func stubSeasonalitySignals(supplierID string, targetDay time.Time) []DemandSignal {
	if supplierID == "" {
		return nil
	}
	weekday := targetDay.Weekday()
	if weekday != time.Friday && weekday != time.Saturday {
		return nil
	}
	return []DemandSignal{{
		SupplierID: supplierID,
		Qty:        1,
		Confidence: 0.35,
		Source:     "seasonality_stub",
		TargetDate: targetDay,
	}}
}

// externalWeatherSignals is a placeholder for pegasus weather-ingest (low confidence until wired).
func externalWeatherSignals(supplierID string, targetDay time.Time) []DemandSignal {
	if supplierID == "" {
		return nil
	}
	month := targetDay.Month()
	if month < time.June || month > time.August {
		return nil
	}
	return []DemandSignal{{
		SupplierID: supplierID,
		Qty:        2,
		Confidence: 0.4,
		Source:     "weather_forecast_stub",
		TargetDate: targetDay,
	}}
}

// externalPOSSignals is a placeholder for retailer POS calendar feeds.
func externalPOSSignals(supplierID string, targetDay time.Time) []DemandSignal {
	if supplierID == "" {
		return nil
	}
	day := targetDay.Day()
	lastDay := time.Date(targetDay.Year(), targetDay.Month()+1, 0, 0, 0, 0, 0, targetDay.Location()).Day()
	if day != 1 && day != 15 && day != lastDay {
		return nil
	}
	return []DemandSignal{{
		SupplierID: supplierID,
		Qty:        3,
		Confidence: 0.45,
		Source:     "pos_calendar_stub",
		TargetDate: targetDay,
	}}
}
