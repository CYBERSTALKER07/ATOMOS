import re

with open("apps/backend-go/events/events.go", "r") as f:
    content = f.read()

content = content.replace('EventOrderReassigned            = "ORDER_REASSIGNED"\n', 'EventOrderReassigned            = "ORDER_REASSIGNED"\n\tEventReassignHandshakeCompleted = "REASSIGN_HANDSHAKE_COMPLETED"\n')

with open("apps/backend-go/events/events.go", "w") as f:
    f.write(content)
