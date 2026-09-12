import re

with open("apps/backend-go/stocklots/load_ledger.go", "r") as f:
    content = f.read()

pattern = re.compile(r'"fmt"\n\t"strings"')
replacement = r""""fmt"
	"os"
	"strings" """
content = pattern.sub(replacement, content)

with open("apps/backend-go/stocklots/load_ledger.go", "w") as f:
    f.write(content)
