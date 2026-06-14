package manifest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
)

// SupplierTruckRow is the durable projection of a supplier/payload manifest.
type SupplierTruckRow struct {
	ManifestID       string
	SupplierID       string
	WarehouseID      string
	RouteID          string
	TruckID          string
	DriverID         string
	State            string
	TotalVolumeVU    float64
	MaxVolumeVU      float64
	StopCount        int64
	LoadingStartedAt *time.Time
	SealedAt         *time.Time
	DispatchedAt     *time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// SupplierManifestOrderRow is one ManifestOrders junction row.
type SupplierManifestOrderRow struct {
	ManifestID    string
	OrderID       string
	SequenceIndex int64
	LoadingOrder  int64
	VolumeVU      float64
	State         string
	RemovedReason string
	UpdatedAt     time.Time
}

// SupplierExceptionRow is one ManifestExceptions row.
type SupplierExceptionRow struct {
	ExceptionID  string
	OrderID      string
	ManifestID   string
	SupplierID   string
	Reason       string
	Metadata     string
	AttemptCount int64
	CreatedAt    time.Time
	EscalatedAt  *time.Time
}

// ErrOrderVersionConflict is returned when an OrderPatch carries ExpectedVersion
// that does not match the stored Orders.Version row.
var ErrOrderVersionConflict = errors.New("manifest: order version conflict")

// OrderPatch updates Orders columns touched by manifest transitions.
type OrderPatch struct {
	OrderID         string
	Status          string
	ManifestID      string
	DriverID        string
	VehicleID       string
	RouteID         string
	UpdatedAt       time.Time
	ExpectedVersion int64 // optional optimistic gate; 0 skips check
	Version         int64 // populated by resolveOrderPatchVersions before write
}

// FactoryTruckRow is the durable projection of a factory manifest.
type FactoryTruckRow struct {
	ManifestID       string
	FactoryID        string
	SupplierID       string
	DriverID         string
	VehicleID        string
	State            string
	TotalVolumeVU    float64
	MaxVolumeVU      float64
	StopCount        int64
	TransferCount    int64
	LoadingStartedAt *time.Time
	SealedAt         *time.Time
	DispatchedAt     *time.Time
	CompletedAt      *time.Time
	CancelledAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// SupplierWriteBatch is the mutation set for one supplier manifest commit.
type SupplierWriteBatch struct {
	Manifests    []SupplierTruckRow
	Orders       []SupplierManifestOrderRow
	Exceptions   []SupplierExceptionRow
	OrderPatches []OrderPatch
}

// FactoryTransferRow is the durable projection of a factory inter-hub transfer.
type FactoryTransferRow struct {
	TransferID     string
	FactoryID      string
	SupplierID     string
	OrderID        string
	ManifestID     string
	State          string
	TotalVolumeVU  float64
	DriverID       string
	VehicleID      string
	ReassignDepth  int64
	ExceptionCount int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// FactoryWriteBatch is the mutation set for one factory manifest commit.
type FactoryWriteBatch struct {
	Manifests []FactoryTruckRow
	Transfers []FactoryTransferRow
}

// Store reads and writes manifest tables in Spanner.
type Store struct {
	client          *spanner.Client
	geometryBuilder *routing.GeometryBuilder
}

// NewStore constructs a manifest Spanner store.
func NewStore(client *spanner.Client) *Store {
	return &Store{client: client}
}

// SetGeometryBuilder configures optional OSRM-backed route geometry resolution.
func (s *Store) SetGeometryBuilder(builder *routing.GeometryBuilder) {
	if s != nil {
		s.geometryBuilder = builder
	}
}

// CommitSupplier applies supplier manifest mutations and outbox events atomically.
func (s *Store) CommitSupplier(
	ctx context.Context,
	batch *SupplierWriteBatch,
	emit func(outbox.TxnBuffer) error,
) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("manifest store: nil client")
	}
	if batch == nil {
		batch = &SupplierWriteBatch{}
	}

	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return s.CommitSupplierTxn(ctx, txn, batch, emit)
	})
	if err != nil {
		return fmt.Errorf("supplier manifest transaction: %w", err)
	}
	return nil
}

// CommitSupplierTxn applies mutations using an existing transaction.
func (s *Store) CommitSupplierTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	batch *SupplierWriteBatch,
	emit func(outbox.TxnBuffer) error,
) error {
	if len(batch.OrderPatches) > 0 {
		if err := resolveOrderPatchVersions(ctx, txn, batch.OrderPatches); err != nil {
			return err
		}
	}

	buf := &txnBuffer{}
	if emit != nil {
		if err := emit(buf); err != nil {
			return err
		}
	}

	mutations, err := supplierMutations(batch)
	if err != nil {
		return err
	}
	mutations = append(mutations, outboxMutations(buf)...)
	return txn.BufferWrite(mutations)
}

// CommitFactory applies factory manifest mutations and outbox events atomically.
func (s *Store) CommitFactory(
	ctx context.Context,
	batch *FactoryWriteBatch,
	emit func(outbox.TxnBuffer) error,
) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("manifest store: nil client")
	}
	if batch == nil {
		batch = &FactoryWriteBatch{}
	}

	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &txnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := factoryMutations(batch)
		mutations = append(mutations, outboxMutations(buf)...)
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("factory manifest transaction: %w", err)
	}
	return nil
}

// ListSupplierManifests returns all supplier manifests for a supplier id.
func (s *Store) ListSupplierManifests(ctx context.Context, supplierID string) ([]SupplierTruckRow, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("manifest store: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, SupplierId, WarehouseId, RouteId, TruckId, DriverId, State,
			TotalVolumeVU, MaxVolumeVU, StopCount, LoadingStartedAt, SealedAt, DispatchedAt,
			CompletedAt, CreatedAt, UpdatedAt
			FROM SupplierTruckManifests WHERE SupplierId = @supplierId ORDER BY CreatedAt DESC`,
		Params: map[string]any{"supplierId": supplierID},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var rows []SupplierTruckRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list supplier manifests: %w", err)
		}
		parsed, err := scanSupplierManifest(row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, parsed)
	}
	return rows, nil
}

// ListSupplierManifestExceptions returns loading-gate exceptions for one supplier.
func (s *Store) ListSupplierManifestExceptions(ctx context.Context, supplierID string, limit int) ([]SupplierExceptionRow, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("manifest store: nil client")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, fmt.Errorf("manifest store: supplier id required")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	stmt := spanner.Statement{
		SQL: `SELECT ExceptionId, OrderId, ManifestId, SupplierId, Reason, Metadata, AttemptCount, EscalatedAt, CreatedAt
		      FROM ManifestExceptions@{FORCE_INDEX=Idx_ManifestExceptions_BySupplier}
		      WHERE SupplierId = @supplierId
		      ORDER BY CreatedAt DESC
		      LIMIT @limit`,
		Params: map[string]any{
			"supplierId": supplierID,
			"limit":      int64(limit),
		},
	}
	iter := s.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	rows := make([]SupplierExceptionRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return rows, nil
		}
		if err != nil {
			return nil, fmt.Errorf("list supplier manifest exceptions: %w", err)
		}

		var parsed SupplierExceptionRow
		var metadata spanner.NullString
		var escalatedAt spanner.NullTime
		if err := row.Columns(
			&parsed.ExceptionID,
			&parsed.OrderID,
			&parsed.ManifestID,
			&parsed.SupplierID,
			&parsed.Reason,
			&metadata,
			&parsed.AttemptCount,
			&escalatedAt,
			&parsed.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan supplier manifest exception: %w", err)
		}
		if metadata.Valid {
			parsed.Metadata = metadata.StringVal
		}
		if escalatedAt.Valid {
			parsed.EscalatedAt = &escalatedAt.Time
		}
		rows = append(rows, parsed)
	}
}

// DriversOnActiveManifests returns driver IDs that already hold an open manifest.
// Active means DRAFT, LOADING, SEALED, or DISPATCHED — drivers are not re-dispatched until COMPLETED.
func (s *Store) DriversOnActiveManifests(ctx context.Context, supplierID, warehouseID string, driverIDs []string) (map[string]bool, error) {
	out := make(map[string]bool, len(driverIDs))
	if s == nil || s.client == nil || len(driverIDs) == 0 {
		return out, nil
	}
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	if supplierID == "" && warehouseID == "" {
		return out, nil
	}

	var stmt spanner.Statement
	if warehouseID != "" {
		stmt = spanner.Statement{
			SQL: `SELECT DISTINCT DriverId
			      FROM SupplierTruckManifests@{FORCE_INDEX=Idx_SupplierManifests_ByWarehouse}
			      WHERE WarehouseId = @warehouseId
			        AND DriverId IN UNNEST(@driverIds)
			        AND State IN ('DRAFT', 'LOADING', 'SEALED', 'DISPATCHED')`,
			Params: map[string]any{
				"warehouseId": warehouseID,
				"driverIds":   driverIDs,
			},
		}
	} else {
		stmt = spanner.Statement{
			SQL: `SELECT DISTINCT DriverId
			      FROM SupplierTruckManifests@{FORCE_INDEX=Idx_SupplierManifests_BySupplierId}
			      WHERE SupplierId = @supplierId
			        AND DriverId IN UNNEST(@driverIds)
			        AND State IN ('DRAFT', 'LOADING', 'SEALED', 'DISPATCHED')`,
			Params: map[string]any{
				"supplierId": supplierID,
				"driverIds":  driverIDs,
			},
		}
	}

	iter := s.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("drivers on active manifests: %w", err)
		}
		var driverID string
		if err := row.Columns(&driverID); err != nil {
			return nil, fmt.Errorf("drivers on active manifests scan: %w", err)
		}
		if driverID != "" {
			out[driverID] = true
		}
	}
}

// ListSupplierManifestOrders returns junction rows for one manifest.
func (s *Store) ListSupplierManifestOrders(ctx context.Context, manifestID string) ([]SupplierManifestOrderRow, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("manifest store: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, OrderId, SequenceIndex, LoadingOrder, VolumeVU, State,
			RemovedReason, UpdatedAt
			FROM ManifestOrders WHERE ManifestId = @manifestId ORDER BY SequenceIndex ASC`,
		Params: map[string]any{"manifestId": manifestID},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var rows []SupplierManifestOrderRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list manifest orders: %w", err)
		}
		parsed, err := scanSupplierManifestOrder(row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, parsed)
	}
	return rows, nil
}

// ListFactoryTransfers returns durable factory transfers for a factory node.
func (s *Store) ListFactoryTransfers(ctx context.Context, factoryID string) ([]FactoryTransferRow, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("manifest store: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT TransferId, FactoryId, SupplierId, OrderId, ManifestId, State,
			TotalVolumeVU, DriverId, VehicleId, ReassignDepth, ExceptionCount,
			CreatedAt, UpdatedAt
			FROM FactoryInternalTransfers WHERE FactoryId = @factoryId ORDER BY UpdatedAt DESC`,
		Params: map[string]any{"factoryId": factoryID},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var rows []FactoryTransferRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list factory transfers: %w", err)
		}
		parsed, err := scanFactoryTransfer(row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, parsed)
	}
	return rows, nil
}

// ListFactoryManifests returns factory manifests for a factory node.
func (s *Store) ListFactoryManifests(ctx context.Context, factoryID string) ([]FactoryTruckRow, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("manifest store: nil client")
	}
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, FactoryId, SupplierId, DriverId, VehicleId, State,
			TotalVolumeVU, MaxVolumeVU, StopCount, TransferCount,
			LoadingStartedAt, SealedAt, DispatchedAt, CompletedAt, CancelledAt,
			CreatedAt, UpdatedAt
			FROM FactoryTruckManifests WHERE FactoryId = @factoryId ORDER BY CreatedAt DESC`,
		Params: map[string]any{"factoryId": factoryID},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var rows []FactoryTruckRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list factory manifests: %w", err)
		}
		parsed, err := scanFactoryManifest(row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, parsed)
	}
	return rows, nil
}

// EnsureFactoryDemoTransfers inserts demo factory transfers when missing.
func (s *Store) EnsureFactoryDemoTransfers(ctx context.Context, rows []FactoryTransferRow) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("manifest store: nil client")
	}
	for _, transfer := range rows {
		_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			_, err := txn.ReadRow(ctx, "FactoryInternalTransfers", spanner.Key{transfer.TransferID}, []string{"TransferId"})
			if err == nil {
				return nil
			}
			if err != spanner.ErrRowNotFound {
				return err
			}
			batch := &FactoryWriteBatch{Transfers: []FactoryTransferRow{transfer}}
			return txn.BufferWrite(factoryMutations(batch))
		})
		if err != nil {
			return fmt.Errorf("ensure factory demo transfer %s: %w", transfer.TransferID, err)
		}
	}
	return nil
}

// EnsureFactoryDemoManifests inserts demo factory manifests when missing.
func (s *Store) EnsureFactoryDemoManifests(ctx context.Context, rows []FactoryTruckRow) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("manifest store: nil client")
	}
	for _, m := range rows {
		_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			_, err := txn.ReadRow(ctx, "FactoryTruckManifests", spanner.Key{m.ManifestID}, []string{"ManifestId"})
			if err == nil {
				return nil
			}
			if err != spanner.ErrRowNotFound {
				return err
			}
			batch := &FactoryWriteBatch{Manifests: []FactoryTruckRow{m}}
			return txn.BufferWrite(factoryMutations(batch))
		})
		if err != nil {
			return fmt.Errorf("ensure factory demo manifest %s: %w", m.ManifestID, err)
		}
	}
	return nil
}

// EnsureSupplierDemoManifests inserts demo manifests when missing (SSMR / local dev).
func (s *Store) EnsureSupplierDemoManifests(ctx context.Context, supplierID string, rows []SupplierTruckRow, orders map[string][]SupplierManifestOrderRow) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("manifest store: nil client")
	}
	for _, m := range rows {
		_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			_, err := txn.ReadRow(ctx, "SupplierTruckManifests", spanner.Key{m.ManifestID}, []string{"ManifestId"})
			if err == nil {
				return nil
			}
			if err != spanner.ErrRowNotFound {
				return err
			}

			batch := &SupplierWriteBatch{
				Manifests: []SupplierTruckRow{m},
				Orders:    orders[m.ManifestID],
			}
			muts, err := supplierMutations(batch)
			if err != nil {
				return err
			}
			return txn.BufferWrite(muts)
		})
		if err != nil {
			return fmt.Errorf("ensure demo manifest %s: %w", m.ManifestID, err)
		}
	}
	return nil
}

type txnBuffer struct {
	events []outbox.Event
	audits []outbox.AuditEntry
}

func (b *txnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (b *txnBuffer) BufferAudit(_ context.Context, e outbox.AuditEntry) error {
	b.audits = append(b.audits, e)
	return nil
}

func supplierMutations(batch *SupplierWriteBatch) ([]*spanner.Mutation, error) {
	var mutations []*spanner.Mutation
	for _, m := range batch.Manifests {
		row := map[string]any{
			"ManifestId":    m.ManifestID,
			"SupplierId":    m.SupplierID,
			"TruckId":       m.TruckID,
			"DriverId":      m.DriverID,
			"State":         m.State,
			"TotalVolumeVU": m.TotalVolumeVU,
			"MaxVolumeVU":   m.MaxVolumeVU,
			"StopCount":     m.StopCount,
			"CreatedAt":     commitOrTime(m.CreatedAt),
			"UpdatedAt":     spanner.CommitTimestamp,
		}
		if m.WarehouseID != "" {
			row["WarehouseId"] = m.WarehouseID
		}
		if m.RouteID != "" {
			row["RouteId"] = m.RouteID
		}
		if m.LoadingStartedAt != nil {
			row["LoadingStartedAt"] = m.LoadingStartedAt.UTC()
		}
		if m.SealedAt != nil {
			row["SealedAt"] = m.SealedAt.UTC()
		}
		if m.DispatchedAt != nil {
			row["DispatchedAt"] = m.DispatchedAt.UTC()
		}
		if m.CompletedAt != nil {
			row["CompletedAt"] = m.CompletedAt.UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("SupplierTruckManifests", row))
	}

	for _, o := range batch.Orders {
		row := map[string]any{
			"ManifestId":    o.ManifestID,
			"OrderId":       o.OrderID,
			"SequenceIndex": o.SequenceIndex,
			"LoadingOrder":  o.LoadingOrder,
			"VolumeVU":      o.VolumeVU,
			"State":         o.State,
			"UpdatedAt":     commitOrTime(o.UpdatedAt),
		}
		if o.RemovedReason != "" {
			row["RemovedReason"] = o.RemovedReason
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("ManifestOrders", row))
	}

	for _, e := range batch.Exceptions {
		row := map[string]any{
			"ExceptionId":  e.ExceptionID,
			"OrderId":      e.OrderID,
			"SupplierId":   e.SupplierID,
			"Reason":       e.Reason,
			"AttemptCount": e.AttemptCount,
			"CreatedAt":    commitOrTime(e.CreatedAt),
		}
		if e.ManifestID != "" {
			row["ManifestId"] = e.ManifestID
		}
		if e.Metadata != "" {
			row["Metadata"] = e.Metadata
		}
		if e.EscalatedAt != nil {
			row["EscalatedAt"] = e.EscalatedAt.UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("ManifestExceptions", row))
	}

	for _, p := range batch.OrderPatches {
		row := map[string]any{
			"OrderId":   p.OrderID,
			"Status":    p.Status,
			"UpdatedAt": commitOrTime(p.UpdatedAt),
		}
		if p.ManifestID != "" {
			row["ManifestId"] = p.ManifestID
		}
		if p.DriverID != "" {
			row["DriverId"] = p.DriverID
		}
		if p.VehicleID != "" {
			row["VehicleId"] = p.VehicleID
		}
		if p.RouteID != "" {
			row["RouteId"] = p.RouteID
		}
		if p.Version > 0 {
			row["Version"] = p.Version
		}
		mutations = append(mutations, spanner.UpdateMap("Orders", row))
	}

	return mutations, nil
}

// OrderPatchVersionByID maps order IDs to the post-patch version assigned during commit prep.
func OrderPatchVersionByID(patches []OrderPatch) map[string]int64 {
	out := make(map[string]int64, len(patches))
	for _, patch := range patches {
		if patch.Version > 0 && patch.OrderID != "" {
			out[patch.OrderID] = patch.Version
		}
	}
	return out
}

func resolveOrderPatchVersions(ctx context.Context, txn *spanner.ReadWriteTransaction, patches []OrderPatch) error {
	if txn == nil || len(patches) == 0 {
		return nil
	}
	seen := make(map[string]int64, len(patches))
	for i := range patches {
		orderID := strings.TrimSpace(patches[i].OrderID)
		if orderID == "" {
			continue
		}
		if version, ok := seen[orderID]; ok {
			patches[i].Version = version
			continue
		}
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"Version"})
		if err != nil {
			return fmt.Errorf("read order %s version: %w", orderID, err)
		}
		var stored int64
		if err := row.Columns(&stored); err != nil {
			return fmt.Errorf("scan order %s version: %w", orderID, err)
		}
		if expected := patches[i].ExpectedVersion; expected > 0 && stored != expected {
			return fmt.Errorf("%w: order %s expected %d got %d", ErrOrderVersionConflict, orderID, expected, stored)
		}
		next := stored + 1
		seen[orderID] = next
		patches[i].Version = next
	}
	return nil
}

func factoryMutations(batch *FactoryWriteBatch) []*spanner.Mutation {
	var mutations []*spanner.Mutation
	for _, t := range batch.Transfers {
		row := map[string]any{
			"TransferId":     t.TransferID,
			"FactoryId":      t.FactoryID,
			"SupplierId":     t.SupplierID,
			"State":          t.State,
			"TotalVolumeVU":  t.TotalVolumeVU,
			"ReassignDepth":  t.ReassignDepth,
			"ExceptionCount": t.ExceptionCount,
			"CreatedAt":      commitOrTime(t.CreatedAt),
			"UpdatedAt":      spanner.CommitTimestamp,
		}
		if t.OrderID != "" {
			row["OrderId"] = t.OrderID
		}
		if t.ManifestID != "" {
			row["ManifestId"] = t.ManifestID
		}
		if t.DriverID != "" {
			row["DriverId"] = t.DriverID
		}
		if t.VehicleID != "" {
			row["VehicleId"] = t.VehicleID
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("FactoryInternalTransfers", row))
	}
	for _, m := range batch.Manifests {
		row := map[string]any{
			"ManifestId":    m.ManifestID,
			"FactoryId":     m.FactoryID,
			"SupplierId":    m.SupplierID,
			"State":         m.State,
			"TotalVolumeVU": m.TotalVolumeVU,
			"MaxVolumeVU":   m.MaxVolumeVU,
			"StopCount":     m.StopCount,
			"TransferCount": m.TransferCount,
			"CreatedAt":     commitOrTime(m.CreatedAt),
			"UpdatedAt":     spanner.CommitTimestamp,
		}
		if m.DriverID != "" {
			row["DriverId"] = m.DriverID
		}
		if m.VehicleID != "" {
			row["VehicleId"] = m.VehicleID
		}
		if m.LoadingStartedAt != nil {
			row["LoadingStartedAt"] = m.LoadingStartedAt.UTC()
		}
		if m.SealedAt != nil {
			row["SealedAt"] = m.SealedAt.UTC()
		}
		if m.DispatchedAt != nil {
			row["DispatchedAt"] = m.DispatchedAt.UTC()
		}
		if m.CompletedAt != nil {
			row["CompletedAt"] = m.CompletedAt.UTC()
		}
		if m.CancelledAt != nil {
			row["CancelledAt"] = m.CancelledAt.UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("FactoryTruckManifests", row))
	}
	return mutations
}

func outboxMutations(buf *txnBuffer) []*spanner.Mutation {
	var mutations []*spanner.Mutation
	for _, e := range buf.events {
		createdAt := e.CreatedAt.UTC()
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		row := map[string]any{
			"EventId":       e.EventID,
			"AggregateType": e.AggregateType,
			"AggregateId":   e.AggregateID,
			"TopicName":     e.TopicName,
			"Payload":       e.Payload,
			"CreatedAt":     createdAt,
			"PublishedAt":   nil,
		}
		if e.PublishedAt != nil {
			row["PublishedAt"] = e.PublishedAt.UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
	}
	for _, a := range buf.audits {
		mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
	}
	return mutations
}

func commitOrTime(t time.Time) any {
	if t.IsZero() {
		return spanner.CommitTimestamp
	}
	return t.UTC()
}

func scanSupplierManifest(row *spanner.Row) (SupplierTruckRow, error) {
	var (
		m              SupplierTruckRow
		warehouseID    spanner.NullString
		routeID        spanner.NullString
		loadingStarted spanner.NullTime
		sealedAt       spanner.NullTime
		dispatchedAt   spanner.NullTime
		completedAt    spanner.NullTime
	)
	if err := row.Columns(
		&m.ManifestID, &m.SupplierID, &warehouseID, &routeID, &m.TruckID, &m.DriverID, &m.State,
		&m.TotalVolumeVU, &m.MaxVolumeVU, &m.StopCount,
		&loadingStarted, &sealedAt, &dispatchedAt, &completedAt,
		&m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return SupplierTruckRow{}, fmt.Errorf("scan supplier manifest: %w", err)
	}
	if warehouseID.Valid {
		m.WarehouseID = warehouseID.StringVal
	}
	if routeID.Valid {
		m.RouteID = routeID.StringVal
	}
	if loadingStarted.Valid {
		t := loadingStarted.Time
		m.LoadingStartedAt = &t
	}
	if sealedAt.Valid {
		t := sealedAt.Time
		m.SealedAt = &t
	}
	if dispatchedAt.Valid {
		t := dispatchedAt.Time
		m.DispatchedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		m.CompletedAt = &t
	}
	return m, nil
}

func scanFactoryManifest(row *spanner.Row) (FactoryTruckRow, error) {
	var (
		m              FactoryTruckRow
		driverID       spanner.NullString
		vehicleID      spanner.NullString
		loadingStarted spanner.NullTime
		sealedAt       spanner.NullTime
		dispatchedAt   spanner.NullTime
		completedAt    spanner.NullTime
		cancelledAt    spanner.NullTime
	)
	if err := row.Columns(
		&m.ManifestID, &m.FactoryID, &m.SupplierID, &driverID, &vehicleID, &m.State,
		&m.TotalVolumeVU, &m.MaxVolumeVU, &m.StopCount, &m.TransferCount,
		&loadingStarted, &sealedAt, &dispatchedAt, &completedAt, &cancelledAt,
		&m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return FactoryTruckRow{}, fmt.Errorf("scan factory manifest: %w", err)
	}
	if driverID.Valid {
		m.DriverID = driverID.StringVal
	}
	if vehicleID.Valid {
		m.VehicleID = vehicleID.StringVal
	}
	if loadingStarted.Valid {
		t := loadingStarted.Time
		m.LoadingStartedAt = &t
	}
	if sealedAt.Valid {
		t := sealedAt.Time
		m.SealedAt = &t
	}
	if dispatchedAt.Valid {
		t := dispatchedAt.Time
		m.DispatchedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		m.CompletedAt = &t
	}
	if cancelledAt.Valid {
		t := cancelledAt.Time
		m.CancelledAt = &t
	}
	return m, nil
}

func scanFactoryTransfer(row *spanner.Row) (FactoryTransferRow, error) {
	var (
		t          FactoryTransferRow
		orderID    spanner.NullString
		manifestID spanner.NullString
		driverID   spanner.NullString
		vehicleID  spanner.NullString
	)
	if err := row.Columns(
		&t.TransferID, &t.FactoryID, &t.SupplierID, &orderID, &manifestID, &t.State,
		&t.TotalVolumeVU, &driverID, &vehicleID, &t.ReassignDepth, &t.ExceptionCount,
		&t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return FactoryTransferRow{}, fmt.Errorf("scan factory transfer: %w", err)
	}
	if orderID.Valid {
		t.OrderID = orderID.StringVal
	}
	if manifestID.Valid {
		t.ManifestID = manifestID.StringVal
	}
	if driverID.Valid {
		t.DriverID = driverID.StringVal
	}
	if vehicleID.Valid {
		t.VehicleID = vehicleID.StringVal
	}
	return t, nil
}

func scanSupplierManifestOrder(row *spanner.Row) (SupplierManifestOrderRow, error) {
	var (
		o             SupplierManifestOrderRow
		removedReason spanner.NullString
	)
	if err := row.Columns(
		&o.ManifestID, &o.OrderID, &o.SequenceIndex, &o.LoadingOrder, &o.VolumeVU, &o.State,
		&removedReason, &o.UpdatedAt,
	); err != nil {
		return SupplierManifestOrderRow{}, fmt.Errorf("scan manifest order: %w", err)
	}
	if removedReason.Valid {
		o.RemovedReason = removedReason.StringVal
	}
	return o, nil
}
