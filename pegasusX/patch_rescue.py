import re

with open("apps/backend-go/driver/rescue.go", "r") as f:
    content = f.read()

content = content.replace('s.driverHub.Broadcast(context.Background(), "fleet_broadcast", payload)', 's.driverHub.Broadcast(context.Background(), "warehouse:"+outWarehouseID, payload)')

with open("apps/backend-go/driver/rescue.go", "w") as f:
    f.write(content)
