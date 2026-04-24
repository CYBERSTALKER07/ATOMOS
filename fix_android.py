import os

replacements = [
    (
        "apps/retailer-app-ios/retailerapp/reatilerapp/Screens/CheckoutView.swift",
        'case "Click": return "CLICK"',
        ''
    ),
    (
        "apps/retailer-app-ios/retailerapp/reatilerapp/Screens/CheckoutView.swift",
        'case "Payme": return "PAYME"',
        ''
    ),
    (
        "apps/retailer-app-ios/retailerapp/reatilerapp/Screens/CheckoutView.swift",
        'default: return "CLICK"',
        'default: return "GLOBAL_PAY"'
    ),
    (
        "apps/retailer-app-ios/retailerapp/reatilerapp/Screens/DeliveryPaymentSheetView.swift",
        '.filter { ["CLICK", "PAYME", "GLOBAL_PAY"].contains($0) }',
        '.filter { ["GLOBAL_PAY", "UZCARD", "CASH"].contains($0) }'
    ),
    (
        "apps/retailer-app-ios/retailerapp/reatilerapp/Screens/DeliveryPaymentSheetView.swift",
        'let gateways = configuredGateways.isEmpty ? ["PAYME", "CLICK", "GLOBAL_PAY"] : Array(NSOrderedSet(array: configuredGateways)) as? [String] ?? configuredGateways',
        'let gateways = configuredGateways.isEmpty ? ["GLOBAL_PAY", "UZCARD", "CASH"] : Array(NSOrderedSet(array: configuredGateways)) as? [String] ?? configuredGateways'
    ),
    (
        "apps/retailer-app-ios/retailerapp/reatilerapp/Screens/DeliveryPaymentSheetView.swift",
        'case "CLICK":\n                return ("Click", "Pay via Click app", "iphone.gen1.circle.fill", .blue)',
        ''
    ),
    (
        "apps/retailer-app-ios/retailerapp/reatilerapp/Screens/DeliveryPaymentSheetView.swift",
        'case "PAYME":\n                return ("Payme", "Pay via Payme app", "iphone.gen2.circle.fill", .teal)',
        ''
    ),
    (
        "apps/retailer-app-ios/retailerapp/reatilerappTests/RetailerServiceTests.swift",
        'let gateways = ["CLICK", "PAYME", "CASH", "GLOBAL_PAY"]',
        'let gateways = ["GLOBAL_PAY", "CASH", "UZCARD"]'
    ),
    (
        "apps/retailer-app-ios/retailerapp/reatilerappTests/RetailerServiceTests.swift",
        '#expect(gateways.contains("CLICK"))',
        ''
    ),
    (
        "apps/retailer-app-ios/retailerapp/reatilerappTests/CartManagerTests.swift",
        'paymentGateway: "CLICK",',
        'paymentGateway: "GLOBAL_PAY",'
    ),
    (
        "apps/retailer-app-ios/retailerapp/reatilerappTests/CartManagerTests.swift",
        '#expect(payload.paymentGateway == "CLICK")',
        '#expect(payload.paymentGateway == "GLOBAL_PAY")'
    )
]

for file_path, old_str, new_str in replacements:
    full_path = os.path.join(".", "the-lab-monorepo", file_path)
    if os.path.exists(full_path):
        with open(full_path, "r") as f:
            content = f.read()
        
        if old_str in content:
            content = content.replace(old_str, new_str)
            with open(full_path, "w") as f:
                f.write(content)
            print(f"Updated {file_path}")
        else:
            print(f"Could not find exact string in {file_path}")
    else:
        print(f"File not found: {file_path}")
