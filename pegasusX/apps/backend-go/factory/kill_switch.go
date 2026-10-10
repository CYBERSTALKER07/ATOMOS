package factory

import "strings"

func isSystemPlanningSource(source string) bool {
	switch strings.ToUpper(strings.TrimSpace(source)) {
	case TransferSourceThreshold, TransferSourcePredicted:
		return true
	default:
		return false
	}
}

func isKillSwitchCancellableState(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case TransferStateCreated, "DRAFT", TransferStateApproved:
		return true
	default:
		return false
	}
}

type killSwitchRow struct {
	TransferID string
	Source     string
	State      string
}

// ClassifyKillSwitch cancels SYSTEM_THRESHOLD / SYSTEM_PREDICTED drafts; leaves MANUAL.
func ClassifyKillSwitch(rows []killSwitchRow) (cancelIDs, keepIDs []string) {
	for _, row := range rows {
		if isSystemPlanningSource(row.Source) && isKillSwitchCancellableState(row.State) {
			cancelIDs = append(cancelIDs, row.TransferID)
			continue
		}
		keepIDs = append(keepIDs, row.TransferID)
	}
	return cancelIDs, keepIDs
}

func transferSLALevel(elapsedHours, promisedHours float64) string {
	if promisedHours <= 0 {
		promisedHours = 24
	}
	ratio := elapsedHours / promisedHours
	switch {
	case ratio >= 2:
		return "AUTO_REROUTE"
	case ratio >= 1.5:
		return "CRITICAL"
	case ratio >= 1:
		return "WARNING"
	default:
		return ""
	}
}
