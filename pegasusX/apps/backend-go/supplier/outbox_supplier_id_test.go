package supplier

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func TestPortalOutboxMutationUsesEventRowMap(t *testing.T) {
	row := outbox.EventRowMap(outbox.Event{
		EventID:       "e1",
		AggregateType: "Supplier",
		AggregateID:   "sup-1",
		TopicName:     "main",
		Payload:       []byte(`{"supplier_id":"sup-1"}`),
		SupplierID:    "sup-1",
	})
	if got := row["SupplierId"]; got != "sup-1" {
		t.Fatalf("EventRowMap SupplierId=%v", got)
	}
}

func TestSupplierPackageOutboxInsertsUseEventRowMap(t *testing.T) {
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
		src := string(body)
		if strings.Contains(src, `InsertOrUpdateMap("OutboxEvents", map[`) {
			t.Errorf("%s: hand-rolled OutboxEvents map; SupplierId is NOT NULL — use portalOutboxMutation / outbox.EventRowMap", name)
		}
	}
}
