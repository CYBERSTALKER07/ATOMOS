package payload

import (
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

// PersistenceSnapshot captures in-memory state to project into Spanner after a mutation.
type PersistenceSnapshot struct {
	Manifests      []ManifestRow
	ManifestOrders map[string][]ManifestOrder
	Orders         []OrderRow
	Exceptions     []ManifestException
}

func (s *Service) buildPersistenceSnapshotLocked() *PersistenceSnapshot {
	ordersCopy := append([]OrderRow(nil), s.orders...)
	manifestsCopy := append([]ManifestRow(nil), s.manifests...)
	ordersMap := make(map[string][]ManifestOrder, len(s.manifestOrders))
	for k, v := range s.manifestOrders {
		copied := append([]ManifestOrder(nil), v...)
		ordersMap[k] = copied
	}
	exceptionsCopy := append([]ManifestException(nil), s.exceptions...)
	return &PersistenceSnapshot{
		Manifests:      manifestsCopy,
		ManifestOrders: ordersMap,
		Orders:         ordersCopy,
		Exceptions:     exceptionsCopy,
	}
}

func supplierBatchFromSnapshot(supplierID string, snap *PersistenceSnapshot) *manifest.SupplierWriteBatch {
	if snap == nil {
		return &manifest.SupplierWriteBatch{}
	}
	batch := &manifest.SupplierWriteBatch{
		Manifests: make([]manifest.SupplierTruckRow, 0, len(snap.Manifests)),
	}
	for _, m := range snap.Manifests {
		batch.Manifests = append(batch.Manifests, supplierTruckFromManifestRow(supplierID, m))
	}
	for manifestID, rows := range snap.ManifestOrders {
		for i, mo := range rows {
			batch.Orders = append(batch.Orders, supplierOrderFromManifestOrder(manifestID, mo, int64(i+1)))
		}
	}
	// Project order patches for payload state durability (Gap 5 correction).
	for _, o := range snap.Orders {
		updatedAt, _ := time.Parse(time.RFC3339Nano, o.UpdatedAt)

		// Find driver ID from manifests if possible
		var driverID string
		for _, m := range snap.Manifests {
			if m.ManifestID == o.ManifestID {
				driverID = m.DriverID
				break
			}
		}

		batch.OrderPatches = append(batch.OrderPatches, manifest.OrderPatch{
			OrderID:    o.OrderID,
			Status:     o.Status,
			ManifestID: o.ManifestID,
			RouteID:    o.RouteID,
			VehicleID:  o.VehicleID,
			DriverID:   driverID,
			UpdatedAt:  updatedAt,
		})
	}
	for _, e := range snap.Exceptions {
		createdAt, _ := time.Parse(time.RFC3339Nano, e.CreatedAt)
		row := manifest.SupplierExceptionRow{
			ExceptionID:  e.ExceptionID,
			OrderID:      e.OrderID,
			ManifestID:   e.ManifestID,
			SupplierID:   supplierID,
			Reason:       e.Reason,
			Metadata:     e.Metadata,
			AttemptCount: e.AttemptCount,
			CreatedAt:    createdAt,
		}
		if e.Escalated {
			now := createdAt
			if now.IsZero() {
				now = time.Now().UTC()
			}
			row.EscalatedAt = &now
		}
		batch.Exceptions = append(batch.Exceptions, row)
	}
	return batch
}

func supplierTruckFromManifestRow(supplierID string, m ManifestRow) manifest.SupplierTruckRow {
	createdAt, _ := time.Parse(time.RFC3339Nano, m.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, m.UpdatedAt)
	row := manifest.SupplierTruckRow{
		ManifestID:    m.ManifestID,
		SupplierID:    supplierID,
		RouteID:       routeIDForManifest(m),
		TruckID:       m.VehicleID,
		DriverID:      m.DriverID,
		State:         m.State,
		TotalVolumeVU: float64(m.TotalVolumeVU),
		MaxVolumeVU:   float64(m.MaxVolumeVU),
		StopCount:     int64(m.StopCount),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
	if t, err := time.Parse(time.RFC3339Nano, m.LoadingStartedAt); err == nil && !t.IsZero() {
		row.LoadingStartedAt = &t
	}
	if t, err := time.Parse(time.RFC3339Nano, m.SealedAt); err == nil && !t.IsZero() {
		row.SealedAt = &t
	}
	return row
}

func supplierOrderFromManifestOrder(manifestID string, mo ManifestOrder, seq int64) manifest.SupplierManifestOrderRow {
	updatedAt, _ := time.Parse(time.RFC3339Nano, mo.UpdatedAt)
	return manifest.SupplierManifestOrderRow{
		ManifestID:    manifestID,
		OrderID:       mo.OrderID,
		SequenceIndex: seq,
		LoadingOrder:  seq,
		VolumeVU:      float64(mo.VolumeVU),
		State:         mo.State,
		RemovedReason: mo.Reason,
		UpdatedAt:     updatedAt,
	}
}

func manifestRowFromSupplierTruck(m manifest.SupplierTruckRow) ManifestRow {
	row := ManifestRow{
		ManifestID:    m.ManifestID,
		VehicleID:     m.TruckID,
		DriverID:      m.DriverID,
		State:         m.State,
		TotalVolumeVU: int64(m.TotalVolumeVU),
		MaxVolumeVU:   int64(m.MaxVolumeVU),
		StopCount:     int(m.StopCount),
		CreatedAt:     m.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     m.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if m.LoadingStartedAt != nil {
		row.LoadingStartedAt = m.LoadingStartedAt.UTC().Format(time.RFC3339Nano)
	}
	if m.SealedAt != nil {
		row.SealedAt = m.SealedAt.UTC().Format(time.RFC3339Nano)
	}
	return row
}

func manifestOrderFromSupplierRow(o manifest.SupplierManifestOrderRow) ManifestOrder {
	return ManifestOrder{
		ManifestID: o.ManifestID,
		OrderID:    o.OrderID,
		State:      o.State,
		VolumeVU:   int64(o.VolumeVU),
		Reason:     o.RemovedReason,
		UpdatedAt:  o.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
