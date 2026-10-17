package retailer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetailerPackageOutboxInsertsUseEventRowMap(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), `InsertOrUpdateMap("OutboxEvents", map[`) {
			t.Errorf("%s: hand-rolled OutboxEvents map; SupplierId is NOT NULL — use outbox.EventRowMap", name)
		}
	}
}
