import re

with open("apps/backend-go/warehouse/demand_products.go", "r") as f:
    content = f.read()

# Replace:
# 	for sku, row := range s.inventory {
#		appendRow(sku, row.ProductName, int64(row.Quantity))
#	}
#	for _, product := range s.products {
#		if !product.IsActive {
#			continue
#		}
#		stock := int64(0)
#		if inv, ok := s.inventory[product.ProductID]; ok {
#			stock = int64(inv.Quantity)
#		}
#		appendRow(product.ProductID, product.Name, stock)
#	}

# We can query inventoryList from repo and use it
replacement = """
	inventoryList, _ := s.repo.GetInventoryList(ctx, warehouseID)
	for sku, row := range inventoryList {
		appendRow(sku, row.ProductName, int64(row.Quantity))
	}
	s.mu.RLock()
	for _, product := range s.products {
		if !product.IsActive {
			continue
		}
		stock := int64(0)
		if inv, ok := inventoryList[product.ProductID]; ok {
			stock = int64(inv.Quantity)
		}
		appendRow(product.ProductID, product.Name, stock)
	}
	s.mu.RUnlock()
"""

content = re.sub(
    r"\tfor sku, row := range s\.inventory \{[\s\S]+?appendRow\(product\.ProductID, product\.Name, stock\)\n\t\}",
    replacement,
    content
)

with open("apps/backend-go/warehouse/demand_products.go", "w") as f:
    f.write(content)

