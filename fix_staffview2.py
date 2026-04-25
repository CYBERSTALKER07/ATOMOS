import re
with open("the-lab-monorepo/apps/warehouse-app-ios/WarehouseApp/Views/Staff/StaffView.swift", "r") as f:
    content = f.read()

content = re.sub(
    r"\\\.",
    ".",
    content
)

with open("the-lab-monorepo/apps/warehouse-app-ios/WarehouseApp/Views/Staff/StaffView.swift", "w") as f:
    f.write(content)
