import re

with open('apps/backend-go/order/warehouse_ops.go', 'r') as f:
    content = f.read()

old_block = """	return s.warehouseTransition(ctx, resolvedOps, orderID, reason, func(current *Order) (Status, error) {
		switch current.Status {
		case StatusPending, StatusLoaded:
			return StatusDelayed, nil
		default:
			return "", fmt.Errorf("%w: %s cannot be delayed", ErrInvalidStatusTransition, current.Status)
		}
	}, false, delayMeta)"""

new_block = """	return s.warehouseTransition(ctx, resolvedOps, orderID, reason, func(current *Order) (Status, error) {
		switch current.Status {
		case StatusPending, StatusLoaded:
			return StatusDelayed, nil
		default:
			return "", fmt.Errorf("%w: %s cannot be delayed", ErrErrInvalidStatusTransition, current.Status)
		}
	}, true, delayMeta)"""

# I need to fix the typo ErrErrInvalidStatusTransition
new_block = """	return s.warehouseTransition(ctx, resolvedOps, orderID, reason, func(current *Order) (Status, error) {
		switch current.Status {
		case StatusPending, StatusLoaded:
			return StatusDelayed, nil
		default:
			return "", fmt.Errorf("%w: %s cannot be delayed", ErrInvalidStatusTransition, current.Status)
		}
	}, true, delayMeta)"""

if old_block in content:
    content = content.replace(old_block, new_block)
    with open('apps/backend-go/order/warehouse_ops.go', 'w') as f:
        f.write(content)
    print("Patched WarehouseMarkDelayed successfully.")
else:
    print("Could not find the target block to replace.")

