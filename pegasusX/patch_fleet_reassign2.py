import re

with open("apps/backend-go/payload/fleet_compat.go", "r") as f:
    content = f.read()

pattern = re.compile(r'oldRouteID := s\.orders\[oIdx\]\.RouteID\n\t\t\tvol := s\.orders\[oIdx\]\.VolumeVU\n\t\t\t\n\t\t\toldMIdx := s\.findManifestIndexLocked\(oldRouteID\)\n\t\t\tnewMIdx := s\.findManifestIndexLocked\(req\.NewRouteID\)\n\t\t\t\n\t\t\tif err := tx\.UpdateOrderAssignment\(ctx, orderID, req\.NewRouteID, driverID\); err != nil \{\n\t\t\t\treturn err\n\t\t\t\}\n\t\t\t\n\t\t\tif oldMIdx >= 0 \{\n\t\t\t\ts\.manifests\[oldMIdx\]\.TotalVolumeVU -= vol\n\t\t\t\ts\.manifests\[oldMIdx\]\.UpdatedAt = now\n\t\t\t\t_ = tx\.SaveManifest\(ctx, s\.manifests\[oldMIdx\]\)\n\t\t\t\t_ = tx\.DeleteManifestOrder\(ctx, oldRouteID, orderID\)\n\t\t\t\}\n\t\t\tif newMIdx >= 0 \{\n\t\t\t\ts\.manifests\[newMIdx\]\.TotalVolumeVU \+= vol\n\t\t\t\ts\.manifests\[newMIdx\]\.UpdatedAt = now\n\t\t\t\t_ = tx\.SaveManifest\(ctx, s\.manifests\[newMIdx\]\)\n\t\t\t\t\n\t\t\t\tmo := ManifestOrder\{\n\t\t\t\t\tManifestID: req\.NewRouteID,\n\t\t\t\t\tOrderID:    orderID,\n\t\t\t\t\tState:      s\.orders\[oIdx\]\.State,\n\t\t\t\t\tVolumeVU:   vol,\n\t\t\t\t\tUpdatedAt:  now,\n\t\t\t\t\}\n\t\t\t\t_ = tx\.SaveManifestOrder\(ctx, mo, time\.Now\(\)\.UnixNano\(\)\)\n\t\t\t\}')

replacement = r"""oldRouteID := s.orders[oIdx].RouteID
			
			oldMIdx := s.findManifestIndexLocked(oldRouteID)
			newMIdx := s.findManifestIndexLocked(req.NewRouteID)
			
			// Find volume from ManifestOrders
			var vol int64
			oldOrders, _ := tx.ListManifestOrders(ctx, oldRouteID)
			for _, mo := range oldOrders {
				if mo.OrderID == orderID {
					vol = mo.VolumeVU
					break
				}
			}
			
			if err := tx.UpdateOrderAssignment(ctx, orderID, req.NewRouteID, driverID); err != nil {
				return err
			}
			
			if oldMIdx >= 0 {
				s.manifests[oldMIdx].TotalVolumeVU -= vol
				if s.manifests[oldMIdx].TotalVolumeVU < 0 {
					s.manifests[oldMIdx].TotalVolumeVU = 0
				}
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
					State:      s.orders[oIdx].Status,
					VolumeVU:   vol,
					UpdatedAt:  now,
				}
				_ = tx.SaveManifestOrder(ctx, mo, time.Now().UnixNano())
			}"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/payload/fleet_compat.go", "w") as f:
    f.write(content)
