import re

with open("apps/backend-go/warehouse/ops_portal.go", "r") as f:
    content = f.read()

# in handleOpsInventory GET branch:
# Replace s.mu.RLock() / for sku, row := range s.inventory { ... } / s.mu.RUnlock()
# with repo call

content = re.sub(
    r"s\.mu\.RLock\(\)\s+items := make\(\[\]map\[string\]any, 0, len\(s\.inventory\)\)\s+for sku, row := range s\.inventory \{[\s\S]+?s\.mu\.RUnlock\(\)",
    r"""inventoryList, err := s.repo.GetInventoryList(r.Context(), "wh-1")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_fetch_inventory"})
			return
		}
		items := make([]map[string]any, 0, len(inventoryList))
		for sku, row := range inventoryList {
			qty := row.Quantity
			isLow := qty < 20
			if lowOnly && !isLow {
				continue
			}
			items = append(items, map[string]any{
				"product_id":       sku,
				"sku_id":           row.SKU,
				"product_name":     row.ProductName,
				"quantity":         qty,
				"is_low_stock":     isLow,
				"last_updated":     row.UpdatedAt,
			})
		}""",
    content
)

# in handleOpsInventory PATCH branch:
content = re.sub(
    r"s\.mu\.Lock\(\)\s+key := body\.ProductID\s+if row, ok := s\.inventory\[key\]; ok \{[\s\S]+?s\.mu\.Unlock\(\)",
    r"""err := s.repo.UpdateInventoryQuantity(r.Context(), "wh-1", body.ProductID, body.Quantity, func(buf outbox.TxnBuffer) error {
			// omit event emission for simple patch if none is required
			return nil
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_update_inventory"})
			return
		}""",
    content
)

# the other s.inventory uses in ops_portal.go:
# handleOpsDashboard calculates low stock items
content = re.sub(
    r"s\.mu\.RLock\(\)\s+for _, row := range s\.inventory \{[\s\S]+?staffCount := int64\(len\(s\.staff\)\)",
    r"""inventoryList, _ := s.repo.GetInventoryList(r.Context(), "wh-1")
	for _, row := range inventoryList {
		if row.Quantity < 20 {
			lowStock++
		}
	}
	s.mu.RLock()
	staffCount := int64(len(s.staff))""",
    content
)

# remove seed from ensurePortalSeed
content = re.sub(
    r"s\.inventory\[\"prod-1\"\] = InventoryRow[\s\S]+?UpdatedAt: now\}",
    "",
    content
)

with open("apps/backend-go/warehouse/ops_portal.go", "w") as f:
    f.write(content)

