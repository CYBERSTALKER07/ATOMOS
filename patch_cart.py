import re

with open("pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Services/CartManager.swift", "r") as f:
    text = f.read()

sync_method = """

    // MARK: - Server Sync
    func sync() async {
        guard let retailerId = AuthManager.shared.currentUser?.id else { return }
        
        let cartItems = items.map {
            CartSyncItem(productId: $0.product.id, variantId: $0.variant.id, quantity: $0.quantity)
        }
        let request = CartSyncRequest(retailerId: retailerId, items: cartItems)
        
        do {
            let response = try await APIClient.shared.syncCart(request: request)
            
            // For now, just print synced state, could update local warnings based on `response.warnings`
            print("Cart synced. Total value: \\(response.totalValue), Warnings: \\(response.warnings.count)")
        } catch {
            print("Failed to sync cart: \\(error)")
        }
    }
}
"""

text = text.replace("}\n", "}\n") # Just to make sure we're replacing the last bracket if we wanted... Wait.

# I'll just remove the last '}' and append the sync method and the ending '}'
text = text.rstrip()
if text.endswith("}"):
    text = text[:-1] + sync_method
else:
    print("Could not find ending bracket.")

with open("pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Services/CartManager.swift", "w") as f:
    f.write(text)

