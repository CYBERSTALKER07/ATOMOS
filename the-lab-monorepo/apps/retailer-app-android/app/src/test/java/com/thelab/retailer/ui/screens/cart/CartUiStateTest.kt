package com.thelab.retailer.ui.screens.cart

import com.thelab.retailer.data.model.CartItem
import com.thelab.retailer.data.model.Product
import com.thelab.retailer.data.model.Variant
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CartUiStateTest {

    private val variant1 = Variant("v1", "1L", "Single", 1, "1000ml", 10_000.0)
    private val variant2 = Variant("v2", "2L", "Twin", 2, "2000ml", 25_000.0)
    private val product1 = Product(id = "p1", name = "Milk", description = "", variants = listOf(variant1))
    private val product2 = Product(id = "p2", name = "Juice", description = "", variants = listOf(variant2))

    private fun cartWith(vararg quantities: Pair<Product, Pair<Variant, Int>>): CartUiState {
        val items = quantities.map { (product, vq) ->
            CartItem(
                id = "${product.id}_${vq.first.id}",
                product = product,
                variant = vq.first,
                quantity = vq.second
            )
        }
        return CartUiState(items = items)
    }

    @Test
    fun isEmpty_noItems() {
        assertTrue(CartUiState().isEmpty)
    }

    @Test
    fun isEmpty_withItems() {
        val state = cartWith(product1 to (variant1 to 1))
        assertFalse(state.isEmpty)
    }

    @Test
    fun totalItems_sumsQuantities() {
        val state = cartWith(product1 to (variant1 to 3), product2 to (variant2 to 2))
        assertEquals(5, state.totalItems)
    }

    @Test
    fun subtotal_sumsItemTotals() {
        // 3 * 10_000 + 2 * 25_000 = 80_000
        val state = cartWith(product1 to (variant1 to 3), product2 to (variant2 to 2))
        assertEquals(80_000.0, state.subtotal, 0.01)
    }

    @Test
    fun shipping_freeAbove50k() {
        val state = cartWith(product2 to (variant2 to 3)) // 75_000
        assertEquals(0.0, state.shipping, 0.01)
    }

    @Test
    fun shipping_chargedBelow50k() {
        val state = cartWith(product1 to (variant1 to 1)) // 10_000
        assertEquals(15_000.0, state.shipping, 0.01)
    }

    @Test
    fun shipping_exactlyAt50k() {
        val variantExact = Variant("ve", "5L", "Bulk", 1, "5000ml", 50_000.0)
        val prodExact = Product(id = "pe", name = "Exact", description = "", variants = listOf(variantExact))
        val state = cartWith(prodExact to (variantExact to 1)) // 50_000
        assertEquals(15_000.0, state.shipping, 0.01) // > not >=
    }

    @Test
    fun discount_5percentAbove500k() {
        val expensiveVariant = Variant("vx", "Big", "Pack", 1, "10kg", 600_000.0)
        val expensiveProduct = Product(id = "px", name = "Expensive", description = "", variants = listOf(expensiveVariant))
        val state = cartWith(expensiveProduct to (expensiveVariant to 1))
        assertEquals(30_000.0, state.discount, 0.01) // 5% of 600k
    }

    @Test
    fun discount_zeroBelow500k() {
        val state = cartWith(product1 to (variant1 to 10)) // 100_000
        assertEquals(0.0, state.discount, 0.01)
    }

    @Test
    fun total_subtotalPlusShippingMinusDiscount() {
        // Subtotal 80_000, shipping free (>50k), no discount (<500k)
        val state = cartWith(product1 to (variant1 to 3), product2 to (variant2 to 2))
        assertEquals(80_000.0, state.total, 0.01) // 80k + 0 - 0
    }

    @Test
    fun total_withShippingAndNoDiscount() {
        val state = cartWith(product1 to (variant1 to 1)) // 10_000 + 15_000
        assertEquals(25_000.0, state.total, 0.01)
    }

    @Test
    fun displaySubtotal_formatsAmount() {
        val state = cartWith(product1 to (variant1 to 3)) // 30_000
        assertEquals("30,000", state.displaySubtotal)
    }

    @Test
    fun displayShipping_free() {
        val state = cartWith(product2 to (variant2 to 3)) // >50k
        assertEquals("Free", state.displayShipping)
    }

    @Test
    fun displayShipping_notFree() {
        val state = cartWith(product1 to (variant1 to 1))
        assertEquals("15,000", state.displayShipping)
    }

    @Test
    fun displayDiscount_zeroWhenNone() {
        val state = cartWith(product1 to (variant1 to 1))
        assertEquals("0", state.displayDiscount)
    }

    @Test
    fun firstProductName_returnsFirst() {
        val state = cartWith(product1 to (variant1 to 1), product2 to (variant2 to 1))
        assertEquals("Milk", state.firstProductName)
    }

    @Test
    fun firstProductName_defaultsToOrder() {
        assertEquals("Order", CartUiState().firstProductName)
    }

    @Test
    fun selectedGlobalPayntLabel_cash() {
        val state = CartUiState(selectedGlobalPayntGateway = "CASH")
        assertEquals("Cash", state.selectedGlobalPayntLabel)
    }

    @Test
    fun selectedGlobalPayntLabel_global_pay() {
        val state = CartUiState(selectedGlobalPayntGateway = "GLOBAL_PAY")
        assertEquals("GlobalPay", state.selectedGlobalPayntLabel)
    }

    @Test
    fun selectedGlobalPayntLabel_globalPay() {
        val state = CartUiState(selectedGlobalPayntGateway = "GLOBAL_PAY")
        assertEquals("GlobalPay", state.selectedGlobalPayntLabel)
    }

    @Test
    fun selectedGlobalPayntLabel_cash() {
        val state = CartUiState(selectedGlobalPayntGateway = "CASH")
        assertEquals("Cash on Delivery", state.selectedGlobalPayntLabel)
    }

    @Test
    fun selectedGlobalPayntLabel_unknown_fallsBackToCash() {
        val state = CartUiState(selectedGlobalPayntGateway = "UNKNOWN_GATEWAY")
        assertEquals("Cash", state.selectedGlobalPayntLabel)
    }

    @Test
    fun selectedGlobalPayntLabel_handlesWhitespaceAndCase() {
        val state = CartUiState(selectedGlobalPayntGateway = "  global_pay  ")
        assertEquals("GlobalPay", state.selectedGlobalPayntLabel)
    }
}
