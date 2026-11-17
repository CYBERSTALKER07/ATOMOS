package periods

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidatePeriodID parses and normalizes a monthly accounting period formatted as "YYYY-MM".
func ValidatePeriodID(periodID string) (year int, month int, normalized string, err error) {
	trimmed := strings.TrimSpace(periodID)
	parts := strings.Split(trimmed, "-")
	if len(parts) != 2 || len(parts[0]) != 4 || len(parts[1]) != 2 {
		return 0, 0, "", fmt.Errorf("%w: period_id must be formatted YYYY-MM, got %q", ErrInvalidInput, periodID)
	}
	yr, err := strconv.Atoi(parts[0])
	if err != nil || yr < 2000 || yr > 2100 {
		return 0, 0, "", fmt.Errorf("%w: invalid period year in %q", ErrInvalidInput, periodID)
	}
	mo, err := strconv.Atoi(parts[1])
	if err != nil || mo < 1 || mo > 12 {
		return 0, 0, "", fmt.Errorf("%w: invalid period month in %q", ErrInvalidInput, periodID)
	}
	return yr, mo, fmt.Sprintf("%04d-%02d", yr, mo), nil
}

// ResolveEntityTimezone resolves the canonical IANA timezone name and location for entityID.
// When explicitTZ is supplied, it is validated directly; otherwise auth/timezone_pack and GL market
// resolution are used (defaulting to Asia/Tashkent).
func ResolveEntityTimezone(ctx context.Context, entityID, explicitTZ string) (string, *time.Location, error) {
	tz := strings.TrimSpace(explicitTZ)
	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return "", nil, fmt.Errorf("%w: invalid timezone %q: %v", ErrInvalidInput, explicitTZ, err)
		}
		return tz, loc, nil
	}
	if name, err := auth.TimezoneNameFromContext(ctx, entityID); err == nil && strings.TrimSpace(name) != "" {
		if loc, lErr := time.LoadLocation(strings.TrimSpace(name)); lErr == nil {
			return strings.TrimSpace(name), loc, nil
		}
	}
	loc := gl.ResolveEntityLocation(ctx, entityID, "")
	if loc == nil {
		loc = time.UTC
	}
	return loc.String(), loc, nil
}

// ComputePeriodWindow calculates the UTC start/end timestamps [startUTC, endUTC) and local
// month-end cutoff string for periodID in tzName.
func ComputePeriodWindow(periodID, tzName string) (startUTC, endUTC time.Time, cutoffLocal, resolvedTZ string, err error) {
	yr, mo, _, err := ValidatePeriodID(periodID)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", err
	}
	tz := strings.TrimSpace(tzName)
	if tz == "" {
		tz = "Asia/Tashkent"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", fmt.Errorf("%w: invalid timezone %q: %v", ErrInvalidInput, tzName, err)
	}
	localStart := time.Date(yr, time.Month(mo), 1, 0, 0, 0, 0, loc)
	localEnd := localStart.AddDate(0, 1, 0)
	// Store the final second of the local month for human/UI display alongside exclusive UTC bounds.
	cutoffLocal = localEnd.Add(-time.Second).Format("2006-01-02T15:04:05-07:00")
	return localStart.UTC(), localEnd.UTC(), cutoffLocal, loc.String(), nil
}

// EnrichPeriodWindow populates computed window/cutoff fields on p without overwriting persisted metadata.
func EnrichPeriodWindow(p *Period) {
	if p == nil {
		return
	}
	if strings.TrimSpace(p.Timezone) == "" {
		p.Timezone = "Asia/Tashkent"
	}
	startUTC, endUTC, cutoffLocal, resolvedTZ, err := ComputePeriodWindow(p.PeriodID, p.Timezone)
	if err == nil {
		p.PeriodStartUTC = startUTC
		p.PeriodEndUTC = endUTC
		p.CutoffLocal = cutoffLocal
		p.Timezone = resolvedTZ
	}
}

// CheckPeriodInTxn resolves the effective posting period (and OriginalPeriodId when late
// operational events roll forward from SOFT_CLOSED or CLOSED periods) inside txn.
// Missing Periods rows default to OPEN so standalone GL callers require no pre-seeded periods.
// Wired into gl.SetPeriodLockChecker during application bootstrap.
func CheckPeriodInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	entityID string,
	occurredAt time.Time,
	sourceType string,
	requestedPeriodID string,
) (effectivePeriodID, originalPeriodID string, err error) {
	if txn == nil {
		return "", "", gl.ErrNilTransaction
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return "", "", fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	explicitRequested := strings.TrimSpace(requestedPeriodID) != ""
	basePeriod := strings.TrimSpace(requestedPeriodID)
	if basePeriod == "" {
		basePeriod = gl.ComputePeriodID(ctx, entityID, "", occurredAt)
	}

	st, rowTZ, found, err := readPeriodStatusInTxn(ctx, txn, entityID, basePeriod)
	if err != nil {
		return "", "", err
	}

	// If no explicit PeriodID was requested and an existing Period row carries an entity-specific
	// timezone (e.g. UTC vs Asia/Tashkent boundary crossing), align occurredAt to that timezone.
	if !explicitRequested {
		if found && strings.TrimSpace(rowTZ) != "" {
			if loc, lErr := time.LoadLocation(strings.TrimSpace(rowTZ)); lErr == nil {
				tzPeriod := occurredAt.In(loc).Format("2006-01")
				if tzPeriod != basePeriod {
					basePeriod = tzPeriod
					st, _, found, err = readPeriodStatusInTxn(ctx, txn, entityID, basePeriod)
					if err != nil {
						return "", "", err
					}
				}
			}
		} else if !found && (occurredAt.UTC().Day() <= 2 || occurredAt.UTC().Day() >= 28) {
			// Check immediately adjacent month rows in case a boundary occurredAt falls into
			// a neighbor period whose configured Timezone shifts the midnight cutoff.
			for _, candidate := range adjacentPeriodIDs(basePeriod) {
				candSt, candTZ, candFound, candErr := readPeriodStatusInTxn(ctx, txn, entityID, candidate)
				if candErr != nil {
					return "", "", candErr
				}
				if !candFound || strings.TrimSpace(candTZ) == "" {
					continue
				}
				if loc, lErr := time.LoadLocation(strings.TrimSpace(candTZ)); lErr == nil {
					if occurredAt.In(loc).Format("2006-01") == candidate {
						basePeriod = candidate
						st = candSt
						found = true
						break
					}
				}
			}
		}
	}

	if !found {
		return basePeriod, "", nil
	}

	switch strings.ToUpper(strings.TrimSpace(st)) {
	case "", StatusOpen, StatusReopened:
		return basePeriod, "", nil
	case StatusSoftClosed:
		if gl.IsFinanceAdjustmentSourceType(sourceType) {
			return basePeriod, "", nil
		}
		eff, rollErr := findEarliestOpenPeriodAfterInTxn(ctx, txn, entityID, basePeriod)
		if rollErr != nil {
			return "", "", rollErr
		}
		return eff, basePeriod, nil
	case StatusClosed:
		if gl.IsFinanceAdjustmentSourceType(sourceType) {
			return "", "", fmt.Errorf("%w: period %s is CLOSED for entity %s", gl.ErrPeriodClosed, basePeriod, entityID)
		}
		eff, rollErr := findEarliestOpenPeriodAfterInTxn(ctx, txn, entityID, basePeriod)
		if rollErr != nil {
			return "", "", rollErr
		}
		return eff, basePeriod, nil
	default:
		return "", "", fmt.Errorf("%w: period %s has blocking status %s", gl.ErrPeriodClosed, basePeriod, st)
	}
}

func readPeriodStatusInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, entityID, periodID string) (statusStr, tzStr string, found bool, err error) {
	row, readErr := txn.ReadRow(ctx, "Periods", spanner.Key{entityID, periodID}, []string{"Status", "Timezone"})
	if readErr != nil {
		if status.Code(readErr) == codes.NotFound || spanner.ErrCode(readErr) == codes.NotFound {
			return "", "", false, nil
		}
		return "", "", false, fmt.Errorf("read Periods(%s, %s): %w", entityID, periodID, readErr)
	}
	var st string
	var tz spanner.NullString
	if err := row.Columns(&st, &tz); err != nil {
		return "", "", false, fmt.Errorf("scan Periods(%s, %s): %w", entityID, periodID, err)
	}
	if tz.Valid {
		tzStr = strings.TrimSpace(tz.StringVal)
	}
	return strings.ToUpper(strings.TrimSpace(st)), tzStr, true, nil
}

func findEarliestOpenPeriodAfterInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, entityID, basePeriod string) (string, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT PeriodId, Status
		      FROM Periods
		      WHERE EntityId = @entityId AND PeriodId > @basePeriod
		      ORDER BY PeriodId ASC`,
		Params: map[string]any{
			"entityId":   entityID,
			"basePeriod": basePeriod,
		},
	})
	defer iter.Stop()

	statusByPeriod := make(map[string]string)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", fmt.Errorf("query future Periods for rollover: %w", err)
		}
		var pid, st string
		if err := row.Columns(&pid, &st); err != nil {
			return "", fmt.Errorf("scan future Period: %w", err)
		}
		statusByPeriod[strings.TrimSpace(pid)] = strings.ToUpper(strings.TrimSpace(st))
	}

	cand := basePeriod
	for i := 0; i < 120; i++ {
		next, ok := stepPeriodMonth(cand, 1)
		if !ok {
			return "", fmt.Errorf("%w: invalid base period %q", gl.ErrInvalidJournalInput, basePeriod)
		}
		cand = next
		st, exists := statusByPeriod[cand]
		if !exists || st == "" || st == StatusOpen || st == StatusReopened {
			return cand, nil
		}
	}
	return "", fmt.Errorf("%w: no open period found after %s", gl.ErrPeriodClosed, basePeriod)
}

func adjacentPeriodIDs(periodID string) []string {
	var out []string
	if prev, ok := stepPeriodMonth(periodID, -1); ok {
		out = append(out, prev)
	}
	if next, ok := stepPeriodMonth(periodID, 1); ok {
		out = append(out, next)
	}
	return out
}

func stepPeriodMonth(periodID string, delta int) (string, bool) {
	yr, mo, _, err := ValidatePeriodID(periodID)
	if err != nil {
		return "", false
	}
	totalMonths := yr*12 + (mo - 1) + delta
	if totalMonths < 2000*12 {
		return "", false
	}
	newYr := totalMonths / 12
	newMo := (totalMonths % 12) + 1
	return fmt.Sprintf("%04d-%02d", newYr, newMo), true
}
