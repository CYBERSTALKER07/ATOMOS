import os

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/cart/CartViewModel.kt"
with open(target, "r") as f:
    text = f.read()

# We need to trigger syncCartContent when cart items change.
old_add = """    fun addToCart(product: Product, variant: Variant, quantity: Int = 1) {
        _uiState.update { state ->
            val existing = state.items.find { it.product.id == product.id && it.variant.id == variant.id }
            if (existing \!= null) {
                state.copy(items = state.items.map { if (it == existing) it.copy(quantity = it.quantity + quantity) else it })
            } else {
                state.copy(items = state.items + CartItem(product, variant, quantity))
            }
        }
    }"""
new_add = """    fun addToCart(product: Product, variant: Variant, quantity: Int = 1) {
        _uiState.update { state ->
            val existing = state.items.find { it.product.id == product.id && it.variant.id == variant.id }
            if (existing \!= null) {
                state.copy(items = state.items.map { if (it == existing) it.copy(quantity = it.quantity + quantity) else it })
            } else {
                state.copy(items = state.items + CartItem(product, variant, quantity))
            }
        }
        syncCartContent()
    }"""

old_update = """    fun updateQuantity(product: Product, variant: Variant, quantity: Int) {
        if (quantity <= 0) {
            removeItem(product, variant)
            return
        }
        _uiState.update { state ->
            state.copy(items = state.items.map { 
                if (it.product.id == product.id && it.variant.id == variant.id) it.copy(quantity = quantity) 
                else it 
            })
        }
    }"""
new_update = """    fun updateQuantity(product: Product, variant: Variant, quantity: Int) {
        if (quantity <= 0) {
            removeItem(product, variant)
            return
        }
        _uiState.update { state ->
            state.copy(items = state.items.map { 
                if (it.product.id == product.id && it.variant.id == variant.id) it.copy(quantity = quantity) 
                else it 
            })
        }
        syncCartContent()
    }"""

old_remove = """    fun removeItem(product: Product, variant: Variant) {
        _uiState.update { state ->
            state.copy(
                items = state.items.filterNot { it.product.id == product.id && it.variant.id == variant.id },
                removedItemMessage = "${product.name} removed from cart"
            )
        }
    }"""
new_remove = """    fun removeItem(product: Product, variant: Variant) {
        _uiState.update { state ->
            state.copy(
                items = state.items.filterNot { it.product.id == product.id && it.variant.id == variant.id },
                removedItemMessage = "${product.name} removed from cart"
            )
        }
        syncCartContent()
    }"""

text = text.replace(old_add, new_add).replace(old_update, new_update).replace(old_remove, new_remove)

with open(target, "w") as f:
    f.write(text)

