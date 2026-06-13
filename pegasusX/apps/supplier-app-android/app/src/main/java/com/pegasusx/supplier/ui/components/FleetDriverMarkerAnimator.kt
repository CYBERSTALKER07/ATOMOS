package com.pegasusx.supplier.ui.components

import com.pegasusx.supplier.data.model.SupplierDriverLocationWire
import com.pegasusx.supplier.data.model.SupplierFleetLiveRoute

data class AnimatedDriverPoint(
    val driverId: String,
    val lat: Double,
    val lng: Double,
    val color: String,
    val stale: Boolean,
)

class FleetDriverMarkerAnimator(
    private val durationMs: Long = 1_200L,
) {
    private data class AnimState(
        val driverId: String,
        var fromLat: Double,
        var fromLng: Double,
        var toLat: Double,
        var toLng: Double,
        var startMs: Long,
        val color: String,
        val stale: Boolean,
    )

    private val states = linkedMapOf<String, AnimState>()

    fun updateTargets(
        routes: List<SupplierFleetLiveRoute>,
        colorForIndex: (Int) -> String,
    ) {
        val active = mutableSetOf<String>()
        routes.forEachIndexed { index, route ->
            val location = route.driverLocation
            if (!route.liveLocationAvailable || location == null) {
                return@forEachIndexed
            }
            val lat = location.resolvedLatitude()
            val lng = location.resolvedLongitude()
            if (!lat.isFinite() || !lng.isFinite()) {
                return@forEachIndexed
            }
            active += route.driverId
            val color = colorForIndex(index)
            val stale = route.locationStale == true
            val existing = states[route.driverId]
            if (existing == null) {
                states[route.driverId] = AnimState(
                    driverId = route.driverId,
                    fromLat = lat,
                    fromLng = lng,
                    toLat = lat,
                    toLng = lng,
                    startMs = System.currentTimeMillis(),
                    color = color,
                    stale = stale,
                )
            } else if (existing.toLat != lat || existing.toLng != lng) {
                val now = System.currentTimeMillis()
                val progress = progressAt(now, existing)
                val currentLat = lerp(existing.fromLat, existing.toLat, progress)
                val currentLng = lerp(existing.fromLng, existing.toLng, progress)
                existing.fromLat = currentLat
                existing.fromLng = currentLng
                existing.toLat = lat
                existing.toLng = lng
                existing.startMs = now
            }
        }
        states.keys.retainAll(active)
    }

    fun snapshot(nowMs: Long): List<AnimatedDriverPoint> =
        states.values.map { state ->
            val t = easeOut(progressAt(nowMs, state))
            AnimatedDriverPoint(
                driverId = state.driverId,
                lat = lerp(state.fromLat, state.toLat, t),
                lng = lerp(state.fromLng, state.toLng, t),
                color = state.color,
                stale = state.stale,
            )
        }

    private fun progressAt(nowMs: Long, state: AnimState): Double {
        if (durationMs <= 0L) return 1.0
        return ((nowMs - state.startMs).toDouble() / durationMs.toDouble()).coerceIn(0.0, 1.0)
    }

    private fun lerp(a: Double, b: Double, t: Double): Double = a + (b - a) * t

    private fun easeOut(t: Double): Double = t * (2.0 - t)
}

private fun SupplierDriverLocationWire.resolvedLatitude(): Double =
    if (latitude != 0.0) latitude else lat

private fun SupplierDriverLocationWire.resolvedLongitude(): Double =
    if (longitude != 0.0) longitude else lng
