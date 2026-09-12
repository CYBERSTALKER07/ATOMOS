package supplier

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadObjectPrefersLocalRoot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "imports", "sup", "sess", "raw.csv")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("sku,qty\nA,1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opener := &ImportObjectOpener{localRoot: root, bucket: "unused-bucket"}
	body, err := opener.ReadObject(context.Background(), "imports/sup/sess/raw.csv")
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}
	if string(body) != "sku,qty\nA,1\n" {
		t.Fatalf("body=%q", body)
	}
}

func TestBuildImportStagedRows_EmptyWarehouseSetInvalidates(t *testing.T) {
	rows := []map[string]any{
		{"product_id": "SSMR-SKU-1", "warehouse_id": "ssmr-warehouse-1", "quantity_on_hand": "82"},
	}
	mappings := []importMappingCandidate{
		{SourceColumn: "warehouse_id", TargetField: "warehouse_id", Confidence: 1},
		{SourceColumn: "product_id", TargetField: "product_id", Confidence: 1},
		{SourceColumn: "quantity_on_hand", TargetField: "quantity_on_hand", Confidence: 1},
	}
	staged, summary := buildImportStagedRows("sup-1", rows, mappings, map[string]struct{}{}, func(string) bool { return true })
	if summary["invalid_rows"] != 1 {
		t.Fatalf("summary=%v", summary)
	}
	if len(staged) != 1 || len(staged[0].ValidationErrors) == 0 || staged[0].ValidationErrors[0] != "warehouse_not_in_topology" {
		t.Fatalf("staged=%+v", staged)
	}
}

func TestBuildImportStagedRows_TopologyWarehouseValid(t *testing.T) {
	rows := []map[string]any{
		{"product_id": "SSMR-SKU-1", "warehouse_id": "ssmr-warehouse-1", "quantity_on_hand": "82"},
	}
	mappings := []importMappingCandidate{
		{SourceColumn: "warehouse_id", TargetField: "warehouse_id", Confidence: 1},
		{SourceColumn: "product_id", TargetField: "product_id", Confidence: 1},
		{SourceColumn: "quantity_on_hand", TargetField: "quantity_on_hand", Confidence: 1},
	}
	staged, summary := buildImportStagedRows("sup-1", rows, mappings, map[string]struct{}{"ssmr-warehouse-1": {}}, func(string) bool { return true })
	if summary["valid_rows"] != 1 {
		t.Fatalf("summary=%v staged=%+v", summary, staged)
	}
	if len(staged) != 1 || len(staged[0].ValidationErrors) != 0 {
		t.Fatalf("staged=%+v", staged)
	}
}

func TestNewImportObjectOpenerFromEnv_LocalRootSkipsGCSClient(t *testing.T) {
	t.Setenv("IMPORT_LOCAL_FILE_ROOT", t.TempDir())
	t.Setenv("GCS_BUCKET_NAME", "pegasus-503013-ssmr-assets")
	opener, err := NewImportObjectOpenerFromEnv(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opener.Close() })
	if opener.client != nil {
		t.Fatal("local root must not construct a GCS client")
	}
	if opener.localRoot == "" {
		t.Fatal("expected local root")
	}
}
