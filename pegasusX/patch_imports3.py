import re

with open("apps/backend-go/payload/ship_units.go", "r") as f:
    content = f.read()

pattern = re.compile(r'"github\.com/google/uuid"')
replacement = r""""github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox" """
content = pattern.sub(replacement, content)

with open("apps/backend-go/payload/ship_units.go", "w") as f:
    f.write(content)
