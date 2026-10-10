package com.pegasusx.driver.ui.screens.home

import com.pegasusx.driver.data.model.OrderState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class RemainingStopsTest {

    @Test
    fun shopClosedAndFiscalFailedAreFirstClassRemainingStops() {
        val stops = RemainingStops.remainingStops(
            listOf(
                RemainingStop("a", "Open", OrderState.IN_TRANSIT, 1),
                RemainingStop("b", "Closed shop", OrderState.ARRIVED_SHOP_CLOSED, 2),
                RemainingStop("c", "Fiscal", OrderState.FISCAL_FAILED, 3),
                RemainingStop("d", "Done", OrderState.COMPLETED, 4),
                RemainingStop("e", "Cancel", OrderState.CANCELLED, 5),
            ),
        )
        assertEquals(listOf("a", "b", "c"), stops.map { it.id })
        assertTrue(stops.any { it.state == OrderState.ARRIVED_SHOP_CLOSED && it.firstClass })
        assertTrue(stops.any { it.state == OrderState.FISCAL_FAILED && it.firstClass })
        assertFalse(stops.any { it.state == OrderState.COMPLETED })
    }

    @Test
    fun firstClassHelpers() {
        assertTrue(RemainingStops.isFirstClass(OrderState.ARRIVED_SHOP_CLOSED))
        assertTrue(RemainingStops.isFirstClass(OrderState.FISCAL_FAILED))
        assertFalse(RemainingStops.isFirstClass(OrderState.ARRIVED))
        assertFalse(RemainingStops.isFirstClass(OrderState.IN_TRANSIT))
    }

    @Test
    fun pulseFailureDoesNotTreatAsEmptyTimeline() {
        val src = java.io.File("src/main/java/com/pegasusx/driver/ui/screens/home/HomeScreen.kt").readText()
        assertTrue(src.contains("PulseHonesty.FAILED"))
        assertTrue(src.contains("error = pulseError"))
        assertTrue(!src.contains("pulseEvents = emptyList()"))
    }
}
