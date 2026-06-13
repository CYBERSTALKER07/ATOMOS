package com.pegasusx.driver.util

import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.sin
import kotlin.math.sqrt

object Haversine {
    private const val EARTH_RADIUS_METERS = 6_371_000.0

    /** Projects a point [distanceMeters] along [bearingDegrees] from (lat, lon). */
    fun offsetMeters(
        lat: Double,
        lon: Double,
        bearingDegrees: Double,
        distanceMeters: Double,
    ): Pair<Double, Double> {
        val bearing = Math.toRadians(bearingDegrees)
        val lat1 = Math.toRadians(lat)
        val lon1 = Math.toRadians(lon)
        val angularDistance = distanceMeters / EARTH_RADIUS_METERS
        val lat2 = kotlin.math.asin(
            kotlin.math.sin(lat1) * kotlin.math.cos(angularDistance) +
                kotlin.math.cos(lat1) * kotlin.math.sin(angularDistance) * kotlin.math.cos(bearing),
        )
        val lon2 = lon1 + kotlin.math.atan2(
            kotlin.math.sin(bearing) * kotlin.math.sin(angularDistance) * kotlin.math.cos(lat1),
            kotlin.math.cos(angularDistance) - kotlin.math.sin(lat1) * kotlin.math.sin(lat2),
        )
        return Math.toDegrees(lat2) to Math.toDegrees(lon2)
    }

    /** Returns distance in meters between two lat/lng points. */
    fun distanceMeters(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Double {
        val dLat = Math.toRadians(lat2 - lat1)
        val dLon = Math.toRadians(lon2 - lon1)
        val a = sin(dLat / 2) * sin(dLat / 2) +
                cos(Math.toRadians(lat1)) * cos(Math.toRadians(lat2)) *
                sin(dLon / 2) * sin(dLon / 2)
        val c = 2 * atan2(sqrt(a), sqrt(1 - a))
        return EARTH_RADIUS_METERS * c
    }
}
