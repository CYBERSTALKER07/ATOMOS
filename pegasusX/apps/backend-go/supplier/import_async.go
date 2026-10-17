package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

var errImportAlreadyClaimed = errors.New("import session already claimed")

// ImportIngestSummary is the durable outcome of discovery + staging for one import body.
type ImportIngestSummary struct {
	Status            string
	RowsStaged        int
	SuggestedMappings int
	ValidRows         int
	InvalidRows       int
	DiscoveryModel    string
	LowConfidence     bool
}

// IngestImportBytes runs deterministic discovery and stages rows for an import session.
func (r *ImportRepository) IngestImportBytes(
	ctx context.Context,
	supplierID, sessionID string,
	body []byte,
	delimiter rune,
	warehouseIDs map[string]struct{},
) (ImportIngestSummary, error) {
	var zero ImportIngestSummary
	if r == nil || r.client == nil {
		return zero, errors.New("spanner unavailable")
	}
	if len(body) == 0 {
		return zero, errors.New("import_empty")
	}

	outcome, _, rows, discoverErr := discoverImportDelimited(body, delimiter)
	if discoverErr != nil {
		return zero, discoverErr
	}
	if len(rows) == 0 {
		return zero, fmt.Errorf("import_empty")
	}
	if len(rows) > importDiscoveryMaxRows {
		return zero, fmt.Errorf("import_too_many_rows")
	}

	productCache := map[string]bool{}
	productExists := func(productID string) bool {
		if cached, ok := productCache[productID]; ok {
			return cached
		}
		exists, lookupErr := r.ProductExists(ctx, supplierID, productID)
		if lookupErr != nil {
			exists = false
		}
		productCache[productID] = exists
		return exists
	}

	stagedRows, stagingSummary := buildImportStagedRows(supplierID, rows, outcome.Mappings, warehouseIDs, productExists)
	nextStatus, lowConfidence := resolveImportDiscoveryStatus(outcome.Mappings)

	errorSummary := map[string]any{
		"anomalies":          outcome.Anomalies,
		"low_confidence":     lowConfidence,
		"provider":           outcome.Model,
		"suggested_mappings": len(outcome.Mappings),
		"staging":            stagingSummary,
	}
	mappingDoc := map[string]any{
		"mappings":     outcome.Mappings,
		"anomalies":    outcome.Anomalies,
		"model":        outcome.Model,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}

	if err := r.PersistDiscovery(ctx, supplierID, sessionID, nextStatus, mappingDoc, errorSummary); err != nil {
		return zero, err
	}
	if err := r.SaveStagedRows(ctx, supplierID, sessionID, stagedRows); err != nil {
		return zero, err
	}

	validRows, _ := stagingSummary["valid_rows"].(int)
	invalidRows, _ := stagingSummary["invalid_rows"].(int)

	return ImportIngestSummary{
		Status:            nextStatus,
		RowsStaged:        len(stagedRows),
		SuggestedMappings: len(outcome.Mappings),
		ValidRows:         validRows,
		InvalidRows:       invalidRows,
		DiscoveryModel:    outcome.Model,
		LowConfidence:     lowConfidence,
	}, nil
}

// MarkSessionDiscovering claims an UPLOADED session for async discovery.
func (r *ImportRepository) MarkSessionDiscovering(ctx context.Context, supplierID, sessionID string) (bool, error) {
	if r == nil || r.client == nil {
		return false, errors.New("spanner unavailable")
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			return readErr
		}
		var current string
		if err := row.Columns(&current); err != nil {
			return err
		}
		current = normalizeImportStatus(current)
		if current == "DISCOVERING" || current == "DISCOVERED" || current == "MAPPING_REQUIRED" {
			return errImportAlreadyClaimed
		}
		if current != "UPLOADED" {
			return errImportAlreadyClaimed
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "updated_at"},
			[]any{supplierID, sessionID, "DISCOVERING", spanner.CommitTimestamp},
		)})
	})
	if err == nil {
		return true, nil
	}
	if errors.Is(err, errImportAlreadyClaimed) {
		return false, nil
	}
	if spanner.ErrCode(err) == codes.NotFound {
		return false, nil
	}
	return false, err
}

// MarkSessionImportFailed records a terminal FAILED status with an error summary.
func (r *ImportRepository) MarkSessionImportFailed(ctx context.Context, supplierID, sessionID, reason string) error {
	if r == nil || r.client == nil {
		return errors.New("spanner unavailable")
	}
	summaryBytes, err := json.Marshal(map[string]any{"error": reason})
	if err != nil {
		return err
	}
	summaryJSON, err := importToNullJSON(summaryBytes)
	if err != nil {
		return err
	}

	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			return readErr
		}
		var current string
		if err := row.Columns(&current); err != nil {
			return err
		}
		if normalizeImportStatus(current) == "FAILED" {
			return nil
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "error_summary", "updated_at"},
			[]any{supplierID, sessionID, "FAILED", summaryJSON, spanner.CommitTimestamp},
		)})
	})
	return err
}

// LoadWarehouseIDSet returns warehouse ids owned by the supplier for import validation.
func (r *ImportRepository) LoadWarehouseIDSet(ctx context.Context, supplierID string) (map[string]struct{}, error) {
	if r == nil || r.client == nil {
		return nil, errors.New("spanner unavailable")
	}
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId
		      FROM Warehouses
		      WHERE SupplierId = @supplierId`,
		Params: map[string]any{"supplierId": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	ids := map[string]struct{}{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query supplier warehouses: %w", err)
		}
		var warehouseID string
		if err := row.Columns(&warehouseID); err != nil {
			return nil, fmt.Errorf("scan warehouse id: %w", err)
		}
		warehouseID = strings.TrimSpace(warehouseID)
		if warehouseID != "" {
			ids[warehouseID] = struct{}{}
		}
	}
	return ids, nil
}

// ImportObjectOpener resolves supplier import objects from GCS or a local dev root.
type ImportObjectOpener struct {
	bucket    string
	client    *storage.Client
	localRoot string
}

// NewImportObjectOpenerFromEnv builds an opener for ai-worker / async import processing.
func NewImportObjectOpenerFromEnv(ctx context.Context) (*ImportObjectOpener, error) {
	opener := &ImportObjectOpener{
		bucket:    strings.TrimSpace(os.Getenv("GCS_BUCKET_NAME")),
		localRoot: strings.TrimSpace(os.Getenv("IMPORT_LOCAL_FILE_ROOT")),
	}
	// Local root is the sandbox/emulator plane. Skip GCS so discovery cannot
	// hang on a real bucket that is not the live import path.
	if opener.localRoot == "" && opener.bucket != "" {
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("init gcs client: %w", err)
		}
		opener.client = client
	}
	return opener, nil
}

// Close releases cloud storage resources when configured.
func (o *ImportObjectOpener) Close() error {
	if o == nil || o.client == nil {
		return nil
	}
	return o.client.Close()
}

// ReadObject loads the full import payload referenced by gcsPath.
func (o *ImportObjectOpener) ReadObject(ctx context.Context, gcsPath string) ([]byte, error) {
	if o == nil {
		return nil, errors.New("import object opener unavailable")
	}
	path := strings.TrimPrefix(strings.TrimSpace(gcsPath), "/")
	if path == "" {
		return nil, errors.New("object path is empty")
	}

	if o.localRoot != "" {
		localPath := filepath.Join(o.localRoot, filepath.FromSlash(path))
		if body, err := os.ReadFile(localPath); err == nil {
			return body, nil
		} else if o.client == nil || o.bucket == "" {
			return nil, err
		}
	}

	if o.client != nil && o.bucket != "" {
		rc, err := o.client.Bucket(o.bucket).Object(path).NewReader(ctx)
		if err == nil {
			defer rc.Close()
			return io.ReadAll(rc)
		}
		if o.localRoot == "" {
			return nil, err
		}
	}

	return nil, fmt.Errorf("import object %q is not available (configure GCS_BUCKET_NAME or IMPORT_LOCAL_FILE_ROOT)", path)
}

// ProcessImportUploaded handles INVENTORY_IMPORT_UPLOADED for async GCS/local discovery.
// warehouseIDs is the wizard topology set when the portal already loaded it; empty falls back to Warehouses.
func (r *ImportRepository) ProcessImportUploaded(ctx context.Context, opener *ImportObjectOpener, supplierID, sessionID, gcsPath string, warehouseIDs map[string]struct{}) error {
	acquired, err := r.MarkSessionDiscovering(ctx, supplierID, sessionID)
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}

	fail := func(reason string) error {
		if markErr := r.MarkSessionImportFailed(ctx, supplierID, sessionID, reason); markErr != nil {
			return markErr
		}
		return nil
	}

	body, readErr := opener.ReadObject(ctx, gcsPath)
	if readErr != nil {
		return fail(fmt.Sprintf("sample read failed: %v", readErr))
	}

	delimiter := detectImportDelimiter(gcsPath, body)
	if len(warehouseIDs) == 0 {
		loaded, topoErr := r.LoadWarehouseIDSet(ctx, supplierID)
		if topoErr != nil {
			return fail(fmt.Sprintf("load warehouses failed: %v", topoErr))
		}
		warehouseIDs = loaded
	}

	if _, ingestErr := r.IngestImportBytes(ctx, supplierID, sessionID, body, delimiter, warehouseIDs); ingestErr != nil {
		return fail(fmt.Sprintf("discovery failed: %v", ingestErr))
	}
	return nil
}

// ParseInventoryImportUploadedEvent decodes an outbox relayed inventory import upload event.
func ParseInventoryImportUploadedEvent(payload []byte) (events.InventoryImportEvent, error) {
	var evt events.InventoryImportEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return events.InventoryImportEvent{}, err
	}
	if strings.TrimSpace(evt.Type) == "" {
		evt.Type = events.EventInventoryImportUploaded
	}
	return evt, nil
}

func detectImportDelimiter(gcsPath string, body []byte) rune {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(strings.TrimSpace(gcsPath)), "."))
	if ext == "tsv" {
		return '\t'
	}
	if ext == "csv" || ext == "txt" || ext == "xlsx" || ext == "xls" {
		if bytesContainTab(body) {
			return '\t'
		}
		return ','
	}
	if bytesContainTab(body) {
		return '\t'
	}
	return ','
}

func bytesContainTab(body []byte) bool {
	for _, b := range body {
		if b == '\t' {
			return true
		}
	}
	return false
}
