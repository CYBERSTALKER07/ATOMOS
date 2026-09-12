import re

with open("apps/backend-go/ws/connection.go", "r") as f:
    content = f.read()

content = content.replace('import (\n\t"context"', 'import (\n\t"context"\n\t"fmt"')

with open("apps/backend-go/ws/connection.go", "w") as f:
    f.write(content)
