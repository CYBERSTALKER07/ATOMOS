import re

with open("apps/backend-go/driver/rescue.go", "r") as f:
    content = f.read()

pattern1 = re.compile(r'_, err := spannerRepo\.client\.ReadWriteTransaction\(r\.Context\(\), func\(ctx context\.Context, txn \*spanner\.ReadWriteTransaction\) error \{\n\t\t// Phase 1: Mark truck as needing rescue')

replacement1 = r"""var outWarehouseID string
	_, err := spannerRepo.client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Phase 1: Mark truck as needing rescue"""
content = content.replace(pattern1.pattern, replacement1)
content = re.sub(r'_, err := spannerRepo\.client\.ReadWriteTransaction\(r\.Context\(\), func\(ctx context\.Context, txn \*spanner\.ReadWriteTransaction\) error \{\n\t\t// Phase 1: Mark truck as needing rescue', replacement1, content)


pattern2 = re.compile(r'supplierID = sp\.StringVal\n\t\t\t\twarehouseID = wh\.StringVal\n\t\t\t\}\n\t\t\}')
replacement2 = r"""supplierID = sp.StringVal
				warehouseID = wh.StringVal
				outWarehouseID = warehouseID
			}
		}"""
content = content.replace(pattern2.pattern, replacement2)
content = re.sub(r'supplierID = sp\.StringVal\n\t\t\t\twarehouseID = wh\.StringVal\n\t\t\t\}\n\t\t\}', replacement2, content)

pattern3 = re.compile(r'\t\t// Admin setting stub\n\t\tautoActionAI := false // In production, this would query WarehouseSettings\n\n\t\tif autoActionAI \{\n\t\t\t// Phase 3: AI Action -> Find nearest driver via GEORADIUS\n\t\t\t// This is a stub for the geosearch logic\n\t\t\t// redis\.GeoRadius\(ctx, "fleet:locations", lng, lat, \.\.\. \)\n\t\t\t// Instead of broadcast, it assigns directly\.\n\t\t\t// \(Implementation simplified for brevity\)\n\t\t\} else \{\n\t\t\t// Phase 3: Broadcast Rescue -> Notify all drivers\n\t\t\tif s\.driverHub != nil \{\n\t\t\t\tpayload, _ := json\.Marshal\(map\[string\]any\{\n\t\t\t\t\t"type":           "RESCUE_BROADCAST",\n\t\t\t\t\t"rescue_id":      rescueID,\n\t\t\t\t\t"broken_driver":  driverID,\n\t\t\t\t\t"warehouse_id":   warehouseID,\n\t\t\t\t\}\)\n\t\t\t\ts\.driverHub\.Broadcast\(context\.Background\(\), "fleet_broadcast", payload\)\n\t\t\t\}\n\t\t\}')

replacement3 = r"""		// Phase 3: AI Action / Broadcast will be done post-commit."""
content = re.sub(pattern3, replacement3, content)

pattern4 = re.compile(r'	if err != nil \{\n\t\ts\.log\.Error\("failed to request rescue", "error", err\)\n\t\twriteJSON\(w, http\.StatusInternalServerError, map\[string\]string\{"error": "internal_error"\}\)\n\t\treturn\n\t\}')
replacement4 = r"""	if err != nil {
		s.log.Error("failed to request rescue", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	autoActionAI := false // In production, this would query WarehouseSettings
	if !autoActionAI {
		// Phase 3: Broadcast Rescue -> Notify all drivers (Post-Commit)
		if s.driverHub != nil {
			payload, _ := json.Marshal(map[string]any{
				"type":           "RESCUE_BROADCAST",
				"rescue_id":      rescueID,
				"broken_driver":  driverID,
				"warehouse_id":   outWarehouseID,
			})
			s.driverHub.Broadcast(context.Background(), "fleet_broadcast", payload)
		}
	}"""
content = re.sub(pattern4, replacement4, content)

with open("apps/backend-go/driver/rescue.go", "w") as f:
    f.write(content)

