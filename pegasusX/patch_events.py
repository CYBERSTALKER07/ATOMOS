import re

with open("apps/backend-go/events/events.go", "r") as f:
    content = f.read()

events_to_add = """
	EventWarehouseTransferCreated     = "WAREHOUSE_TRANSFER_CREATED"
	EventWarehouseTransferReceived    = "WAREHOUSE_TRANSFER_RECEIVED"
"""

if "EventWarehouseTransferCreated" not in content:
    content = content.replace('EventWarehouseDispatchLockChanged = "WAREHOUSE_DISPATCH_LOCK_CHANGED"', 'EventWarehouseDispatchLockChanged = "WAREHOUSE_DISPATCH_LOCK_CHANGED"\n' + events_to_add)

with open("apps/backend-go/events/events.go", "w") as f:
    f.write(content)

