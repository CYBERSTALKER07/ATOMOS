import re

with open('apps/backend-go/order/warehouse_ops.go', 'r') as f:
    content = f.read()

old_block = """	current.LineItems = lineItems
	current.TotalMinor = total"""

new_block = """	if current.OriginalTotalMinor == 0 {
		current.OriginalTotalMinor = current.TotalMinor
	}
	current.LineItems = lineItems
	current.TotalMinor = total"""

if old_block in content:
    content = content.replace(old_block, new_block)
    with open('apps/backend-go/order/warehouse_ops.go', 'w') as f:
        f.write(content)
    print("Patched WarehouseEditPreorder successfully.")
else:
    print("Could not find the target block to replace.")

