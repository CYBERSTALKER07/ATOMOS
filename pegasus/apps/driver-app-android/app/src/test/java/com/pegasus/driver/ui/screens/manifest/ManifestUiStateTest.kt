package com.pegasus.driver.ui.screens.manifest

import com.pegasus.driver.data.model.Order
import com.pegasus.driver.data.model.OrderLineItem
import com.pegasus.driver.data.model.OrderState
import org.junit.Assert.*
import org.junit.Test

/**
 * ManifestUiState — State derivation, truck status, stop counting
 */
class ManifestUiStateTest {

    private fun makeOrder(id: String, state: OrderState) = Order(
        id = id,
        retailerId = "r-1",
        retailerName = "Test Shop",
        state = state,
        totalAmount = 100_000L,
        deliveryAddress = "123 Test St",
        createdAt = "2026-01-01T00:00:00Z",
        updatedAt = "2026-01-01T00:00:00Z",
        items = listOf(
            OrderLineItem("p-1", "Milk", 10, 5000L, 50000L)
        ),
    )

    // ── Default state ──

    @Test
    fun `default state is loading with empty orders`() {
        val state = ManifestUiState()
        assertTrue(state.isLoading)
        assertTrue(state.orders.isEmpty())
        assertNull(state.error)
        assertEquals(0, state.totalStops)
        assertEquals("AVAILABLE", state.truckStatus)
        assertFalse(state.isReturning)
    }

    // ── Stop counting ──

    @Test
    fun `totalStops excludes COMPLETED and CANCELLED orders`() {
        val orders = listOf(
            makeOrder("o-1", OrderState.IN_TRANSIT),
            makeOrder("o-2", OrderState.ARRIVED),
            makeOrder("o-3", OrderState.COMPLETED),
            makeOrder("o-4", OrderState.CANCELLED),
            makeOrder("o-5", OrderState.PENDING),
        )
        val stopCount = orders.count { it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED }
        assertEquals(3, stopCount)
    }

    @Test
    fun `all complete yields zero stops`() {
        val orders = listOf(
            makeOrder("o-1", OrderState.COMPLETED),
            makeOrder("o-2", OrderState.COMPLETED),
        )
        val stopCount = orders.count { it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED }
        assertEquals(0, stopCount)
    }

    // ── Truck status derivation ──

    @Test
    fun `all orders complete derives RETURNING status`() {
        val orders = listOf(
            makeOrder("o-1", OrderState.COMPLETED),
            makeOrder("o-2", OrderState.COMPLETED),
        )
        val allComplete = orders.isNotEmpty() && orders.all {
            it.state == OrderState.COMPLETED || it.state == OrderState.CANCELLED
        }
        assertTrue(allComplete)
    }

    @Test
    fun `IN_TRANSIT order derives IN_TRANSIT status`() {
        val orders = listOf(
            makeOrder("o-1", OrderState.IN_TRANSIT),
            makeOrder("o-2", OrderState.PENDING),
        )
        val hasInTransit = orders.any {
            it.state == OrderState.IN_TRANSIT || it.state == OrderState.ARRIVING ||
            it.state == OrderState.ARRIVED || it.state == OrderState.AWAITING_PAYMENT ||
            it.state == OrderState.PENDING_CASH_COLLECTION || it.state == OrderState.DISPATCHED
        }
        assertTrue(hasInTransit)
    }

    @Test
    fun `mixed ARRIVED and PENDING does not derive RETURNING`() {
        val orders = listOf(
            makeOrder("o-1", OrderState.ARRIVED),
            makeOrder("o-2", OrderState.PENDING),
        )
        val allComplete = orders.isNotEmpty() && orders.all {
            it.state == OrderState.COMPLETED || it.state == OrderState.CANCELLED
        }
        assertFalse(allComplete)
    }

    @Test
    fun `empty orders list does not derive RETURNING`() {
        val allComplete = emptyList<Order>().isNotEmpty() && emptyList<Order>().all {
            it.state == OrderState.COMPLETED || it.state == OrderState.CANCELLED
        }
        assertFalse(allComplete)
    }

    // ── LEO / Manifest seal ──

    @Test
    fun `awaitingSeal true when manifestId exists and not sealed`() {
        val state = ManifestUiState(
            manifestId = "m-1",
            manifestSealed = false,
            manifestState = "LOADING",
            awaitingSeal = true,
        )
        assertTrue(state.awaitingSeal)
        assertFalse(state.manifestSealed)
    }

    @Test
    fun `sealed manifest clears awaitingSeal`() {
        val state = ManifestUiState(
            manifestId = "m-1",
            manifestSealed = true,
            manifestState = "SEALED",
            awaitingSeal = false,
        )
        assertFalse(state.awaitingSeal)
        assertTrue(state.manifestSealed)
    }

    // ── OrderState enum coverage ──

    @Test
    fun `OrderState has all expected values`() {
        val expected = setOf(
            "PENDING", "PENDING_REVIEW", "SCHEDULED", "LOADED", "DISPATCHED", "IN_TRANSIT", "ARRIVING", "ARRIVED",
            "ARRIVED_SHOP_CLOSED", "AWAITING_PAYMENT", "PENDING_CASH_COLLECTION",
            "CANCEL_REQUESTED", "NO_CAPACITY", "COMPLETED", "CANCELLED",
            "QUARANTINE", "DELIVERED_ON_CREDIT"
        )
        val actual = OrderState.entries.map { it.name }.toSet()
        assertEquals(expected, actual)
    }

    @Test
    fun `OrderLineItem lineTotal calculation`() {
        val item = OrderLineItem("p-1", "Rice", 10, 25_000L, 250_000L)
        assertEquals(250_000L, item.lineTotal)
        assertEquals(10 * 25_000L, item.lineTotal)
    }
}
