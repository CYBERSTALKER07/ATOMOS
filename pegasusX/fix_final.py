import re

with open("apps/backend-go/warehouse/demand_products.go", "r") as f:
    content = f.read()
content = content.replace("s.repo.GetInventoryList(ctx, warehouseID)", "s.repo.GetInventoryList(r.Context(), warehouseID)")
with open("apps/backend-go/warehouse/demand_products.go", "w") as f:
    f.write(content)

with open("apps/backend-go/warehouse/ops_portal.go", "r") as f:
    content = f.read()
content = re.sub(r"\s+s\.inventory\[\"prod-2\"\].+UpdatedAt: now\}\n", "\n", content)
with open("apps/backend-go/warehouse/ops_portal.go", "w") as f:
    f.write(content)

