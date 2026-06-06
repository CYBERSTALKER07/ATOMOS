import re

with open("apps/backend-go/order/repository_spanner.go", "r") as f:
    content = f.read()

content = content.replace("strings.Join(orderSelectColumns, \", \")", "orderSelectColumns")
content = content.replace("scanOrderRow(row)", "scanOrderRowRow(row)")

with open("apps/backend-go/order/repository_spanner.go", "w") as f:
    f.write(content)

