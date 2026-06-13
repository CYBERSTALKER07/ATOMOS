package com.pegasusx.driver.data.telemetry

import com.pegasusx.driver.data.model.RouteStep
import org.junit.Assert.assertEquals
import org.junit.Test

class RouteNavigationTest {

    @Test
    fun advanceNavigationStepIndex_staysOnFirstStepUntilPassed() {
        val steps = listOf(
            RouteStep("Head north", 0.0, 0.0, "DEPART NORTH", 41.2995, 69.2401),
            RouteStep("Turn right", 0.0, 0.0, "RIGHT TURN", 41.3005, 69.2411),
        )
        val index = advanceNavigationStepIndex(0, steps, 41.2975, 69.2375)
        assertEquals(0, index)
    }

    @Test
    fun advanceNavigationStepIndex_advancesAfterPassingManeuver() {
        val steps = listOf(
            RouteStep("Head north", 0.0, 0.0, "DEPART NORTH", 41.2995, 69.2401),
            RouteStep("Turn right", 0.0, 0.0, "RIGHT TURN", 41.3005, 69.2411),
        )
        val index = advanceNavigationStepIndex(0, steps, 41.3005, 69.2411)
        assertEquals(1, index)
    }
}
