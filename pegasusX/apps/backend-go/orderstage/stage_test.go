package orderstage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type fixture struct {
	Default         string            `json:"default"`
	ProgressPercent map[string]int    `json:"progress_percent"`
	Mapping         map[string]string `json:"mapping"`
}

// loadFixture reads contracts/order-customer-stage.json relative to this file.
func loadFixture(t *testing.T) fixture {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(here), "..", "..", "..", "contracts", "order-customer-stage.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if len(f.Mapping) == 0 {
		t.Fatal("fixture mapping is empty")
	}
	return f
}

func TestFromStringMatchesFixture(t *testing.T) {
	f := loadFixture(t)
	for status, want := range f.Mapping {
		if got := FromString(status); string(got) != want {
			t.Errorf("FromString(%q) = %s, fixture says %s", status, got, want)
		}
	}
}

func TestFromStringNormalizesInput(t *testing.T) {
	if got := FromString("  arriving "); got != OutForDelivery {
		t.Errorf("FromString with whitespace/lowercase = %s, want %s", got, OutForDelivery)
	}
}

func TestUnknownFallsBackToFixtureDefault(t *testing.T) {
	f := loadFixture(t)
	if got := FromString("SOMETHING_NEW"); string(got) != f.Default {
		t.Errorf("unknown status = %s, fixture default %s", got, f.Default)
	}
}

func TestProgressPercentMatchesFixture(t *testing.T) {
	f := loadFixture(t)
	for stage, want := range f.ProgressPercent {
		if got := ProgressPercent(Stage(stage)); got != want {
			t.Errorf("ProgressPercent(%s) = %d, fixture says %d", stage, got, want)
		}
	}
}
