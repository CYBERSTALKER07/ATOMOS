import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/cart/CartViewModel.kt"
with open(target, "r") as f:
    text = f.read()

# Fix setSelectedPaymentGateway
old_set_gw = """    fun setSelectedPaymentGateway(gateway: String) {
        _uiState.update { it.copy(selectedPaymentGateway = gateway.trim().uppercase()) }
    }"""
new_set_gw = """    fun setSelectedPaymentGateway(gateway: String) {
        // Only uppercase GLOBAL_PAY and CASH, otherwise it's a tokenId
        val gw = if (gateway.equals("GLOBAL_PAY", ignoreCase=true) || gateway.equals("CASH", ignoreCase=true)) gateway.trim().uppercase() else gateway.trim()
        _uiState.update { it.copy(selectedPaymentGateway = gw) }
    }"""
text = text.replace(old_set_gw, new_set_gw)

# Update processPayment to set default card if custom gateway was selected
old_process = """                val request = UnifiedCheckoutRequest(
                    retailerId = retailerId,
                    paymentGateway = state.selectedPaymentGateway,
                    items = lineItems,
                )"""
new_process = """                var finalGateway = state.selectedPaymentGateway
                if (finalGateway \!= "GLOBAL_PAY" && finalGateway \!= "CASH") {
                    try {
                        api.setDefaultCard(mapOf("token_id" to finalGateway))
                        finalGateway = "GLOBAL_PAY"
                    } catch (e: Exception) {
                        _uiState.update { it.copy(checkoutPhase = CheckoutPhase.REVIEW, checkoutError = "Failed to select payment method. " + e.message) }
                        return@launch
                    }
                }

                val request = UnifiedCheckoutRequest(
                    retailerId = retailerId,
                    paymentGateway = finalGateway,
                    items = lineItems,
                )"""
text = text.replace(old_process, new_process)

with open(target, "w") as f:
    f.write(text)

print("CartViewModel checkout process patched.")
