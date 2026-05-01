package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
)

// DayWindow represents the open/close times for a single day.
type DayWindow struct {
	Open  string `json:"open"`  // "HH:MM" in UTC+5
	Close string `json:"close"` // "HH:MM" in UTC+5
}

// OperatingScheduleMap is the weekly schedule keyed by lowercase day name.
// Keys: "mon", "tue", "wed", "thu", "fri", "sat", "sun"
// A missing key means the supplier is closed on that day.
type OperatingScheduleMap map[string]DayWindow

// OperatingSchedule is the enhanced schedule wrapper that supports a 24/7 bypass.
// JSON format: {"is_24h": true, "schedules": {"mon":{"open":"09:00","close":"18:00"}, ...}}
// Backward compatible: bare map {"mon":{...}} is treated as is_24h=false.
type OperatingSchedule struct {
	Is24h     bool                 `json:"is_24h"`
	Schedules OperatingScheduleMap `json:"schedules"`
}

// parseOperatingSchedule decodes the schedule JSON, handling both legacy flat-map
// and the new {"is_24h":...,"schedules":{...}} envelope.
func parseOperatingSchedule(raw string) (OperatingSchedule, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return OperatingSchedule{}, nil
	}

	// Try the new envelope format first
	var sched OperatingSchedule
	if err := json.Unmarshal([]byte(trimmed), &sched); err == nil && sched.Schedules != nil {
		return sched, nil
	}

	// Fallback: legacy flat map {"mon":{...}}
	var flat OperatingScheduleMap
	if err := json.Unmarshal([]byte(trimmed), &flat); err != nil {
		return OperatingSchedule{}, err
	}
	return OperatingSchedule{Is24h: false, Schedules: flat}, nil
}

// dayKey returns the lowercase 3-letter day abbreviation for a given time.
func dayKey(t time.Time) string {
	return strings.ToLower(t.Weekday().String()[:3])
}

// resolveIsActive computes whether a supplier is currently accepting orders.
//
// Logic:
//  1. manualOff == true → always CLOSED (operator override)
//  2. scheduleJSON == "" or "{}" → no schedule configured → always OPEN
//  3. is_24h == true → always OPEN (perpetual node)
//  4. parse schedule → look up today's window → check if now (UTC+5) is within open/close → return bool
func resolveIsActive(scheduleJSON string, manualOff bool, now time.Time) bool {
	if manualOff {
		return false
	}

	// Normalize — empty or bare object means no restrictions (always open)
	trimmed := strings.TrimSpace(scheduleJSON)
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return true
	}

	sched, err := parseOperatingSchedule(trimmed)
	if err != nil {
		// Unparseable schedule — fail open (treat as always open)
		log.Printf("[shift] Failed to parse operating schedule JSON: %v", err)
		return true
	}

	// 24/7 bypass — perpetual availability
	if sched.Is24h {
		return true
	}

	if len(sched.Schedules) == 0 {
		return true
	}

	// All schedule comparison is done in UTC+5 (Uzbekistan)
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		// Fallback: manual UTC+5 offset
		loc = time.FixedZone("UTC+5", 5*60*60)
	}
	local := now.In(loc)

	window, ok := sched.Schedules[dayKey(local)]
	if !ok {
		// Day not in schedule → closed on this day
		return false
	}

	openTime, errO := parseHHMM(window.Open, local)
	closeTime, errC := parseHHMM(window.Close, local)
	if errO != nil || errC != nil {
		// Malformed times — fail open
		return true
	}

	return !local.Before(openTime) && local.Before(closeTime)
}

// parseHHMM parses "HH:MM" and returns a time.Time on the same date as ref.
func parseHHMM(hhmm string, ref time.Time) (time.Time, error) {
	parts := strings.SplitN(hhmm, ":", 2)
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", hhmm)
	}
	var hour, minute int
	if _, err := fmt.Sscanf(parts[0], "%d", &hour); err != nil {
		return time.Time{}, err
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minute); err != nil {
		return time.Time{}, err
	}
	return time.Date(ref.Year(), ref.Month(), ref.Day(), hour, minute, 0, 0, ref.Location()), nil
}

// ── PATCH /v1/supplier/shift ──────────────────────────────────────────────

type shiftPatchRequest struct {
	ManualOffShift    *bool           `json:"manual_off_shift"`
	OperatingSchedule json.RawMessage `json:"operating_schedule"`
}

// HandleSupplierShift toggles ManualOffShift and/or updates OperatingSchedule for
// the authenticated supplier.
//
// PATCH /v1/supplier/shift
// Body (either or both fields):
//
//	{"manual_off_shift": true}
//	{"operating_schedule": {"mon":{"open":"09:00","close":"18:00"}, ...}}
func HandleSupplierShift(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		var req shiftPatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
			return
		}

		if req.ManualOffShift == nil && len(req.OperatingSchedule) == 0 {
			http.Error(w, "Bad Request: provide manual_off_shift or operating_schedule", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		// Build Spanner mutation columns dynamically
		cols := []string{"SupplierId"}
		vals := []interface{}{supplierID}

		if req.ManualOffShift != nil {
			cols = append(cols, "ManualOffShift")
			vals = append(vals, *req.ManualOffShift)
		}

		if len(req.OperatingSchedule) > 0 {
			// Validate JSON is parseable as OperatingScheduleMap before storing
			var sched OperatingScheduleMap
			if err := json.Unmarshal(req.OperatingSchedule, &sched); err != nil {
				http.Error(w, "Bad Request: operating_schedule must be a valid schedule object", http.StatusBadRequest)
				return
			}
			schedStr := string(req.OperatingSchedule)
			cols = append(cols, "OperatingSchedule")
			vals = append(vals, spanner.NullJSON{Value: json.RawMessage(schedStr), Valid: true})
		}

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Suppliers", cols, vals),
			})
		})
		if err != nil {
			log.Printf("[shift] Failed to update shift for supplier %s: %v", supplierID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Read back current state to return is_active
		stmt := spanner.Statement{
			SQL:    `SELECT IFNULL(ManualOffShift, false), COALESCE(TO_JSON_STRING(OperatingSchedule), '{}') FROM Suppliers WHERE SupplierId = @id`,
			Params: map[string]interface{}{"id": supplierID},
		}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err != nil {
			log.Printf("[shift] Failed to read back shift state for supplier %s: %v", supplierID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var manualOff bool
		var schedJSON string
		if err := row.Columns(&manualOff, &schedJSON); err != nil {
			log.Printf("[shift] Failed to parse shift row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		isActive := resolveIsActive(schedJSON, manualOff, time.Now())

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"manual_off_shift":   manualOff,
			"operating_schedule": json.RawMessage(schedJSON),
			"is_active":          isActive,
		})
	}
}
