import re

with open("apps/backend-go/payload/progress.go", "r") as f:
    content = f.read()

content = content.replace('s.payloadHub.Broadcast(context.Background(), "warehouse_ops", payload)', 
'''if whID := s.resolveWarehouseScope(r.Context()); whID != "" {
			s.payloadHub.Broadcast(context.Background(), "warehouse:"+whID, payload)
		}''')

with open("apps/backend-go/payload/progress.go", "w") as f:
    f.write(content)
