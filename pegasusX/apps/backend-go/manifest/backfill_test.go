package manifest

import "testing"

func TestNormalizeBackfillOptions_defaults(t *testing.T) {
	opts := normalizeBackfillOptions(BackfillOptions{})
	if opts.Limit != 100 {
		t.Fatalf("limit = %d, want 100", opts.Limit)
	}
	if len(opts.States) != len(defaultBackfillStates) {
		t.Fatalf("states len = %d, want %d", len(opts.States), len(defaultBackfillStates))
	}
	for i, state := range defaultBackfillStates {
		if opts.States[i] != state {
			t.Fatalf("states[%d] = %q, want %q", i, opts.States[i], state)
		}
	}
}

func TestNormalizeBackfillOptions_preservesCustomLimit(t *testing.T) {
	opts := normalizeBackfillOptions(BackfillOptions{Limit: 25, States: []string{"SEALED"}})
	if opts.Limit != 25 {
		t.Fatalf("limit = %d, want 25", opts.Limit)
	}
	if len(opts.States) != 1 || opts.States[0] != "SEALED" {
		t.Fatalf("states = %#v, want [SEALED]", opts.States)
	}
}
