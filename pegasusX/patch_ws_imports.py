import re

with open("apps/backend-go/ws/connection.go", "r") as f:
    content = f.read()

content = content.replace('import (\n\t"context"\n\t"sync"\n\t"time"', 'import (\n\t"context"\n\t"fmt"\n\t"sync"\n\t"time"')

with open("apps/backend-go/ws/connection.go", "w") as f:
    f.write(content)
