package com.thelab.retailer.data.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class OrderStatusTest {

    @Test
    fun displayName_pending() {
        assertEquals("Order Placed", OrderStatus.PENDING.displayName)
    }

    @Test
    fun displayName_loaded() {
        assertEquals("Approved", OrderStatus.LOADED.displayName)
    }

    @Test
    fun displayName_dispatched() {
        assertEquals("Dispatched", OrderStatus.DISPATCHED.displayName)
    }

    @Test
    fun displayName_inTransit() {
        assertEquals("Active", OrderStatus.IN_TRANSIT.displayName)
    }

    @Test
    fun displayName_arrived() {
        assertEquals("Driver Arrived", OrderStatus.ARRIVED.displayName)
    }

    @Test
    fun displayName_awaitingGlobalPaynt() {
        assertEquals("GlobalPaynt Required", OrderStatus.AWAITING_GLOBAL_PAYNT.displayName)
    }

    @Test
    fun displayName_pendingCash() {
        assertEquals("Cash Collection", OrderStatus.PENDING_CASH_COLLECTION.displayName)
    }

    @Test
    fun displayName_completed() {
        assertEquals("Delivered", OrderStatus.COMPLETED.displayName)
    }

    @Test
    fun displayName_cancelled() {
        assertEquals("Cancelled", OrderStatus.CANCELLED.displayName)
    }

    @Test
    fun isActive_activeStates() {
        val active = listOf(
            OrderStatus.LOADED, OrderStatus.DISPATCHED, OrderStatus.IN_TRANSIT,
            OrderStatus.ARRIVED, OrderStatus.AWAITING_GLOBAL_PAYNT, OrderStatus.PENDING_CASH_COLLECTION
        )
        for (s in active) assertTrue("$s should be active", s.isActive)
    }

    @Test
    fun isActive_nonActiveStates() {
        val inactive = listOf(OrderStatus.PENDING, OrderStatus.COMPLETED, OrderStatus.CANCELLED)
        for (s in inactive) assertFalse("$s should not be active", s.isActive)
    }

    @Test
    fun progressFraction_completed_is1() {
        assertEquals(1.0f, OrderStatus.COMPLETED.progressFraction)
    }

    @Test
    fun progressFraction_cancelled_is0() {
        assertEquals(0f, OrderStatus.CANCELLED.progressFraction)
    }

    @Test
    fun progressFraction_monotonicallyIncreases() {
        val ordered = listOf(
            OrderStatus.PENDING, OrderStatus.LOADED, OrderStatus.DISPATCHED,
            OrderStatus.IN_TRANSIT, OrderStatus.ARRIVED, OrderStatus.COMPLETED
        )
        for (i in 0 until ordered.size - 1) {
            assertTrue(
                "${ordered[i]} < ${ordered[i + 1]}",
                ordered[i].progressFraction < ordered[i + 1].progressFraction
            )
        }
    }

    @Test
    fun canCancel_onlyPending() {
        assertTrue(OrderStatus.PENDING.canCancel)
        val others = OrderStatus.entries.filter { it != OrderStatus.PENDING }
        for (s in others) assertFalse("$s should not be cancellable", s.canCancel)
    }

    @Test
    fun hasDeliveryToken_dispatched_inTransit_arrived() {
        assertTrue(OrderStatus.DISPATCHED.hasDeliveryToken)
        assertTrue(OrderStatus.IN_TRANSIT.hasDeliveryToken)
        assertTrue(OrderStatus.ARRIVED.hasDeliveryToken)
    }

    @Test
    fun hasDeliveryToken_false_pending() {
        assertFalse(OrderStatus.PENDING.hasDeliveryToken)
        assertFalse(OrderStatus.COMPLETED.hasDeliveryToken)
        assertFalse(OrderStatus.CANCELLED.hasDeliveryToken)
    }

    @Test
    fun timelineStepIndex_ordered() {
        assertEquals(0, OrderStatus.PENDING.timelineStepIndex)
        assertEquals(1, OrderStatus.LOADED.timelineStepIndex)
        assertEquals(2, OrderStatus.DISPATCHED.timelineStepIndex)
        assertEquals(3, OrderStatus.IN_TRANSIT.timelineStepIndex)
        assertEquals(4, OrderStatus.ARRIVED.timelineStepIndex)
        assertEquals(5, OrderStatus.COMPLETED.timelineStepIndex)
        assertEquals(-1, OrderStatus.CANCELLED.timelineStepIndex)
    }

    @Test
    fun timelineSteps_has6Entries() {
        assertEquals(6, OrderStatus.timelineSteps.size)
    }

    @Test
    fun timelineSteps_firstIsPlaced_lastIsDelivered() {
        assertEquals("Placed", OrderStatus.timelineSteps.first().first)
        assertEquals("Delivered", OrderStatus.timelineSteps.last().first)
    }

    @Test
    fun ringLabel_completed_isDone() {
        assertEquals("Done", OrderStatus.COMPLETED.ringLabel)
    }

    @Test
    fun ringLabel_cancelled_isX() {
        assertEquals("X", OrderStatus.CANCELLED.ringLabel)
    }

    @Test
    fun ringLabel_awaitingGlobalPaynt_isPay() {
        assertEquals("Pay", OrderStatus.AWAITING_GLOBAL_PAYNT.ringLabel)
    }
}
