import re

# Fix analytics/revenue.go
path1 = "apps/backend-go/analytics/revenue.go"
with open(path1, "r") as f:
    text = f.read()

# 1. Remove duplicate 'Cash int64  `json:"cash"`' struct field
parts = text.split('Cash int64  `json:"cash"`')
if len(parts) >= 3:
    # There are two occurrences. Keep the surrounding text but remove one.
    text = parts[0] + 'Cash int64  `json:"cash"`' + parts[2]

# 2. Remove duplicate 'CAST(... CASH ...) as Cash,' in SQL
select_line = "CAST(SUM(CASE WHEN o.PaymentGateway = 'CASH' THEN o.Amount ELSE 0 END) AS INT64) as Cash,"
parts = text.split(select_line)
if len(parts) >= 2:
    text = parts[0] + parts[1] # just remove the first one

# 3. Remove duplicate 'var cash sql.NullInt64' block
# Actually, the error was "cash redeclared in this block", so we just find `var cash sql.NullInt64` which is probably twice because of click->cash
parts = text.split('var cash sql.NullInt64')
if len(parts) >= 3:
    text = parts[0] + 'var cash sql.NullInt64' + parts[2]

# 4. Remove duplicate `b.Cash = cash.Int64`
parts = text.split('b.Cash = cash.Int64')
if len(parts) >= 3:
    text = parts[0] + 'b.Cash = cash.Int64' + parts[2]

with open(path1, "w") as f:
    f.write(text)

# Fix vault/capabilities.go
path2 = "apps/backend-go/vault/capabilities.go"
with open(path2, "r") as f:
    text = f.read()

# There are two '"GLOBAL_PAY": {' entries. Let's find the first one and the second one.
keys = text.split('"GLOBAL_PAY": {')
if len(keys) == 3:
    # We want to remove the second block completely until the next top-level key or end of dict.
    # The first block ends with `},` and then some whitespace and the next key.
    # We can just remove the second one.
    import re
    # match `"GLOBAL_PAY": { ... },`
    text = re.sub(r'"GLOBAL_PAY": \{.*?\n\s+\},', '', text, count=1, flags=re.DOTALL)

with open(path2, "w") as f:
    f.write(text)
print("Done fixing syntax")
