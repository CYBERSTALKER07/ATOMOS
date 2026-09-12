import re

with open("apps/backend-go/order/reassign_handshake.go", "r") as f:
    content = f.read()

content = content.replace('s.client.ReadWriteTransaction', 's.spannerClient.ReadWriteTransaction')

with open("apps/backend-go/order/reassign_handshake.go", "w") as f:
    f.write(content)
