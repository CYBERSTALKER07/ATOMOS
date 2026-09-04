import re

with open("apps/backend-go/auth/route_guard.go", "r") as f:
    content = f.read()

content = content.replace('import "github.com/go-chi/chi/v5"', 'import (\n\t"net/http"\n\n\t"github.com/go-chi/chi/v5"\n)')

with open("apps/backend-go/auth/route_guard.go", "w") as f:
    f.write(content)
