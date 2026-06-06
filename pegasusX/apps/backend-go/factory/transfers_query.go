package factory

import (
	"net/http"
	"strconv"
	"strings"
)

func parseTransferStateFilters(r *http.Request) []string {
	filters := make([]string, 0, 4)
	if state := strings.TrimSpace(r.URL.Query().Get("state")); state != "" {
		filters = append(filters, strings.ToUpper(state))
	}
	if states := strings.TrimSpace(r.URL.Query().Get("states")); states != "" {
		for _, state := range strings.Split(states, ",") {
			normalized := strings.ToUpper(strings.TrimSpace(state))
			if normalized != "" {
				filters = append(filters, normalized)
			}
		}
	}
	return filters
}

func parseTransferLimit(raw string) int {
	if raw == "" {
		return 100
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 100
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func parseTransferOffset(raw string) int {
	if raw == "" {
		return 0
	}
	offset, err := strconv.Atoi(raw)
	if err != nil || offset < 0 {
		return 0
	}
	return offset
}

func filterTransferRows(rows []TransferRow, stateFilters []string) []TransferRow {
	if len(stateFilters) == 0 {
		return rows
	}
	allowed := make(map[string]struct{}, len(stateFilters))
	for _, state := range stateFilters {
		allowed[state] = struct{}{}
	}
	filtered := make([]TransferRow, 0, len(rows))
	for i := range rows {
		if _, ok := allowed[strings.ToUpper(strings.TrimSpace(rows[i].State))]; ok {
			filtered = append(filtered, rows[i])
		}
	}
	return filtered
}

func dispatchableTransferState(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "CREATED", "APPROVED", "LOADING":
		return true
	default:
		return false
	}
}

func isValidTransferTransition(from, to string) bool {
	allowed, ok := validTransferTransitions[strings.ToUpper(strings.TrimSpace(from))]
	if !ok {
		return false
	}
	target := strings.ToUpper(strings.TrimSpace(to))
	for _, state := range allowed {
		if state == target {
			return true
		}
	}
	return false
}

var validTransferTransitions = map[string][]string{
	"CREATED":    {"APPROVED", "CANCELLED"},
	"APPROVED":   {"LOADING", "CANCELLED"},
	"LOADING":    {"DISPATCHED"},
	"DISPATCHED": {"IN_TRANSIT"},
	"IN_TRANSIT": {"ARRIVED"},
	"ARRIVED":    {"RECEIVED"},
}
