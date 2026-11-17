package events

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestM05OntologyEventContract(t *testing.T) {
	if EventOntologyObjectChanged != "ontology.object.changed" {
		t.Fatalf("EventOntologyObjectChanged = %q, want %q", EventOntologyObjectChanged, "ontology.object.changed")
	}
	if EventOntologyObjectUpdated != "ONTOLOGY_OBJECT_UPDATED" {
		t.Fatalf("EventOntologyObjectUpdated = %q, want %q", EventOntologyObjectUpdated, "ONTOLOGY_OBJECT_UPDATED")
	}
	if AggregateOntologyObject != "OntologyObject" {
		t.Fatalf("AggregateOntologyObject = %q, want %q", AggregateOntologyObject, "OntologyObject")
	}
	if TopicIntel != "pegasusx-intel" {
		t.Fatalf("TopicIntel = %q, want %q", TopicIntel, "pegasusx-intel")
	}
	if TopicConnectors != "pegasusx-connectors" {
		t.Fatalf("TopicConnectors = %q, want %q", TopicConnectors, "pegasusx-connectors")
	}

	ev := OntologyObjectChangedEvent{
		BaseEvent: BaseEvent{
			Type:      EventOntologyObjectChanged,
			Version:   1,
			Timestamp: "2026-11-17T00:40:00Z",
		},
		CellID:           "uz-tashkent-01",
		ObjectType:       "Order",
		ObjectID:         "ord-101",
		SupplierID:       "sup-1",
		Version:          3,
		ChangeKind:       "UPSERT",
		SourceEventType:  EventOrderCreated,
		SourceEventID:    "evt-900",
		SecurityMarkings: []string{"MARKING:FINANCE_ONLY"},
		UpdatedAt:        "2026-11-17T00:40:00Z",
	}

	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal OntologyObjectChangedEvent: %v", err)
	}
	var decMap map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&decMap); err != nil {
		t.Fatalf("unmarshal OntologyObjectChangedEvent: %v", err)
	}
	for k, want := range map[string]any{
		"type":              EventOntologyObjectChanged,
		"cell_id":           "uz-tashkent-01",
		"object_type":       "Order",
		"object_id":         "ord-101",
		"supplier_id":       "sup-1",
		"change_kind":       "UPSERT",
		"source_event_type": EventOrderCreated,
		"source_event_id":   "evt-900",
		"updated_at":        "2026-11-17T00:40:00Z",
	} {
		if decMap[k] != want {
			t.Fatalf("event[%q] = %v, want %v", k, decMap[k], want)
		}
	}
	if decMap["version"] != json.Number("3") {
		t.Fatalf("version = %v, want 3", decMap["version"])
	}

	envelopePath := filepath.Clean(filepath.Join("..", "..", "..", "packages", "types", "src", "envelope.ts"))
	envelopeSrc, err := os.ReadFile(envelopePath)
	if err != nil {
		t.Fatalf("read %s: %v", envelopePath, err)
	}
	if !strings.Contains(string(envelopeSrc), `"ontology.object.changed"`) {
		t.Fatalf("%s missing \"ontology.object.changed\" in EventType union", envelopePath)
	}
}

func TestM05OntologySchemaRegistryAndTypesParity(t *testing.T) {
	ontologyDir := filepath.Clean(filepath.Join("..", "..", "..", "contracts", "ontology"))
	indexPath := filepath.Join(ontologyDir, "index.json")
	rawIndex, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read %s: %v", indexPath, err)
	}

	var index struct {
		Version          string   `json:"version"`
		SchemaVersion    string   `json:"schema_version"`
		PrimaryKey       []string `json:"primary_key"`
		MonetaryUnit     string   `json:"monetary_unit"`
		AllowedLinkTypes []string `json:"allowed_link_types"`
		SecurityMarkings []struct {
			Marking string `json:"marking"`
		} `json:"security_markings"`
		ObjectTypes []struct {
			ObjectType      string   `json:"object_type"`
			SchemaVersion   string   `json:"schema_version"`
			SchemaFile      string   `json:"schema_file"`
			MinorUnitFields []string `json:"minor_unit_fields"`
		} `json:"object_types"`
	}
	if err := json.Unmarshal(rawIndex, &index); err != nil {
		t.Fatalf("unmarshal %s: %v", indexPath, err)
	}
	if index.Version != "1.0.0" || index.SchemaVersion != "v1" || index.MonetaryUnit != "minor_unit_int64" {
		t.Fatalf("unexpected registry header: %+v", index)
	}
	if strings.Join(index.PrimaryKey, ",") != "cell_id,object_type,object_id" {
		t.Fatalf("unexpected primary_key: %v", index.PrimaryKey)
	}

	wantTypes := []string{
		"Supplier", "LegalEntity", "Warehouse", "Factory", "SKU", "Lot", "Shop",
		"Order", "OrderLine", "Manifest", "Vehicle", "Driver", "NasiyaInvoice",
		"CreditLine", "Payment", "FiscalDoc", "Proposal",
	}
	if len(index.ObjectTypes) != len(wantTypes) {
		t.Fatalf("object_types count = %d, want %d", len(index.ObjectTypes), len(wantTypes))
	}

	wantLinks := map[string]bool{
		"supplies": true, "stocks": true, "ordered": true, "shipped_on": true,
		"owes": true, "substitutes": true, "recalled_with": true,
	}
	for _, lt := range index.AllowedLinkTypes {
		delete(wantLinks, lt)
	}
	if len(wantLinks) != 0 {
		t.Fatalf("missing allowed_link_types: %v", wantLinks)
	}

	markingSet := make(map[string]bool, len(index.SecurityMarkings))
	for _, sm := range index.SecurityMarkings {
		markingSet[sm.Marking] = true
	}
	for _, requiredMarking := range []string{"MARKING:FINANCE_ONLY", "MARKING:PII"} {
		if !markingSet[requiredMarking] {
			t.Fatalf("index.json missing security marking %s", requiredMarking)
		}
	}

	tsOntologyPath := filepath.Clean(filepath.Join("..", "..", "..", "packages", "types", "src", "ontology.ts"))
	tsRaw, err := os.ReadFile(tsOntologyPath)
	if err != nil {
		t.Fatalf("read %s: %v", tsOntologyPath, err)
	}
	tsSrc := string(tsRaw)

	for _, entry := range index.ObjectTypes {
		if !strings.Contains(tsSrc, `"`+entry.ObjectType+`"`) {
			t.Errorf("%s missing object type %q", tsOntologyPath, entry.ObjectType)
		}
		schemaPath := filepath.Join(ontologyDir, entry.SchemaFile)
		schemaBytes, err := os.ReadFile(schemaPath)
		if err != nil {
			t.Errorf("read %s: %v", schemaPath, err)
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal(schemaBytes, &doc); err != nil {
			t.Errorf("unmarshal %s: %v", schemaPath, err)
			continue
		}
		if doc["x-object-type"] != entry.ObjectType {
			t.Errorf("%s x-object-type = %v, want %s", entry.SchemaFile, doc["x-object-type"], entry.ObjectType)
		}
		if ap, ok := doc["additionalProperties"].(bool); !ok || ap {
			t.Errorf("%s top-level additionalProperties = %v, want false", entry.SchemaFile, doc["additionalProperties"])
		}
		topProps, _ := doc["properties"].(map[string]any)
		innerPropsObj, _ := topProps["properties"].(map[string]any)
		if ap, ok := innerPropsObj["additionalProperties"].(bool); !ok || ap {
			t.Errorf("%s inner properties additionalProperties = %v, want false", entry.SchemaFile, innerPropsObj["additionalProperties"])
		}
		innerFields, _ := innerPropsObj["properties"].(map[string]any)
		for fieldName, rawDef := range innerFields {
			def, _ := rawDef.(map[string]any)
			typ, _ := def["type"].(string)
			if strings.HasSuffix(fieldName, "_minor") || strings.Contains(fieldName, "amount") || strings.Contains(fieldName, "price") {
				if typ != "integer" {
					t.Errorf("%s.%s has type %q, want \"integer\" (zero float money fields)", entry.ObjectType, fieldName, typ)
				}
			}
		}
		for _, mf := range entry.MinorUnitFields {
			def, ok := innerFields[mf].(map[string]any)
			if !ok || def["type"] != "integer" {
				t.Errorf("%s minor_unit_field %q missing or not integer in %s", entry.ObjectType, mf, entry.SchemaFile)
			}
		}
	}
}
