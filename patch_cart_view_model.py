import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/cart/CartViewModel.kt"
with open(target, "r") as f:
    text = f.read()

# Add import for CheckoutPaymentOption
if "import com.pegasus.retailer.ui.components.CheckoutPaymentOption" not in text:
    text = text.replace("import com.pegasus.retailer.ui.components.CheckoutPhase", "import com.pegasus.retailer.ui.components.CheckoutPhase\nimport com.pegasus.retailer.ui.components.CheckoutPaymentOption")

# Add paymentOptions to CartUiState
if "val paymentOptions: List<CheckoutPaymentOption>" not in text:
    text = text.replace("val oosItems: List<String> = emptyList(),", "val oosItems: List<String> = emptyList(),\n    val paymentOptions: List<CheckoutPaymentOption> = emptyList(),")

# Replace checkoutPaymentLabel to use paymentOptions
old_label_func = """private fun checkoutPaymentLabel(gateway: String): String {
    return when (gateway.trim().uppercase()) {
        
        "GLOBAL_PAY" -> "GlobalPay"
        "GLOBAL_PAY" -> "GlobalPay"
        "CASH" -> "Cash on Delivery"
        else -> "Cash"
    }
}"""
new_label_func = """private fun checkoutPaymentLabel(gateway: String, options: List<CheckoutPaymentOption>): String {
    return options.find { it.gateway == gateway }?.label ?: when (gateway.toUpperCase()) {
        "GLOBAL_PAY" -> "Global Pay"
        "CASH" -> "Cash on Delivery"
        else -> gateway
    }
}"""
text = text.replace(old_label_func, new_label_func)

# Fix selectedPaymentLabel to pass options
text = text.replace("val selectedPaymentLabel: String get() = checkoutPaymentLabel(selectedPaymentGateway)", "val selectedPaymentLabel: String get() = checkoutPaymentLabel(selectedPaymentGateway, paymentOptions)")

# In CartViewModel, when initialized or when loaded, fetch cards
fetch_cards_code = """
    init { 
        flushPendingOrders()
        fetchPaymentOptions()
    }

    private fun fetchPaymentOptions() = viewModelScope.launch {
        try {
            val cardsResp = api.getSavedCards()
            val dynamicOptions = cardsResp.cards.map { card ->
                CheckoutPaymentOption(gateway = card.tokenId, label = "•••• " + card.panMask.takeLast(4))
            }
            val standardOptions = listOf(
                CheckoutPaymentOption(gateway = "GLOBAL_PAY", label = "GlobalPay (New)"),
                CheckoutPaymentOption(gateway = "CASH", label = "Cash on Delivery")
            )
            _uiState.update { it.copy(paymentOptions = dynamicOptions + standardOptions) }
        } catch (e: Exception) {
            val standardOptions = listOf(
                CheckoutPaymentOption(gateway = "GLOBAL_PAY", label = "GlobalPay (New)"),
                CheckoutPaymentOption(gateway = "CASH", label = "Cash on Delivery")
            )
            _uiState.update { it.copy(paymentOptions = standardOptions) }
        }
    }
"""

text = text.replace("    init { flushPendingOrders() }", fetch_cards_code.strip())

with open(target, "w") as f:
    f.write(text)

print("CartViewModel patched.")
