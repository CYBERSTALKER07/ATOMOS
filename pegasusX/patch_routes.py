import re

with open("apps/backend-go/payloaderoutes/routes.go", "r") as f:
    content = f.read()

content = content.replace('r.Post("/v1/auth/payloader/login", d.Service.HandlePayloaderLogin)', 'r.With(auth.RequireDeviceCert).Post("/v1/auth/payloader/login", d.Service.HandlePayloaderLogin)')

with open("apps/backend-go/payloaderoutes/routes.go", "w") as f:
    f.write(content)
