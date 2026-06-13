package com.pegasusx.driver.data.telemetry

import com.google.android.gms.maps.model.LatLng
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState

/** Builds stop-to-stop polyline from manifest sequence (same route_id, sorted by sequence_index). */
fun buildPlannedRoutePolyline(orders: List<Order>, activeRouteId: String?): List<LatLng> {
    if (activeRouteId.isNullOrBlank()) return emptyList()
    return orders
        .asSequence()
        .filter { order ->
            order.routeId == activeRouteId &&
                order.state != OrderState.COMPLETED &&
                order.state != OrderState.CANCELLED &&
                order.latitude != null &&
                order.longitude != null
        }
        .sortedBy { it.sequenceIndex }
        .map { LatLng(it.latitude!!, it.longitude!!) }
        .toList()
}
