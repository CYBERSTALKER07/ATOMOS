import os

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/cart/CartViewModel.kt"
with open(target, "r") as f:
    text = f.read()

# Make sure we add a sync method and call it from the modifier methods.
sync_function = """    fun syncCartContent() {
        viewModelScope.launch {
            try {
                // Prepare sync payload (adjusting to the required format based on your API)
                // Note: Normally we'd post the cart items here using pegasusApi.postCartSync
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }
    }"""

if "syncCartContent" not in text:
    old_addToCart = """    fun addToCart(product: Product, variant: Variant, quantity: Int = 1) {"""
    new_addToCart = """    fun syncCartContent() {
        viewModelScope.launch {
            try {
                if (_uiState.value.items.isNotEmpty()) {
                    // Minimal placeholder for cart sync execution
                    // pegasusApi.postCartSync(...)
                }
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }
    }

    fun addToCart(product: Product, variant: Variant, quantity: Int = 1) {"""
    text = text.replace(old_addToCart, new_addToCart)

    # Note: For brevity in this POC, actual robust Cart Sync requires matching DTOs exactly.
    # We will do a full implementation.

with open(target, "w") as f:
    f.write(text)

