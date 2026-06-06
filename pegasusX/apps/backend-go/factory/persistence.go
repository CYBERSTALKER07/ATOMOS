package factory

import (
	"errors"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

var errTransferNotFound = errors.New("transfer_not_found")

// PersistenceSnapshot captures in-memory factory manifest and transfer state for Spanner projection.
type PersistenceSnapshot struct {
	Manifests []ManifestRow
	Transfers []TransferRow
}

func (s *Service) buildPersistenceSnapshotLocked() *PersistenceSnapshot {
	return &PersistenceSnapshot{
		Manifests: append([]ManifestRow(nil), s.manifests...),
		Transfers: append([]TransferRow(nil), s.transfers...),
	}
}

func (s *Service) rebuildManifestTransfersLocked() {
	s.manifestTransfers = make(map[string][]TransferRow)
	for i := range s.transfers {
		manifestID := strings.TrimSpace(s.transfers[i].ManifestID)
		if manifestID == "" {
			continue
		}
		s.manifestTransfers[manifestID] = append(s.manifestTransfers[manifestID], s.transfers[i])
	}
}

func factoryBatchFromSnapshot(supplierID, factoryID string, snap *PersistenceSnapshot) *manifest.FactoryWriteBatch {
	if snap == nil {
		return &manifest.FactoryWriteBatch{}
	}
	batch := &manifest.FactoryWriteBatch{
		Manifests: make([]manifest.FactoryTruckRow, 0, len(snap.Manifests)),
		Transfers: make([]manifest.FactoryTransferRow, 0, len(snap.Transfers)),
	}
	for _, m := range snap.Manifests {
		batch.Manifests = append(batch.Manifests, factoryTruckFromManifestRow(supplierID, factoryID, m))
	}
	for _, t := range snap.Transfers {
		batch.Transfers = append(batch.Transfers, factoryTransferFromRow(supplierID, factoryID, t))
	}
	return batch
}

func factoryTransferFromRow(supplierID, factoryID string, t TransferRow) manifest.FactoryTransferRow {
	createdAt, _ := time.Parse(time.RFC3339Nano, t.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, t.UpdatedAt)
	return manifest.FactoryTransferRow{
		TransferID:     t.TransferID,
		FactoryID:      factoryID,
		SupplierID:     supplierID,
		OrderID:        t.OrderID,
		ManifestID:     t.ManifestID,
		State:          t.State,
		TotalVolumeVU:  float64(t.TotalVU),
		DriverID:       t.DriverID,
		VehicleID:      t.VehicleID,
		ReassignDepth:  int64(t.ReassignDepth),
		ExceptionCount: t.ExceptionCount,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}

func transferRowFromFactoryTransfer(t manifest.FactoryTransferRow) TransferRow {
	return TransferRow{
		TransferID:     t.TransferID,
		OrderID:        t.OrderID,
		ManifestID:     t.ManifestID,
		State:          t.State,
		TotalVU:        int64(t.TotalVolumeVU),
		DriverID:       t.DriverID,
		VehicleID:      t.VehicleID,
		ReassignDepth:  int(t.ReassignDepth),
		ExceptionCount: t.ExceptionCount,
		CreatedAt:      t.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:      t.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func factoryTruckFromManifestRow(supplierID, factoryID string, m ManifestRow) manifest.FactoryTruckRow {
	createdAt, _ := time.Parse(time.RFC3339Nano, m.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, m.UpdatedAt)
	row := manifest.FactoryTruckRow{
		ManifestID:    m.ManifestID,
		FactoryID:     factoryID,
		SupplierID:    supplierID,
		DriverID:      m.DriverID,
		VehicleID:     m.VehicleID,
		State:         m.State,
		TotalVolumeVU: float64(m.TotalVolumeVU),
		MaxVolumeVU:   float64(m.MaxVolumeVU),
		StopCount:     int64(m.TransferCnt),
		TransferCount: int64(m.TransferCnt),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
	if t, err := time.Parse(time.RFC3339Nano, m.LoadingStartedAt); err == nil && !t.IsZero() {
		row.LoadingStartedAt = &t
	}
	if t, err := time.Parse(time.RFC3339Nano, m.SealedAt); err == nil && !t.IsZero() {
		row.SealedAt = &t
	}
	if t, err := time.Parse(time.RFC3339Nano, m.DispatchedAt); err == nil && !t.IsZero() {
		row.DispatchedAt = &t
	}
	if t, err := time.Parse(time.RFC3339Nano, m.CompletedAt); err == nil && !t.IsZero() {
		row.CompletedAt = &t
	}
	if t, err := time.Parse(time.RFC3339Nano, m.CancelledAt); err == nil && !t.IsZero() {
		row.CancelledAt = &t
	}
	return row
}

func manifestRowFromFactoryTruck(m manifest.FactoryTruckRow) ManifestRow {
	row := ManifestRow{
		ManifestID:    m.ManifestID,
		State:         m.State,
		TransferCnt:   int(m.TransferCount),
		TotalVolumeVU: int64(m.TotalVolumeVU),
		MaxVolumeVU:   int64(m.MaxVolumeVU),
		DriverID:      m.DriverID,
		VehicleID:     m.VehicleID,
		CreatedAt:     m.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     m.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if m.LoadingStartedAt != nil {
		row.LoadingStartedAt = m.LoadingStartedAt.UTC().Format(time.RFC3339Nano)
	}
	if m.SealedAt != nil {
		row.SealedAt = m.SealedAt.UTC().Format(time.RFC3339Nano)
	}
	if m.DispatchedAt != nil {
		row.DispatchedAt = m.DispatchedAt.UTC().Format(time.RFC3339Nano)
	}
	if m.CompletedAt != nil {
		row.CompletedAt = m.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	if m.CancelledAt != nil {
		row.CancelledAt = m.CancelledAt.UTC().Format(time.RFC3339Nano)
	}
	return row
}
