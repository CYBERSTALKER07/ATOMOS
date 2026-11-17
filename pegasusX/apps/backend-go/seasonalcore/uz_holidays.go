package seasonalcore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

// HolidayEntry defines one deterministic seasonal holiday window in YYYY-MM-DD bounds.
type HolidayEntry struct {
	Key        string  `json:"key"`
	TemplateID string  `json:"template_id"`
	Name       string  `json:"name"`
	StartDate  string  `json:"start_date"`
	EndDate    string  `json:"end_date"`
	Multiplier float64 `json:"multiplier"`
}

// uzbekistanHolidays2025_2035 is the deterministic 2025-2035 Uzbekistan holiday
// calendar (Muftiate lunar holiday windows + fixed national peaks).
// No astronomical runtime calculation is used so planning multipliers remain repeatable.
var uzbekistanHolidays2025_2035 = []HolidayEntry{
	// 2025
	{Key: "uz_ramadan_2025", TemplateID: "uz_ramadan", Name: "Ramadan 2025", StartDate: "2025-03-01", EndDate: "2025-03-29", Multiplier: 1.30},
	{Key: "uz_navruz_2025", TemplateID: "uz_navruz", Name: "Navruz 2025", StartDate: "2025-03-20", EndDate: "2025-03-24", Multiplier: 1.35},
	{Key: "uz_ramazon_hayit_2025", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2025", StartDate: "2025-03-30", EndDate: "2025-04-01", Multiplier: 1.45},
	{Key: "uz_qurbon_hayit_2025", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2025", StartDate: "2025-06-06", EndDate: "2025-06-09", Multiplier: 1.40},
	{Key: "uz_independence_2025", TemplateID: "uz_independence_day", Name: "Independence Day 2025", StartDate: "2025-08-30", EndDate: "2025-09-02", Multiplier: 1.20},

	// 2026
	{Key: "uz_ramadan_2026", TemplateID: "uz_ramadan", Name: "Ramadan 2026", StartDate: "2026-02-18", EndDate: "2026-03-19", Multiplier: 1.30},
	{Key: "uz_ramazon_hayit_2026", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2026", StartDate: "2026-03-20", EndDate: "2026-03-22", Multiplier: 1.45},
	{Key: "uz_navruz_2026", TemplateID: "uz_navruz", Name: "Navruz 2026", StartDate: "2026-03-20", EndDate: "2026-03-24", Multiplier: 1.35},
	{Key: "uz_qurbon_hayit_2026", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2026", StartDate: "2026-05-27", EndDate: "2026-05-30", Multiplier: 1.40},
	{Key: "uz_independence_2026", TemplateID: "uz_independence_day", Name: "Independence Day 2026", StartDate: "2026-08-30", EndDate: "2026-09-02", Multiplier: 1.20},

	// 2027
	{Key: "uz_ramadan_2027", TemplateID: "uz_ramadan", Name: "Ramadan 2027", StartDate: "2027-02-08", EndDate: "2027-03-08", Multiplier: 1.30},
	{Key: "uz_ramazon_hayit_2027", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2027", StartDate: "2027-03-09", EndDate: "2027-03-11", Multiplier: 1.45},
	{Key: "uz_navruz_2027", TemplateID: "uz_navruz", Name: "Navruz 2027", StartDate: "2027-03-20", EndDate: "2027-03-24", Multiplier: 1.35},
	{Key: "uz_qurbon_hayit_2027", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2027", StartDate: "2027-05-16", EndDate: "2027-05-19", Multiplier: 1.40},
	{Key: "uz_independence_2027", TemplateID: "uz_independence_day", Name: "Independence Day 2027", StartDate: "2027-08-30", EndDate: "2027-09-02", Multiplier: 1.20},

	// 2028
	{Key: "uz_ramadan_2028", TemplateID: "uz_ramadan", Name: "Ramadan 2028", StartDate: "2028-01-28", EndDate: "2028-02-25", Multiplier: 1.30},
	{Key: "uz_ramazon_hayit_2028", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2028", StartDate: "2028-02-26", EndDate: "2028-02-28", Multiplier: 1.45},
	{Key: "uz_navruz_2028", TemplateID: "uz_navruz", Name: "Navruz 2028", StartDate: "2028-03-20", EndDate: "2028-03-24", Multiplier: 1.35},
	{Key: "uz_qurbon_hayit_2028", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2028", StartDate: "2028-05-05", EndDate: "2028-05-08", Multiplier: 1.40},
	{Key: "uz_independence_2028", TemplateID: "uz_independence_day", Name: "Independence Day 2028", StartDate: "2028-08-30", EndDate: "2028-09-02", Multiplier: 1.20},

	// 2029
	{Key: "uz_ramadan_2029", TemplateID: "uz_ramadan", Name: "Ramadan 2029", StartDate: "2029-01-16", EndDate: "2029-02-13", Multiplier: 1.30},
	{Key: "uz_ramazon_hayit_2029", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2029", StartDate: "2029-02-14", EndDate: "2029-02-16", Multiplier: 1.45},
	{Key: "uz_navruz_2029", TemplateID: "uz_navruz", Name: "Navruz 2029", StartDate: "2029-03-20", EndDate: "2029-03-24", Multiplier: 1.35},
	{Key: "uz_qurbon_hayit_2029", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2029", StartDate: "2029-04-24", EndDate: "2029-04-27", Multiplier: 1.40},
	{Key: "uz_independence_2029", TemplateID: "uz_independence_day", Name: "Independence Day 2029", StartDate: "2029-08-30", EndDate: "2029-09-02", Multiplier: 1.20},

	// 2030 (two Hijri Ramadans begin in Gregorian 2030)
	{Key: "uz_ramadan_2030_1451", TemplateID: "uz_ramadan", Name: "Ramadan 2030 (1451 AH)", StartDate: "2030-01-05", EndDate: "2030-02-03", Multiplier: 1.30},
	{Key: "uz_ramazon_hayit_2030", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2030", StartDate: "2030-02-04", EndDate: "2030-02-06", Multiplier: 1.45},
	{Key: "uz_navruz_2030", TemplateID: "uz_navruz", Name: "Navruz 2030", StartDate: "2030-03-20", EndDate: "2030-03-24", Multiplier: 1.35},
	{Key: "uz_qurbon_hayit_2030", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2030", StartDate: "2030-04-13", EndDate: "2030-04-16", Multiplier: 1.40},
	{Key: "uz_independence_2030", TemplateID: "uz_independence_day", Name: "Independence Day 2030", StartDate: "2030-08-30", EndDate: "2030-09-02", Multiplier: 1.20},
	{Key: "uz_ramadan_2030_1452", TemplateID: "uz_ramadan", Name: "Ramadan 2030-2031 (1452 AH)", StartDate: "2030-12-26", EndDate: "2031-01-24", Multiplier: 1.30},

	// 2031
	{Key: "uz_ramazon_hayit_2031", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2031", StartDate: "2031-01-25", EndDate: "2031-01-27", Multiplier: 1.45},
	{Key: "uz_navruz_2031", TemplateID: "uz_navruz", Name: "Navruz 2031", StartDate: "2031-03-20", EndDate: "2031-03-24", Multiplier: 1.35},
	{Key: "uz_qurbon_hayit_2031", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2031", StartDate: "2031-04-03", EndDate: "2031-04-06", Multiplier: 1.40},
	{Key: "uz_independence_2031", TemplateID: "uz_independence_day", Name: "Independence Day 2031", StartDate: "2031-08-30", EndDate: "2031-09-02", Multiplier: 1.20},
	{Key: "uz_ramadan_2031", TemplateID: "uz_ramadan", Name: "Ramadan 2031-2032 (1453 AH)", StartDate: "2031-12-15", EndDate: "2032-01-13", Multiplier: 1.30},

	// 2032
	{Key: "uz_ramazon_hayit_2032", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2032", StartDate: "2032-01-14", EndDate: "2032-01-16", Multiplier: 1.45},
	{Key: "uz_navruz_2032", TemplateID: "uz_navruz", Name: "Navruz 2032", StartDate: "2032-03-20", EndDate: "2032-03-24", Multiplier: 1.35},
	{Key: "uz_qurbon_hayit_2032", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2032", StartDate: "2032-03-22", EndDate: "2032-03-25", Multiplier: 1.40},
	{Key: "uz_independence_2032", TemplateID: "uz_independence_day", Name: "Independence Day 2032", StartDate: "2032-08-30", EndDate: "2032-09-02", Multiplier: 1.20},
	{Key: "uz_ramadan_2032", TemplateID: "uz_ramadan", Name: "Ramadan 2032-2033 (1454 AH)", StartDate: "2032-12-04", EndDate: "2033-01-01", Multiplier: 1.30},

	// 2033
	{Key: "uz_ramazon_hayit_2033_1454", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2033 (1454 AH)", StartDate: "2033-01-02", EndDate: "2033-01-04", Multiplier: 1.45},
	{Key: "uz_qurbon_hayit_2033", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2033", StartDate: "2033-03-11", EndDate: "2033-03-14", Multiplier: 1.40},
	{Key: "uz_navruz_2033", TemplateID: "uz_navruz", Name: "Navruz 2033", StartDate: "2033-03-20", EndDate: "2033-03-24", Multiplier: 1.35},
	{Key: "uz_independence_2033", TemplateID: "uz_independence_day", Name: "Independence Day 2033", StartDate: "2033-08-30", EndDate: "2033-09-02", Multiplier: 1.20},
	{Key: "uz_ramadan_2033_1455", TemplateID: "uz_ramadan", Name: "Ramadan 2033 (1455 AH)", StartDate: "2033-11-23", EndDate: "2033-12-21", Multiplier: 1.30},
	{Key: "uz_ramazon_hayit_2033_1455", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2033 (1455 AH)", StartDate: "2033-12-22", EndDate: "2033-12-24", Multiplier: 1.45},

	// 2034
	{Key: "uz_qurbon_hayit_2034", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2034", StartDate: "2034-03-01", EndDate: "2034-03-04", Multiplier: 1.40},
	{Key: "uz_navruz_2034", TemplateID: "uz_navruz", Name: "Navruz 2034", StartDate: "2034-03-20", EndDate: "2034-03-24", Multiplier: 1.35},
	{Key: "uz_independence_2034", TemplateID: "uz_independence_day", Name: "Independence Day 2034", StartDate: "2034-08-30", EndDate: "2034-09-02", Multiplier: 1.20},
	{Key: "uz_ramadan_2034", TemplateID: "uz_ramadan", Name: "Ramadan 2034", StartDate: "2034-11-12", EndDate: "2034-12-10", Multiplier: 1.30},
	{Key: "uz_ramazon_hayit_2034", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2034", StartDate: "2034-12-11", EndDate: "2034-12-13", Multiplier: 1.45},

	// 2035
	{Key: "uz_qurbon_hayit_2035", TemplateID: "uz_qurbon_hayit", Name: "Qurbon Hayit 2035", StartDate: "2035-02-18", EndDate: "2035-02-21", Multiplier: 1.40},
	{Key: "uz_navruz_2035", TemplateID: "uz_navruz", Name: "Navruz 2035", StartDate: "2035-03-20", EndDate: "2035-03-24", Multiplier: 1.35},
	{Key: "uz_independence_2035", TemplateID: "uz_independence_day", Name: "Independence Day 2035", StartDate: "2035-08-30", EndDate: "2035-09-02", Multiplier: 1.20},
	{Key: "uz_ramadan_2035", TemplateID: "uz_ramadan", Name: "Ramadan 2035", StartDate: "2035-11-01", EndDate: "2035-11-30", Multiplier: 1.30},
	{Key: "uz_ramazon_hayit_2035", TemplateID: "uz_ramazon_hayit", Name: "Ramazon Hayit 2035", StartDate: "2035-12-01", EndDate: "2035-12-03", Multiplier: 1.45},
}

// UzbekistanHolidays2025_2035 returns a copy of the 2025-2035 Uzbekistan holiday schedule.
func UzbekistanHolidays2025_2035() []HolidayEntry {
	out := make([]HolidayEntry, len(uzbekistanHolidays2025_2035))
	copy(out, uzbekistanHolidays2025_2035)
	return out
}

// OverrideIDForHoliday returns a deterministic 36-char UUID for (supplierID, holidayKey)
// so re-seeding Uzbekistan holidays is strictly idempotent.
func OverrideIDForHoliday(supplierID, holidayKey string) string {
	seedStr := "uz_holiday:" + strings.TrimSpace(supplierID) + ":" + strings.TrimSpace(holidayKey)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(seedStr)).String()
}

// ComposeMultipliers composes all active seasonal multipliers multiplicatively and
// clamps the resulting factor into [MinMultiplier, MaxMultiplier].
func ComposeMultipliers(mults []float64) float64 {
	combined := 1.0
	hasPositive := false
	for _, m := range mults {
		if m > 0 {
			combined *= m
			hasPositive = true
		}
	}
	if !hasPositive {
		return 1.0
	}
	return ClampMultiplier(combined)
}

// ActiveUzbekistanHolidaysOn returns every scheduled Uzbekistan holiday window active on on (UTC date).
func ActiveUzbekistanHolidaysOn(on time.Time) []HolidayEntry {
	day := civil.DateOf(on.UTC())
	out := make([]HolidayEntry, 0)
	for _, h := range uzbekistanHolidays2025_2035 {
		sd, err1 := civil.ParseDate(h.StartDate)
		ed, err2 := civil.ParseDate(h.EndDate)
		if err1 != nil || err2 != nil {
			continue
		}
		if !day.Before(sd) && !day.After(ed) {
			out = append(out, h)
		}
	}
	return out
}

// SeedUzbekistanHolidays idempotently upserts the 2025-2035 Uzbekistan holiday schedule
// into SeasonalTemplateOverrides for supplierID. Returns the number of rows written.
func SeedUzbekistanHolidays(ctx context.Context, client *spanner.Client, supplierID string) (int, error) {
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return 0, errors.New("supplier_id_required")
	}
	if client == nil {
		return 0, nil
	}
	mutations := make([]*spanner.Mutation, 0, len(uzbekistanHolidays2025_2035))
	for _, h := range uzbekistanHolidays2025_2035 {
		sd, err := civil.ParseDate(h.StartDate)
		if err != nil {
			return 0, fmt.Errorf("parse start_date %s: %w", h.Key, err)
		}
		ed, err := civil.ParseDate(h.EndDate)
		if err != nil {
			return 0, fmt.Errorf("parse end_date %s: %w", h.Key, err)
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("SeasonalTemplateOverrides", map[string]any{
			"SupplierId": sid,
			"OverrideId": OverrideIDForHoliday(sid, h.Key),
			"TemplateId": h.TemplateID,
			"Name":       h.Name,
			"StartDate":  sd,
			"EndDate":    ed,
			"IsActive":   true,
			"Multiplier": ClampMultiplier(h.Multiplier),
			"CreatedAt":  spanner.CommitTimestamp,
		}))
	}
	if _, err := client.Apply(ctx, mutations); err != nil {
		return 0, fmt.Errorf("seed uzbekistan holidays: %w", err)
	}
	return len(mutations), nil
}
