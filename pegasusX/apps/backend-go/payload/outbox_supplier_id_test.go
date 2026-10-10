package payload

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func TestPayloadOutboxFlushUsesEventRowMap(t *testing.T) {
	row := outbox.EventRowMap(outbox.Event{
		EventID:       "e1",
		AggregateType: "Manifest",
		AggregateID:   "mf-1",
		TopicName:     "main",
		Payload:       []byte(`{"supplier_id":"sup-1"}`),
		SupplierID:    "sup-1",
	})
	if got := row["SupplierId"]; got != "sup-1" {
		t.Fatalf("EventRowMap SupplierId=%v", got)
	}
}

func TestPayloadPackageOutboxInsertsUseEventRowMap(t *testing.T) {
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
