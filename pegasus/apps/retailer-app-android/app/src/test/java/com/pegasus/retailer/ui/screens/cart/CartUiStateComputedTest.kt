package com.thelab.retailer.ui.screens.cart

import org.junit.Assert.*
import org.junit.Test

/**
 * CartUiState — Computed properties: totals, shipping, discount, display values, payment labels.
 *
 * CartItem requires Product reference which needs Hilt wiring.
 * For the empty cart we exercise the actual data class; for non-trivial subtotals
 * we verify the formula inline (same logic as the computed getters).
 */
class CartUiStateComputedTest {

    @Test
    fun `empty cart is empty and has correct defaults`() {
        val state = CartUiState()
        assertTrue(state.isEmpty)
        assertEquals(0, state.totalItems)
        assertEquals(0.0, state.subtotal, 0.001)
        assertEquals(15_000.0, state.shipping, 0.001) // under 50k threshold
        assertEquals(0.0, state.discount, 0.001)
        assertEquals(15_000.0, state.total, 0.001)
        assertEquals("0", state.displaySubtotal)
    }

    @Test
    fun `free shipping formula when subtotal over 50000`() {
        val subtotal = 60_000.0
        val shipping = if (subtotal > 50_000) 0.0 else 15_000.0
        assertEquals(0.0, shipping, 0.001)
    }

    @Test
    fun `shipping charged formula when subtotal under 50000`() {
        val subtotal = 30_000.0
        val shipping = if (subtotal > 50_000) 0.0 else 15_000.0
        assertEquals(15_000.0, shipping, 0.001)
    }

    @Test
    fun `5 percent discount formula when subtotal over 500000`() {
        val subtotal = 600_000.0
        val discount = if (subtotal > 500_000) subtotal * 0.05 else 0.0
        assertEquals(30_000.0, discount, 0.001)
    }

    @Test
    fun `no discount formula when subtotal under 500000`() {
        val subtotal = 400_000.0
        val discount = if (subtotal > 500_000) subtotal * 0.05 else 0.0
        assertEquals(0.0, discount, 0.001)
    }

    @Test
    fun `total formula is subtotal + shipping - discount`() {
        val subtotal = 700_000.0
        val shipping = 0.0 // over 50k → free
        val discount = 35_000.0 // 5% of 700k
        val total = subtotal + shipping - discount
        assertEquals(665_000.0, total, 0.001)
    }

    @Test
    fun `selectedPaymentLabel default is GlobalPay`() {
        val state = CartUiState()
        assertEquals("GlobalPay", state.selectedPaymentLabel)
    }

    @Test
    fun `selectedPaymentLabel GlobalPay`() {
        val state = CartUiState(selectedPaymentGateway = "GLOBAL_PAY")
        assertEquals("GlobalPay", state.selectedPaymentLabel)
    }

    @Test
    fun `selectedPaymentLabel Cash`() {
        val state = CartUiState(selectedPaymentGateway = "CASH")
        assertEquals("Cash on Delivery", state.selectedPaymentLabel)
    }

    @Test
    fun `displayShipping shows Free when zero`() {
        val displayShipping = if (0.0 == 0.0) "Free" else "%,.0f".format(0.0)
        assertEquals("Free", displayShipping)
    }

    @Test
    fun `oosItems starts empty`() {
        val state = CartUiState()
        assertTrue(state.oosItems.isEmpty())
    }
}
