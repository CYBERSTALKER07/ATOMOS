package com.pegasusx.driver.data.telemetry

import com.pegasusx.driver.data.model.RouteCoordinate
import com.pegasusx.driver.util.Haversine
import kotlin.math.max

const val OFF_ROUTE_THRESHOLD_METERS = 45.0
const val SUSTAINED_OFF_ROUTE_MS = 4_000L
const val MIN_REROUTE_INTERVAL_MS = 30_000L

enum class RouteDeviationAction {
    None,
    Reroute,
}

fun distanceToPolylineMeters(lat: Double, lng: Double, polyline: List<RouteCoordinate>): Double {
    if (polyline.isEmpty()) {
        return Double.MAX_VALUE
    }
    if (polyline.size == 1) {
        val point = polyline.first()
        return Haversine.distanceMeters(lat, lng, point.lat, point.lng)
    }
    var minDistance = Double.MAX_VALUE
    for (index in 0 until polyline.lastIndex) {
        val start = polyline[index]
        val end = polyline[index + 1]
        val distance = distancePointToSegmentMeters(lat, lng, start.lat, start.lng, end.lat, end.lng)
        minDistance = minOf(minDistance, distance)
    }
    return minDistance
}

fun distancePointToSegmentMeters(
    lat: Double,
    lng: Double,
    startLat: Double,
    startLng: Double,
    endLat: Double,
    endLng: Double,
): Double {
    val deltaLng = endLng - startLng
    val deltaLat = endLat - startLat
    if (deltaLng == 0.0 && deltaLat == 0.0) {
        return Haversine.distanceMeters(lat, lng, startLat, startLng)
    }
    var t = ((lng - startLng) * deltaLng + (lat - startLat) * deltaLat) /
        (deltaLng * deltaLng + deltaLat * deltaLat)
    t = max(0.0, minOf(1.0, t))
    val closestLat = startLat + t * deltaLat
    val closestLng = startLng + t * deltaLng
    return Haversine.distanceMeters(lat, lng, closestLat, closestLng)
}

class RouteDeviationTracker {
    private var offRouteSinceMs: Long? = null
    private var lastRerouteAtMs: Long = 0L

    fun evaluate(
        nowMs: Long,
        lat: Double,
        lng: Double,
        polyline: List<RouteCoordinate>,
    ): RouteDeviationAction {
        if (polyline.size < 2) {
            offRouteSinceMs = null
            return RouteDeviationAction.None
        }
        val distance = distanceToPolylineMeters(lat, lng, polyline)
        if (distance <= OFF_ROUTE_THRESHOLD_METERS) {
            offRouteSinceMs = null
            return RouteDeviationAction.None
        }
        if (offRouteSinceMs == null) {
            offRouteSinceMs = nowMs
            return RouteDeviationAction.None
        }
        if (nowMs - offRouteSinceMs!! < SUSTAINED_OFF_ROUTE_MS) {
            return RouteDeviationAction.None
        }
        if (lastRerouteAtMs > 0L && nowMs - lastRerouteAtMs < MIN_REROUTE_INTERVAL_MS) {
            return RouteDeviationAction.None
        }
        lastRerouteAtMs = nowMs
        offRouteSinceMs = null
        return RouteDeviationAction.Reroute
    }

    fun reset() {
        offRouteSinceMs = null
    }
}
