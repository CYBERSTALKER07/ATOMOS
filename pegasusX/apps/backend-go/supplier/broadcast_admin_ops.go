package supplier

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

// broadcastAdminMessage fans a supplier admin broadcast to role-scoped WS rooms.
// Always delivers to the supplier hub; additional hubs depend on targetRole (ALL fans everywhere).
func (s *Service) broadcastAdminMessage(ctx context.Context, supplierID, targetRole string, payload []byte) {
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" || len(payload) == 0 {
		return
	}
	role := strings.ToUpper(strings.TrimSpace(targetRole))
	if role == "" {
		role = "ALL"
	}

	if s.portalSupplierHub != nil {
		s.portalSupplierHub.Broadcast(ctx, "supplier:"+supplierID, payload)
	}

	fanAll := role == "ALL"
	if fanAll || role == "RETAILER" {
		if s.portalRetailerHub != nil {
			s.portalRetailerHub.Broadcast(ctx, ws.SupplierPromoRoom(supplierID), payload)
		}
	}
	if fanAll || role == "PAYLOAD" {
		if s.portalPayloadHub != nil {
			s.portalPayloadHub.Broadcast(ctx, "payload:"+supplierID, payload)
		}
	}

	if s.repo == nil {
		return
	}
	if fanAll || role == "DRIVER" {
		if s.portalDriverHub != nil {
			if drivers, err := s.repo.ListFleetDrivers(ctx, supplierID); err == nil {
				seen := make(map[string]struct{}, len(drivers))
				for _, d := range drivers {
					driverID := strings.TrimSpace(d.DriverID)
					if driverID == "" {
						continue
					}
					if _, ok := seen[driverID]; ok {
						continue
					}
					seen[driverID] = struct{}{}
					s.portalDriverHub.Broadcast(ctx, "driver:"+driverID, payload)
				}
			}
		}
	}
	if fanAll || role == "WAREHOUSE" || role == "FACTORY" {
		topology, err := s.repo.GetTopology(ctx, supplierID)
		if err != nil {
			return
		}
		if fanAll || role == "WAREHOUSE" {
			if s.portalWarehouseHub != nil {
				for _, wh := range topology.Warehouses {
					whID := strings.TrimSpace(wh.WarehouseID)
					if whID == "" {
						continue
					}
					s.portalWarehouseHub.Broadcast(ctx, "warehouse:"+whID, payload)
				}
			}
		}
		if fanAll || role == "FACTORY" {
			if s.portalFactoryHub != nil {
				for _, fc := range topology.Factories {
					fcID := strings.TrimSpace(fc.FactoryID)
					if fcID == "" {
						continue
					}
					s.portalFactoryHub.Broadcast(ctx, "factory:"+fcID, payload)
				}
			}
		}
	}
}
