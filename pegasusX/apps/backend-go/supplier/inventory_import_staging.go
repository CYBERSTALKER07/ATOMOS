package supplier

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

const inventoryImportStagingFileName = "direct_csv_import"

type inventoryImportStagingEntry struct {
	RowIndex         int64
	RawData          map[string]any
	CleanedData      map[string]any
	ValidationErrors []string
}

type inventoryImportStagingSummary struct {
	Applied int `json:"applied"`
	Skipped int `json:"skipped"`
}

// persistInventoryImportStaging writes SupplierImportSessions + SupplierImportStagedRows
// so warehouse analytics can surface import_anomaly_queue from validation_errors.
func (s *Service) persistInventoryImportStaging(
	ctx context.Context,
	supplierID string,
	entries []inventoryImportStagingEntry,
	applied, skipped int,
) (string, error) {
	if s.portalSpanner == nil || len(entries) == 0 {
		return "", nil
	}

	sessionID := uuid.NewString()
	summaryJSON, err := json.Marshal(inventoryImportStagingSummary{
		Applied: applied,
		Skipped: skipped,
	})
	if err != nil {
		return "", fmt.Errorf("marshal import staging summary: %w", err)
	}

	_, err = s.portalSpanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := make([]*spanner.Mutation, 0, len(entries)+1)
		mutations = append(mutations, spanner.InsertOrUpdate(
			"SupplierImportSessions",
			[]string{
				"supplier_id", "session_id", "status", "file_name", "total_rows",
				"error_summary", "created_at", "updated_at",
			},
			[]any{
				supplierID,
				sessionID,
				"APPLIED",
				inventoryImportStagingFileName,
				int64(len(entries)),
				spanner.NullJSON{Value: json.RawMessage(summaryJSON), Valid: true},
				spanner.CommitTimestamp,
				spanner.CommitTimestamp,
			},
		))

		for _, entry := range entries {
			rawJSON, err := inventoryImportToNullJSON(entry.RawData)
			if err != nil {
				return fmt.Errorf("raw_data row %d: %w", entry.RowIndex, err)
			}
			cleanedJSON, err := inventoryImportToNullJSON(entry.CleanedData)
			if err != nil {
				return fmt.Errorf("cleaned_data row %d: %w", entry.RowIndex, err)
			}
			mutations = append(mutations, spanner.InsertOrUpdate(
				"SupplierImportStagedRows",
				[]string{
					"supplier_id", "session_id", "row_index", "raw_data", "cleaned_data",
					"validation_errors", "is_new_product", "created_at", "updated_at",
				},
				[]any{
					supplierID,
					sessionID,
					entry.RowIndex,
					rawJSON,
					cleanedJSON,
					entry.ValidationErrors,
					false,
					spanner.CommitTimestamp,
					spanner.CommitTimestamp,
				},
			))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return "", fmt.Errorf("persist inventory import staging: %w", err)
	}
	return sessionID, nil
}

func inventoryImportToNullJSON(value map[string]any) (spanner.NullJSON, error) {
	if len(value) == 0 {
		return spanner.NullJSON{}, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return spanner.NullJSON{}, err
	}
	return spanner.NullJSON{Value: json.RawMessage(encoded), Valid: true}, nil
}

func inventoryImportStagingRaw(row inventoryImportRow) map[string]any {
	return map[string]any{
		"product_id":         row.ProductID,
		"warehouse_id":       row.WarehouseID,
		"quantity_on_hand":   row.QuantityOnHand,
		"reorder_threshold":  row.ReorderThreshold,
	}
}
