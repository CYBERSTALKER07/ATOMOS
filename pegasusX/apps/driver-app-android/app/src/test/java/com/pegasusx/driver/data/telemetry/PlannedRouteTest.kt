package com.pegasusx.driver.data.telemetry

import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState
import org.junit.Assert.assertEquals
import org.junit.Test

class PlannedRouteTest {
    @Test
    fun buildPlannedRoutePolyline_sortsBySequenceIndexOnSameRoute() {
        val orders = listOf(
            order("o3", routeId = "r1", sequence = 2, lat = 41.30, lng = 69.25),
            order("o1", routeId = "r1", sequence = 0, lat = 41.29, lng = 69.24),
            order("o2", routeId = "r1", sequence = 1, lat = 41.295, lng = 69.245),
            order("other", routeId = "r2", sequence = 0, lat = 40.0, lng = 70.0),
        )

        val points = buildPlannedRoutePolyline(orders, "r1")

        assertEquals(3, points.size)
        assertEquals(41.29, points[0].latitude, 0.0001)
        assertEquals(41.295, points[1].latitude, 0.0001)
        assertEquals(41.30, points[2].latitude, 0.0001)
    }

    private fun order(
        id: String,
        routeId: String,
        sequence: Int,
        lat: Double,
        lng: Double,
    ): Order = Order(
        id = id,
        retailerId = "ret-1",
        retailerName = "Shop",
        driverId = "drv-1",
        state = OrderState.IN_TRANSIT,
        totalAmount = 1000,
        deliveryAddress = "Addr",
        latitude = lat,
        longitude = lng,
        createdAt = "2026-01-01T00:00:00Z",
        updatedAt = "2026-01-01T00:00:00Z",
        routeId = routeId,
        sequenceIndex = sequence,
    )
}
