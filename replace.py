import re

with open("/Users/shakhzod/Desktop/V.O.I.D/the-lab-monorepo/apps/retailer-app-ios/retailerapp/reatilerapp/Models/Order.swift", "r") as f:
    text = f.read()

with open("patch_order.swift", "r") as f:
    patch = f.read()

# Replace from `// MARK: - Order Line Item` to `// MARK: - Tracking Order (for delivery map)`
pattern = r'// MARK: - Order Line Item.*?// MARK: - Tracking Order \(for delivery map\)'
replaced = re.sub(pattern, patch + "\n// MARK: - Tracking Order (for delivery map)", text, flags=re.DOTALL)

with open("/Users/shakhzod/Desktop/V.O.I.D/the-lab-monorepo/apps/retailer-app-ios/retailerapp/reatilerapp/Models/Order.swift", "w") as f:
    f.write(replaced)

print("Replaced\!")
