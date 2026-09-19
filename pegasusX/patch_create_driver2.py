import re

with open("apps/backend-go/driver/crud_handlers.go", "r") as f:
    content = f.read()

content = content.replace("req.PinHash = string(hash)", "h := string(hash)\n\t\t\treq.PinHash = &h")

with open("apps/backend-go/driver/crud_handlers.go", "w") as f:
    f.write(content)
