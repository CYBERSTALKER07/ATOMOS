package gl

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PeriodLockCheckerFunc resolves the effective posting period (and original period when
// late operational events roll forward out of SOFT_CLOSED/CLOSED periods) inside txn.
type PeriodLockCheckerFunc func(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	entityID string,
	occurredAt time.Time,
	sourceType string,
	requestedPeriodID string,
) (effectivePeriodID, originalPeriodID string, err error)

// PeriodLockChecker is the interface variant for period lock validation.
type PeriodLockChecker interface {
	CheckPeriodInTxn(
		ctx context.Context,
		txn *spanner.ReadWriteTransaction,
		entityID string,
		occurredAt time.Time,
		sourceType string,
		requestedPeriodID string,
	) (effectivePeriodID, originalPeriodID string, err error)
}

var (
	periodLockCheckerMu sync.RWMutex
	periodLockChecker   PeriodLockCheckerFunc
)

// SetPeriodLockChecker wires an external period lock checker (e.g., periods.CheckPeriodInTxn).
// Pass nil to revert to DefaultCheckPeriodInTxn.
func SetPeriodLockChecker(fn PeriodLockCheckerFunc) {
	periodLockCheckerMu.Lock()
	defer periodLockCheckerMu.Unlock()
	periodLockChecker = fn
}

func activePeriodLockChecker() PeriodLockCheckerFunc {
	periodLockCheckerMu.RLock()
	fn := periodLockChecker
	periodLockCheckerMu.RUnlock()
	if fn != nil {
		return fn
	}
	return DefaultCheckPeriodInTxn
}

// IsFinanceAdjustmentSourceType reports whether sourceType is a manual/finance adjustment
// allowed in SOFT_CLOSED periods and blocked (409 period_closed) in CLOSED periods.
func IsFinanceAdjustmentSourceType(sourceType string) bool {
	switch strings.ToUpper(strings.TrimSpace(sourceType)) {
	case "MANUAL", "PERIOD_ADJUSTMENT", "REVERSAL", "FX_REVALUATION", "INTERCOMPANY_TRANSFER":
		return true
	default:
		return false
	}
}

// DefaultCheckPeriodInTxn enforces Periods table locks inside txn.
// Missing Periods rows default to OPEN (preserving standalone GL tests).
func DefaultCheckPeriodInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	entityID string,
	occurredAt time.Time,
	sourceType string,
	requestedPeriodID string,
) (effectivePeriodID, originalPeriodID string, err error) {
	if txn == nil {
		return "", "", ErrNilTransaction
	}
	entityID = strings.TrimSpace(entityID)
	basePeriod := strings.TrimSpace(requestedPeriodID)
	if basePeriod == "" {
		basePeriod = ComputePeriodID(ctx, entityID, "", occurredAt)
	}

	row, readErr := txn.ReadRow(ctx, "Periods", spanner.Key{entityID, basePeriod}, []string{"Status"})
	if readErr != nil {
		if status.Code(readErr) == codes.NotFound || spanner.ErrCode(readErr) == codes.NotFound {
			return basePeriod, "", nil
		}
		return "", "", fmt.Errorf("read Periods(%s, %s): %w", entityID, basePeriod, readErr)
	}
	var st string
	if err := row.Columns(&st); err != nil {
		return "", "", fmt.Errorf("scan Periods(%s, %s): %w", entityID, basePeriod, err)
	}
	st = strings.ToUpper(strings.TrimSpace(st))
	switch st {
	case "", "OPEN", "REOPENED":
		return basePeriod, "", nil
	case "SOFT_CLOSED":
		if IsFinanceAdjustmentSourceType(sourceType) {
			return basePeriod, "", nil
		}
		eff, rollErr := findEarliestOpenPeriodAfterInTxn(ctx, txn, entityID, basePeriod)
		if rollErr != nil {
			return "", "", rollErr
		}
		return eff, basePeriod, nil
	case "CLOSED":
		if IsFinanceAdjustmentSourceType(sourceType) {
			return "", "", fmt.Errorf("%w: period %s is CLOSED for entity %s", ErrPeriodClosed, basePeriod, entityID)
		}
		eff, rollErr := findEarliestOpenPeriodAfterInTxn(ctx, txn, entityID, basePeriod)
		if rollErr != nil {
			return "", "", rollErr
		}
		return eff, basePeriod, nil
	default:
		return "", "", fmt.Errorf("%w: period %s has blocking status %s", ErrPeriodClosed, basePeriod, st)
	}
}

func findEarliestOpenPeriodAfterInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	entityID, basePeriod string,
) (string, error) {
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
		next, ok := incrementPeriodMonth(cand)
		if !ok {
			return "", fmt.Errorf("%w: invalid base period %q", ErrInvalidJournalInput, basePeriod)
		}
		cand = next
		st, exists := statusByPeriod[cand]
		if !exists || st == "" || st == "OPEN" || st == "REOPENED" {
			return cand, nil
		}
	}
	return "", fmt.Errorf("%w: no open period found after %s", ErrPeriodClosed, basePeriod)
}

func incrementPeriodMonth(periodID string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(periodID), "-")
	if len(parts) != 2 {
		return "", false
	}
	yr, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", false
	}
	mo, err := strconv.Atoi(parts[1])
	if err != nil || mo < 1 || mo > 12 {
		return "", false
	}
	mo++
	if mo > 12 {
		mo = 1
		yr++
	}
	return fmt.Sprintf("%04d-%02d", yr, mo), true
}
