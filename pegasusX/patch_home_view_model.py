import re

with open("apps/payload-app-ios/payload-app-ios/ViewModels/HomeViewModel.swift", "r") as f:
    content = f.read()

pattern = re.compile(r'guard let truckId = selectedTruckId,\n\s*let order = orders\.first\(where: \{ \$0\.orderId == orderId \}\),\n\s*canSealOrder\(orderId\) else \{ return \}\n\s*sealingOrderId = orderId\n\s*error = nil\n\s*defer \{ sealingOrderId = nil \}\n\s*do \{\n\s*let resp = try await api\.sealOrder\(orderId: orderId, terminalId: truckId\)')

replacement = r"""guard let truckId = selectedTruckId,
              let order = orders.first(where: { $0.orderId == orderId }),
              let manifestId = manifest?.manifestId,
              canSealOrder(orderId) else { return }
        sealingOrderId = orderId
        error = nil
        defer { sealingOrderId = nil }
        do {
            let resp = try await api.sealOrder(manifestId: manifestId, orderId: orderId, terminalId: truckId)"""

content = pattern.sub(replacement, content)

with open("apps/payload-app-ios/payload-app-ios/ViewModels/HomeViewModel.swift", "w") as f:
    f.write(content)
