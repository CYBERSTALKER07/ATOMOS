import re
with open("the-lab-monorepo/apps/warehouse-app-ios/WarehouseApp/Views/Staff/StaffView.swift", "r") as f:
    content = f.read()

content = re.sub(
    r"\.alert\(\"Staff Created\"\, isPresented: Binding\([\s\S]*?\)\) \{",
    """\.alert("Staff Created", isPresented: Binding(
                get: { createdPin \!= nil },
                set: { if \!$0 { createdPin = nil } }
            )) {""",
    content
)

with open("the-lab-monorepo/apps/warehouse-app-ios/WarehouseApp/Views/Staff/StaffView.swift", "w") as f:
    f.write(content)
