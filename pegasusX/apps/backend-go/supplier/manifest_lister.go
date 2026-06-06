package supplier

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

// ManifestLister adapts supplier manifest projections for the payload service.
type ManifestLister struct {
	Service *Service
}

// ListPortalManifests implements payload.PortalManifestLister.
func (l *ManifestLister) ListPortalManifests(ctx context.Context, supplierID string) ([]manifest.PortalRow, error) {
	if l == nil || l.Service == nil {
		return nil, nil
	}
	rows, err := l.Service.listSupplierManifests(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	out := make([]manifest.PortalRow, len(rows))
	for i := range rows {
		out[i] = manifest.PortalRow{
			ManifestID:   rows[i].ManifestID,
			Status:       rows[i].Status,
			OrdersCount:  rows[i].OrdersCount,
			DriverID:     rows[i].DriverID,
			DriverName:   rows[i].DriverName,
			VehiclePlate: rows[i].VehiclePlate,
			VehicleID:    rows[i].VehicleID,
			TotalVu:      rows[i].TotalVu,
			UpdatedAt:    rows[i].UpdatedAt,
		}
	}
	return out, nil
}
