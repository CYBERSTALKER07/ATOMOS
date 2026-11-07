package order

import (
	"fmt"
	"sort"
	"strings"
)

// TransitionEdge models a directed edge between two canonical order statuses.
type TransitionEdge struct {
	From        Status `json:"from"`
	To          Status `json:"to"`
	Description string `json:"description,omitempty"`
	Actor       string `json:"actor,omitempty"`
}

// OrderStateGraph encodes the authoritative order lifecycle transition matrix
// matching ValidateStatusTransition.
var OrderStateGraph = map[Status][]Status{
	StatusScheduled: {
		StatusAutoAccepted,
		StatusPending,
		StatusCancelled,
		StatusCancelRequested,
	},
	StatusAutoAccepted: {
		StatusPending,
		StatusCancelled,
		StatusCancelRequested,
	},
	StatusBackordered: {
		StatusPending,
		StatusScheduled,
		StatusCancelled,
	},
	StatusPending: {
		StatusLoaded,
		StatusDelayed,
		StatusCancelled,
	},
	StatusDelayed: {
		StatusPending,
	},
	StatusLoaded: {
		StatusInTransit,
		StatusPending,
		StatusDelayed,
		StatusCancelled,
		StatusCancelRequested,
	},
	StatusInTransit: {
		StatusArrived,
		StatusPending,
		StatusCancelled,
		StatusCancelRequested,
	},
	StatusArrived: {
		StatusAwaitingPayment,
		StatusPendingCashCollection,
		StatusDeliveredOnCredit,
		StatusShopClosedPending,
		StatusCancelRequested,
	},
	StatusShopClosedPending: {
		StatusAwaitingPayment,
		StatusPendingCashCollection,
		StatusDeliveredOnCredit,
		StatusArrived,
		StatusCancelRequested,
		StatusCancelled,
	},
	StatusAwaitingPayment: {
		StatusFiscalizing,
		StatusPendingCashCollection,
		StatusDeliveredOnCredit,
	},
	StatusPendingCashCollection: {
		StatusFiscalizing,
	},
	StatusDeliveredOnCredit: {
		StatusFiscalizing,
	},
	StatusFiscalizing: {
		StatusCompleted,
		StatusFiscalFailed,
	},
	StatusFiscalFailed: {
		StatusFiscalizing,
		StatusCompleted,
	},
	StatusCancelRequested: {
		StatusCancelled,
		StatusLoaded,
		StatusInTransit,
		StatusArrived,
	},
	StatusCancelled: {
		StatusReconciliationRequired,
	},
	StatusReconciliationRequired: {
		StatusCompleted,
		StatusCancelled,
	},
	StatusCompleted: {}, // Terminal: zero outbound transitions
}

// CanTransition reports whether transitioning from `from` to `to` is permitted.
func CanTransition(from, to Status) bool {
	if from == to {
		return true // Idempotent no-op transitions are always allowed
	}
	destinations, exists := OrderStateGraph[from]
	if !exists {
		return false
	}
	for _, target := range destinations {
		if target == to {
			return true
		}
	}
	return false
}

// NextStates returns a copy of permitted successor states for a status.
func NextStates(from Status) []Status {
	targets, exists := OrderStateGraph[from]
	if !exists {
		return nil
	}
	out := make([]Status, len(targets))
	copy(out, targets)
	return out
}

// IsTerminal reports whether the status represents a final state with no outbound exits.
func IsTerminal(s Status) bool {
	targets, exists := OrderStateGraph[s]
	return exists && len(targets) == 0
}

// AllStatuses returns a sorted list of all canonical statuses registered in the graph.
func AllStatuses() []Status {
	keys := make([]string, 0, len(OrderStateGraph))
	for k := range OrderStateGraph {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	out := make([]Status, len(keys))
	for i, k := range keys {
		out[i] = Status(k)
	}
	return out
}

// ExportMermaid exports the state machine graph as a Mermaid flowchart string.
func ExportMermaid() string {
	var sb strings.Builder
	sb.WriteString("flowchart TD\n")

	keys := make([]string, 0, len(OrderStateGraph))
	for k := range OrderStateGraph {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	for _, k := range keys {
		from := Status(k)
		for _, to := range OrderStateGraph[from] {
			sb.WriteString(fmt.Sprintf("    %s --> %s\n", from, to))
		}
	}
	return sb.String()
}

// ExportDOT exports the state machine graph in Graphviz DOT format.
func ExportDOT() string {
	var sb strings.Builder
	sb.WriteString("digraph OrderLifecycle {\n")
	sb.WriteString("    rankdir=TB;\n")
	sb.WriteString("    node [shape=box, style=\"rounded,filled\", fillcolor=\"#ECEFF1\", fontname=\"Helvetica\"];\n")

	keys := make([]string, 0, len(OrderStateGraph))
	for k := range OrderStateGraph {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	for _, k := range keys {
		from := Status(k)
		for _, to := range OrderStateGraph[from] {
			sb.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\";\n", from, to))
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}
