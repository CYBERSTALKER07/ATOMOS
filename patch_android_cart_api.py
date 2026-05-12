import os

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/cart/CartViewModel.kt"
with open(target, "r") as f:
    text = f.read()

# We need to map uiState.items into a payload for postCartSync. 
# In PegasusApi.kt postCartSync takes a body.
old_sync = """    fun syncCartContent() {
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
    }"""
    
new_sync = """    fun syncCartContent() {
        viewModelScope.launch {
            try {
                val itemsPayload = _uiState.value.items.map {
                    mapOf(
                        "sku_id" to it.product.id,
                        "variant_id" to it.variant.id,
                        "quantity" to it.quantity
                    )
                }
                pegasusApi.postCartSync(mapOf("items" to itemsPayload))
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }
    }"""

text = text.replace(old_sync, new_sync)

with open(target, "w") as f:
    f.write(text)

print("Patched CartViewModel syncCartContent.")
