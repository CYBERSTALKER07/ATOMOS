package com.pegasus.retailer.ui.screens.orders

import com.pegasus.retailer.data.model.Order
import com.pegasus.retailer.data.model.OrderStatus
import org.junit.Assert.*
import org.junit.Test

/**
 * OrdersUiState — computed property tests for order filtering and status grouping
 */
class OrdersUiStateTest {

    private fun makeOrder(id: String, status: OrderStatus) = Order(
        id = id,
        supplierId = "sup-1",
        supplierName = "Test Supplier",
        retailerId = "r-1",
        status = status,
        totalAmount = 100_000.0,
        createdAt = "2026-01-01T00:00:00Z",
        updatedAt = "2026-01-01T00:00:00Z",
    )

    @Test
    fun `default state has empty orders and no loading`() {
        val state = OrdersUiState()
        assertFalse(state.isLoading)
        assertTrue(state.allOrders.isEmpty())
        assertTrue(state.predictions.isEmpty())
        assertNull(state.error)
    }

    @Test
    fun `activeOrders filters LOADED, DISPATCHED, IN_TRANSIT, ARRIVED`() {
        val state = OrdersUiState(
            allOrders = listOf(
                makeOrder("o-1", OrderStatus.LOADED),
                makeOrder("o-2", OrderStatus.DISPATCHED),
                makeOrder("o-3", OrderStatus.IN_TRANSIT),
                makeOrder("o-4", OrderStatus.ARRIVED),
                makeOrder("o-5", OrderStatus.PENDING),
                makeOrder("o-6", OrderStatus.COMPLETED),
            )
        )
        assertEquals(4, state.activeOrders.size)
        assertTrue(state.activeOrders.all {
            it.status in listOf(OrderStatus.LOADED, OrderStatus.DISPATCHED, OrderStatus.IN_TRANSIT, OrderStatus.ARRIVED)
        })
    }

    @Test
    fun `pendingOrders filters only PENDING`() {
        val state = OrdersUiState(
            allOrders = listOf(
                makeOrder("o-1", OrderStatus.PENDING),
                makeOrder("o-2", OrderStatus.PENDING),
                makeOrder("o-3", OrderStatus.IN_TRANSIT),
                makeOrder("o-4", OrderStatus.COMPLETED),
            )
        )
        assertEquals(2, state.pendingOrders.size)
        assertTrue(state.pendingOrders.all { it.status == OrderStatus.PENDING })
    }

    @Test
    fun `empty orders yields empty active and pending`() {
        val state = OrdersUiState(allOrders = emptyList())
        assertTrue(state.activeOrders.isEmpty())
        assertTrue(state.pendingOrders.isEmpty())
    }

    @Test
    fun `all COMPLETED yields empty active and pending`() {
        val state = OrdersUiState(
            allOrders = listOf(
                makeOrder("o-1", OrderStatus.COMPLETED),
                makeOrder("o-2", OrderStatus.COMPLETED),
            )
        )
        assertTrue(state.activeOrders.isEmpty())
        assertTrue(state.pendingOrders.isEmpty())
    }
}
