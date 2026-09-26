import re

with open("apps/backend-go/order/driver_edges.go", "r") as f:
    content = f.read()

content = content.replace('IdempotencyKey: fmt.Sprintf("split-%s-cash-%d", current.OrderID, now.UnixNano()),', 'IdempotencyKey: fmt.Sprintf("split-%s-cash", current.OrderID),')
content = content.replace('IdempotencyKey: fmt.Sprintf("split-%s-card-%d", current.OrderID, now.UnixNano()),', 'IdempotencyKey: fmt.Sprintf("split-%s-card", current.OrderID),')

with open("apps/backend-go/order/driver_edges.go", "w") as f:
    f.write(content)
