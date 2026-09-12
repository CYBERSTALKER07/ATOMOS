import re

with open("apps/backend-go/payload/fleet_compat.go", "r") as f:
    content = f.read()

pattern = re.compile(r'(\t\t\tdriverID := s\.driverIDForRouteLocked\(req\.NewRouteID\)\n\t\t\tif err := tx\.UpdateOrderAssignment\(ctx, orderID, req\.NewRouteID, driverID\); err != nil \{\n\t\t\t\treturn err\n\t\t\t\}\n\t\t\ts\.orders\[oIdx\]\.RouteID = req\.NewRouteID\n\t\t\ts\.orders\[oIdx\]\.UpdatedAt = now\n\t\t\treassigned\+\+)', re.DOTALL)

replacement = r"""			driverID := s.driverIDForRouteLocked(req.NewRouteID)
			
			// Adjust Manifests
			oldRouteID := s.orders[oIdx].RouteID
			vol := s.orders[oIdx].VolumeVU
			
			oldMIdx := s.findManifestIndexLocked(oldRouteID)
			newMIdx := s.findManifestIndexLocked(req.NewRouteID)
			
			if err := tx.UpdateOrderAssignment(ctx, orderID, req.NewRouteID, driverID); err != nil {
				return err
			}
			
			if oldMIdx >= 0 {
				s.manifests[oldMIdx].TotalVolumeVU -= vol
				s.manifests[oldMIdx].UpdatedAt = now
				_ = tx.SaveManifest(ctx, s.manifests[oldMIdx])
				_ = tx.DeleteManifestOrder(ctx, oldRouteID, orderID)
			}
			if newMIdx >= 0 {
				s.manifests[newMIdx].TotalVolumeVU += vol
				s.manifests[newMIdx].UpdatedAt = now
				_ = tx.SaveManifest(ctx, s.manifests[newMIdx])
				
				mo := ManifestOrder{
					ManifestID: req.NewRouteID,
					OrderID:    orderID,
					State:      s.orders[oIdx].State,
					VolumeVU:   vol,
					UpdatedAt:  now,
				}
				_ = tx.SaveManifestOrder(ctx, mo, time.Now().UnixNano())
			}

			s.orders[oIdx].RouteID = req.NewRouteID
			s.orders[oIdx].UpdatedAt = now
			reassigned++"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/payload/fleet_compat.go", "w") as f:
    f.write(content)
