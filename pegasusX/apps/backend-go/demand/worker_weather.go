package demand

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

type WeatherConfig struct {
	BaseURL        string
	UpdateInterval time.Duration
	LookaheadDays  int
	Locations      []Location
}

type Location struct {
	Scope string
	Lat   float64
	Lng   float64
}

type openMeteoResponse struct {
	Daily struct {
		Time             []string  `json:"time"`
		Temperature2mMax []float64 `json:"temperature_2m_max"`
		Temperature2mMin []float64 `json:"temperature_2m_min"`
		PrecipitationSum []float64 `json:"precipitation_sum"`
	} `json:"daily"`
}

// RunWeatherIngestionWorker polls the forecast API and upserts DemandSignals.
func (s *Service) RunWeatherIngestionWorker(ctx context.Context, cfg WeatherConfig) {
	ticker := time.NewTicker(cfg.UpdateInterval)
	defer ticker.Stop()

	// Initial run
	_ = s.ingestWeather(ctx, cfg)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.ingestWeather(ctx, cfg)
		}
	}
}

var LastWeatherIngestionTime time.Time

func (s *Service) ingestWeather(ctx context.Context, cfg WeatherConfig) error {
	// Discover active locations directly from Retailers
	stmt := spanner.Statement{
		SQL: `
			SELECT H3Cell, ANY_VALUE(Lat) as Lat, ANY_VALUE(Lng) as Lng
			FROM Retailers
			WHERE H3Cell IS NOT NULL
			GROUP BY H3Cell
		`,
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var locations []Location
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var h3 string
		var lat, lng float64
		if err := row.Columns(&h3, &lat, &lng); err != nil {
			return err
		}
		locations = append(locations, Location{
			Scope: "h3:" + h3,
			Lat:   lat,
			Lng:   lng,
		})
	}

	// Fallback to configured locations if database is empty
	if len(locations) == 0 {
		locations = cfg.Locations
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var allSignals []DemandSignal

	for _, loc := range locations {
		url := fmt.Sprintf("%s?latitude=%f&longitude=%f&daily=temperature_2m_max,temperature_2m_min,precipitation_sum&forecast_days=%d&timezone=auto",
			cfg.BaseURL, loc.Lat, loc.Lng, cfg.LookaheadDays)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		var forecast openMeteoResponse
		if err := json.NewDecoder(resp.Body).Decode(&forecast); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for i, dateStr := range forecast.Daily.Time {
			if i >= len(forecast.Daily.Temperature2mMax) || i >= len(forecast.Daily.PrecipitationSum) || i >= len(forecast.Daily.Temperature2mMin) {
				continue
			}
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}
			tempMax := forecast.Daily.Temperature2mMax[i]
			tempMin := forecast.Daily.Temperature2mMin[i]
			precip := forecast.Daily.PrecipitationSum[i]

			multiplier := calculateWeatherMultiplier(tempMax, precip) // Phase 1 uses max temp for multiplier
			meta, _ := json.Marshal(map[string]any{
				"tempMaxC": tempMax,
				"tempMinC": tempMin,
				"precipMm": precip,
				"provider": "open-meteo",
			})

			// Generate deterministic ID
			seed := fmt.Sprintf("weather:%s:%s", loc.Scope, date.Format("2006-01-02"))
			id := uuid.NewMD5(uuid.NameSpaceOID, []byte(seed)).String()

			allSignals = append(allSignals, DemandSignal{
				SignalId:   id,
				Type:       "WEATHER",
				Scope:      loc.Scope,
				StartAt:    date,
				EndAt:      date.Add(24 * time.Hour).Add(-time.Nanosecond), // EOD
				Multiplier: multiplier,
				Meta:       meta,
				CreatedAt:  time.Now().UTC(),
				CreatedBy:  "system:weather-worker",
			})
		}
	}

	if len(allSignals) > 0 {
		err := s.UpsertSignals(ctx, allSignals)
		if err == nil {
			LastWeatherIngestionTime = time.Now().UTC()
		}
		return err
	}

	LastWeatherIngestionTime = time.Now().UTC()
	return nil
}

// calculateWeatherMultiplier calculates a continuous demand multiplier based on extreme weather conditions.
// Uses linear scaling for more dynamic, enterprise-grade modeling.
func calculateWeatherMultiplier(tempC, precipMm float64) float64 {
	m := 1.0

	// Heat waves increase demand
	if tempC > 30.0 {
		m += (tempC - 30.0) * 0.02
	}

	// Freezing cold increases demand
	if tempC < 5.0 {
		m += (5.0 - tempC) * 0.02
	}

	// Heavy precipitation dampens demand (or creates supply constraints modeled as demand drops)
	if precipMm > 0 {
		m -= precipMm * 0.01
	}

	return math.Max(0.70, math.Min(1.30, m))
}

// UpsertSignals idempotently writes signals using InsertOrUpdate.
func (s *Service) UpsertSignals(ctx context.Context, signals []DemandSignal) error {
	var muts []*spanner.Mutation
	for _, sig := range signals {
		var sku spanner.NullString
		if sig.Sku != nil {
			sku = spanner.NullString{StringVal: *sig.Sku, Valid: true}
		}
		var meta spanner.NullJSON
		if len(sig.Meta) > 0 {
			meta = spanner.NullJSON{Value: string(sig.Meta), Valid: true}
		}
		muts = append(muts, spanner.InsertOrUpdateMap("DemandSignals", map[string]any{
			"SignalId":   sig.SignalId,
			"Type":       string(sig.Type),
			"Scope":      sig.Scope,
			"Sku":        sku,
			"StartAt":    sig.StartAt,
			"EndAt":      sig.EndAt,
			"Multiplier": sig.Multiplier,
			"Meta":       meta,
			"CreatedAt":  sig.CreatedAt,
			"CreatedBy":  sig.CreatedBy,
		}))
	}

	// Batch by 500
	for i := 0; i < len(muts); i += 500 {
		end := i + 500
		if end > len(muts) {
			end = len(muts)
		}
		if _, err := s.spanner.Apply(ctx, muts[i:end]); err != nil {
			return err
		}
	}
	return nil
}
