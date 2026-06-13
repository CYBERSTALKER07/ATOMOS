package com.pegasusx.driver.data.telemetry

import com.pegasusx.driver.data.model.RouteCoordinate
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class RouteDeviationTest {

    @Test
    fun distanceToPolylineMeters_nearSegment() {
        val polyline = listOf(
            RouteCoordinate(41.2995, 69.2401),
            RouteCoordinate(41.3005, 69.2411),
        )
        val distance = distanceToPolylineMeters(41.3000, 69.2406, polyline)
        assertTrue(distance <= 30.0)
    }

    @Test
    fun routeDeviationTracker_requiresSustainedOffRoute() {
        val tracker = RouteDeviationTracker()
        val polyline = listOf(
            RouteCoordinate(41.2995, 69.2401),
            RouteCoordinate(41.3005, 69.2411),
        )
        val action = tracker.evaluate(1_000L, 41.3050, 69.2500, polyline)
        assertEquals(RouteDeviationAction.None, action)
    }

    @Test
    fun routeDeviationTracker_rerouteAfterSustainedOffRoute() {
        val tracker = RouteDeviationTracker()
        val polyline = listOf(
            RouteCoordinate(41.2995, 69.2401),
            RouteCoordinate(41.3005, 69.2411),
        )
        tracker.evaluate(1_000L, 41.3050, 69.2500, polyline)
        val action = tracker.evaluate(6_000L, 41.3050, 69.2500, polyline)
        assertEquals(RouteDeviationAction.Reroute, action)
    }
}
