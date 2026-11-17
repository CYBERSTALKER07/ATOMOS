package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

// IdentityMappingEntry captures an external-to-internal entity ID mapping stored in MemoryApplyStore.
type IdentityMappingEntry struct {
	SupplierID       string `json:"supplier_id"`
	Pack             Pack   `json:"pack"`
	ExternalID       string `json:"external_id"`
	InternalID       string `json:"internal_id"`
	CreatedBySession string `json:"created_by_session"`
}

// MemorySupplierMeta captures supplier-level defaults needed during dry-run and chunk apply.
type MemorySupplierMeta struct {
	HomeCell string
	Currency string
}

// MemoryApplyStore provides a transactional, snapshot-capturing in-memory EntityApplyStore and DryRunLiveOverlayReader.
// Chunk failures roll back atomically without leaving partial mutations in memory.
type MemoryApplyStore struct {
	mu             sync.Mutex
	NowFn          func() time.Time
	Supplier       map[string]MemorySupplierMeta
	Warehouses     map[string]StoredWarehouse
	PlantMaps      map[string]string // supplierID|lower(extPlantID) -> warehouseID
	Products       map[string]StoredProduct
	Retailers      map[string]StoredCustomer
	Links          map[string]StoredCustomerLink // supplierID|retailerID
	Inventory      map[string]StoredStock        // supplierID|warehouseID|productID
	Lots           map[string]StoredLot          // lotID
	Locations      map[string]bool               // warehouseID|locationID
	Orders         map[string]order.Order
	ARRepo         *ar.MemoryRepository
	CreditProfiles map[string]ar.CreditProfileState // supplierID|retailerID
	JournalEntries map[string]*payment.JournalEntry // entryID
	IdentityMap    map[string]IdentityMappingEntry
	Snapshots      map[string][]MigrationSnapshotRecord
	SnapshotSeq    map[string]int64
}

// NewMemoryApplyStore initializes an empty transactional in-memory entity store.
func NewMemoryApplyStore() *MemoryApplyStore {
	return &MemoryApplyStore{
		Supplier:       make(map[string]MemorySupplierMeta),
		Warehouses:     make(map[string]StoredWarehouse),
		PlantMaps:      make(map[string]string),
		Products:       make(map[string]StoredProduct),
		Retailers:      make(map[string]StoredCustomer),
		Links:          make(map[string]StoredCustomerLink),
		Inventory:      make(map[string]StoredStock),
		Lots:           make(map[string]StoredLot),
		Locations:      make(map[string]bool),
		Orders:         make(map[string]order.Order),
		ARRepo:         ar.NewMemoryRepository(),
		CreditProfiles: make(map[string]ar.CreditProfileState),
		JournalEntries: make(map[string]*payment.JournalEntry),
		IdentityMap:    make(map[string]IdentityMappingEntry),
		Snapshots:      make(map[string][]MigrationSnapshotRecord),
		SnapshotSeq:    make(map[string]int64),
	}
}

func (s *MemoryApplyStore) ensureInitLocked() {
	if s.Supplier == nil {
		s.Supplier = make(map[string]MemorySupplierMeta)
	}
	if s.Warehouses == nil {
		s.Warehouses = make(map[string]StoredWarehouse)
	}
	if s.PlantMaps == nil {
		s.PlantMaps = make(map[string]string)
	}
	if s.Products == nil {
		s.Products = make(map[string]StoredProduct)
	}
	if s.Retailers == nil {
		s.Retailers = make(map[string]StoredCustomer)
	}
	if s.Links == nil {
		s.Links = make(map[string]StoredCustomerLink)
	}
	if s.Inventory == nil {
		s.Inventory = make(map[string]StoredStock)
	}
	if s.Lots == nil {
		s.Lots = make(map[string]StoredLot)
	}
	if s.Locations == nil {
		s.Locations = make(map[string]bool)
	}
	if s.Orders == nil {
		s.Orders = make(map[string]order.Order)
	}
	if s.ARRepo == nil {
		s.ARRepo = ar.NewMemoryRepository()
	}
	if s.CreditProfiles == nil {
		s.CreditProfiles = make(map[string]ar.CreditProfileState)
	}
	if s.JournalEntries == nil {
		s.JournalEntries = make(map[string]*payment.JournalEntry)
	}
	if s.IdentityMap == nil {
		s.IdentityMap = make(map[string]IdentityMappingEntry)
	}
	if s.Snapshots == nil {
		s.Snapshots = make(map[string][]MigrationSnapshotRecord)
	}
	if s.SnapshotSeq == nil {
		s.SnapshotSeq = make(map[string]int64)
	}
}

func (s *MemoryApplyStore) nowUTC() time.Time {
	if s != nil && s.NowFn != nil {
		return s.NowFn().UTC()
	}
	return time.Now().UTC()
}

// LoadLiveOverlay implements DryRunLiveOverlayReader over current live in-memory business entities.
func (s *MemoryApplyStore) LoadLiveOverlay(_ context.Context, supplierID string) (LiveOverlaySnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := NewEmptyLiveOverlay()
	if meta, ok := s.Supplier[supplierID]; ok {
		if strings.TrimSpace(meta.Currency) != "" {
			out.SupplierCurrency = strings.ToUpper(strings.TrimSpace(meta.Currency))
		}
		out.HomeCell = meta.HomeCell
	}

	whIDs := make([]string, 0, len(s.Warehouses))
	for id, wh := range s.Warehouses {
		if wh.SupplierID == supplierID {
			whIDs = append(whIDs, id)
		}
	}
	sort.Strings(whIDs)
	for _, id := range whIDs {
		wh := s.Warehouses[id]
		lw := LiveWarehouse{WarehouseID: wh.WarehouseID, Name: wh.Name}
		out.WarehousesByExt[strings.ToLower(wh.WarehouseID)] = lw
		out.WarehousesByName[strings.ToLower(strings.TrimSpace(wh.Name))] = lw
		if out.DefaultWarehouseID == "" {
			out.DefaultWarehouseID = wh.WarehouseID
		}
	}
	for pk, whID := range s.PlantMaps {
		prefix := supplierID + "|"
		if !strings.HasPrefix(pk, prefix) {
			continue
		}
		extPlant := strings.TrimPrefix(pk, prefix)
		if wh, ok := s.Warehouses[whID]; ok {
			out.WarehousesByExt[extPlant] = LiveWarehouse{WarehouseID: wh.WarehouseID, ExternalID: extPlant, Name: wh.Name}
		}
	}

	for _, p := range s.Products {
		if p.SupplierID != supplierID {
			continue
		}
		lp := LiveProduct{
			ProductID:         p.ProductID,
			SKU:               p.ProductID,
			Barcode:           p.Barcode,
			Name:              p.Name,
			Unit:              p.Unit,
			PriceMinor:        p.PriceMinor,
			Currency:          p.Currency,
			RequiresColdChain: p.RequiresColdChain,
			IsPerishable:      p.IsPerishable,
		}
		out.ProductsByID[strings.ToLower(p.ProductID)] = lp
		out.ProductsBySKU[strings.ToLower(p.ProductID)] = lp
		if b := strings.ToLower(strings.TrimSpace(p.Barcode)); b != "" {
			out.ProductsByBarcode[b] = lp
		}
		if n := strings.ToLower(strings.TrimSpace(p.Name)); n != "" {
			out.ProductsByName[n] = lp
		}
	}

	for _, im := range s.IdentityMap {
		if im.SupplierID != supplierID {
			continue
		}
		switch im.Pack {
		case PackWarehouses:
			if wh, ok := s.Warehouses[im.InternalID]; ok {
				out.WarehousesByExt[strings.ToLower(im.ExternalID)] = LiveWarehouse{
					WarehouseID: wh.WarehouseID, ExternalID: im.ExternalID, Name: wh.Name,
				}
			}
		case PackProductsPrices:
			if lp, ok := out.ProductsByID[strings.ToLower(im.InternalID)]; ok {
				lp.SKU = im.ExternalID
				out.ProductsBySKU[strings.ToLower(im.ExternalID)] = lp
			}
		}
	}

	for _, ret := range s.Retailers {
		lc := LiveCustomer{
			RetailerID:   ret.RetailerID,
			Phone:        ret.Phone,
			CustomerName: ret.Name,
		}
		if link, ok := s.Links[supplierLinkKey(supplierID, ret.RetailerID)]; ok {
			lc.INN = link.TaxIdTIN
			lc.CustomerName = firstNonEmpty(link.LegalName, ret.Name)
		}
		if lc.INN == "" && out.HomeCell != "" && ret.DeliveryAddr != "" {
			// HomeCell matching checks supplier links first; also index phone/name for conflict checks.
		}
		if ph := NormalizePhoneUZ(lc.Phone); ph != "" {
			out.CustomersByPhone[ph] = lc
		}
		if lc.INN != "" {
			out.CustomersByINN[lc.INN] = lc
		}
		if nm := strings.ToLower(strings.TrimSpace(lc.CustomerName)); nm != "" {
			out.CustomersByName[nm] = lc
		}
	}

	for _, stk := range s.Inventory {
		if stk.SupplierID != supplierID {
			continue
		}
		unit := ""
		if p, ok := s.Products[stk.ProductID]; ok {
			unit = p.Unit
		}
		out.StockByWHProduct[stk.WarehouseID+"|"+stk.ProductID] = LiveStock{
			WarehouseID: stk.WarehouseID,
			ProductID:   stk.ProductID,
			OnHand:      stk.QuantityOnHand,
			Reserved:    stk.QuantityReserved,
			Unit:        unit,
		}
		out.StockTotalByProd[stk.ProductID] += stk.QuantityOnHand
	}

	for _, lot := range s.Lots {
		if lot.SupplierID != supplierID {
			continue
		}
		var exp time.Time
		if lot.ExpiryDate != nil {
			exp = *lot.ExpiryDate
		}
		out.LotsByKey[lot.WarehouseID+"|"+lot.ProductID+"|"+strings.ToLower(lot.LotCode)] = LiveLot{
			LotID:       lot.LotID,
			WarehouseID: lot.WarehouseID,
			ProductID:   lot.ProductID,
			LotCode:     lot.LotCode,
			ExpiryDate:  exp,
			OnHand:      lot.QuantityOnHand,
			Reserved:    lot.QuantityReserved,
		}
	}

	for _, o := range s.Orders {
		if o.SupplierID != supplierID || o.SupersededByRollback || strings.TrimSpace(o.ExternalOrderNo) == "" {
			continue
		}
		out.OrdersByExtNo[strings.ToLower(strings.TrimSpace(o.ExternalOrderNo))] = LiveOrder{
			OrderID:         o.OrderID,
			ExternalOrderNo: o.ExternalOrderNo,
			OrderSource:     string(o.Source),
			Status:          string(o.Status),
		}
	}

	if s.ARRepo != nil {
		for _, inv := range s.ARRepo.ExportInvoices() {
			if inv.SupplierID != supplierID || strings.TrimSpace(inv.ExternalID) == "" {
				continue
			}
			out.ArInvoicesByExt[strings.ToLower(strings.TrimSpace(inv.ExternalID))] = LiveARInvoice{
				InvoiceID:    inv.InvoiceID,
				ExternalID:   inv.ExternalID,
				BalanceMinor: inv.BalanceMinor,
				Currency:     inv.Currency,
				Status:       inv.Status,
			}
		}
	}

	return out, nil
}

// RunChunkTx executes fn inside an atomic staged clone of MemoryApplyStore; on any error all chunk mutations discard.
func (s *MemoryApplyStore) RunChunkTx(_ context.Context, supplierID, sessionID string, fn func(tx ApplyTx) error) error {
	s.mu.Lock()
	s.ensureInitLocked()
	stagedAR := ar.NewMemoryRepository()
	stagedAR.ImportInvoices(s.ARRepo.ExportInvoices())
	ledgerEntries, ledgerKeys := s.ARRepo.ExportLedger()
	stagedAR.ImportLedger(ledgerEntries, ledgerKeys)

	staged := memoryTxState{
		supplier:       cloneMap(s.Supplier),
		warehouses:     cloneMap(s.Warehouses),
		plantMaps:      cloneMap(s.PlantMaps),
		products:       cloneMap(s.Products),
		retailers:      cloneMap(s.Retailers),
		links:          cloneMap(s.Links),
		inventory:      cloneMap(s.Inventory),
		lots:           cloneMap(s.Lots),
		locations:      cloneMap(s.Locations),
		orders:         cloneMap(s.Orders),
		arRepo:         stagedAR,
		creditProfiles: cloneMap(s.CreditProfiles),
		journalEntries: cloneMap(s.JournalEntries),
		identityMap:    cloneMap(s.IdentityMap),
	}
	seq := s.SnapshotSeq[sessionID]
	now := s.nowUTC()
	s.mu.Unlock()

	tx := &memoryApplyTx{
		supplierID:   supplierID,
		sessionID:    sessionID,
		now:          now,
		seq:          seq,
		state:        staged,
		snapRecorded: make(map[string]bool),
	}
	if err := fn(tx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.Supplier = tx.state.supplier
	s.Warehouses = tx.state.warehouses
	s.PlantMaps = tx.state.plantMaps
	s.Products = tx.state.products
	s.Retailers = tx.state.retailers
	s.Links = tx.state.links
	s.Inventory = tx.state.inventory
	s.Lots = tx.state.lots
	s.Locations = tx.state.locations
	s.Orders = tx.state.orders
	s.ARRepo.ImportInvoices(tx.state.arRepo.ExportInvoices())
	comEntries, comKeys := tx.state.arRepo.ExportLedger()
	s.ARRepo.ImportLedger(comEntries, comKeys)
	s.CreditProfiles = tx.state.creditProfiles
	s.JournalEntries = tx.state.journalEntries
	s.IdentityMap = tx.state.identityMap
	s.Snapshots[sessionID] = append(s.Snapshots[sessionID], tx.snapshots...)
	s.SnapshotSeq[sessionID] = tx.seq
	return nil
}

// ListSnapshotsForSession returns committed snapshots for sessionID ordered newest-first (LIFO rollback order).
func (s *MemoryApplyStore) ListSnapshotsForSession(sessionID string) []MigrationSnapshotRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := append([]MigrationSnapshotRecord(nil), s.Snapshots[sessionID]...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].SnapshotSeq > cp[j].SnapshotSeq })
	return cp
}

// ExecuteRollbackInMemory applies LIFO snapshots to MemoryApplyStore and removes session identity mappings.
func (s *MemoryApplyStore) ExecuteRollbackInMemory(supplierID, sessionID string) (RollbackResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snaps := append([]MigrationSnapshotRecord(nil), s.Snapshots[sessionID]...)
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].SnapshotSeq > snaps[j].SnapshotSeq })

	var res RollbackResult
	for _, snap := range snaps {
		res.RevertedSnapshots++
		switch snap.TargetTable {
		case "Warehouses":
			id, _ := extractPKField(snap.RowKeyJSON, "WarehouseId")
			if snap.OpKind == SnapshotOpInsert {
				delete(s.Warehouses, id)
			} else if len(snap.BeforeJSON) > 0 {
				var row StoredWarehouse
				if err := json.Unmarshal(snap.BeforeJSON, &row); err != nil {
					return RollbackResult{}, err
				}
				s.Warehouses[id] = row
			}
		case "PartnerPlantMaps":
			sid, _ := extractPKField(snap.RowKeyJSON, "TenantId")
			ext, _ := extractPKField(snap.RowKeyJSON, "ExternalPlantId")
			pk := plantMapKey(sid, ext)
			if snap.OpKind == SnapshotOpInsert {
				delete(s.PlantMaps, pk)
			} else if len(snap.BeforeJSON) > 0 {
				var m map[string]string
				if err := json.Unmarshal(snap.BeforeJSON, &m); err != nil {
					return RollbackResult{}, err
				}
				s.PlantMaps[pk] = m["WarehouseId"]
			}
		case "Products":
			id, _ := extractPKField(snap.RowKeyJSON, "ProductId")
			if snap.OpKind == SnapshotOpInsert {
				delete(s.Products, id)
			} else if len(snap.BeforeJSON) > 0 {
				var row StoredProduct
				if err := json.Unmarshal(snap.BeforeJSON, &row); err != nil {
					return RollbackResult{}, err
				}
				s.Products[id] = row
			}
		case "SupplierRetailerLinks":
			sid, _ := extractPKField(snap.RowKeyJSON, "SupplierId")
			rid, _ := extractPKField(snap.RowKeyJSON, "RetailerId")
			lk := supplierLinkKey(sid, rid)
			if snap.OpKind == SnapshotOpInsert {
				delete(s.Links, lk)
			} else if len(snap.BeforeJSON) > 0 {
				var row StoredCustomerLink
				if err := json.Unmarshal(snap.BeforeJSON, &row); err != nil {
					return RollbackResult{}, err
				}
				s.Links[lk] = row
			}
		case "SupplierInventoryV2":
			sid, _ := extractPKField(snap.RowKeyJSON, "SupplierId")
			whID, _ := extractPKField(snap.RowKeyJSON, "WarehouseId")
			prodID, _ := extractPKField(snap.RowKeyJSON, "ProductId")
			sk := inventoryKey(sid, whID, prodID)
			if snap.OpKind == SnapshotOpInsert {
				delete(s.Inventory, sk)
			} else if len(snap.BeforeJSON) > 0 {
				var row StoredStock
				if err := json.Unmarshal(snap.BeforeJSON, &row); err != nil {
					return RollbackResult{}, err
				}
				s.Inventory[sk] = row
			}
		case "StockLots":
			id, _ := extractPKField(snap.RowKeyJSON, "LotId")
			if snap.OpKind == SnapshotOpInsert {
				delete(s.Lots, id)
			} else if len(snap.BeforeJSON) > 0 {
				var raw map[string]any
				if err := decodeJSONWithNumbersMap(snap.BeforeJSON, &raw); err != nil {
					return RollbackResult{}, err
				}
				s.Lots[id] = storedLotFromMap(raw)
			}
		case "Orders":
			oid, _ := extractPKField(snap.RowKeyJSON, "OrderId")
			if snap.OpKind == SnapshotOpInsert {
				if ord, ok := s.Orders[oid]; ok {
					ord.SupersededByRollback = true
					ord.Status = order.StatusCancelled
					ord.UpdatedAt = s.nowUTC()
					s.Orders[oid] = ord
					res.SupersededOrders++
				}
			}
		case "ArInvoices":
			invID, _ := extractPKField(snap.RowKeyJSON, "InvoiceId")
			if s.ARRepo != nil {
				if inv, err := s.ARRepo.GetByID(context.Background(), invID); err == nil && inv.Status != ar.StatusVoid && inv.Status != ar.StatusPaid {
					bal := inv.BalanceMinor
					if bal > 0 {
						revJE, revErr := BuildReversalJournalEntry(sessionID, inv.InvoiceID, inv.OrderID, inv.SupplierID, inv.RetailerID, inv.Currency, bal, s.nowUTC())
						if revErr != nil {
							return RollbackResult{}, revErr
						}
						res.ReversalEntryIDs = append(res.ReversalEntryIDs, revJE.EntryID)
						res.ReversedAmountMinor += bal
						s.JournalEntries[revJE.EntryID] = revJE
						relKey := inv.SupplierID + "|" + inv.RetailerID
						if cp, ok := s.CreditProfiles[relKey]; ok {
							cp.CurrentBalanceMinor -= bal
							if cp.CurrentBalanceMinor < 0 {
								cp.CurrentBalanceMinor = 0
							}
							avail := cp.CreditLimitMinor - cp.CurrentBalanceMinor - cp.ReservedMinor
							if avail < 0 {
								avail = 0
							}
							cp.AvailableCreditMinor = avail
							cp.Version++
							s.CreditProfiles[relKey] = cp
						}
					}
					if err := s.ARRepo.VoidInvoice(context.Background(), invID, "mig_rollback:"+sessionID+":"+invID); err != nil {
						return RollbackResult{}, err
					}
					res.ReversedInvoices++
				}
			}
		}
	}

	for k, im := range s.IdentityMap {
		if im.SupplierID == supplierID && im.CreatedBySession == sessionID {
			delete(s.IdentityMap, k)
		}
	}
	return res, nil
}

type memoryTxState struct {
	supplier       map[string]MemorySupplierMeta
	warehouses     map[string]StoredWarehouse
	plantMaps      map[string]string
	products       map[string]StoredProduct
	retailers      map[string]StoredCustomer
	links          map[string]StoredCustomerLink
	inventory      map[string]StoredStock
	lots           map[string]StoredLot
	locations      map[string]bool
	orders         map[string]order.Order
	arRepo         *ar.MemoryRepository
	creditProfiles map[string]ar.CreditProfileState
	journalEntries map[string]*payment.JournalEntry
	identityMap    map[string]IdentityMappingEntry
}

type memoryApplyTx struct {
	supplierID   string
	sessionID    string
	now          time.Time
	seq          int64
	state        memoryTxState
	snapshots    []MigrationSnapshotRecord
	snapRecorded map[string]bool
}

func (tx *memoryApplyTx) Now() time.Time     { return tx.now }
func (tx *memoryApplyTx) SupplierID() string { return tx.supplierID }
func (tx *memoryApplyTx) SessionID() string  { return tx.sessionID }

func (tx *memoryApplyTx) ReadSupplierMeta(_ context.Context) (string, string, error) {
	if meta, ok := tx.state.supplier[tx.supplierID]; ok {
		cur := strings.ToUpper(strings.TrimSpace(meta.Currency))
		if cur == "" {
			cur = "UZS"
		}
		return meta.HomeCell, cur, nil
	}
	return "", "UZS", nil
}

func (tx *memoryApplyTx) LookupIdentity(_ context.Context, pack Pack, extID string) (string, bool, error) {
	im, ok := tx.state.identityMap[identityKey(tx.supplierID, pack, extID)]
	if !ok {
		return "", false, nil
	}
	return im.InternalID, true, nil
}

func (tx *memoryApplyTx) PutIdentity(_ context.Context, pack Pack, extID, internalID string) error {
	key := identityKey(tx.supplierID, pack, extID)
	tx.state.identityMap[key] = IdentityMappingEntry{
		SupplierID:       tx.supplierID,
		Pack:             pack,
		ExternalID:       strings.TrimSpace(extID),
		InternalID:       strings.TrimSpace(internalID),
		CreatedBySession: tx.sessionID,
	}
	return nil
}

func (tx *memoryApplyTx) RecordSnapshot(_ context.Context, targetTable string, keyObj any, opKind string, beforeObj any) error {
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
	tx.snapshots = append(tx.snapshots, rec)
	return nil
}

func (tx *memoryApplyTx) FindWarehouse(_ context.Context, extID, name string) (StoredWarehouse, bool, error) {
	ext := strings.TrimSpace(extID)
	if ext != "" {
		if wh, ok := tx.state.warehouses[ext]; ok && wh.SupplierID == tx.supplierID {
			return wh, true, nil
		}
		if mappedWH, ok := tx.state.plantMaps[plantMapKey(tx.supplierID, ext)]; ok {
			if wh, ok2 := tx.state.warehouses[mappedWH]; ok2 && wh.SupplierID == tx.supplierID {
				return wh, true, nil
			}
		}
	}
	wantName := strings.ToLower(strings.TrimSpace(name))
	if wantName != "" {
		for _, wh := range tx.state.warehouses {
			if wh.SupplierID == tx.supplierID && strings.ToLower(strings.TrimSpace(wh.Name)) == wantName {
				return wh, true, nil
			}
		}
	}
	return StoredWarehouse{}, false, nil
}

func (tx *memoryApplyTx) SaveWarehouse(ctx context.Context, wh StoredWarehouse, extPlantID string, before *StoredWarehouse) error {
	if before == nil {
		if err := tx.RecordSnapshot(ctx, "Warehouses", map[string]any{"WarehouseId": wh.WarehouseID}, SnapshotOpInsert, nil); err != nil {
			return err
		}
	} else {
		if err := tx.RecordSnapshot(ctx, "Warehouses", map[string]any{"WarehouseId": wh.WarehouseID}, SnapshotOpUpdate, snapshotWarehouseMap(*before)); err != nil {
			return err
		}
	}
	tx.state.warehouses[wh.WarehouseID] = wh
	if ext := strings.TrimSpace(extPlantID); ext != "" {
		pk := plantMapKey(tx.supplierID, ext)
		prevWH, existed := tx.state.plantMaps[pk]
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
		tx.state.plantMaps[pk] = wh.WarehouseID
	}
	return nil
}

func (tx *memoryApplyTx) FindProduct(_ context.Context, internalID, sku, barcode, name string) (StoredProduct, bool, error) {
	for _, candidate := range []string{strings.TrimSpace(internalID), strings.TrimSpace(sku)} {
		if candidate != "" {
			if p, ok := tx.state.products[candidate]; ok && p.SupplierID == tx.supplierID {
				return p, true, nil
			}
		}
	}
	if bc := strings.TrimSpace(barcode); bc != "" {
		for _, p := range tx.state.products {
			if p.SupplierID == tx.supplierID && strings.EqualFold(strings.TrimSpace(p.Barcode), bc) {
				return p, true, nil
			}
		}
	}
	if nm := strings.ToLower(strings.TrimSpace(name)); nm != "" {
		for _, p := range tx.state.products {
			if p.SupplierID == tx.supplierID && strings.ToLower(strings.TrimSpace(p.Name)) == nm {
				return p, true, nil
			}
		}
	}
	return StoredProduct{}, false, nil
}

func (tx *memoryApplyTx) TotalProductOnHand(_ context.Context, productID string) (int64, error) {
	var total int64
	for _, stk := range tx.state.inventory {
		if stk.SupplierID == tx.supplierID && stk.ProductID == productID {
			total += stk.QuantityOnHand
		}
	}
	return total, nil
}

func (tx *memoryApplyTx) SaveProduct(ctx context.Context, p StoredProduct, before *StoredProduct) error {
	if before == nil {
		if err := tx.RecordSnapshot(ctx, "Products", map[string]any{"ProductId": p.ProductID}, SnapshotOpInsert, nil); err != nil {
			return err
		}
	} else {
		if err := tx.RecordSnapshot(ctx, "Products", map[string]any{"ProductId": p.ProductID}, SnapshotOpUpdate, snapshotProductMap(*before)); err != nil {
			return err
		}
	}
	tx.state.products[p.ProductID] = p
	return nil
}

func (tx *memoryApplyTx) FindCustomer(_ context.Context, _ string, extID, inn, phone string) (StoredCustomer, bool, error) {
	if id := strings.TrimSpace(extID); id != "" {
		if ret, ok := tx.state.retailers[id]; ok {
			ret.Link = tx.state.links[supplierLinkKey(tx.supplierID, ret.RetailerID)]
			return ret, true, nil
		}
	}
	normINN := NormalizeINN(inn)
	if normINN != "" {
		// Check supplier's existing links first, then cross-supplier retailer links matching INN.
		if link, ok := tx.findLinkByINN(normINN, true); ok {
			if ret, rOK := tx.state.retailers[link.RetailerID]; rOK {
				ret.Link = link
				return ret, true, nil
			}
		}
		if link, ok := tx.findLinkByINN(normINN, false); ok {
			if ret, rOK := tx.state.retailers[link.RetailerID]; rOK {
				ret.Link = tx.state.links[supplierLinkKey(tx.supplierID, ret.RetailerID)]
				return ret, true, nil
			}
		}
	}
	normPhone := NormalizePhoneUZ(phone)
	if normPhone != "" {
		for _, ret := range tx.state.retailers {
			if NormalizePhoneUZ(ret.Phone) == normPhone {
				ret.Link = tx.state.links[supplierLinkKey(tx.supplierID, ret.RetailerID)]
				return ret, true, nil
			}
		}
	}
	return StoredCustomer{}, false, nil
}

func (tx *memoryApplyTx) findLinkByINN(inn string, sameSupplierOnly bool) (StoredCustomerLink, bool) {
	keys := make([]string, 0, len(tx.state.links))
	for k := range tx.state.links {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		l := tx.state.links[k]
		if sameSupplierOnly && l.SupplierID != tx.supplierID {
			continue
		}
		if NormalizeINN(l.TaxIdTIN) == inn {
			return l, true
		}
	}
	return StoredCustomerLink{}, false
}

func (tx *memoryApplyTx) SaveCustomer(ctx context.Context, ret StoredCustomer, createRetailer bool, beforeLink *StoredCustomerLink) error {
	if createRetailer {
		tx.state.retailers[ret.RetailerID] = StoredCustomer{
			RetailerID:   ret.RetailerID,
			Phone:        ret.Phone,
			Name:         ret.Name,
			CountryCode:  firstNonEmpty(ret.CountryCode, "UZ"),
			DeliveryAddr: ret.DeliveryAddr,
		}
	}
	keyObj := map[string]any{"SupplierId": ret.Link.SupplierID, "RetailerId": ret.Link.RetailerID}
	if beforeLink == nil {
		if err := tx.RecordSnapshot(ctx, "SupplierRetailerLinks", keyObj, SnapshotOpInsert, nil); err != nil {
			return err
		}
	} else {
		if err := tx.RecordSnapshot(ctx, "SupplierRetailerLinks", keyObj, SnapshotOpUpdate, snapshotCustomerLinkMap(*beforeLink)); err != nil {
			return err
		}
	}
	tx.state.links[supplierLinkKey(ret.Link.SupplierID, ret.Link.RetailerID)] = ret.Link
	return nil
}

func (tx *memoryApplyTx) GetInventory(_ context.Context, warehouseID, productID string) (StoredStock, bool, error) {
	stk, ok := tx.state.inventory[inventoryKey(tx.supplierID, warehouseID, productID)]
	return stk, ok, nil
}

func (tx *memoryApplyTx) UpsertStock(ctx context.Context, stk StoredStock, before *StoredStock) error {
	keyObj := map[string]any{
		"SupplierId":  stk.SupplierID,
		"WarehouseId": stk.WarehouseID,
		"ProductId":   stk.ProductID,
	}
	if before == nil {
		if err := tx.RecordSnapshot(ctx, "SupplierInventoryV2", keyObj, SnapshotOpInsert, nil); err != nil {
			return err
		}
	} else {
		if err := tx.RecordSnapshot(ctx, "SupplierInventoryV2", keyObj, SnapshotOpUpdate, snapshotStockMap(*before)); err != nil {
			return err
		}
	}
	tx.state.inventory[inventoryKey(stk.SupplierID, stk.WarehouseID, stk.ProductID)] = stk
	return nil
}

func (tx *memoryApplyTx) FindLot(_ context.Context, warehouseID, productID, lotCode string) (StoredLot, bool, error) {
	want := strings.ToLower(strings.TrimSpace(lotCode))
	for _, lot := range tx.state.lots {
		if lot.SupplierID == tx.supplierID &&
			lot.WarehouseID == warehouseID &&
			lot.ProductID == productID &&
			strings.ToLower(strings.TrimSpace(lot.LotCode)) == want {
			return lot, true, nil
		}
	}
	return StoredLot{}, false, nil
}

func (tx *memoryApplyTx) PutawayOrAdjustLot(ctx context.Context, op LotMigrationOp, existing *StoredLot, beforeStock *StoredStock) (LotMigrationResult, error) {
	locID := firstNonEmpty(strings.TrimSpace(op.LocationID), "recv-default")
	tx.state.locations[op.WarehouseID+"|"+locID] = true

	var result LotMigrationResult
	if existing == nil {
		if op.TargetQuantity == 0 {
			return LotMigrationResult{LotID: op.NewLotID, DeltaQty: 0, Action: "skipped", FinalOnHand: 0}, nil
		}
		if err := tx.RecordSnapshot(ctx, "StockLots", map[string]any{"LotId": op.NewLotID}, SnapshotOpInsert, nil); err != nil {
			return LotMigrationResult{}, err
		}
		tx.state.lots[op.NewLotID] = StoredLot{
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
		result = LotMigrationResult{
			LotID:       op.NewLotID,
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
		if err := tx.RecordSnapshot(ctx, "StockLots", map[string]any{"LotId": existing.LotID}, SnapshotOpUpdate, snapshotLotMap(*existing)); err != nil {
			return LotMigrationResult{}, err
		}
		updated := *existing
		updated.LocationID = locID
		updated.QuantityOnHand = op.TargetQuantity
		if op.ExpiryDate != nil {
			updated.ExpiryDate = op.ExpiryDate
		}
		if op.ManufacturedDate != nil {
			updated.ManufacturedDate = op.ManufacturedDate
		}
		if updated.Status == "" {
			updated.Status = "AVAILABLE"
		}
		tx.state.lots[updated.LotID] = updated
		result = LotMigrationResult{
			LotID:       updated.LotID,
			DeltaQty:    delta,
			Action:      "updated",
			FinalOnHand: updated.QuantityOnHand,
		}
	}

	// Recompute SupplierInventoryV2 rollup across AVAILABLE lots for this (supplier, warehouse, product).
	var qohSum, resSum int64
	for _, lot := range tx.state.lots {
		if lot.SupplierID == tx.supplierID &&
			lot.WarehouseID == op.WarehouseID &&
			lot.ProductID == op.ProductID &&
			lot.Status == "AVAILABLE" {
			qohSum += lot.QuantityOnHand
			resSum += lot.QuantityReserved
		}
	}
	var thresh int64
	if beforeStock != nil {
		thresh = beforeStock.ReorderThreshold
	}
	nextStk := StoredStock{
		SupplierID:       tx.supplierID,
		WarehouseID:      op.WarehouseID,
		ProductID:        op.ProductID,
		QuantityOnHand:   qohSum,
		QuantityReserved: resSum,
		ReorderThreshold: thresh,
	}
	if err := tx.UpsertStock(ctx, nextStk, beforeStock); err != nil {
		return LotMigrationResult{}, err
	}
	return result, nil
}

func (tx *memoryApplyTx) ResolveCustomerForFinancials(ctx context.Context, homeCell, extID, inn, phone, name string) (StoredCustomer, bool, error) {
	if cust, ok, err := tx.FindCustomer(ctx, homeCell, extID, inn, phone); err != nil || ok {
		return cust, ok, err
	}
	wantName := strings.ToLower(strings.TrimSpace(name))
	if wantName != "" {
		for lk, link := range tx.state.links {
			if !strings.HasPrefix(lk, tx.supplierID+"|") {
				continue
			}
			ret, ok := tx.state.retailers[link.RetailerID]
			if !ok {
				continue
			}
			if strings.ToLower(strings.TrimSpace(link.LegalName)) == wantName ||
				strings.ToLower(strings.TrimSpace(ret.Name)) == wantName {
				ret.Link = link
				return ret, true, nil
			}
		}
	}
	return StoredCustomer{}, false, nil
}

func (tx *memoryApplyTx) ResolveWarehouseForOrder(ctx context.Context, whHint, defaultWarehouseID string) (StoredWarehouse, bool, error) {
	hint := strings.TrimSpace(whHint)
	if hint != "" {
		if wh, ok, err := tx.FindWarehouse(ctx, hint, hint); err != nil || ok {
			return wh, ok, err
		}
		return StoredWarehouse{}, false, nil
	}
	if strings.TrimSpace(defaultWarehouseID) != "" {
		return tx.FindWarehouse(ctx, defaultWarehouseID, "")
	}
	for _, wh := range tx.state.warehouses {
		if wh.SupplierID == tx.supplierID && wh.IsActive {
			return wh, true, nil
		}
	}
	return StoredWarehouse{}, false, nil
}

func (tx *memoryApplyTx) ImportHistoricalOrder(ctx context.Context, req order.HistoricalOrderRequest) (bool, bool, string, error) {
	prep, _, err := order.PrepareHistoricalOrder(req)
	if err != nil {
		return false, false, "", err
	}
	if existing, ok := tx.state.orders[prep.OrderID]; ok {
		if len(prep.LineItems) > 0 && !containsOrderLineItem(existing.LineItems, prep.LineItems[0]) {
			existing.LineItems = append(existing.LineItems, prep.LineItems...)
			existing.Version++
			existing.UpdatedAt = tx.now
			tx.state.orders[existing.OrderID] = existing
			return false, true, existing.OrderID, nil
		}
		return false, false, existing.OrderID, nil
	}
	if err := tx.RecordSnapshot(ctx, "Orders", map[string]any{"OrderId": prep.OrderID}, SnapshotOpInsert, nil); err != nil {
		return false, false, "", err
	}
	tx.state.orders[prep.OrderID] = prep
	return true, false, prep.OrderID, nil
}

func (tx *memoryApplyTx) ImportMigratedOpenOrder(ctx context.Context, req order.MigratedOpenOrderRequest) (bool, bool, string, error) {
	prep, _, err := order.PrepareMigratedOpenOrder(req)
	if err != nil {
		return false, false, "", err
	}
	if existing, ok := tx.state.orders[prep.OrderID]; ok {
		if len(prep.LineItems) > 0 && !containsOrderLineItem(existing.LineItems, prep.LineItems[0]) {
			existing.LineItems = append(existing.LineItems, prep.LineItems...)
			existing.TotalMinor += prep.LineItems[0].Quantity * prep.LineItems[0].UnitPrice
			existing.OriginalTotalMinor = existing.TotalMinor
			existing.Version++
			existing.UpdatedAt = tx.now
			tx.state.orders[existing.OrderID] = existing
			return false, true, existing.OrderID, nil
		}
		return false, false, existing.OrderID, nil
	}
	if err := tx.RecordSnapshot(ctx, "Orders", map[string]any{"OrderId": prep.OrderID}, SnapshotOpInsert, nil); err != nil {
		return false, false, "", err
	}
	tx.state.orders[prep.OrderID] = prep
	return true, false, prep.OrderID, nil
}

func containsOrderLineItem(lines []order.LineItem, candidate order.LineItem) bool {
	for _, li := range lines {
		if strings.EqualFold(li.SKU, candidate.SKU) && li.Quantity == candidate.Quantity && li.UnitPrice == candidate.UnitPrice {
			return true
		}
	}
	return false
}

func (tx *memoryApplyTx) OpenMigrationNasiyaAR(ctx context.Context, req ar.OpenFromMigrationRequest) (*ar.OpenFromMigrationResult, error) {
	res, err := tx.state.arRepo.OpenFromMigration(req, tx.state.creditProfiles)
	if err != nil {
		return nil, err
	}
	if res.AlreadyExisted {
		return res, nil
	}
	// Persist paired synthetic MIGRATION order so order state matches Spanner Idx_ArInvoices_ByOrder invariant.
	if _, exists := tx.state.orders[res.SyntheticOrderID]; !exists {
		synOrder := order.Order{
			OrderID:            res.SyntheticOrderID,
			SupplierID:         tx.supplierID,
			RetailerID:         res.Invoice.RetailerID,
			WarehouseID:        req.WarehouseID,
			Status:             order.Status("DELIVERED"),
			Source:             order.OrderSourceMigration,
			ConfirmationStatus: order.ConfirmationStatusConfirmed,
			LineItems:          []order.LineItem{},
			TotalMinor:         res.Invoice.PrincipalMinor,
			OriginalTotalMinor: res.Invoice.PrincipalMinor,
			Currency:           res.Invoice.Currency,
			FiscalStatus:       order.FiscalStatusHistoricalSkipped,
			ExternalOrderNo:    res.Invoice.ExternalID,
			Version:            1,
			CreatedAt:          res.Invoice.CreditLeaveAt,
			UpdatedAt:          tx.now,
		}
		if err := tx.RecordSnapshot(ctx, "Orders", map[string]any{"OrderId": synOrder.OrderID}, SnapshotOpInsert, nil); err != nil {
			return nil, err
		}
		tx.state.orders[synOrder.OrderID] = synOrder
	}
	if err := tx.RecordSnapshot(ctx, "ArInvoices", map[string]any{"InvoiceId": res.Invoice.InvoiceID}, SnapshotOpInsert, nil); err != nil {
		return nil, err
	}
	if je, ok := res.JournalEntry.(*payment.JournalEntry); ok && je != nil {
		tx.state.journalEntries[je.EntryID] = je
	}
	return res, nil
}

// ReleaseOpenOrders reserves in-memory stock/lots and releases MIGRATION_OPEN orders to live PENDING/SCHEDULED.
func (s *MemoryApplyStore) ReleaseOpenOrders(_ context.Context, supplierID string, items []order.ReleaseOpenOrderItem, actorID string, now time.Time) (order.ReleaseOpenOrdersResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureInitLocked()
	if now.IsZero() {
		now = s.nowUTC()
	}

	if len(items) == 0 {
		var ids []string
		for oid, o := range s.Orders {
			if o.SupplierID == supplierID && o.Status == order.StatusMigrationOpen && !o.SupersededByRollback {
				ids = append(ids, oid)
			}
		}
		sort.Strings(ids)
		for _, oid := range ids {
			items = append(items, order.ReleaseOpenOrderItem{OrderID: oid})
		}
	}

	res := order.ReleaseOpenOrdersResult{
		ReleasedOrderIDs: make([]string, 0, len(items)),
		Errors:           make(map[string]string),
	}
	for _, it := range items {
		o, ok := s.Orders[it.OrderID]
		if !ok || o.SupplierID != supplierID {
			res.Errors[it.OrderID] = "order not found"
			res.SkippedOrderIDs = append(res.SkippedOrderIDs, it.OrderID)
			continue
		}
		target, err := order.ResolveReleaseTargetStatus(o, it.TargetStatus)
		if err != nil {
			res.Errors[it.OrderID] = err.Error()
			res.SkippedOrderIDs = append(res.SkippedOrderIDs, it.OrderID)
			continue
		}

		// Transactional per-order copy of Inventory + Lots to guarantee all-or-nothing line reservation.
		stagedInv := cloneMap(s.Inventory)
		stagedLots := cloneMap(s.Lots)
		reserveOK := true
		for _, line := range o.LineItems {
			prodID := strings.TrimSpace(line.SKU)
			sk := inventoryKey(supplierID, o.WarehouseID, prodID)
			stk, exists := stagedInv[sk]
			if !exists || stk.QuantityOnHand-stk.QuantityReserved < line.Quantity {
				res.Errors[it.OrderID] = fmt.Sprintf("insufficient stock for product %s at warehouse %s", prodID, o.WarehouseID)
				res.SkippedOrderIDs = append(res.SkippedOrderIDs, it.OrderID)
				reserveOK = false
				break
			}
			stk.QuantityReserved += line.Quantity
			stagedInv[sk] = stk

			// FEFO lot allocation if lots exist for this (supplier, warehouse, product).
			var matchingLots []StoredLot
			for _, lot := range stagedLots {
				if lot.SupplierID == supplierID && lot.WarehouseID == o.WarehouseID && lot.ProductID == prodID && lot.Status == "AVAILABLE" {
					matchingLots = append(matchingLots, lot)
				}
			}
			if len(matchingLots) > 0 {
				sort.Slice(matchingLots, func(i, j int) bool {
					ei, ej := matchingLots[i].ExpiryDate, matchingLots[j].ExpiryDate
					if ei == nil && ej != nil {
						return false
					}
					if ei != nil && ej == nil {
						return true
					}
					if ei != nil && ej != nil && !ei.Equal(*ej) {
						return ei.Before(*ej)
					}
					return matchingLots[i].LotID < matchingLots[j].LotID
				})
				rem := line.Quantity
				for _, lot := range matchingLots {
					if rem <= 0 {
						break
					}
					avail := lot.QuantityOnHand - lot.QuantityReserved
					if avail <= 0 {
						continue
					}
					take := rem
					if take > avail {
						take = avail
					}
					lot.QuantityReserved += take
					stagedLots[lot.LotID] = lot
					rem -= take
				}
				if rem > 0 {
					res.Errors[it.OrderID] = fmt.Sprintf("insufficient lot stock for product %s", prodID)
					res.SkippedOrderIDs = append(res.SkippedOrderIDs, it.OrderID)
					reserveOK = false
					break
				}
			}
		}
		if !reserveOK {
			continue
		}

		s.Inventory = stagedInv
		s.Lots = stagedLots
		o.Status = target
		o.Source = order.OrderSourceManual
		if strings.TrimSpace(o.ExternalFiscalRef) != "" {
			o.FiscalStatus = order.FiscalStatusHistoricalSkipped
		} else if strings.TrimSpace(o.FiscalStatus) == "" {
			o.FiscalStatus = order.FiscalStatusNone
		}
		o.Version++
		o.UpdatedAt = now.UTC()
		_ = actorID
		s.Orders[o.OrderID] = o
		res.ReleasedOrderIDs = append(res.ReleasedOrderIDs, o.OrderID)
	}
	if len(res.Errors) == 0 {
		res.Errors = nil
	}
	return res, nil
}

// --- Shared helpers reused by both MemoryApplyStore and SpannerApplyStore ---

func identityKey(supplierID string, pack Pack, extID string) string {
	return supplierID + "|" + string(pack) + "|" + strings.ToLower(strings.TrimSpace(extID))
}

func plantMapKey(supplierID, extPlantID string) string {
	return supplierID + "|" + strings.ToLower(strings.TrimSpace(extPlantID))
}

func supplierLinkKey(supplierID, retailerID string) string {
	return supplierID + "|" + retailerID
}

func inventoryKey(supplierID, warehouseID, productID string) string {
	return supplierID + "|" + warehouseID + "|" + productID
}

func cloneMap[K comparable, V any](src map[K]V) map[K]V {
	dst := make(map[K]V, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func snapshotWarehouseMap(wh StoredWarehouse) map[string]any {
	m := map[string]any{
		"WarehouseId": wh.WarehouseID,
		"SupplierId":  wh.SupplierID,
		"Name":        wh.Name,
		"IsActive":    wh.IsActive,
		"IsOnShift":   wh.IsOnShift,
	}
	if wh.Address != "" {
		m["Address"] = wh.Address
	} else {
		m["Address"] = nil
	}
	return m
}

func snapshotPlantMap(supplierID, extPlantID, warehouseID string) map[string]any {
	return map[string]any{
		"TenantType":      "SUPPLIER",
		"TenantId":        supplierID,
		"ExternalPlantId": extPlantID,
		"WarehouseId":     warehouseID,
	}
}

func snapshotProductMap(p StoredProduct) map[string]any {
	m := map[string]any{
		"ProductId":         p.ProductID,
		"SupplierId":        p.SupplierID,
		"CategoryId":        p.CategoryID,
		"Name":              p.Name,
		"PriceMinor":        p.PriceMinor,
		"Currency":          p.Currency,
		"Unit":              p.Unit,
		"HandlingClass":     p.HandlingClass,
		"IsPerishable":      p.IsPerishable,
		"RequiresColdChain": p.RequiresColdChain,
		"IsActive":          p.IsActive,
		"Version":           p.Version,
	}
	if p.Description != "" {
		m["Description"] = p.Description
	} else {
		m["Description"] = nil
	}
	if p.Barcode != "" {
		m["Barcode"] = p.Barcode
	} else {
		m["Barcode"] = nil
	}
	if p.MinShelfLifeDays > 0 {
		m["MinShelfLifeDays"] = p.MinShelfLifeDays
	} else {
		m["MinShelfLifeDays"] = nil
	}
	return m
}

func snapshotCustomerLinkMap(l StoredCustomerLink) map[string]any {
	m := map[string]any{
		"SupplierId":  l.SupplierID,
		"RetailerId":  l.RetailerID,
		"DedupStatus": firstNonEmpty(l.DedupStatus, "CLEAN"),
	}
	if l.TaxIdTIN != "" {
		m["TaxIdTIN"] = l.TaxIdTIN
	} else {
		m["TaxIdTIN"] = nil
	}
	if l.NormalizedPhone != "" {
		m["NormalizedPhone"] = l.NormalizedPhone
	} else {
		m["NormalizedPhone"] = nil
	}
	if l.LegalName != "" {
		m["LegalName"] = l.LegalName
	} else {
		m["LegalName"] = nil
	}
	return m
}

func snapshotStockMap(stk StoredStock) map[string]any {
	return map[string]any{
		"SupplierId":       stk.SupplierID,
		"WarehouseId":      stk.WarehouseID,
		"ProductId":        stk.ProductID,
		"QuantityOnHand":   stk.QuantityOnHand,
		"QuantityReserved": stk.QuantityReserved,
		"ReorderThreshold": stk.ReorderThreshold,
	}
}

func snapshotLotMap(l StoredLot) map[string]any {
	m := map[string]any{
		"LotId":            l.LotID,
		"SupplierId":       l.SupplierID,
		"WarehouseId":      l.WarehouseID,
		"ProductId":        l.ProductID,
		"LocationId":       l.LocationID,
		"LotCode":          l.LotCode,
		"QuantityOnHand":   l.QuantityOnHand,
		"QuantityReserved": l.QuantityReserved,
		"Status":           l.Status,
	}
	if l.ExpiryDate != nil && !l.ExpiryDate.IsZero() {
		m["ExpiryDate"] = l.ExpiryDate.UTC().Format("2006-01-02")
	} else {
		m["ExpiryDate"] = nil
	}
	if l.ManufacturedDate != nil && !l.ManufacturedDate.IsZero() {
		m["ManufacturedDate"] = l.ManufacturedDate.UTC().Format("2006-01-02")
	} else {
		m["ManufacturedDate"] = nil
	}
	return m
}

func decodeJSONWithNumbersMap(raw []byte, dst *map[string]any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	for k, v := range *dst {
		if num, ok := v.(json.Number); ok {
			if iv, err := num.Int64(); err == nil {
				(*dst)[k] = iv
			}
		}
	}
	return nil
}

func storedLotFromMap(m map[string]any) StoredLot {
	l := StoredLot{
		LotID:       fmt.Sprint(m["LotId"]),
		SupplierID:  fmt.Sprint(m["SupplierId"]),
		WarehouseID: fmt.Sprint(m["WarehouseId"]),
		ProductID:   fmt.Sprint(m["ProductId"]),
		LocationID:  fmt.Sprint(m["LocationId"]),
		LotCode:     fmt.Sprint(m["LotCode"]),
		Status:      fmt.Sprint(m["Status"]),
	}
	if v, ok := m["QuantityOnHand"].(int64); ok {
		l.QuantityOnHand = v
	}
	if v, ok := m["QuantityReserved"].(int64); ok {
		l.QuantityReserved = v
	}
	if s, ok := m["ExpiryDate"].(string); ok && s != "" {
		if t, err := ParseFlexibleDate(s); err == nil {
			l.ExpiryDate = &t
		}
	}
	if s, ok := m["ManufacturedDate"].(string); ok && s != "" {
		if t, err := ParseFlexibleDate(s); err == nil {
			l.ManufacturedDate = &t
		}
	}
	return l
}
