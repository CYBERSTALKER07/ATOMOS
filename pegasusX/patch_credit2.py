import re

with open("apps/backend-go/credit/repository.go", "r") as f:
    content = f.read()

content = content.replace("ErrProfileNotActive", "ErrCreditNotEnabled")
content = content.replace("ErrInsufficientCredit", "ErrLimitBreached")

with open("apps/backend-go/credit/repository.go", "w") as f:
    f.write(content)

with open("apps/backend-go/credit/service.go", "r") as f:
    content = f.read()

# add imports
if '"errors"' not in content:
    content = content.replace('import (', 'import (\n\t"errors"\n\t"cloud.google.com/go/spanner"\n', 1)

with open("apps/backend-go/credit/service.go", "w") as f:
    f.write(content)

