package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SpannerApplyStore executes chunked migration applies inside atomic Spanner ReadWriteTransactions
// with in-transaction read-your-writes caching, snapshot capture, and FEFO lot rollups.
type SpannerApplyStore struct {
	client *spanner.Client
	NowFn  func() time.Time
}

// NewSpannerApplyStore constructs a production Spanner-backed EntityApplyStore.
func NewSpannerApplyStore(client *spanner.Client) *SpannerApplyStore {
	return &SpannerApplyStore{client: client}
}

func (s *SpannerApplyStore) nowUTC() time.Time {
	if s != nil && s.NowFn != nil {
		return s.NowFn().UTC()
	}
	return time.Now().UTC()
}

// RunChunkTx runs fn in a single Spanner ReadWriteTransaction and commits deduplicated mutations and snapshots.
func (s *SpannerApplyStore) RunChunkTx(ctx context.Context, supplierID, sessionID string, fn func(tx ApplyTx) error) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("spanner apply store: nil client")
	}
	now := s.nowUTC()
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		seq, err := queryMaxSnapshotSeq(ctx, txn, supplierID, sessionID)
		if err != nil {
			return err
		}
		tx := &spannerApplyTx{
			txn:             txn,
			supplierID:      supplierID,
			sessionID:       sessionID,
			now:             now,
			seq:             seq,
			identities:      make(map[string]string),
			checkedIDs:      make(map[string]bool),
			warehouses:      make(map[string]StoredWarehouse),
			whByName:        make(map[string]string),
			plantMaps:       make(map[string]string),
			products:        make(map[string]StoredProduct),
			prodByBarcode:   make(map[string]string),
			prodByName:      make(map[string]string),
			checkedProdIDs:  make(map[string]bool),
			checkedProdBC:   make(map[string]bool),
			checkedProdName: make(map[string]bool),
			customers:       make(map[string]StoredCustomer),
			custByINN:       make(map[string]string),
			custByPhone:     make(map[string]string),
			custByLegalName: make(map[string]string),
			checkedCustIDs:  make(map[string]bool),
			checkedCustINNs: make(map[string]bool),
			checkedCustPh:   make(map[string]bool),
			checkedCustName: make(map[string]bool),
			stocks:          make(map[string]StoredStock),
			checkedStocks:   make(map[string]bool),
			lotsByID:        make(map[string]StoredLot),
			lotsByKey:       make(map[string]string),
			checkedLotPairs: make(map[string]bool),
			checkedLotKeys:  make(map[string]bool),
			lotOverrides:    make(map[string]map[string]stocklots.StockLotSnapshot),
			locations:       make(map[string]bool),
			snapRecorded:    make(map[string]bool),
			insertedWH:      make(map[string]bool),
			insertedProd:    make(map[string]bool),
			insertedLot:     make(map[string]bool),
			insertedLink:    make(map[string]bool),
			whMuts:          make(map[string]*spanner.Mutation),
			plantMuts:       make(map[string]*spanner.Mutation),
			prodMuts:        make(map[string]*spanner.Mutation),
			retMuts:         make(map[string]*spanner.Mutation),
			linkMuts:        make(map[string]*spanner.Mutation),
			stockMuts:       make(map[string]*spanner.Mutation),
			lotMuts:         make(map[string]*spanner.Mutation),
			orderMuts:       make(map[string]*spanner.Mutation),
			orderCols:       make(map[string]map[string]any),
			idMuts:          make(map[string]*spanner.Mutation),
			rollupPairs:     make(map[string]whProdPair),
			ordersByID:      make(map[string]order.Order),
			ordersByExt:     make(map[string]string),
			checkedOrderExt: make(map[string]bool),
			arCache:         ar.NewMigrationTxnCache(),
		}
		if err := fn(tx); err != nil {
			return err
		}
		return tx.flush(ctx)
	})
	return err
}

type whProdPair struct {
	whID   string
	prodID string
}

type spannerApplyTx struct {
	txn             *spanner.ReadWriteTransaction
	supplierID      string
	sessionID       string
	now             time.Time
	seq             int64
	metaLoaded      bool
	homeCell        string
	currency        string
	whLoaded        bool
	identities      map[string]string
	checkedIDs      map[string]bool
	warehouses      map[string]StoredWarehouse
	whByName        map[string]string
	plantMaps       map[string]string
	products        map[string]StoredProduct
	prodByBarcode   map[string]string
	prodByName      map[string]string
	checkedProdIDs  map[string]bool
	checkedProdBC   map[string]bool
	checkedProdName map[string]bool
	customers       map[string]StoredCustomer
	custByINN       map[string]string
	custByPhone     map[string]string
	custByLegalName map[string]string
	checkedCustIDs  map[string]bool
	checkedCustINNs map[string]bool
	checkedCustPh   map[string]bool
	checkedCustName map[string]bool
	stocks          map[string]StoredStock
	checkedStocks   map[string]bool
	lotsByID        map[string]StoredLot
	lotsByKey       map[string]string // whID|prodID|lower(lotCode) -> lotID
	checkedLotPairs map[string]bool
	checkedLotKeys  map[string]bool
	lotOverrides    map[string]map[string]stocklots.StockLotSnapshot
	locations       map[string]bool
	snapRecorded    map[string]bool
	insertedWH      map[string]bool
	insertedProd    map[string]bool
	insertedLot     map[string]bool
	insertedLink    map[string]bool

	whMuts          map[string]*spanner.Mutation
	plantMuts       map[string]*spanner.Mutation
	prodMuts        map[string]*spanner.Mutation
	retMuts         map[string]*spanner.Mutation
	linkMuts        map[string]*spanner.Mutation
	stockMuts       map[string]*spanner.Mutation
	lotMuts         map[string]*spanner.Mutation
	orderMuts       map[string]*spanner.Mutation
	orderCols       map[string]map[string]any
	idMuts          map[string]*spanner.Mutation
	snapMuts        []*spanner.Mutation
	rollupPairs     map[string]whProdPair
	ordersByID      map[string]order.Order
	ordersByExt     map[string]string
	checkedOrderExt map[string]bool
	arCache         *ar.MigrationTxnCache
}

func (tx *spannerApplyTx) Now() time.Time     { return tx.now }
func (tx *spannerApplyTx) SupplierID() string { return tx.supplierID }
func (tx *spannerApplyTx) SessionID() string  { return tx.sessionID }

func (tx *spannerApplyTx) ReadSupplierMeta(ctx context.Context) (string, string, error) {
	if tx.metaLoaded {
		return tx.homeCell, tx.currency, nil
	}
	tx.currency = "UZS"
	row, err := tx.txn.ReadRow(ctx, "Suppliers", spanner.Key{tx.supplierID}, []string{"Currency", "HomeCell"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			tx.metaLoaded = true
			return "", "UZS", nil
		}
		return "", "", err
	}
	var cur string
	var cell spanner.NullString
	if err := row.Columns(&cur, &cell); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(cur) != "" {
		tx.currency = strings.ToUpper(strings.TrimSpace(cur))
	}
	tx.homeCell = cell.StringVal
	tx.metaLoaded = true
	return tx.homeCell, tx.currency, nil
}

func (tx *spannerApplyTx) LookupIdentity(ctx context.Context, pack Pack, extID string) (string, bool, error) {
	key := identityKey(tx.supplierID, pack, extID)
	if id, ok := tx.identities[key]; ok {
		return id, true, nil
	}
	if tx.checkedIDs[key] {
		return "", false, nil
	}
	tx.checkedIDs[key] = true
	row, err := tx.txn.ReadRow(ctx, "MigrationIdentityMap",
		spanner.Key{tx.supplierID, string(pack), strings.TrimSpace(extID)},
		[]string{"internal_id"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return "", false, nil
		}
		return "", false, err
	}
	var internalID string
	if err := row.Columns(&internalID); err != nil {
		return "", false, err
	}
	tx.identities[key] = internalID
	return internalID, true, nil
}

func (tx *spannerApplyTx) PutIdentity(_ context.Context, pack Pack, extID, internalID string) error {
	trimmedExt := strings.TrimSpace(extID)
	trimmedInt := strings.TrimSpace(internalID)
	key := identityKey(tx.supplierID, pack, trimmedExt)
	tx.identities[key] = trimmedInt
	tx.idMuts[key] = spanner.InsertOrUpdateMap("MigrationIdentityMap", map[string]any{
		"supplier_id":        tx.supplierID,
		"pack":               string(pack),
		"external_id":        trimmedExt,
		"internal_id":        trimmedInt,
		"created_by_session": tx.sessionID,
		"created_at":         spanner.CommitTimestamp,
	})
	return nil
}

func (tx *spannerApplyTx) RecordSnapshot(_ context.Context, targetTable string, keyObj any, opKind string, beforeObj any) error {
	keyBytes, err := json.Marshal(keyObj)
	if err != nil {
		return err
	}
	dedup := targetTable + "|" + string(keyBytes)
	if tx.snapRecorded[dedup] {
		return nil
	}
	tx.snapRecorded[dedup] = true
	tx.seq++
	rec, err := BuildSnapshotRecord(tx.supplierID, tx.sessionID, tx.seq, targetTable, keyObj, opKind, beforeObj, tx.now)
	if err != nil {
		return err
	}
	beforeJSON, err := toNullJSON(rec.BeforeJSON)
	if err != nil {
		return err
	}
	tx.snapMuts = append(tx.snapMuts, spanner.InsertOrUpdateMap("MigrationSnapshots", map[string]any{
		"supplier_id":  tx.supplierID,
		"session_id":   tx.sessionID,
		"snapshot_seq": rec.SnapshotSeq,
		"target_table": rec.TargetTable,
		"row_key_json": rec.RowKeyJSON,
		"op_kind":      rec.OpKind,
		"before_json":  beforeJSON,
		"expires_at":   rec.ExpiresAt.UTC(),
		"created_at":   spanner.CommitTimestamp,
	}))
	return nil
}

func (tx *spannerApplyTx) FindWarehouse(ctx context.Context, extID, name string) (StoredWarehouse, bool, error) {
	ext := strings.TrimSpace(extID)
	if ext != "" {
		if wh, ok := tx.warehouses[ext]; ok {
			return wh, true, nil
		}
		if mapped, ok := tx.plantMaps[plantMapKey(tx.supplierID, ext)]; ok {
			if wh, ok2 := tx.warehouses[mapped]; ok2 {
				return wh, true, nil
			}
		}
		if !tx.whLoaded {
			if wh, ok, err := tx.readWarehouseRow(ctx, ext); err != nil || ok {
				return wh, ok, err
			}
			pmRow, err := tx.txn.ReadRow(ctx, "PartnerPlantMaps", spanner.Key{"SUPPLIER", tx.supplierID, ext}, []string{"WarehouseId"})
			if err == nil {
				var whID string
				if colErr := pmRow.Columns(&whID); colErr == nil && whID != "" {
					tx.plantMaps[plantMapKey(tx.supplierID, ext)] = whID
					return tx.readWarehouseRow(ctx, whID)
				}
			} else if spanner.ErrCode(err) != codes.NotFound {
				return StoredWarehouse{}, false, err
			}
		}
	}
	wantName := strings.ToLower(strings.TrimSpace(name))
	if wantName != "" {
		if id, ok := tx.whByName[wantName]; ok {
			if wh, ok2 := tx.warehouses[id]; ok2 {
				return wh, true, nil
			}
		}
		if tx.whLoaded {
			return StoredWarehouse{}, false, nil
		}
		iter := tx.txn.Query(ctx, spanner.Statement{
			SQL: `SELECT WarehouseId, SupplierId, Name, Address, IsActive, IsOnShift
			      FROM Warehouses
			      WHERE SupplierId = @sid AND LOWER(Name) = @name
			      LIMIT 1`,
			Params: map[string]any{"sid": tx.supplierID, "name": wantName},
		})
		defer iter.Stop()
		row, err := iter.Next()
		if err == nil {
			wh, scanErr := scanStoredWarehouse(row)
			if scanErr != nil {
				return StoredWarehouse{}, false, scanErr
			}
			tx.cacheWarehouse(wh)
			return wh, true, nil
		}
		if err != iterator.Done {
			return StoredWarehouse{}, false, err
		}
	}
	return StoredWarehouse{}, false, nil
}

func (tx *spannerApplyTx) readWarehouseRow(ctx context.Context, warehouseID string) (StoredWarehouse, bool, error) {
	if wh, ok := tx.warehouses[warehouseID]; ok {
		return wh, true, nil
	}
	row, err := tx.txn.ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
		[]string{"WarehouseId", "SupplierId", "Name", "Address", "IsActive", "IsOnShift"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return StoredWarehouse{}, false, nil
		}
		return StoredWarehouse{}, false, err
	}
	wh, err := scanStoredWarehouse(row)
	if err != nil {
		return StoredWarehouse{}, false, err
	}
	if wh.SupplierID != tx.supplierID {
		return StoredWarehouse{}, false, nil
	}
	tx.cacheWarehouse(wh)
	return wh, true, nil
}

func (tx *spannerApplyTx) cacheWarehouse(wh StoredWarehouse) {
	tx.warehouses[wh.WarehouseID] = wh
	if n := strings.ToLower(strings.TrimSpace(wh.Name)); n != "" {
		tx.whByName[n] = wh.WarehouseID
	}
}

func (tx *spannerApplyTx) SaveWarehouse(ctx context.Context, wh StoredWarehouse, extPlantID string, before *StoredWarehouse) error {
	if before == nil || tx.insertedWH[wh.WarehouseID] {
		if before == nil {
			tx.insertedWH[wh.WarehouseID] = true
			if err := tx.RecordSnapshot(ctx, "Warehouses", map[string]any{"WarehouseId": wh.WarehouseID}, SnapshotOpInsert, nil); err != nil {
				return err
			}
		}
		cols := map[string]any{
			"WarehouseId":      wh.WarehouseID,
			"SupplierId":       wh.SupplierID,
			"Name":             wh.Name,
			"Address":          spanner.NullString{StringVal: wh.Address, Valid: wh.Address != ""},
			"Lat":              0.0,
			"Lng":              0.0,
			"CoverageRadiusKm": 50.0,
			"IsActive":         wh.IsActive,
			"IsOnShift":        wh.IsOnShift,
			"CreatedAt":        spanner.CommitTimestamp,
			"UpdatedAt":        spanner.CommitTimestamp,
		}
		tx.whMuts[wh.WarehouseID] = spanner.InsertOrUpdateMap("Warehouses", cols)
	} else {
		if err := tx.RecordSnapshot(ctx, "Warehouses", map[string]any{"WarehouseId": wh.WarehouseID}, SnapshotOpUpdate, snapshotWarehouseMap(*before)); err != nil {
			return err
		}
		tx.whMuts[wh.WarehouseID] = spanner.UpdateMap("Warehouses", map[string]any{
			"WarehouseId": wh.WarehouseID,
			"Name":        wh.Name,
			"Address":     spanner.NullString{StringVal: wh.Address, Valid: wh.Address != ""},
			"IsActive":    wh.IsActive,
			"IsOnShift":   wh.IsOnShift,
			"UpdatedAt":   spanner.CommitTimestamp,
		})
	}
	tx.cacheWarehouse(wh)

	if ext := strings.TrimSpace(extPlantID); ext != "" {
		pk := plantMapKey(tx.supplierID, ext)
		prevWH, existed := tx.plantMaps[pk]
		if !existed && !tx.whLoaded {
			row, err := tx.txn.ReadRow(ctx, "PartnerPlantMaps", spanner.Key{"SUPPLIER", tx.supplierID, ext}, []string{"WarehouseId"})
			if err == nil {
				if colErr := row.Columns(&prevWH); colErr == nil {
					existed = true
				}
			} else if spanner.ErrCode(err) != codes.NotFound {
				return err
			}
		}
		keyObj := map[string]any{"TenantType": "SUPPLIER", "TenantId": tx.supplierID, "ExternalPlantId": ext}
		if !existed {
			if err := tx.RecordSnapshot(ctx, "PartnerPlantMaps", keyObj, SnapshotOpInsert, nil); err != nil {
				return err
			}
		} else if prevWH != wh.WarehouseID {
			if err := tx.RecordSnapshot(ctx, "PartnerPlantMaps", keyObj, SnapshotOpUpdate, snapshotPlantMap(tx.supplierID, ext, prevWH)); err != nil {
				return err
			}
		}
		tx.plantMaps[pk] = wh.WarehouseID
		tx.plantMuts[pk] = spanner.InsertOrUpdateMap("PartnerPlantMaps", map[string]any{
			"TenantType":      "SUPPLIER",
			"TenantId":        tx.supplierID,
			"ExternalPlantId": ext,
			"WarehouseId":     wh.WarehouseID,
			"UpdatedAt":       spanner.CommitTimestamp,
		})
	}
	return nil
}

func (tx *spannerApplyTx) cacheProduct(p StoredProduct) {
	tx.products[p.ProductID] = p
	if bc := strings.ToLower(strings.TrimSpace(p.Barcode)); bc != "" {
		tx.prodByBarcode[bc] = p.ProductID
	}
	if nm := strings.ToLower(strings.TrimSpace(p.Name)); nm != "" {
		tx.prodByName[nm] = p.ProductID
	}
}

func (tx *spannerApplyTx) FindProduct(ctx context.Context, internalID, sku, barcode, name string) (StoredProduct, bool, error) {
	for _, candidate := range []string{strings.TrimSpace(internalID), strings.TrimSpace(sku)} {
		if candidate == "" {
			continue
		}
		if p, ok := tx.products[candidate]; ok && p.SupplierID == tx.supplierID {
			return p, true, nil
		}
		if !tx.checkedProdIDs[candidate] {
			tx.checkedProdIDs[candidate] = true
			p, ok, err := tx.readProductByID(ctx, candidate)
			if err != nil || ok {
				return p, ok, err
			}
		}
	}
	if bc := strings.TrimSpace(barcode); bc != "" {
		if id, ok := tx.prodByBarcode[strings.ToLower(bc)]; ok {
			if p, ok2 := tx.products[id]; ok2 && p.SupplierID == tx.supplierID {
				return p, true, nil
			}
		}
		if !tx.checkedProdBC[bc] {
			tx.checkedProdBC[bc] = true
			p, ok, err := tx.queryOneProduct(ctx,
				`WHERE SupplierId = @sid AND Barcode = @val`, map[string]any{"sid": tx.supplierID, "val": bc})
			if err != nil || ok {
				return p, ok, err
			}
		}
	}
	if nm := strings.ToLower(strings.TrimSpace(name)); nm != "" {
		if id, ok := tx.prodByName[nm]; ok {
			if p, ok2 := tx.products[id]; ok2 && p.SupplierID == tx.supplierID {
				return p, true, nil
			}
		}
		if !tx.checkedProdName[nm] {
			tx.checkedProdName[nm] = true
			p, ok, err := tx.queryOneProduct(ctx,
				`WHERE SupplierId = @sid AND LOWER(Name) = @val`, map[string]any{"sid": tx.supplierID, "val": nm})
			if err != nil || ok {
				return p, ok, err
			}
		}
	}
	return StoredProduct{}, false, nil
}

const productSelectCols = `ProductId, SupplierId, CategoryId, Name, Description, PriceMinor, Currency, Unit, Barcode, HandlingClass, IsPerishable, RequiresColdChain, MinShelfLifeDays, IsActive, Version`

var productColsSlice = []string{
	"ProductId", "SupplierId", "CategoryId", "Name", "Description", "PriceMinor", "Currency",
	"Unit", "Barcode", "HandlingClass", "IsPerishable", "RequiresColdChain", "MinShelfLifeDays", "IsActive", "Version",
}

func (tx *spannerApplyTx) readProductByID(ctx context.Context, productID string) (StoredProduct, bool, error) {
	row, err := tx.txn.ReadRow(ctx, "Products", spanner.Key{productID}, productColsSlice)
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return StoredProduct{}, false, nil
		}
		return StoredProduct{}, false, err
	}
	p, err := scanStoredProduct(row)
	if err != nil {
		return StoredProduct{}, false, err
	}
	if p.SupplierID != tx.supplierID {
		return StoredProduct{}, false, nil
	}
	tx.cacheProduct(p)
	return p, true, nil
}

func (tx *spannerApplyTx) queryOneProduct(ctx context.Context, whereSQL string, params map[string]any) (StoredProduct, bool, error) {
	iter := tx.txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT ` + productSelectCols + ` FROM Products ` + whereSQL + ` LIMIT 1`,
		Params: params,
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return StoredProduct{}, false, nil
	}
	if err != nil {
		return StoredProduct{}, false, err
	}
	p, err := scanStoredProduct(row)
	if err != nil {
		return StoredProduct{}, false, err
	}
	tx.cacheProduct(p)
	return p, true, nil
}

func (tx *spannerApplyTx) TotalProductOnHand(ctx context.Context, productID string) (int64, error) {
	iter := tx.txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT WarehouseId, QuantityOnHand FROM SupplierInventoryV2 WHERE SupplierId = @sid AND ProductId = @pid`,
		Params: map[string]any{"sid": tx.supplierID, "pid": productID},
	})
	defer iter.Stop()
	byWH := make(map[string]int64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		var whID string
		var qoh int64
		if err := row.Columns(&whID, &qoh); err != nil {
			return 0, err
		}
		byWH[whID] = qoh
	}
	for _, stk := range tx.stocks {
		if stk.SupplierID == tx.supplierID && stk.ProductID == productID {
			byWH[stk.WarehouseID] = stk.QuantityOnHand
		}
	}
	var total int64
	for _, qoh := range byWH {
		total += qoh
	}
	return total, nil
}

func (tx *spannerApplyTx) SaveProduct(ctx context.Context, p StoredProduct, before *StoredProduct) error {
	if before == nil || tx.insertedProd[p.ProductID] {
		if before == nil {
			tx.insertedProd[p.ProductID] = true
			if err := tx.RecordSnapshot(ctx, "Products", map[string]any{"ProductId": p.ProductID}, SnapshotOpInsert, nil); err != nil {
				return err
			}
		}
		tx.prodMuts[p.ProductID] = spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":         p.ProductID,
			"SupplierId":        p.SupplierID,
			"CategoryId":        p.CategoryID,
			"Name":              p.Name,
			"Description":       spanner.NullString{StringVal: p.Description, Valid: p.Description != ""},
			"ImageURL":          spanner.NullString{},
			"PriceMinor":        p.PriceMinor,
			"Currency":          p.Currency,
			"StockQuantity":     int64(0),
			"Unit":              p.Unit,
			"UnitVolumeVU":      1.0,
			"SaleUnit":          "PIECE",
			"UnitsPerPack":      spanner.NullInt64{},
			"Barcode":           spanner.NullString{StringVal: p.Barcode, Valid: p.Barcode != ""},
			"HandlingClass":     firstNonEmpty(p.HandlingClass, "GENERAL"),
			"IsHazardous":       false,
			"IsPerishable":      p.IsPerishable,
			"RequiresColdChain": p.RequiresColdChain,
			"MinShelfLifeDays":  spanner.NullInt64{Int64: p.MinShelfLifeDays, Valid: p.MinShelfLifeDays > 0},
			"IsActive":          p.IsActive,
			"Version":           p.Version,
			"CreatedAt":         spanner.CommitTimestamp,
			"UpdatedAt":         spanner.CommitTimestamp,
		})
	} else {
		if err := tx.RecordSnapshot(ctx, "Products", map[string]any{"ProductId": p.ProductID}, SnapshotOpUpdate, snapshotProductMap(*before)); err != nil {
			return err
		}
		tx.prodMuts[p.ProductID] = spanner.UpdateMap("Products", map[string]any{
			"ProductId":         p.ProductID,
			"CategoryId":        p.CategoryID,
			"Name":              p.Name,
			"Description":       spanner.NullString{StringVal: p.Description, Valid: p.Description != ""},
			"PriceMinor":        p.PriceMinor,
			"Currency":          p.Currency,
			"Unit":              p.Unit,
			"Barcode":           spanner.NullString{StringVal: p.Barcode, Valid: p.Barcode != ""},
			"HandlingClass":     firstNonEmpty(p.HandlingClass, "GENERAL"),
			"IsPerishable":      p.IsPerishable,
			"RequiresColdChain": p.RequiresColdChain,
			"MinShelfLifeDays":  spanner.NullInt64{Int64: p.MinShelfLifeDays, Valid: p.MinShelfLifeDays > 0},
			"IsActive":          p.IsActive,
			"Version":           p.Version,
			"UpdatedAt":         spanner.CommitTimestamp,
		})
	}
	tx.cacheProduct(p)
	return nil
}

func (tx *spannerApplyTx) cacheCustomer(ret StoredCustomer) {
	tx.customers[ret.RetailerID] = ret
	if tin := NormalizeINN(ret.Link.TaxIdTIN); tin != "" {
		tx.custByINN[tin] = ret.RetailerID
	}
	if ph := NormalizePhoneUZ(ret.Phone); ph != "" {
		tx.custByPhone[ph] = ret.RetailerID
	}
	if ph := NormalizePhoneUZ(ret.Link.NormalizedPhone); ph != "" {
		tx.custByPhone[ph] = ret.RetailerID
	}
	if ln := strings.ToLower(strings.TrimSpace(ret.Link.LegalName)); ln != "" && ret.Link.SupplierID == tx.supplierID {
		tx.custByLegalName[ln] = ret.RetailerID
	}
}

func (tx *spannerApplyTx) FindCustomer(ctx context.Context, homeCell, extID, inn, phone string) (StoredCustomer, bool, error) {
	if id := strings.TrimSpace(extID); id != "" {
		if ret, ok := tx.customers[id]; ok {
			return ret, true, nil
		}
		if !tx.checkedCustIDs[id] {
			tx.checkedCustIDs[id] = true
			if ret, ok, err := tx.readCustomerByRetailerID(ctx, id); err != nil || ok {
				return ret, ok, err
			}
		}
	}
	normINN := NormalizeINN(inn)
	if normINN != "" {
		if id, ok := tx.custByINN[normINN]; ok {
			if ret, ok2 := tx.customers[id]; ok2 {
				return ret, true, nil
			}
		}
		if !tx.checkedCustINNs[normINN] {
			tx.checkedCustINNs[normINN] = true
			if ret, ok, err := tx.queryCustomerByINN(ctx, homeCell, normINN); err != nil || ok {
				if ok {
					tx.custByINN[normINN] = ret.RetailerID
				}
				return ret, ok, err
			}
		}
	}
	normPhone := NormalizePhoneUZ(phone)
	if normPhone != "" {
		if id, ok := tx.custByPhone[normPhone]; ok {
			if ret, ok2 := tx.customers[id]; ok2 {
				return ret, true, nil
			}
		}
		if !tx.checkedCustPh[normPhone] {
			tx.checkedCustPh[normPhone] = true
			if ret, ok, err := tx.queryCustomerByPhone(ctx, normPhone); err != nil || ok {
				if ok {
					tx.custByPhone[normPhone] = ret.RetailerID
				}
				return ret, ok, err
			}
		}
	}
	return StoredCustomer{}, false, nil
}

func (tx *spannerApplyTx) readCustomerByRetailerID(ctx context.Context, retailerID string) (StoredCustomer, bool, error) {
	row, err := tx.txn.ReadRow(ctx, "Retailers", spanner.Key{retailerID},
		[]string{"RetailerId", "Phone", "Name", "CountryCode", "DeliveryAddress"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return StoredCustomer{}, false, nil
		}
		return StoredCustomer{}, false, err
	}
	var ret StoredCustomer
	var addr spanner.NullString
	if err := row.Columns(&ret.RetailerID, &ret.Phone, &ret.Name, &ret.CountryCode, &addr); err != nil {
		return StoredCustomer{}, false, err
	}
	ret.DeliveryAddr = addr.StringVal
	link, hasLink, err := tx.readSupplierRetailerLink(ctx, retailerID)
	if err != nil {
		return StoredCustomer{}, false, err
	}
	if hasLink {
		ret.Link = link
	}
	tx.cacheCustomer(ret)
	return ret, true, nil
}

func (tx *spannerApplyTx) readSupplierRetailerLink(ctx context.Context, retailerID string) (StoredCustomerLink, bool, error) {
	row, err := tx.txn.ReadRow(ctx, "SupplierRetailerLinks", spanner.Key{tx.supplierID, retailerID},
		[]string{"SupplierId", "RetailerId", "TaxIdTIN", "NormalizedPhone", "LegalName", "DedupClusterId", "DedupStatus"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return StoredCustomerLink{}, false, nil
		}
		return StoredCustomerLink{}, false, err
	}
	var link StoredCustomerLink
	var tin, phone, legal, cluster spanner.NullString
	if err := row.Columns(&link.SupplierID, &link.RetailerID, &tin, &phone, &legal, &cluster, &link.DedupStatus); err != nil {
		return StoredCustomerLink{}, false, err
	}
	link.TaxIdTIN = tin.StringVal
	link.NormalizedPhone = phone.StringVal
	link.LegalName = legal.StringVal
	link.DedupClusterID = cluster.StringVal
	return link, true, nil
}

func (tx *spannerApplyTx) queryCustomerByINN(ctx context.Context, homeCell, inn string) (StoredCustomer, bool, error) {
	iter := tx.txn.Query(ctx, spanner.Statement{
		SQL: `SELECT l.RetailerId, COALESCE(r.HomeCell, r.H3Cell, ''), (l.SupplierId = @sid) AS same_sup
		      FROM SupplierRetailerLinks l
		      JOIN Retailers r ON r.RetailerId = l.RetailerId
		      WHERE l.TaxIdTIN = @inn
		        AND (@homeCell = '' OR COALESCE(r.HomeCell, r.H3Cell, '') = @homeCell OR COALESCE(r.HomeCell, r.H3Cell, '') = '' OR l.SupplierId = @sid)
		      ORDER BY same_sup DESC
		      LIMIT 1`,
		Params: map[string]any{"sid": tx.supplierID, "inn": inn, "homeCell": homeCell},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return StoredCustomer{}, false, nil
	}
	if err != nil {
		return StoredCustomer{}, false, err
	}
	var retailerID, cell string
	var sameSup bool
	if err := row.Columns(&retailerID, &cell, &sameSup); err != nil {
		return StoredCustomer{}, false, err
	}
	return tx.readCustomerByRetailerID(ctx, retailerID)
}

func (tx *spannerApplyTx) queryCustomerByPhone(ctx context.Context, phone string) (StoredCustomer, bool, error) {
	iter := tx.txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT RetailerId FROM Retailers WHERE Phone = @phone LIMIT 1`,
		Params: map[string]any{"phone": phone},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == nil {
		var retailerID string
		if colErr := row.Columns(&retailerID); colErr != nil {
			return StoredCustomer{}, false, colErr
		}
		return tx.readCustomerByRetailerID(ctx, retailerID)
	}
	if err != iterator.Done {
		return StoredCustomer{}, false, err
	}
	return StoredCustomer{}, false, nil
}

func (tx *spannerApplyTx) SaveCustomer(ctx context.Context, ret StoredCustomer, createRetailer bool, beforeLink *StoredCustomerLink) error {
	if createRetailer {
		homeCell, _, _ := tx.ReadSupplierMeta(ctx)
		tx.retMuts[ret.RetailerID] = spanner.InsertOrUpdateMap("Retailers", map[string]any{
			"RetailerId":      ret.RetailerID,
			"Phone":           ret.Phone,
			"Name":            ret.Name,
			"CountryCode":     firstNonEmpty(ret.CountryCode, "UZ"),
			"Lat":             0.0,
			"Lng":             0.0,
			"HomeCell":        spanner.NullString{StringVal: homeCell, Valid: homeCell != ""},
			"DeliveryAddress": spanner.NullString{StringVal: ret.DeliveryAddr, Valid: ret.DeliveryAddr != ""},
			"CreatedAt":       spanner.CommitTimestamp,
		})
	}
	keyObj := map[string]any{"SupplierId": ret.Link.SupplierID, "RetailerId": ret.Link.RetailerID}
	lk := supplierLinkKey(ret.Link.SupplierID, ret.Link.RetailerID)
	if beforeLink == nil {
		tx.insertedLink[lk] = true
		if err := tx.RecordSnapshot(ctx, "SupplierRetailerLinks", keyObj, SnapshotOpInsert, nil); err != nil {
			return err
		}
	} else {
		if err := tx.RecordSnapshot(ctx, "SupplierRetailerLinks", keyObj, SnapshotOpUpdate, snapshotCustomerLinkMap(*beforeLink)); err != nil {
			return err
		}
	}
	linkMap := map[string]any{
		"SupplierId":      ret.Link.SupplierID,
		"RetailerId":      ret.Link.RetailerID,
		"TaxIdTIN":        spanner.NullString{StringVal: ret.Link.TaxIdTIN, Valid: ret.Link.TaxIdTIN != ""},
		"NormalizedPhone": spanner.NullString{StringVal: ret.Link.NormalizedPhone, Valid: ret.Link.NormalizedPhone != ""},
		"LegalName":       spanner.NullString{StringVal: ret.Link.LegalName, Valid: ret.Link.LegalName != ""},
		"DedupStatus":     firstNonEmpty(ret.Link.DedupStatus, "CLEAN"),
		"UpdatedAt":       spanner.CommitTimestamp,
	}
	if beforeLink == nil || tx.insertedLink[lk] {
		linkMap["CreatedAt"] = spanner.CommitTimestamp
	}
	tx.linkMuts[lk] = spanner.InsertOrUpdateMap("SupplierRetailerLinks", linkMap)
	tx.cacheCustomer(ret)
	return nil
}

func (tx *spannerApplyTx) GetInventory(ctx context.Context, warehouseID, productID string) (StoredStock, bool, error) {
	k := inventoryKey(tx.supplierID, warehouseID, productID)
	if stk, ok := tx.stocks[k]; ok {
		return stk, true, nil
	}
	if tx.checkedStocks[k] {
		return StoredStock{}, false, nil
	}
	tx.checkedStocks[k] = true
	row, err := tx.txn.ReadRow(ctx, "SupplierInventoryV2",
		spanner.Key{tx.supplierID, warehouseID, productID},
		[]string{"QuantityOnHand", "QuantityReserved", "ReorderThreshold"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return StoredStock{}, false, nil
		}
		return StoredStock{}, false, err
	}
	stk := StoredStock{
		SupplierID:  tx.supplierID,
		WarehouseID: warehouseID,
		ProductID:   productID,
	}
	if err := row.Columns(&stk.QuantityOnHand, &stk.QuantityReserved, &stk.ReorderThreshold); err != nil {
		return StoredStock{}, false, err
	}
	tx.stocks[k] = stk
	return stk, true, nil
}

func (tx *spannerApplyTx) UpsertStock(ctx context.Context, stk StoredStock, before *StoredStock) error {
	keyObj := map[string]any{"SupplierId": stk.SupplierID, "WarehouseId": stk.WarehouseID, "ProductId": stk.ProductID}
	if before == nil {
		if err := tx.RecordSnapshot(ctx, "SupplierInventoryV2", keyObj, SnapshotOpInsert, nil); err != nil {
			return err
		}
	} else {
		if err := tx.RecordSnapshot(ctx, "SupplierInventoryV2", keyObj, SnapshotOpUpdate, snapshotStockMap(*before)); err != nil {
			return err
		}
	}
	k := inventoryKey(stk.SupplierID, stk.WarehouseID, stk.ProductID)
	tx.stocks[k] = stk
	tx.stockMuts[k] = spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
		"SupplierId":       stk.SupplierID,
		"WarehouseId":      stk.WarehouseID,
		"ProductId":        stk.ProductID,
		"QuantityOnHand":   stk.QuantityOnHand,
		"QuantityReserved": stk.QuantityReserved,
		"ReorderThreshold": stk.ReorderThreshold,
		"UpdatedAt":        spanner.CommitTimestamp,
	})
	return nil
}

func (tx *spannerApplyTx) FindLot(ctx context.Context, warehouseID, productID, lotCode string) (StoredLot, bool, error) {
	lk := strings.ToLower(strings.TrimSpace(warehouseID + "|" + productID + "|" + lotCode))
	if id, ok := tx.lotsByKey[lk]; ok {
		if lot, ok2 := tx.lotsByID[id]; ok2 {
			return lot, true, nil
		}
	}
	pairKey := strings.TrimSpace(warehouseID) + "|" + strings.TrimSpace(productID)
	if tx.checkedLotPairs[pairKey] || tx.checkedLotKeys[lk] {
		return StoredLot{}, false, nil
	}
	tx.checkedLotKeys[lk] = true
	iter := tx.txn.Query(ctx, spanner.Statement{
		SQL: `SELECT LotId, SupplierId, WarehouseId, ProductId, LocationId, LotCode, ExpiryDate, ManufacturedDate, QuantityOnHand, QuantityReserved, Status
		      FROM StockLots
		      WHERE SupplierId = @sid AND WarehouseId = @wid AND ProductId = @pid AND LOWER(LotCode) = @code
		      LIMIT 1`,
		Params: map[string]any{
			"sid":  tx.supplierID,
			"wid":  warehouseID,
			"pid":  productID,
			"code": strings.ToLower(strings.TrimSpace(lotCode)),
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return StoredLot{}, false, nil
	}
	if err != nil {
		return StoredLot{}, false, err
	}
	lot, err := scanStoredLot(row)
	if err != nil {
		return StoredLot{}, false, err
	}
	tx.lotsByID[lot.LotID] = lot
	tx.lotsByKey[lk] = lot.LotID
	return lot, true, nil
}

func (tx *spannerApplyTx) PutawayOrAdjustLot(ctx context.Context, op LotMigrationOp, existing *StoredLot, beforeStock *StoredStock) (LotMigrationResult, error) {
	locID := firstNonEmpty(strings.TrimSpace(op.LocationID), stocklots.DefaultRecvLocationID)
	binKey := op.WarehouseID + "|" + locID
	if !tx.locations[binKey] {
		if _, err := stocklots.UpsertBinInTxn(ctx, tx.txn, stocklots.CreateBinRequest{
			WarehouseID:  op.WarehouseID,
			LocationID:   locID,
			Zone:         "RECV",
			LocationType: "STAGE",
		}); err != nil {
			return LotMigrationResult{}, err
		}
		tx.locations[binKey] = true
	}

	var nextLot StoredLot
	var result LotMigrationResult
	if existing == nil {
		if op.TargetQuantity == 0 {
			return LotMigrationResult{LotID: op.NewLotID, DeltaQty: 0, Action: "skipped", FinalOnHand: 0}, nil
		}
		if err := tx.RecordSnapshot(ctx, "StockLots", map[string]any{"LotId": op.NewLotID}, SnapshotOpInsert, nil); err != nil {
			return LotMigrationResult{}, err
		}
		tx.insertedLot[op.NewLotID] = true
		nextLot = StoredLot{
			LotID:            op.NewLotID,
			SupplierID:       tx.supplierID,
			WarehouseID:      op.WarehouseID,
			ProductID:        op.ProductID,
			LocationID:       locID,
			LotCode:          strings.TrimSpace(op.LotCode),
			ExpiryDate:       op.ExpiryDate,
			ManufacturedDate: op.ManufacturedDate,
			QuantityOnHand:   op.TargetQuantity,
			QuantityReserved: 0,
			Status:           "AVAILABLE",
		}
		cols := buildStockLotMutationCols(nextLot, true)
		tx.lotMuts[nextLot.LotID] = spanner.InsertOrUpdateMap("StockLots", cols)
		result = LotMigrationResult{
			LotID:       nextLot.LotID,
			DeltaQty:    op.TargetQuantity,
			Action:      "created",
			FinalOnHand: op.TargetQuantity,
		}
	} else {
		if op.TargetQuantity < existing.QuantityReserved {
			return LotMigrationResult{}, fmt.Errorf("%w: lot %s target %d < reserved %d",
				ErrOnHandBelowReserved, existing.LotCode, op.TargetQuantity, existing.QuantityReserved)
		}
		delta := op.TargetQuantity - existing.QuantityOnHand
		if delta == 0 {
			return LotMigrationResult{
				LotID:       existing.LotID,
				DeltaQty:    0,
				Action:      "skipped",
				FinalOnHand: existing.QuantityOnHand,
			}, nil
		}
		if !tx.insertedLot[existing.LotID] {
			if err := tx.RecordSnapshot(ctx, "StockLots", map[string]any{"LotId": existing.LotID}, SnapshotOpUpdate, snapshotLotMap(*existing)); err != nil {
				return LotMigrationResult{}, err
			}
		}
		nextLot = *existing
		nextLot.LocationID = locID
		nextLot.QuantityOnHand = op.TargetQuantity
		if op.ExpiryDate != nil {
			nextLot.ExpiryDate = op.ExpiryDate
		}
		if op.ManufacturedDate != nil {
			nextLot.ManufacturedDate = op.ManufacturedDate
		}
		if nextLot.Status == "" {
			nextLot.Status = "AVAILABLE"
		}
		cols := buildStockLotMutationCols(nextLot, tx.insertedLot[existing.LotID])
		tx.lotMuts[nextLot.LotID] = spanner.InsertOrUpdateMap("StockLots", cols)
		result = LotMigrationResult{
			LotID:       nextLot.LotID,
			DeltaQty:    delta,
			Action:      "updated",
			FinalOnHand: nextLot.QuantityOnHand,
		}
	}

	tx.lotsByID[nextLot.LotID] = nextLot
	lk := strings.ToLower(strings.TrimSpace(op.WarehouseID + "|" + op.ProductID + "|" + op.LotCode))
	tx.lotsByKey[lk] = nextLot.LotID

	pairKey := op.WarehouseID + "|" + op.ProductID
	if tx.lotOverrides[pairKey] == nil {
		tx.lotOverrides[pairKey] = make(map[string]stocklots.StockLotSnapshot)
	}
	tx.lotOverrides[pairKey][nextLot.LotID] = stocklots.StockLotSnapshot{
		SupplierID:       tx.supplierID,
		WarehouseID:      op.WarehouseID,
		ProductID:        op.ProductID,
		QuantityOnHand:   nextLot.QuantityOnHand,
		QuantityReserved: nextLot.QuantityReserved,
		Status:           nextLot.Status,
	}
	tx.rollupPairs[pairKey] = whProdPair{whID: op.WarehouseID, prodID: op.ProductID}

	stkKeyObj := map[string]any{"SupplierId": tx.supplierID, "WarehouseId": op.WarehouseID, "ProductId": op.ProductID}
	if beforeStock == nil {
		if err := tx.RecordSnapshot(ctx, "SupplierInventoryV2", stkKeyObj, SnapshotOpInsert, nil); err != nil {
			return LotMigrationResult{}, err
		}
	} else {
		if err := tx.RecordSnapshot(ctx, "SupplierInventoryV2", stkKeyObj, SnapshotOpUpdate, snapshotStockMap(*beforeStock)); err != nil {
			return LotMigrationResult{}, err
		}
	}
	// Keep in-transaction inventory cache current for subsequent rows in this same chunk.
	curStock := StoredStock{SupplierID: tx.supplierID, WarehouseID: op.WarehouseID, ProductID: op.ProductID}
	if beforeStock != nil {
		curStock = *beforeStock
	}
	curStock.QuantityOnHand += result.DeltaQty
	tx.stocks[inventoryKey(tx.supplierID, op.WarehouseID, op.ProductID)] = curStock

	return result, nil
}

func (tx *spannerApplyTx) flush(ctx context.Context) error {
	var muts []*spanner.Mutation
	appendSortedMuts := func(m map[string]*spanner.Mutation) {
		if len(m) == 0 {
			return
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			muts = append(muts, m[k])
		}
	}
	appendSortedMuts(tx.whMuts)
	appendSortedMuts(tx.plantMuts)
	appendSortedMuts(tx.prodMuts)
	appendSortedMuts(tx.retMuts)
	appendSortedMuts(tx.linkMuts)
	appendSortedMuts(tx.stockMuts)
	appendSortedMuts(tx.lotMuts)
	appendSortedMuts(tx.orderMuts)
	appendSortedMuts(tx.idMuts)
	muts = append(muts, tx.snapMuts...)
	if len(muts) > 0 {
		if err := tx.txn.BufferWrite(muts); err != nil {
			return err
		}
	}

	if tx.arCache != nil {
		if err := ar.FlushMigrationCreditProfilesAndGLInTxn(ctx, tx.txn, nil, tx.arCache, tx.now); err != nil {
			return err
		}
	}

	if len(tx.rollupPairs) > 0 {
		keys := make([]string, 0, len(tx.rollupPairs))
		for k := range tx.rollupPairs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			pair := tx.rollupPairs[k]
			if err := stocklots.RollupInventoryV2WithLotOverridesInTxn(ctx, tx.txn, tx.supplierID, pair.whID, pair.prodID, tx.lotOverrides[k]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (tx *spannerApplyTx) ResolveCustomerForFinancials(ctx context.Context, homeCell, extID, inn, phone, name string) (StoredCustomer, bool, error) {
	if cust, ok, err := tx.FindCustomer(ctx, homeCell, extID, inn, phone); err != nil || ok {
		return cust, ok, err
	}
	wantName := strings.TrimSpace(name)
	if wantName != "" {
		lowerName := strings.ToLower(wantName)
		if id, ok := tx.custByLegalName[lowerName]; ok {
			if cust, ok2 := tx.customers[id]; ok2 {
				return cust, true, nil
			}
		}
		if tx.checkedCustName[lowerName] {
			return StoredCustomer{}, false, nil
		}
		tx.checkedCustName[lowerName] = true
		iter := tx.txn.Query(ctx, spanner.Statement{
			SQL: `SELECT RetailerId, COALESCE(TaxIdTIN, ''), COALESCE(NormalizedPhone, ''), COALESCE(LegalName, ''), COALESCE(DedupStatus, 'CLEAN')
			      FROM SupplierRetailerLinks
			      WHERE SupplierId = @sid AND LOWER(COALESCE(LegalName, '')) = LOWER(@nm) LIMIT 1`,
			Params: map[string]any{"sid": tx.supplierID, "nm": wantName},
		})
		row, err := iter.Next()
		iter.Stop()
		if err == nil {
			var link StoredCustomerLink
			link.SupplierID = tx.supplierID
			if cErr := row.Columns(&link.RetailerID, &link.TaxIdTIN, &link.NormalizedPhone, &link.LegalName, &link.DedupStatus); cErr == nil {
				if cust, ok, rErr := tx.readCustomerByRetailerID(ctx, link.RetailerID); rErr == nil && ok {
					cust.Link = link
					tx.cacheCustomer(cust)
					return cust, true, nil
				}
			}
		} else if err != iterator.Done {
			return StoredCustomer{}, false, err
		}
	}
	return StoredCustomer{}, false, nil
}

func (tx *spannerApplyTx) ResolveWarehouseForOrder(ctx context.Context, whHint, defaultWarehouseID string) (StoredWarehouse, bool, error) {
	hint := strings.TrimSpace(whHint)
	if hint != "" {
		return tx.FindWarehouse(ctx, hint, hint)
	}
	if strings.TrimSpace(defaultWarehouseID) != "" {
		return tx.FindWarehouse(ctx, defaultWarehouseID, "")
	}
	if tx.whLoaded {
		var best StoredWarehouse
		found := false
		for _, wh := range tx.warehouses {
			if wh.SupplierID != tx.supplierID || !wh.IsActive {
				continue
			}
			if !found || wh.Name < best.Name {
				best = wh
				found = true
			}
		}
		return best, found, nil
	}
	iter := tx.txn.Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, SupplierId, Name, Address, IsActive, IsOnShift
		      FROM Warehouses WHERE SupplierId = @sid AND IsActive = TRUE ORDER BY Name ASC LIMIT 1`,
		Params: map[string]any{"sid": tx.supplierID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return StoredWarehouse{}, false, nil
	}
	if err != nil {
		return StoredWarehouse{}, false, err
	}
	wh, err := scanStoredWarehouse(row)
	if err != nil {
		return StoredWarehouse{}, false, err
	}
	tx.cacheWarehouse(wh)
	return wh, true, nil
}

func (tx *spannerApplyTx) loadOrderByExternalNo(ctx context.Context, extNo string) (order.Order, bool, error) {
	key := strings.ToLower(strings.TrimSpace(extNo))
	if oid, ok := tx.ordersByExt[key]; ok {
		if ord, ok2 := tx.ordersByID[oid]; ok2 {
			return ord, true, nil
		}
	}
	if tx.checkedOrderExt[key] || tx.txn == nil {
		return order.Order{}, false, nil
	}
	tx.checkedOrderExt[key] = true
	iter := tx.txn.Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, Status, OrderSource, LineItemsJson, TotalMinor, OriginalTotalMinor, Version
		      FROM Orders WHERE SupplierId = @sid AND ExternalOrderNo = @eno LIMIT 1`,
		Params: map[string]any{"sid": tx.supplierID, "eno": strings.TrimSpace(extNo)},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return order.Order{}, false, nil
	}
	if err != nil {
		return order.Order{}, false, err
	}
	var ord order.Order
	var st, src string
	var lineRaw []byte
	if err := row.Columns(&ord.OrderID, &st, &src, &lineRaw, &ord.TotalMinor, &ord.OriginalTotalMinor, &ord.Version); err != nil {
		return order.Order{}, false, err
	}
	ord.SupplierID = tx.supplierID
	ord.ExternalOrderNo = strings.TrimSpace(extNo)
	ord.Status = order.Status(st)
	ord.Source = order.OrderSource(src)
	if len(lineRaw) > 0 {
		if err := json.Unmarshal(lineRaw, &ord.LineItems); err != nil {
			return order.Order{}, false, err
		}
	}
	tx.ordersByID[ord.OrderID] = ord
	tx.ordersByExt[key] = ord.OrderID
	return ord, true, nil
}

func (tx *spannerApplyTx) ImportHistoricalOrder(ctx context.Context, req order.HistoricalOrderRequest) (bool, bool, string, error) {
	prep, insertMap, err := order.PrepareHistoricalOrder(req)
	if err != nil {
		return false, false, "", err
	}
	if existing, found, err := tx.loadOrderByExternalNo(ctx, prep.ExternalOrderNo); err != nil {
		return false, false, "", err
	} else if found {
		if len(prep.LineItems) > 0 && !containsOrderLineItem(existing.LineItems, prep.LineItems[0]) {
			existing.LineItems = append(existing.LineItems, prep.LineItems...)
			var addedTotal int64
			for _, li := range prep.LineItems {
				addedTotal += li.Quantity * li.UnitPrice
			}
			existing.TotalMinor += addedTotal
			existing.OriginalTotalMinor = existing.TotalMinor
			linesRaw, mErr := json.Marshal(existing.LineItems)
			if mErr != nil {
				return false, false, "", mErr
			}
			existing.Version++
			existing.UpdatedAt = tx.now
			tx.ordersByID[existing.OrderID] = existing
			cols, ok := tx.orderCols[existing.OrderID]
			if !ok {
				cols = make(map[string]any)
				tx.orderCols[existing.OrderID] = cols
			}
			cols["OrderId"] = existing.OrderID
			cols["LineItemsJson"] = linesRaw
			cols["TotalMinor"] = existing.TotalMinor
			cols["OriginalTotalMinor"] = existing.OriginalTotalMinor
			cols["Version"] = existing.Version
			cols["UpdatedAt"] = tx.now
			tx.orderMuts[existing.OrderID] = spanner.InsertOrUpdateMap("Orders", cols)
			return false, true, existing.OrderID, nil
		}
		return false, false, existing.OrderID, nil
	}

	if err := tx.RecordSnapshot(ctx, "Orders", map[string]any{"OrderId": prep.OrderID}, SnapshotOpInsert, nil); err != nil {
		return false, false, "", err
	}
	tx.ordersByID[prep.OrderID] = prep
	tx.ordersByExt[strings.ToLower(prep.ExternalOrderNo)] = prep.OrderID
	tx.orderCols[prep.OrderID] = insertMap
	tx.orderMuts[prep.OrderID] = spanner.InsertOrUpdateMap("Orders", insertMap)
	return true, false, prep.OrderID, nil
}

func (tx *spannerApplyTx) ImportMigratedOpenOrder(ctx context.Context, req order.MigratedOpenOrderRequest) (bool, bool, string, error) {
	prep, insertMap, err := order.PrepareMigratedOpenOrder(req)
	if err != nil {
		return false, false, "", err
	}
	if existing, found, err := tx.loadOrderByExternalNo(ctx, prep.ExternalOrderNo); err != nil {
		return false, false, "", err
	} else if found {
		if len(prep.LineItems) > 0 && !containsOrderLineItem(existing.LineItems, prep.LineItems[0]) {
			existing.LineItems = append(existing.LineItems, prep.LineItems...)
			var addedTotal int64
			for _, li := range prep.LineItems {
				addedTotal += li.Quantity * li.UnitPrice
			}
			existing.TotalMinor += addedTotal
			existing.OriginalTotalMinor = existing.TotalMinor
			linesRaw, mErr := json.Marshal(existing.LineItems)
			if mErr != nil {
				return false, false, "", mErr
			}
			existing.Version++
			existing.UpdatedAt = tx.now
			tx.ordersByID[existing.OrderID] = existing
			cols, ok := tx.orderCols[existing.OrderID]
			if !ok {
				cols = make(map[string]any)
				tx.orderCols[existing.OrderID] = cols
			}
			cols["OrderId"] = existing.OrderID
			cols["LineItemsJson"] = linesRaw
			cols["TotalMinor"] = existing.TotalMinor
			cols["OriginalTotalMinor"] = existing.OriginalTotalMinor
			cols["Version"] = existing.Version
			cols["UpdatedAt"] = tx.now
			tx.orderMuts[existing.OrderID] = spanner.InsertOrUpdateMap("Orders", cols)
			return false, true, existing.OrderID, nil
		}
		return false, false, existing.OrderID, nil
	}

	if err := tx.RecordSnapshot(ctx, "Orders", map[string]any{"OrderId": prep.OrderID}, SnapshotOpInsert, nil); err != nil {
		return false, false, "", err
	}
	tx.ordersByID[prep.OrderID] = prep
	tx.ordersByExt[strings.ToLower(prep.ExternalOrderNo)] = prep.OrderID
	tx.orderCols[prep.OrderID] = insertMap
	tx.orderMuts[prep.OrderID] = spanner.InsertOrUpdateMap("Orders", insertMap)
	return true, false, prep.OrderID, nil
}

func (tx *spannerApplyTx) OpenMigrationNasiyaAR(ctx context.Context, req ar.OpenFromMigrationRequest) (*ar.OpenFromMigrationResult, error) {
	res, err := ar.OpenFromMigrationInTxn(ctx, tx.txn, tx.arCache, req)
	if err != nil {
		return nil, err
	}
	if res.AlreadyExisted {
		return res, nil
	}
	if err := tx.RecordSnapshot(ctx, "Orders", map[string]any{"OrderId": res.SyntheticOrderID}, SnapshotOpInsert, nil); err != nil {
		return nil, err
	}
	if err := tx.RecordSnapshot(ctx, "ArInvoices", map[string]any{"InvoiceId": res.Invoice.InvoiceID}, SnapshotOpInsert, nil); err != nil {
		return nil, err
	}
	return res, nil
}

func queryMaxSnapshotSeq(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, sessionID string) (int64, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT COALESCE(MAX(snapshot_seq), 0) FROM MigrationSnapshots
		      WHERE supplier_id = @sid AND session_id = @sess`,
		Params: map[string]any{"sid": supplierID, "sess": sessionID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var seq int64
	if err := row.Columns(&seq); err != nil {
		return 0, err
	}
	return seq, nil
}

func scanStoredWarehouse(row *spanner.Row) (StoredWarehouse, error) {
	var wh StoredWarehouse
	var addr spanner.NullString
	if err := row.Columns(&wh.WarehouseID, &wh.SupplierID, &wh.Name, &addr, &wh.IsActive, &wh.IsOnShift); err != nil {
		return StoredWarehouse{}, err
	}
	wh.Address = addr.StringVal
	return wh, nil
}

func scanStoredProduct(row *spanner.Row) (StoredProduct, error) {
	var p StoredProduct
	var desc, bc spanner.NullString
	var minShelf spanner.NullInt64
	if err := row.Columns(
		&p.ProductID, &p.SupplierID, &p.CategoryID, &p.Name, &desc,
		&p.PriceMinor, &p.Currency, &p.Unit, &bc, &p.HandlingClass,
		&p.IsPerishable, &p.RequiresColdChain, &minShelf, &p.IsActive, &p.Version,
	); err != nil {
		return StoredProduct{}, err
	}
	p.Description = desc.StringVal
	p.Barcode = bc.StringVal
	if minShelf.Valid {
		p.MinShelfLifeDays = minShelf.Int64
	}
	return p, nil
}

func scanStoredLot(row *spanner.Row) (StoredLot, error) {
	var l StoredLot
	var exp, mfg spanner.NullDate
	if err := row.Columns(
		&l.LotID, &l.SupplierID, &l.WarehouseID, &l.ProductID, &l.LocationID,
		&l.LotCode, &exp, &mfg, &l.QuantityOnHand, &l.QuantityReserved, &l.Status,
	); err != nil {
		return StoredLot{}, err
	}
	if exp.Valid {
		t := exp.Date.In(time.UTC)
		l.ExpiryDate = &t
	}
	if mfg.Valid {
		t := mfg.Date.In(time.UTC)
		l.ManufacturedDate = &t
	}
	return l, nil
}

func buildStockLotMutationCols(l StoredLot, isInsert bool) map[string]any {
	cols := map[string]any{
		"LotId":          l.LotID,
		"LocationId":     l.LocationID,
		"QuantityOnHand": l.QuantityOnHand,
		"Status":         firstNonEmpty(l.Status, "AVAILABLE"),
		"UpdatedAt":      spanner.CommitTimestamp,
	}
	if isInsert {
		cols["SupplierId"] = l.SupplierID
		cols["WarehouseId"] = l.WarehouseID
		cols["ProductId"] = l.ProductID
		cols["LotCode"] = l.LotCode
		cols["QuantityReserved"] = l.QuantityReserved
		cols["ReceivedAt"] = spanner.CommitTimestamp
		cols["CreatedAt"] = spanner.CommitTimestamp
	}
	if l.ExpiryDate != nil && !l.ExpiryDate.IsZero() {
		cols["ExpiryDate"] = civil.DateOf(l.ExpiryDate.UTC())
	} else {
		cols["ExpiryDate"] = spanner.NullDate{}
	}
	if l.ManufacturedDate != nil && !l.ManufacturedDate.IsZero() {
		cols["ManufacturedDate"] = civil.DateOf(l.ManufacturedDate.UTC())
	} else {
		cols["ManufacturedDate"] = spanner.NullDate{}
	}
	return cols
}

func dedupNonEmpty(vals []string, lower bool) []string {
	out := make([]string, 0, len(vals))
	seen := make(map[string]bool, len(vals))
	for _, raw := range vals {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if lower {
			v = strings.ToLower(v)
		}
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func (tx *spannerApplyTx) PreloadChunk(ctx context.Context, pack Pack, homeCell, defaultWarehouseID string, rows []applyChunkRow) error {
	if len(rows) == 0 {
		return nil
	}
	switch pack {
	case PackWarehouses:
		if err := tx.preloadWarehouses(ctx); err != nil {
			return err
		}
		extIDs := make([]string, 0, len(rows))
		for _, cr := range rows {
			extIDs = append(extIDs, cr.Fields["external_warehouse_id"])
		}
		return tx.preloadIdentities(ctx, PackWarehouses, extIDs)

	case PackProductsPrices:
		extIDs := make([]string, 0, len(rows))
		skus := make([]string, 0, len(rows))
		barcodes := make([]string, 0, len(rows))
		names := make([]string, 0, len(rows))
		for _, cr := range rows {
			sku := strings.TrimSpace(cr.Fields["sku"])
			bc := strings.TrimSpace(cr.Fields["barcode"])
			ext := firstNonEmpty(sku, bc)
			extIDs = append(extIDs, ext)
			skus = append(skus, sku)
			barcodes = append(barcodes, bc)
			names = append(names, cr.Fields["name"])
		}
		if err := tx.preloadIdentities(ctx, PackProductsPrices, extIDs); err != nil {
			return err
		}
		candIDs := make([]string, 0, len(rows)*2)
		for i, ext := range extIDs {
			if id, ok := tx.identities[identityKey(tx.supplierID, PackProductsPrices, ext)]; ok && id != "" {
				candIDs = append(candIDs, id)
			}
			if skus[i] != "" {
				candIDs = append(candIDs, skus[i])
			}
		}
		return tx.preloadProducts(ctx, candIDs, barcodes, names)

	case PackCustomers:
		extIDs := make([]string, 0, len(rows))
		inns := make([]string, 0, len(rows))
		phones := make([]string, 0, len(rows))
		for _, cr := range rows {
			extIDs = append(extIDs, cr.Fields["external_customer_id"])
			inns = append(inns, NormalizeINN(cr.Fields["inn"]))
			phones = append(phones, NormalizePhoneUZ(cr.Fields["phone"]))
		}
		if err := tx.preloadIdentities(ctx, PackCustomers, extIDs); err != nil {
			return err
		}
		retIDs := make([]string, 0, len(rows)*2)
		for _, rawExt := range extIDs {
			ext := strings.TrimSpace(rawExt)
			if ext == "" {
				continue
			}
			if id, ok := tx.identities[identityKey(tx.supplierID, PackCustomers, ext)]; ok && id != "" {
				retIDs = append(retIDs, id)
			}
			retIDs = append(retIDs, ext)
		}
		return tx.preloadCustomers(ctx, homeCell, retIDs, inns, phones, nil)

	case PackStock, PackLots:
		if err := tx.preloadWarehouses(ctx); err != nil {
			return err
		}
		skus := make([]string, 0, len(rows))
		whRefs := make([]string, 0, len(rows))
		for _, cr := range rows {
			skus = append(skus, strings.TrimSpace(cr.Fields["sku"]))
			whRefs = append(whRefs, strings.TrimSpace(cr.Fields["warehouse_id"]))
		}
		if err := tx.preloadIdentities(ctx, PackProductsPrices, skus); err != nil {
			return err
		}
		if err := tx.preloadIdentities(ctx, PackWarehouses, whRefs); err != nil {
			return err
		}
		candProdIDs := make([]string, 0, len(skus)*2)
		for _, sku := range skus {
			if sku == "" {
				continue
			}
			if id, ok := tx.identities[identityKey(tx.supplierID, PackProductsPrices, sku)]; ok && id != "" {
				candProdIDs = append(candProdIDs, id)
			}
			candProdIDs = append(candProdIDs, sku)
		}
		if err := tx.preloadProducts(ctx, candProdIDs, skus, skus); err != nil {
			return err
		}
		pairs := make([]whProdPair, 0, len(rows))
		for _, cr := range rows {
			sku := strings.TrimSpace(cr.Fields["sku"])
			whRef := strings.TrimSpace(cr.Fields["warehouse_id"])
			if sku == "" {
				continue
			}
			prod, pOK := tx.findProductCached(tx.identities[identityKey(tx.supplierID, PackProductsPrices, sku)], sku, sku, sku)
			if !pOK {
				continue
			}
			mappedWH := defaultWarehouseID
			if whRef != "" {
				mappedWH = firstNonEmpty(tx.identities[identityKey(tx.supplierID, PackWarehouses, whRef)], whRef)
			}
			wh, wOK := tx.findWarehouseCached(mappedWH, whRef)
			if !wOK {
				continue
			}
			pairs = append(pairs, whProdPair{whID: wh.WarehouseID, prodID: prod.ProductID})
		}
		if err := tx.preloadStocks(ctx, pairs); err != nil {
			return err
		}
		if pack == PackLots {
			return tx.preloadLots(ctx, pairs)
		}
		return nil

	case PackHistoricalOrders, PackOpenOrders:
		if err := tx.preloadWarehouses(ctx); err != nil {
			return err
		}
		skus := make([]string, 0, len(rows))
		extOrderNos := make([]string, 0, len(rows))
		inns := make([]string, 0, len(rows))
		phones := make([]string, 0, len(rows))
		names := make([]string, 0, len(rows))
		for _, cr := range rows {
			skus = append(skus, strings.TrimSpace(cr.Fields["sku"]))
			extOrderNos = append(extOrderNos, cr.Fields["external_order_no"])
			inns = append(inns, NormalizeINN(cr.Fields["customer_inn"]))
			phones = append(phones, NormalizePhoneUZ(cr.Fields["customer_phone"]))
			names = append(names, cr.Fields["customer_name"])
		}
		if err := tx.preloadIdentities(ctx, PackProductsPrices, skus); err != nil {
			return err
		}
		candProdIDs := make([]string, 0, len(skus)*2)
		for _, sku := range skus {
			if sku == "" {
				continue
			}
			if id, ok := tx.identities[identityKey(tx.supplierID, PackProductsPrices, sku)]; ok && id != "" {
				candProdIDs = append(candProdIDs, id)
			}
			candProdIDs = append(candProdIDs, sku)
		}
		if err := tx.preloadProducts(ctx, candProdIDs, skus, skus); err != nil {
			return err
		}
		if err := tx.preloadCustomers(ctx, homeCell, nil, inns, phones, names); err != nil {
			return err
		}
		return tx.preloadOrders(ctx, extOrderNos)

	case PackNasiyaAR:
		inns := make([]string, 0, len(rows))
		phones := make([]string, 0, len(rows))
		names := make([]string, 0, len(rows))
		extInvIDs := make([]string, 0, len(rows))
		for _, cr := range rows {
			inns = append(inns, NormalizeINN(cr.Fields["customer_inn"]))
			phones = append(phones, NormalizePhoneUZ(cr.Fields["customer_phone"]))
			names = append(names, cr.Fields["customer_name"])
			extInvIDs = append(extInvIDs, firstNonEmpty(
				strings.TrimSpace(cr.Fields["external_invoice_id"]),
				fmt.Sprintf("nasiya_%s_r%d", tx.sessionID, cr.RowIndex),
			))
		}
		if err := tx.preloadCustomers(ctx, homeCell, nil, inns, phones, names); err != nil {
			return err
		}
		retIDs := make([]string, 0, len(rows))
		for i := range rows {
			if id := tx.findCustomerIDCached(inns[i], phones[i], names[i]); id != "" {
				retIDs = append(retIDs, id)
			}
		}
		return ar.PreloadMigrationTxnCacheInTxn(ctx, tx.txn, tx.arCache, tx.supplierID, extInvIDs, retIDs)
	}
	return nil
}

func (tx *spannerApplyTx) preloadIdentities(ctx context.Context, pack Pack, extIDs []string) error {
	uniq := dedupNonEmpty(extIDs, false)
	keys := make([]spanner.Key, 0, len(uniq))
	pending := make([]string, 0, len(uniq))
	for _, ext := range uniq {
		k := identityKey(tx.supplierID, pack, ext)
		if _, ok := tx.identities[k]; ok {
			continue
		}
		if tx.checkedIDs[k] {
			continue
		}
		pending = append(pending, ext)
		keys = append(keys, spanner.Key{tx.supplierID, string(pack), ext})
	}
	if len(keys) == 0 {
		return nil
	}
	iter := tx.txn.Read(ctx, "MigrationIdentityMap", spanner.KeySetFromKeys(keys...), []string{"external_id", "internal_id"})
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iter.Stop()
			return err
		}
		var extID, internalID string
		if err := row.Columns(&extID, &internalID); err != nil {
			iter.Stop()
			return err
		}
		tx.identities[identityKey(tx.supplierID, pack, extID)] = internalID
	}
	iter.Stop()
	for _, ext := range pending {
		tx.checkedIDs[identityKey(tx.supplierID, pack, ext)] = true
	}
	return nil
}

func (tx *spannerApplyTx) preloadWarehouses(ctx context.Context) error {
	if tx.whLoaded {
		return nil
	}
	whIter := tx.txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT WarehouseId, SupplierId, Name, Address, IsActive, IsOnShift FROM Warehouses WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": tx.supplierID},
	})
	for {
		row, err := whIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			whIter.Stop()
			return err
		}
		wh, err := scanStoredWarehouse(row)
		if err != nil {
			whIter.Stop()
			return err
		}
		tx.cacheWarehouse(wh)
	}
	whIter.Stop()

	pmIter := tx.txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT ExternalPlantId, WarehouseId FROM PartnerPlantMaps WHERE TenantType = 'SUPPLIER' AND TenantId = @sid`,
		Params: map[string]any{"sid": tx.supplierID},
	})
	for {
		row, err := pmIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			pmIter.Stop()
			return err
		}
		var extPlantID, whID string
		if err := row.Columns(&extPlantID, &whID); err != nil {
			pmIter.Stop()
			return err
		}
		if strings.TrimSpace(extPlantID) != "" && whID != "" {
			tx.plantMaps[plantMapKey(tx.supplierID, extPlantID)] = whID
		}
	}
	pmIter.Stop()
	tx.whLoaded = true
	return nil
}

func (tx *spannerApplyTx) preloadProducts(ctx context.Context, prodIDs, barcodes, names []string) error {
	uniqIDs := dedupNonEmpty(prodIDs, false)
	idKeys := make([]spanner.Key, 0, len(uniqIDs))
	pendingIDs := make([]string, 0, len(uniqIDs))
	for _, id := range uniqIDs {
		if _, ok := tx.products[id]; ok {
			continue
		}
		if tx.checkedProdIDs[id] {
			continue
		}
		pendingIDs = append(pendingIDs, id)
		idKeys = append(idKeys, spanner.Key{id})
	}
	if len(idKeys) > 0 {
		iter := tx.txn.Read(ctx, "Products", spanner.KeySetFromKeys(idKeys...), productColsSlice)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			p, err := scanStoredProduct(row)
			if err != nil {
				iter.Stop()
				return err
			}
			if p.SupplierID == tx.supplierID {
				tx.cacheProduct(p)
			}
		}
		iter.Stop()
		for _, id := range pendingIDs {
			tx.checkedProdIDs[id] = true
		}
	}

	uniqBC := dedupNonEmpty(barcodes, false)
	pendingBC := make([]string, 0, len(uniqBC))
	for _, bc := range uniqBC {
		if _, ok := tx.prodByBarcode[strings.ToLower(bc)]; ok {
			continue
		}
		if tx.checkedProdBC[bc] {
			continue
		}
		pendingBC = append(pendingBC, bc)
	}
	if len(pendingBC) > 0 {
		iter := tx.txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT ` + productSelectCols + ` FROM Products WHERE SupplierId = @sid AND Barcode IN UNNEST(@bcs)`,
			Params: map[string]any{"sid": tx.supplierID, "bcs": pendingBC},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			p, err := scanStoredProduct(row)
			if err != nil {
				iter.Stop()
				return err
			}
			tx.cacheProduct(p)
		}
		iter.Stop()
		for _, bc := range pendingBC {
			tx.checkedProdBC[bc] = true
		}
	}

	uniqNames := dedupNonEmpty(names, true)
	pendingNames := make([]string, 0, len(uniqNames))
	for _, nm := range uniqNames {
		if _, ok := tx.prodByName[nm]; ok {
			continue
		}
		if tx.checkedProdName[nm] {
			continue
		}
		pendingNames = append(pendingNames, nm)
	}
	if len(pendingNames) > 0 {
		iter := tx.txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT ` + productSelectCols + ` FROM Products WHERE SupplierId = @sid AND LOWER(Name) IN UNNEST(@nms)`,
			Params: map[string]any{"sid": tx.supplierID, "nms": pendingNames},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			p, err := scanStoredProduct(row)
			if err != nil {
				iter.Stop()
				return err
			}
			tx.cacheProduct(p)
		}
		iter.Stop()
		for _, nm := range pendingNames {
			tx.checkedProdName[nm] = true
		}
	}
	return nil
}

func (tx *spannerApplyTx) preloadSupplierLinksBatch(ctx context.Context, retIDs []string, links map[string]StoredCustomerLink) error {
	uniq := dedupNonEmpty(retIDs, false)
	keys := make([]spanner.Key, 0, len(uniq))
	for _, rid := range uniq {
		if _, ok := links[rid]; ok {
			continue
		}
		keys = append(keys, spanner.Key{tx.supplierID, rid})
	}
	if len(keys) == 0 {
		return nil
	}
	iter := tx.txn.Read(ctx, "SupplierRetailerLinks", spanner.KeySetFromKeys(keys...),
		[]string{"SupplierId", "RetailerId", "TaxIdTIN", "NormalizedPhone", "LegalName", "DedupClusterId", "DedupStatus"})
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iter.Stop()
			return err
		}
		var link StoredCustomerLink
		var tin, phone, legal, cluster spanner.NullString
		if err := row.Columns(&link.SupplierID, &link.RetailerID, &tin, &phone, &legal, &cluster, &link.DedupStatus); err != nil {
			iter.Stop()
			return err
		}
		link.TaxIdTIN = tin.StringVal
		link.NormalizedPhone = phone.StringVal
		link.LegalName = legal.StringVal
		link.DedupClusterID = cluster.StringVal
		links[link.RetailerID] = link
	}
	iter.Stop()
	return nil
}

func (tx *spannerApplyTx) preloadCustomers(ctx context.Context, homeCell string, retIDs, inns, phones, legalNames []string) error {
	retByID := make(map[string]StoredCustomer)
	sameSupLinks := make(map[string]StoredCustomerLink)

	uniqIDs := dedupNonEmpty(retIDs, false)
	idKeys := make([]spanner.Key, 0, len(uniqIDs))
	pendingIDs := make([]string, 0, len(uniqIDs))
	for _, id := range uniqIDs {
		if _, ok := tx.customers[id]; ok {
			continue
		}
		if tx.checkedCustIDs[id] {
			continue
		}
		pendingIDs = append(pendingIDs, id)
		idKeys = append(idKeys, spanner.Key{id})
	}
	if len(idKeys) > 0 {
		iter := tx.txn.Read(ctx, "Retailers", spanner.KeySetFromKeys(idKeys...),
			[]string{"RetailerId", "Phone", "Name", "CountryCode", "DeliveryAddress"})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			var ret StoredCustomer
			var name, addr spanner.NullString
			if err := row.Columns(&ret.RetailerID, &ret.Phone, &name, &ret.CountryCode, &addr); err != nil {
				iter.Stop()
				return err
			}
			ret.Name = name.StringVal
			ret.DeliveryAddr = addr.StringVal
			retByID[ret.RetailerID] = ret
		}
		iter.Stop()
		for _, id := range pendingIDs {
			tx.checkedCustIDs[id] = true
		}
	}

	uniqINNs := dedupNonEmpty(inns, false)
	pendingINNs := make([]string, 0, len(uniqINNs))
	for _, inn := range uniqINNs {
		if _, ok := tx.custByINN[inn]; ok {
			continue
		}
		if tx.checkedCustINNs[inn] {
			continue
		}
		pendingINNs = append(pendingINNs, inn)
	}
	innMatchIDs := make(map[string]string)
	if len(pendingINNs) > 0 {
		iter := tx.txn.Query(ctx, spanner.Statement{
			SQL: `SELECT l.RetailerId, (l.SupplierId = @sid) AS same_sup,
			             r.Phone, COALESCE(r.Name, ''), r.CountryCode, COALESCE(r.DeliveryAddress, ''),
			             COALESCE(l.TaxIdTIN, ''), COALESCE(l.NormalizedPhone, ''), COALESCE(l.LegalName, ''),
			             COALESCE(l.DedupClusterId, ''), COALESCE(l.DedupStatus, 'CLEAN')
			      FROM SupplierRetailerLinks l
			      JOIN Retailers r ON r.RetailerId = l.RetailerId
			      WHERE l.TaxIdTIN IN UNNEST(@inns)
			        AND (@homeCell = '' OR COALESCE(r.HomeCell, r.H3Cell, '') = @homeCell OR COALESCE(r.HomeCell, r.H3Cell, '') = '' OR l.SupplierId = @sid)
			      ORDER BY same_sup DESC`,
			Params: map[string]any{"sid": tx.supplierID, "inns": pendingINNs, "homeCell": homeCell},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			var ret StoredCustomer
			var sameSup bool
			var tin, normPh, legal, cluster, dedupSt string
			if err := row.Columns(&ret.RetailerID, &sameSup, &ret.Phone, &ret.Name, &ret.CountryCode, &ret.DeliveryAddr,
				&tin, &normPh, &legal, &cluster, &dedupSt); err != nil {
				iter.Stop()
				return err
			}
			retByID[ret.RetailerID] = ret
			if sameSup {
				sameSupLinks[ret.RetailerID] = StoredCustomerLink{
					SupplierID:      tx.supplierID,
					RetailerID:      ret.RetailerID,
					TaxIdTIN:        tin,
					NormalizedPhone: normPh,
					LegalName:       legal,
					DedupClusterID:  cluster,
					DedupStatus:     dedupSt,
				}
			}
			if normTIN := NormalizeINN(tin); normTIN != "" {
				if _, exists := innMatchIDs[normTIN]; !exists || sameSup {
					innMatchIDs[normTIN] = ret.RetailerID
				}
			}
		}
		iter.Stop()
		for _, inn := range pendingINNs {
			tx.checkedCustINNs[inn] = true
		}
	}

	uniqPhones := dedupNonEmpty(phones, false)
	pendingPhones := make([]string, 0, len(uniqPhones))
	for _, ph := range uniqPhones {
		if _, ok := tx.custByPhone[ph]; ok {
			continue
		}
		if tx.checkedCustPh[ph] {
			continue
		}
		pendingPhones = append(pendingPhones, ph)
	}
	if len(pendingPhones) > 0 {
		iter := tx.txn.Query(ctx, spanner.Statement{
			SQL: `SELECT RetailerId, Phone, COALESCE(Name, ''), CountryCode, COALESCE(DeliveryAddress, '')
			      FROM Retailers WHERE Phone IN UNNEST(@phones)`,
			Params: map[string]any{"phones": pendingPhones},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			var ret StoredCustomer
			if err := row.Columns(&ret.RetailerID, &ret.Phone, &ret.Name, &ret.CountryCode, &ret.DeliveryAddr); err != nil {
				iter.Stop()
				return err
			}
			retByID[ret.RetailerID] = ret
		}
		iter.Stop()
		for _, ph := range pendingPhones {
			tx.checkedCustPh[ph] = true
		}
	}

	uniqNames := dedupNonEmpty(legalNames, true)
	pendingNames := make([]string, 0, len(uniqNames))
	for _, nm := range uniqNames {
		if _, ok := tx.custByLegalName[nm]; ok {
			continue
		}
		if tx.checkedCustName[nm] {
			continue
		}
		pendingNames = append(pendingNames, nm)
	}
	missingNameRetIDs := make([]string, 0)
	if len(pendingNames) > 0 {
		iter := tx.txn.Query(ctx, spanner.Statement{
			SQL: `SELECT RetailerId, COALESCE(TaxIdTIN, ''), COALESCE(NormalizedPhone, ''), COALESCE(LegalName, ''),
			             COALESCE(DedupClusterId, ''), COALESCE(DedupStatus, 'CLEAN')
			      FROM SupplierRetailerLinks
			      WHERE SupplierId = @sid AND LOWER(COALESCE(LegalName, '')) IN UNNEST(@nms)`,
			Params: map[string]any{"sid": tx.supplierID, "nms": pendingNames},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			var link StoredCustomerLink
			link.SupplierID = tx.supplierID
			if err := row.Columns(&link.RetailerID, &link.TaxIdTIN, &link.NormalizedPhone, &link.LegalName, &link.DedupClusterID, &link.DedupStatus); err != nil {
				iter.Stop()
				return err
			}
			sameSupLinks[link.RetailerID] = link
			if _, ok := retByID[link.RetailerID]; !ok {
				if _, cached := tx.customers[link.RetailerID]; !cached {
					missingNameRetIDs = append(missingNameRetIDs, link.RetailerID)
				}
			}
		}
		iter.Stop()
		for _, nm := range pendingNames {
			tx.checkedCustName[nm] = true
		}
	}
	if len(missingNameRetIDs) > 0 {
		uniqMissing := dedupNonEmpty(missingNameRetIDs, false)
		keys := make([]spanner.Key, 0, len(uniqMissing))
		for _, id := range uniqMissing {
			keys = append(keys, spanner.Key{id})
		}
		iter := tx.txn.Read(ctx, "Retailers", spanner.KeySetFromKeys(keys...),
			[]string{"RetailerId", "Phone", "Name", "CountryCode", "DeliveryAddress"})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			var ret StoredCustomer
			var name, addr spanner.NullString
			if err := row.Columns(&ret.RetailerID, &ret.Phone, &name, &ret.CountryCode, &addr); err != nil {
				iter.Stop()
				return err
			}
			ret.Name = name.StringVal
			ret.DeliveryAddr = addr.StringVal
			retByID[ret.RetailerID] = ret
		}
		iter.Stop()
	}

	allFoundRetIDs := make([]string, 0, len(retByID))
	for rid := range retByID {
		allFoundRetIDs = append(allFoundRetIDs, rid)
	}
	if err := tx.preloadSupplierLinksBatch(ctx, allFoundRetIDs, sameSupLinks); err != nil {
		return err
	}
	for rid, ret := range retByID {
		if link, ok := sameSupLinks[rid]; ok {
			ret.Link = link
		}
		tx.cacheCustomer(ret)
	}
	for normTIN, rid := range innMatchIDs {
		if _, exists := tx.custByINN[normTIN]; !exists {
			tx.custByINN[normTIN] = rid
		}
	}
	return nil
}

func (tx *spannerApplyTx) preloadStocks(ctx context.Context, pairs []whProdPair) error {
	seen := make(map[string]bool, len(pairs))
	keys := make([]spanner.Key, 0, len(pairs))
	pending := make([]whProdPair, 0, len(pairs))
	for _, p := range pairs {
		if p.whID == "" || p.prodID == "" {
			continue
		}
		k := inventoryKey(tx.supplierID, p.whID, p.prodID)
		if seen[k] {
			continue
		}
		seen[k] = true
		if _, ok := tx.stocks[k]; ok {
			continue
		}
		if tx.checkedStocks[k] {
			continue
		}
		pending = append(pending, p)
		keys = append(keys, spanner.Key{tx.supplierID, p.whID, p.prodID})
	}
	if len(keys) == 0 {
		return nil
	}
	iter := tx.txn.Read(ctx, "SupplierInventoryV2", spanner.KeySetFromKeys(keys...),
		[]string{"WarehouseId", "ProductId", "QuantityOnHand", "QuantityReserved", "ReorderThreshold"})
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iter.Stop()
			return err
		}
		stk := StoredStock{SupplierID: tx.supplierID}
		if err := row.Columns(&stk.WarehouseID, &stk.ProductID, &stk.QuantityOnHand, &stk.QuantityReserved, &stk.ReorderThreshold); err != nil {
			iter.Stop()
			return err
		}
		tx.stocks[inventoryKey(tx.supplierID, stk.WarehouseID, stk.ProductID)] = stk
	}
	iter.Stop()
	for _, p := range pending {
		tx.checkedStocks[inventoryKey(tx.supplierID, p.whID, p.prodID)] = true
	}
	return nil
}

func (tx *spannerApplyTx) preloadLots(ctx context.Context, pairs []whProdPair) error {
	pids := make([]string, 0, len(pairs))
	pendingPairs := make([]whProdPair, 0, len(pairs))
	seenPair := make(map[string]bool, len(pairs))
	for _, p := range pairs {
		if p.whID == "" || p.prodID == "" {
			continue
		}
		pk := p.whID + "|" + p.prodID
		if seenPair[pk] || tx.checkedLotPairs[pk] {
			continue
		}
		seenPair[pk] = true
		pendingPairs = append(pendingPairs, p)
		pids = append(pids, p.prodID)
	}
	uniqPIDs := dedupNonEmpty(pids, false)
	if len(uniqPIDs) == 0 {
		return nil
	}
	iter := tx.txn.Query(ctx, spanner.Statement{
		SQL: `SELECT LotId, SupplierId, WarehouseId, ProductId, LocationId, LotCode, ExpiryDate, ManufacturedDate, QuantityOnHand, QuantityReserved, Status
		      FROM StockLots WHERE SupplierId = @sid AND ProductId IN UNNEST(@pids)`,
		Params: map[string]any{"sid": tx.supplierID, "pids": uniqPIDs},
	})
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iter.Stop()
			return err
		}
		lot, err := scanStoredLot(row)
		if err != nil {
			iter.Stop()
			return err
		}
		lk := strings.ToLower(strings.TrimSpace(lot.WarehouseID + "|" + lot.ProductID + "|" + lot.LotCode))
		tx.lotsByID[lot.LotID] = lot
		tx.lotsByKey[lk] = lot.LotID
	}
	iter.Stop()
	for _, p := range pendingPairs {
		tx.checkedLotPairs[p.whID+"|"+p.prodID] = true
	}
	return nil
}

func (tx *spannerApplyTx) preloadOrders(ctx context.Context, extNos []string) error {
	uniq := dedupNonEmpty(extNos, false)
	pending := make([]string, 0, len(uniq))
	for _, eno := range uniq {
		key := strings.ToLower(eno)
		if _, ok := tx.ordersByExt[key]; ok {
			continue
		}
		if tx.checkedOrderExt[key] {
			continue
		}
		pending = append(pending, eno)
	}
	if len(pending) == 0 {
		return nil
	}
	iter := tx.txn.Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, ExternalOrderNo, Status, OrderSource, LineItemsJson, TotalMinor, OriginalTotalMinor, Version
		      FROM Orders WHERE SupplierId = @sid AND ExternalOrderNo IN UNNEST(@enos)`,
		Params: map[string]any{"sid": tx.supplierID, "enos": pending},
	})
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iter.Stop()
			return err
		}
		var ord order.Order
		var eno spanner.NullString
		var st, src string
		var lineRaw []byte
		if err := row.Columns(&ord.OrderID, &eno, &st, &src, &lineRaw, &ord.TotalMinor, &ord.OriginalTotalMinor, &ord.Version); err != nil {
			iter.Stop()
			return err
		}
		ord.SupplierID = tx.supplierID
		ord.ExternalOrderNo = strings.TrimSpace(eno.StringVal)
		ord.Status = order.Status(st)
		ord.Source = order.OrderSource(src)
		if len(lineRaw) > 0 {
			if err := json.Unmarshal(lineRaw, &ord.LineItems); err != nil {
				iter.Stop()
				return err
			}
		}
		tx.ordersByID[ord.OrderID] = ord
		if ord.ExternalOrderNo != "" {
			tx.ordersByExt[strings.ToLower(ord.ExternalOrderNo)] = ord.OrderID
		}
	}
	iter.Stop()
	for _, eno := range pending {
		tx.checkedOrderExt[strings.ToLower(eno)] = true
	}
	return nil
}

func (tx *spannerApplyTx) findWarehouseCached(extID, name string) (StoredWarehouse, bool) {
	if ext := strings.TrimSpace(extID); ext != "" {
		if wh, ok := tx.warehouses[ext]; ok {
			return wh, true
		}
		if mapped, ok := tx.plantMaps[plantMapKey(tx.supplierID, ext)]; ok {
			if wh, ok2 := tx.warehouses[mapped]; ok2 {
				return wh, true
			}
		}
	}
	if wantName := strings.ToLower(strings.TrimSpace(name)); wantName != "" {
		if id, ok := tx.whByName[wantName]; ok {
			if wh, ok2 := tx.warehouses[id]; ok2 {
				return wh, true
			}
		}
	}
	return StoredWarehouse{}, false
}

func (tx *spannerApplyTx) findProductCached(internalID, sku, barcode, name string) (StoredProduct, bool) {
	for _, cand := range []string{strings.TrimSpace(internalID), strings.TrimSpace(sku)} {
		if cand == "" {
			continue
		}
		if p, ok := tx.products[cand]; ok && p.SupplierID == tx.supplierID {
			return p, true
		}
	}
	if bc := strings.ToLower(strings.TrimSpace(barcode)); bc != "" {
		if id, ok := tx.prodByBarcode[bc]; ok {
			if p, ok2 := tx.products[id]; ok2 && p.SupplierID == tx.supplierID {
				return p, true
			}
		}
	}
	if nm := strings.ToLower(strings.TrimSpace(name)); nm != "" {
		if id, ok := tx.prodByName[nm]; ok {
			if p, ok2 := tx.products[id]; ok2 && p.SupplierID == tx.supplierID {
				return p, true
			}
		}
	}
	return StoredProduct{}, false
}

func (tx *spannerApplyTx) findCustomerIDCached(inn, phone, name string) string {
	if normINN := NormalizeINN(inn); normINN != "" {
		if id, ok := tx.custByINN[normINN]; ok {
			return id
		}
	}
	if normPhone := NormalizePhoneUZ(phone); normPhone != "" {
		if id, ok := tx.custByPhone[normPhone]; ok {
			return id
		}
	}
	if nm := strings.ToLower(strings.TrimSpace(name)); nm != "" {
		if id, ok := tx.custByLegalName[nm]; ok {
			return id
		}
	}
	return ""
}
