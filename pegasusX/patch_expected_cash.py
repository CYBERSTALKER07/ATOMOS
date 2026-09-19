import re

with open("apps/backend-go/cashrecon/expected_cash.go", "r") as f:
    content = f.read()

content = content.replace("AND o.CreatedAt >= @start\n  AND o.CreatedAt < @end", "AND pl.CapturedAt >= @start\n  AND pl.CapturedAt < @end")

with open("apps/backend-go/cashrecon/expected_cash.go", "w") as f:
    f.write(content)
