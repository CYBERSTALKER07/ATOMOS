package order

import (
	"strings"
	"testing"
)

func TestOrderStateGraph_ParityWithValidateStatusTransition(t *testing.T) {
	all := AllStatuses()

	for _, from := range all {
		for _, to := range all {
			err := ValidateStatusTransition(string(from), string(to), TransitionOpts{})
			expectedAllowed := err == nil
			actualAllowed := CanTransition(from, to)

			if expectedAllowed != actualAllowed {
				t.Errorf("mismatch for %s -> %s: ValidateStatusTransition allowed=%v, CanTransition allowed=%v (err=%v)",
					from, to, expectedAllowed, actualAllowed, err)
			}
		}
	}
}

func TestOrderStateGraph_TerminalState(t *testing.T) {
	if !IsTerminal(StatusCompleted) {
		t.Errorf("expected COMPLETED to be terminal")
	}
	if IsTerminal(StatusPending) {
		t.Errorf("expected PENDING to not be terminal")
	}
	if IsTerminal(StatusCancelled) {
		t.Errorf("expected CANCELLED to not be terminal (has reconciliation branch)")
	}
}

func TestOrderStateGraph_Exporters(t *testing.T) {
	mermaid := ExportMermaid()
	if !strings.HasPrefix(mermaid, "flowchart TD\n") {
		t.Errorf("expected mermaid flowchart header, got %s", mermaid)
	}
	if !strings.Contains(mermaid, "PENDING --> LOADED") {
		t.Errorf("expected PENDING --> LOADED in mermaid export")
	}

	dot := ExportDOT()
	if !strings.HasPrefix(dot, "digraph OrderLifecycle {\n") {
		t.Errorf("expected DOT digraph header, got %s", dot)
	}
	if !strings.Contains(dot, "\"PENDING\" -> \"LOADED\";") {
		t.Errorf("expected \"PENDING\" -> \"LOADED\"; in DOT export")
	}
}
