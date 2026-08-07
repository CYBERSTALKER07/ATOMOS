package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// ApplyImportSession atomically applies approved staged rows into pegasusX inventory tables.
func (r *ImportRepository) ApplyImportSession(ctx context.Context, supplierID, sessionID string) (ImportApplySummary, error) {
	summary := ImportApplySummary{
		SessionID: sessionID,
		Status:    "APPLIED",
	}
	if r.client == nil {
		return summary, errors.New("spanner unavailable")
	}

	updatedSummary, err := r.applyImportSessionTxn(ctx, supplierID, sessionID)
	if err == nil {
		return updatedSummary, nil
	}

	if errors.Is(err, errImportAlreadyApplied) {
		summary.Idempotent = true
		return summary, nil
	}

	if errors.Is(err, errImportSessionNotFound) ||
		errors.Is(err, errImportStateConflict) ||
		errors.Is(err, errImportAccessDenied) {
		return summary, err
	}

	if markErr := r.markApplyFailure(ctx, supplierID, sessionID, err); markErr != nil && !errors.Is(markErr, errImportSessionNotFound) {
		return summary, fmt.Errorf("apply import session failed: %w (mark failed: %v)", err, markErr)
	}

	if errors.Is(err, errImportNoApplicableRows) {
		return summary, errImportStateConflict
	}

	return summary, err
}

func (r *ImportRepository) applyImportSessionTxn(ctx context.Context, supplierID, sessionID string) (ImportApplySummary, error) {
	summary := ImportApplySummary{
		SessionID: sessionID,
		Status:    "APPLIED",
	}
	if stocklots.LotsEnabled() {
		return summary, fmt.Errorf("inventory import QoH apply forbidden when WMS_LOTS_ENABLED — use putaway / lot adjust instead of absolute SupplierInventoryV2 set")
	}
	affectedWarehouses := map[string]struct{}{}
	affectedProducts := map[string]struct{}{}
	seenRowKeys := map[string]struct{}{}
	defaultCategoryID := ""

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		sessionRow, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errImportSessionNotFound
			}
			return readErr
		}

		var sessionStatus string
		if err := sessionRow.Columns(&sessionStatus); err != nil {
			return err
		}
		sessionStatus = normalizeImportStatus(sessionStatus)
		switch sessionStatus {
		case "APPLIED":
			return errImportAlreadyApplied
		case "APPROVED", "APPLYING":
		default:
			return errImportStateConflict
		}

		batchMutations := make([]*spanner.Mutation, 0, supplierImportMutationBatch)
		flush := func() error {
			if len(batchMutations) == 0 {
				return nil
			}
			if err := txn.BufferWrite(batchMutations); err != nil {
				return err
			}
			batchMutations = batchMutations[:0]
			return nil
		}

		if sessionStatus == "APPROVED" {
			batchMutations = append(batchMutations, spanner.Update(
				"SupplierImportSessions",
				[]string{"supplier_id", "session_id", "status", "updated_at"},
				[]any{supplierID, sessionID, "APPLYING", spanner.CommitTimestamp},
			))
		}

		warehouseOwnerCache := make(map[string]string)
		stmt := spanner.Statement{
			SQL: `SELECT row_index, raw_data, cleaned_data, is_new_product
			      FROM SupplierImportStagedRows
			      WHERE supplier_id = @supplierId
			        AND session_id = @sessionId
			        AND (validation_errors IS NULL OR ARRAY_LENGTH(validation_errors) = 0)
			      ORDER BY row_index`,
			Params: map[string]any{
				"supplierId": supplierID,
				"sessionId":  sessionID,
			},
		}

		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		for {
			row, nextErr := iter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return fmt.Errorf("list staged rows for apply: %w", nextErr)
			}

			var rowIndex int64
			var rawDataJSON spanner.NullJSON
			var cleanedDataJSON spanner.NullJSON
			var isNewProduct bool
			if err := row.Columns(&rowIndex, &rawDataJSON, &cleanedDataJSON, &isNewProduct); err != nil {
				return fmt.Errorf("parse staged row for apply: %w", err)
			}

			rawData := importJSONMap(rawDataJSON)
			cleanedData := importJSONMap(cleanedDataJSON)

			rowSupplierID := importStringValue(cleanedData, rawData, "supplier_id")
			if rowSupplierID != "" && rowSupplierID != supplierID {
				return errImportAccessDenied
			}

			warehouseID := strings.TrimSpace(importStringValue(cleanedData, rawData, "warehouse_id", "warehouse"))
			if warehouseID == "" {
				return fmt.Errorf("row %d missing warehouse_id", rowIndex)
			}

			warehouseOwner, cached := warehouseOwnerCache[warehouseID]
			if !cached {
				warehouseRow, whErr := txn.ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"SupplierId"})
				if whErr != nil {
					if spanner.ErrCode(whErr) == codes.NotFound {
						return errImportAccessDenied
					}
					return fmt.Errorf("read warehouse owner: %w", whErr)
				}
				if err := warehouseRow.Columns(&warehouseOwner); err != nil {
					return fmt.Errorf("parse warehouse owner: %w", err)
				}
				warehouseOwnerCache[warehouseID] = warehouseOwner
			}
			if warehouseOwner != supplierID {
				return errImportAccessDenied
			}

			productID := strings.TrimSpace(importStringValue(cleanedData, rawData, "product_id", "sku_id", "sku", "item_code"))
			if productID == "" {
				if !isNewProduct {
					return fmt.Errorf("row %d missing product_id", rowIndex)
				}
				productID = importDefaultProductID(sessionID, rowIndex)
			}

			productExists := false
			productOwnerRow, productErr := txn.ReadRow(ctx, "Products", spanner.Key{productID}, []string{"SupplierId", "IsActive"})
			if productErr == nil {
				productExists = true
				var ownerSupplierID string
				var isActive bool
				if err := productOwnerRow.Columns(&ownerSupplierID, &isActive); err != nil {
					return fmt.Errorf("parse product owner: %w", err)
				}
				if ownerSupplierID != supplierID {
					return errImportAccessDenied
				}
				if !isActive {
					return fmt.Errorf("row %d product inactive", rowIndex)
				}
			} else if spanner.ErrCode(productErr) != codes.NotFound {
				return fmt.Errorf("read product: %w", productErr)
			}

			if !productExists && !isNewProduct {
				return fmt.Errorf("row %d product_id not found in supplier catalog", rowIndex)
			}

			if isNewProduct && !productExists {
				productName := strings.TrimSpace(importStringValue(cleanedData, rawData, "product_name", "name", "item_name"))
				if productName == "" {
					productName = fmt.Sprintf("Imported product %d", rowIndex+1)
				}
				priceMinor, ok := importInt64Value(cleanedData, rawData, "price_minor", "unit_price", "price")
				if !ok || priceMinor <= 0 {
					priceMinor = 1
				}
				categoryID := strings.TrimSpace(importStringValue(cleanedData, rawData, "category_id", "category"))
				if categoryID == "" {
					if defaultCategoryID == "" {
						categoryID, catErr := r.firstSupplierCategoryID(ctx, txn, supplierID)
						if catErr != nil {
							return catErr
						}
						defaultCategoryID = categoryID
					}
					categoryID = defaultCategoryID
				}
				if categoryID == "" {
					return fmt.Errorf("row %d missing category_id for new product", rowIndex)
				}

				productCols := []string{
					"ProductId", "SupplierId", "CategoryId", "Name", "PriceMinor", "Currency",
					"StockQuantity", "Unit", "UnitVolumeVU", "IsActive", "Version", "CreatedAt", "UpdatedAt",
				}
				productVals := []any{
					productID, supplierID, categoryID, productName, priceMinor, "UZS",
					int64(0), "UNIT", 1.0, true, int64(1), spanner.CommitTimestamp, spanner.CommitTimestamp,
				}
				if description := strings.TrimSpace(importStringValue(cleanedData, rawData, "description")); description != "" {
					productCols = append(productCols, "Description")
					productVals = append(productVals, description)
				}
				if imageURL := strings.TrimSpace(importStringValue(cleanedData, rawData, "image_url")); imageURL != "" {
					productCols = append(productCols, "ImageURL")
					productVals = append(productVals, imageURL)
				}
				batchMutations = append(batchMutations, spanner.InsertOrUpdate("Products", productCols, productVals))
				summary.CreatedProducts++
			}

			qty, hasQty := importInt64Value(cleanedData, rawData, "quantity_on_hand", "quantity_available", "quantity", "qty")
			if !hasQty {
				return fmt.Errorf("row %d missing quantity_on_hand", rowIndex)
			}
			if qty < 0 {
				return fmt.Errorf("row %d quantity_on_hand cannot be negative", rowIndex)
			}

			reorderThreshold, _ := importInt64Value(cleanedData, rawData, "reorder_threshold", "threshold", "reorder")

			rowDeterministicKey := fmt.Sprintf("%s|%s|%s|%d", supplierID, warehouseID, productID, rowIndex)
			if _, alreadySeen := seenRowKeys[rowDeterministicKey]; alreadySeen {
				return fmt.Errorf("row %d duplicate deterministic key", rowIndex)
			}
			seenRowKeys[rowDeterministicKey] = struct{}{}

			inventoryID := uuid.NewString()
			existingInventoryID := ""
			invStmt := spanner.Statement{
				SQL: `SELECT InventoryId FROM InventoryLevels
				      WHERE WarehouseId = @warehouseId AND ProductId = @productId
				      LIMIT 1`,
				Params: map[string]any{
					"warehouseId": warehouseID,
					"productId":   productID,
				},
			}
			invIter := txn.Query(ctx, invStmt)
			invRow, invErr := invIter.Next()
			invIter.Stop()
			if invErr == nil {
				if err := invRow.Columns(&existingInventoryID); err != nil {
					return fmt.Errorf("parse inventory id: %w", err)
				}
			} else if invErr != iterator.Done {
				return fmt.Errorf("lookup inventory level: %w", invErr)
			}
			if existingInventoryID != "" {
				inventoryID = existingInventoryID
			}

			batchMutations = append(batchMutations,
				spanner.InsertOrUpdate(
					"InventoryLevels",
					[]string{
						"InventoryId", "ProductId", "WarehouseId", "SupplierId",
						"QuantityOnHand", "QuantityReserved", "ReorderThreshold", "Version", "UpdatedAt",
					},
					[]any{
						inventoryID, productID, warehouseID, supplierID,
						qty, int64(0), reorderThreshold, int64(1), spanner.CommitTimestamp,
					},
				),
				spanner.InsertOrUpdate(
					"SupplierInventoryV2",
					[]string{"SupplierId", "WarehouseId", "ProductId", "QuantityOnHand", "QuantityReserved", "UpdatedAt"},
					[]any{supplierID, warehouseID, productID, qty, int64(0), spanner.CommitTimestamp},
				),
			)

			summary.AppliedRows++
			affectedWarehouses[warehouseID] = struct{}{}
			affectedProducts[productID] = struct{}{}

			if len(batchMutations) >= supplierImportMutationBatch {
				if err := flush(); err != nil {
					return fmt.Errorf("flush apply mutation batch: %w", err)
				}
			}
		}

		if summary.AppliedRows == 0 {
			return errImportNoApplicableRows
		}

		batchMutations = append(batchMutations, spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "error_summary", "updated_at"},
			[]any{supplierID, sessionID, "APPLIED", nil, spanner.CommitTimestamp},
		))

		if err := flush(); err != nil {
			return fmt.Errorf("flush final apply mutations: %w", err)
		}
		return nil
	})

	if err != nil {
		return summary, err
	}

	if len(affectedWarehouses) > 0 {
		summary.WarehouseIDs = make([]string, 0, len(affectedWarehouses))
		for warehouseID := range affectedWarehouses {
			summary.WarehouseIDs = append(summary.WarehouseIDs, warehouseID)
		}
		sort.Strings(summary.WarehouseIDs)
	}

	if len(affectedProducts) > 0 {
		summary.AppliedProductIDs = make([]string, 0, len(affectedProducts))
		for productID := range affectedProducts {
			summary.AppliedProductIDs = append(summary.AppliedProductIDs, productID)
		}
		sort.Strings(summary.AppliedProductIDs)
	}

	summary.AffectedWarehouses = int64(len(summary.WarehouseIDs))
	summary.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	return summary, nil
}

func (r *ImportRepository) firstSupplierCategoryID(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string) (string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT CategoryId FROM ProductCategories
		      WHERE SupplierId = @supplierId
		      ORDER BY SortOrder ASC
		      LIMIT 1`,
		Params: map[string]any{"supplierId": supplierID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", fmt.Errorf("supplier has no product categories for import")
	}
	if err != nil {
		return "", fmt.Errorf("lookup supplier category: %w", err)
	}
	var categoryID string
	if err := row.Columns(&categoryID); err != nil {
		return "", fmt.Errorf("parse supplier category: %w", err)
	}
	return strings.TrimSpace(categoryID), nil
}

func (r *ImportRepository) markApplyFailure(ctx context.Context, supplierID, sessionID string, applyErr error) error {
	if r.client == nil {
		return errors.New("spanner unavailable")
	}
	errorSummaryPayload := map[string]any{
		"phase":      "APPLY",
		"session_id": sessionID,
		"error":      applyErr.Error(),
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	errorSummaryBytes, marshalErr := json.Marshal(errorSummaryPayload)
	if marshalErr != nil {
		return marshalErr
	}
	errorSummaryJSON, parseErr := importToNullJSON(errorSummaryBytes)
	if parseErr != nil {
		return parseErr
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errImportSessionNotFound
			}
			return readErr
		}

		var currentStatus string
		if err := row.Columns(&currentStatus); err != nil {
			return err
		}
		if normalizeImportStatus(currentStatus) == "APPLIED" {
			return nil
		}

		return txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "error_summary", "updated_at"},
			[]any{supplierID, sessionID, "FAILED", errorSummaryJSON, spanner.CommitTimestamp},
		)})
	})
	return err
}

func importJSONMap(source spanner.NullJSON) map[string]any {
	if !source.Valid {
		return map[string]any{}
	}
	m, ok := source.Value.(map[string]any)
	if ok {
		return m
	}
	if source.Value == nil {
		return map[string]any{}
	}
	encoded, err := json.Marshal(source.Value)
	if err != nil {
		return map[string]any{}
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return map[string]any{}
	}
	return decoded
}

func importLookupValue(cleaned, raw map[string]any, keys ...string) (any, bool) {
	lookup := func(dataset map[string]any, key string) (any, bool) {
		if value, ok := dataset[key]; ok {
			return value, true
		}
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		for candidate, value := range dataset {
			if strings.ToLower(strings.TrimSpace(candidate)) == normalizedKey {
				return value, true
			}
		}
		return nil, false
	}

	for _, key := range keys {
		if value, ok := lookup(cleaned, key); ok {
			return value, true
		}
		if value, ok := lookup(raw, key); ok {
			return value, true
		}
	}
	return nil, false
}

func importStringValue(cleaned, raw map[string]any, keys ...string) string {
	value, ok := importLookupValue(cleaned, raw, keys...)
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func importInt64Value(cleaned, raw map[string]any, keys ...string) (int64, bool) {
	value, ok := importLookupValue(cleaned, raw, keys...)
	if !ok || value == nil {
		return 0, false
	}

	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float32:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return parsed, true
		}
	case string:
		normalized := strings.ReplaceAll(strings.TrimSpace(typed), ",", "")
		if normalized == "" {
			return 0, false
		}
		if parsed, err := strconv.ParseInt(normalized, 10, 64); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func importDefaultProductID(sessionID string, rowIndex int64) string {
	seed := fmt.Sprintf("%s:%d", sessionID, rowIndex)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(seed)).String()
}
