package com.pegasusx.driver.data.telemetry

import android.location.Location
import com.google.android.gms.maps.model.LatLng
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Linearly interpolates between GPS fixes so the map camera glides instead of snapping.
 */
class LocationInterpolator {
    private data class Sample(
        val lat: Double,
        val lng: Double,
        val bearing: Float,
        val speedMps: Float,
        val timeMs: Long,
    )

    private var from: Sample? = null
    private var to: Sample? = null

    private val _position = MutableStateFlow<LatLng?>(null)
    val position: StateFlow<LatLng?> = _position.asStateFlow()

    private val _bearing = MutableStateFlow(0f)
    val bearing: StateFlow<Float> = _bearing.asStateFlow()

    private val _speedMps = MutableStateFlow(0f)
    val speedMps: StateFlow<Float> = _speedMps.asStateFlow()

    fun onGps(location: Location) {
        val sample = Sample(
            lat = location.latitude,
            lng = location.longitude,
            bearing = if (location.hasBearing()) location.bearing else _bearing.value,
            speedMps = location.speed.coerceAtLeast(0f),
            timeMs = System.currentTimeMillis(),
        )
        if (to == null) {
            to = sample
            publish(sample)
            return
        }
        from = to
        to = sample
    }

    fun tick(nowMs: Long = System.currentTimeMillis()) {
        val start = from
        val end = to ?: return
        if (start == null) {
            publish(end)
            return
        }
        val duration = (end.timeMs - start.timeMs).coerceAtLeast(1L)
        val progress = ((nowMs - start.timeMs).toDouble() / duration.toDouble()).coerceIn(0.0, 1.0)
        val lat = start.lat + (end.lat - start.lat) * progress
        val lng = start.lng + (end.lng - start.lng) * progress
        val bearing = start.bearing + (end.bearing - start.bearing) * progress.toFloat()
        val speed = start.speedMps + (end.speedMps - start.speedMps) * progress.toFloat()
        _position.value = LatLng(lat, lng)
        _bearing.value = bearing
        _speedMps.value = speed
    }

    fun clear() {
        from = null
        to = null
        _position.value = null
    }

    private fun publish(sample: Sample) {
        _position.value = LatLng(sample.lat, sample.lng)
        _bearing.value = sample.bearing
        _speedMps.value = sample.speedMps
    }
}
