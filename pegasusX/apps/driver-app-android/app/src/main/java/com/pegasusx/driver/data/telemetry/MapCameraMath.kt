package com.pegasusx.driver.data.telemetry

import com.google.android.gms.maps.model.CameraPosition
import com.google.android.gms.maps.model.LatLng
import com.pegasusx.driver.util.Haversine

object MapCameraMath {
    fun dynamicZoom(speedMps: Float): Float =
        (MapCameraConfig.MAX_ZOOM - (speedMps / MapCameraConfig.SPEED_ZOOM_DIVISOR))
            .coerceIn(MapCameraConfig.MIN_ZOOM, MapCameraConfig.MAX_ZOOM)

    fun lookAheadTarget(lat: Double, lng: Double, bearing: Float, speedMps: Float): LatLng {
        val meters = (speedMps * MapCameraConfig.LOOK_AHEAD_SECONDS)
            .coerceIn(0.0, MapCameraConfig.MAX_LOOK_AHEAD_METERS)
        if (meters <= 1.0) return LatLng(lat, lng)
        val (aheadLat, aheadLng) = Haversine.offsetMeters(lat, lng, bearing.toDouble(), meters)
        return LatLng(aheadLat, aheadLng)
    }

    fun trackingCameraPosition(
        lat: Double,
        lng: Double,
        bearing: Float,
        speedMps: Float,
        fallbackBearing: Float,
    ): CameraPosition {
        val target = lookAheadTarget(lat, lng, bearing, speedMps)
        return CameraPosition.Builder()
            .target(target)
            .zoom(dynamicZoom(speedMps))
            .tilt(MapCameraConfig.NAVIGATION_TILT)
            .bearing(if (bearing.isFinite() && bearing != 0f) bearing else fallbackBearing)
            .build()
    }
}
