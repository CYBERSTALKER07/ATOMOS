package proximity

import "testing"

func TestCellsInRadiusIncludesCenter(t *testing.T) {
	t.Parallel()
	cells, err := CellsInRadius(41.3111, 69.2797, 9, 1)
	if err != nil {
		t.Fatalf("CellsInRadius: %v", err)
	}
	if len(cells) < 2 {
		t.Fatalf("expected center + neighbors, got %d cells", len(cells))
	}
}

func TestCellsInRadiusK0(t *testing.T) {
	t.Parallel()
	cells, err := CellsInRadius(41.3111, 69.2797, 9, 0)
	if err != nil {
		t.Fatalf("CellsInRadius: %v", err)
	}
	if len(cells) != 1 {
		t.Fatalf("k=0 want 1 cell, got %d", len(cells))
	}
}
