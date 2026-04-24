package com.thelab.driver.data.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class OrderModelTest {

    @Test
    fun orderState_allCases() {
        val values = OrderState.values()
        assertEquals("OrderState should have 15 values", 15, values.size)
        // Verify critical states exist
        assertTrue(values.contains(OrderState.PENDING))
        assertTrue(values.contains(OrderState.IN_TRANSIT))
        assertTrue(values.contains(OrderState.ARRIVED))
        assertTrue(values.contains(OrderState.COMPLETED))
        assertTrue(values.contains(OrderState.CANCELLED))
    }

    @Test
    fun orderCopy_preservesFields() {
        val original = Order(
            id = "ORD-COPY",
            retailerId = "RET-001",
            retailerName = "Test Shop",
            driverId = "DRV-001",
            state = OrderState.IN_TRANSIT,
            totalAmount = 150000L,
            deliveryAddress = "123 Main St",
            latitude = 41.2995,
            longitude = 69.2401,
            qrToken = "tok_abc",
            paymentGateway = "GLOBAL_PAY",
            createdAt = "2026-04-12T10:00:00Z",
            updatedAt = "2026-04-12T10:00:00Z",
            items = emptyList(),
            estimatedArrivalAt = null,
            etaDurationSec = null,
            etaDistanceM = null,
            routeId = "ROUTE-01",
            sequenceIndex = 2
        )

        val arrived = original.copy(state = OrderState.ARRIVED)

        assertEquals(OrderState.ARRIVED, arrived.state)
        assertEquals(original.id, arrived.id)
        assertEquals(original.retailerId, arrived.retailerId)
        assertEquals(original.totalAmount, arrived.totalAmount)
        assertEquals(original.routeId, arrived.routeId)
        assertEquals(original.sequenceIndex, arrived.sequenceIndex)
        assertEquals(original.qrToken, arrived.qrToken)
        // Original unchanged
        assertEquals(OrderState.IN_TRANSIT, original.state)
    }

    @Test
    fun orderState_enumNames() {
        assertEquals("PENDING", OrderState.PENDING.name)
        assertEquals("LOADED", OrderState.LOADED.name)
        assertEquals("IN_TRANSIT", OrderState.IN_TRANSIT.name)
        assertEquals("ARRIVED", OrderState.ARRIVED.name)
        assertEquals("AWAITING_PAYMENT", OrderState.AWAITING_PAYMENT.name)
        assertEquals("COMPLETED", OrderState.COMPLETED.name)
        assertEquals("CANCELLED", OrderState.CANCELLED.name)
    }
}
