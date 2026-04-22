package supplier

import (
	"testing"
)

// ── canonicalCategoryIndex ──────────────────────────────────────────────────

func TestCanonicalCategoryIndex_HasWater(t *testing.T) {
	if _, ok := canonicalCategoryIndex["cat-water"]; !ok {
		t.Fatal("canonical index should contain cat-water")
	}
}

func TestCanonicalCategoryIndex_HasOther(t *testing.T) {
	if _, ok := canonicalCategoryIndex["cat-other"]; !ok {
		t.Fatal("canonical index should contain cat-other (last entry)")
	}
}

func TestCanonicalCategoryIndex_MatchesCatalogLength(t *testing.T) {
	if got := len(canonicalCategoryIndex); got != len(canonicalCategories) {
		t.Fatalf("index has %d entries, catalog has %d — mismatch", got, len(canonicalCategories))
	}
}

func TestCanonicalCategories_Has50Entries(t *testing.T) {
	if got := len(canonicalCategories); got != 50 {
		t.Fatalf("canonical catalog has %d entries, want 50", got)
	}
}

func TestCanonicalCategories_SortOrderMonotonic(t *testing.T) {
	for i := 1; i < len(canonicalCategories); i++ {
		if canonicalCategories[i].SortOrder <= canonicalCategories[i-1].SortOrder {
			t.Fatalf("sort order not monotonic at index %d: %d <= %d",
				i, canonicalCategories[i].SortOrder, canonicalCategories[i-1].SortOrder)
		}
	}
}

// ── normalizeValidCategoryIDs ──────────────────────────────────────────────

func TestNormalizeValid_AllValid(t *testing.T) {
	valid, invalid := normalizeValidCategoryIDs([]string{"cat-water", "cat-juice"})
	if len(valid) != 2 || len(invalid) != 0 {
		t.Fatalf("valid=%v invalid=%v", valid, invalid)
	}
}

func TestNormalizeValid_AllInvalid(t *testing.T) {
	valid, invalid := normalizeValidCategoryIDs([]string{"not-real", "also-fake"})
	if len(valid) != 0 || len(invalid) != 2 {
		t.Fatalf("valid=%v invalid=%v", valid, invalid)
	}
}

func TestNormalizeValid_Mixed(t *testing.T) {
	valid, invalid := normalizeValidCategoryIDs([]string{"cat-water", "bogus", "cat-seafood"})
	if len(valid) != 2 || len(invalid) != 1 {
		t.Fatalf("expected 2 valid, 1 invalid; got valid=%v invalid=%v", valid, invalid)
	}
	if invalid[0] != "bogus" {
		t.Fatalf("expected invalid[0]=%q, got %q", "bogus", invalid[0])
	}
}

func TestNormalizeValid_Empty(t *testing.T) {
	valid, invalid := normalizeValidCategoryIDs(nil)
	if len(valid) != 0 || len(invalid) != 0 {
		t.Fatalf("nil input should return empty slices; got valid=%v invalid=%v", valid, invalid)
	}
}

func TestNormalizeValid_DeduplicatesInput(t *testing.T) {
	valid, _ := normalizeValidCategoryIDs([]string{"cat-water", "cat-water", "cat-water"})
	if len(valid) != 1 {
		t.Fatalf("expected 1 after dedup, got %d", len(valid))
	}
}

func TestNormalizeValid_TrimsWhitespace(t *testing.T) {
	valid, _ := normalizeValidCategoryIDs([]string{"  cat-water  "})
	if len(valid) != 1 || valid[0] != "cat-water" {
		t.Fatalf("expected trimmed match, got %v", valid)
	}
}

func TestNormalizeValid_SkipsBlanks(t *testing.T) {
	valid, invalid := normalizeValidCategoryIDs([]string{"", "  ", "cat-water"})
	if len(valid) != 1 || len(invalid) != 0 {
		t.Fatalf("blanks should be skipped; valid=%v invalid=%v", valid, invalid)
	}
}

func TestNormalizeValid_PreservesInputOrder(t *testing.T) {
	valid, _ := normalizeValidCategoryIDs([]string{"cat-seafood", "cat-water", "cat-juice"})
	if valid[0] != "cat-seafood" || valid[1] != "cat-water" || valid[2] != "cat-juice" {
		t.Fatalf("order should be preserved; got %v", valid)
	}
}

// ── categoryDisplayNameByID ────────────────────────────────────────────────

func TestDisplayName_KnownID_Water(t *testing.T) {
	if got := categoryDisplayNameByID("cat-water"); got != "Water" {
		t.Fatalf("cat-water = %q, want %q", got, "Water")
	}
}

func TestDisplayName_KnownID_Seafood(t *testing.T) {
	if got := categoryDisplayNameByID("cat-seafood"); got != "Seafood" {
		t.Fatalf("cat-seafood = %q, want %q", got, "Seafood")
	}
}

func TestDisplayName_UnknownID_ReturnsEmpty(t *testing.T) {
	if got := categoryDisplayNameByID("cat-nonexistent"); got != "" {
		t.Fatalf("unknown ID should return empty, got %q", got)
	}
}

// ── categoryDisplayNames ───────────────────────────────────────────────────

func TestDisplayNames_MultipleValid(t *testing.T) {
	names := categoryDisplayNames([]string{"cat-water", "cat-seafood"})
	if len(names) != 2 || names[0] != "Water" || names[1] != "Seafood" {
		t.Fatalf("expected [Water, Seafood], got %v", names)
	}
}

func TestDisplayNames_SkipsUnknown(t *testing.T) {
	names := categoryDisplayNames([]string{"cat-water", "bogus", "cat-juice"})
	if len(names) != 2 {
		t.Fatalf("expected 2 names (unknown skipped), got %v", names)
	}
}

func TestDisplayNames_Empty(t *testing.T) {
	names := categoryDisplayNames(nil)
	if len(names) != 0 {
		t.Fatalf("nil input should return empty, got %v", names)
	}
}

func TestDisplayNames_PreservesOrder(t *testing.T) {
	names := categoryDisplayNames([]string{"cat-seafood", "cat-water"})
	if names[0] != "Seafood" || names[1] != "Water" {
		t.Fatalf("order should be preserved; got %v", names)
	}
}

// ── primaryCategoryName ────────────────────────────────────────────────────

func TestPrimaryCategoryName_ReturnsFirst(t *testing.T) {
	if got := primaryCategoryName([]string{"cat-water", "cat-juice"}); got != "Water" {
		t.Fatalf("primary should be first element display name, got %q", got)
	}
}

func TestPrimaryCategoryName_Empty(t *testing.T) {
	if got := primaryCategoryName(nil); got != "" {
		t.Fatalf("empty input should return empty, got %q", got)
	}
}

func TestPrimaryCategoryName_UnknownFirst(t *testing.T) {
	if got := primaryCategoryName([]string{"bogus"}); got != "" {
		t.Fatalf("unknown first ID should return empty, got %q", got)
	}
}

// ── containsCategoryID ─────────────────────────────────────────────────────

func TestContainsCategoryID_Present(t *testing.T) {
	if !containsCategoryID([]string{"cat-water", "cat-juice"}, "cat-juice") {
		t.Fatal("should find cat-juice")
	}
}

func TestContainsCategoryID_Absent(t *testing.T) {
	if containsCategoryID([]string{"cat-water"}, "cat-juice") {
		t.Fatal("should not find cat-juice")
	}
}

func TestContainsCategoryID_EmptySlice(t *testing.T) {
	if containsCategoryID(nil, "cat-water") {
		t.Fatal("nil slice should not contain anything")
	}
}
