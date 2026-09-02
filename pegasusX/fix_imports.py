import re

with open("apps/backend-go/factory/repository_spanner.go", "r") as f:
    content = f.read()

imports = """
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
"""

if "stocklots" not in content:
    content = content.replace('"time"', '"time"\n' + imports)

with open("apps/backend-go/factory/repository_spanner.go", "w") as f:
    f.write(content)

