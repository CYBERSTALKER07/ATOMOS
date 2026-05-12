import re

with open("pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Screens/SupplierProductsView.swift", "r") as f:
    text = f.read()

old_toggle = """    private func toggleMySupplier() async {
        isTogglingMySupplier = true
        do {
            let path = isMySupplier ? "/v1/retailer/suppliers/\\(supplier.id)/add" : "/v1/retailer/suppliers/\\(supplier.id)/remove"
            let action = isMySupplier ? "add" : "remove"
            let _: [String: Bool] = try await api.post(
                path: path,
                body: ["supplier_id": supplier.id],
                headers: ["Idempotency-Key": "retailer-supplier-\\(action):\\(supplier.id)"]
            )
        } catch {
            withAnimation(AnimationConstants.express) { isMySupplier.toggle() }
        }
        isTogglingMySupplier = false
    }"""

new_toggle = """    private func toggleMySupplier() async {
        isTogglingMySupplier = true
        do {
            if isMySupplier {
                try await api.favoriteSupplier(supplierId: supplier.id)
            } else {
                try await api.unfavoriteSupplier(supplierId: supplier.id)
            }
        } catch {
            withAnimation(AnimationConstants.express) { isMySupplier.toggle() }
        }
        isTogglingMySupplier = false
    }"""

text = text.replace(old_toggle, new_toggle)

with open("pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Screens/SupplierProductsView.swift", "w") as f:
    f.write(text)

