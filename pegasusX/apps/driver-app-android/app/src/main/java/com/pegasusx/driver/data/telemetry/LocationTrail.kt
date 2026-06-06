package com.pegasusx.driver.data.telemetry

import com.google.android.gms.maps.model.LatLng
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** Bounded recent driver positions for map breadcrumb overlays. */
object LocationTrail {
    private const val maxPoints = 60

    private val _points = MutableStateFlow<List<LatLng>>(emptyList())
    val points: StateFlow<List<LatLng>> = _points.asStateFlow()

    fun record(latitude: Double, longitude: Double) {
        val next = (_points.value + LatLng(latitude, longitude)).takeLast(maxPoints)
        _points.value = next
    }

    fun clear() {
        _points.value = emptyList()
    }
}
