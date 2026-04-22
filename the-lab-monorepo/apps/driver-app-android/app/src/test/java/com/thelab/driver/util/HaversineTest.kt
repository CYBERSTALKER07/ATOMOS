package com.thelab.driver.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import kotlin.math.PI
import kotlin.math.abs

class HaversineTest {

    @Test
    fun samePoint_returnsZero() {
        val d = Haversine.distanceMeters(41.2995, 69.2401, 41.2995, 69.2401)
        assertTrue("Same point should be ~0m, got $d", d < 1.0)
    }

    @Test
    fun knownDistance_tashkentToSamarkand() {
        val d = Haversine.distanceMeters(41.2995, 69.2401, 39.6542, 66.9597)
        val km = d / 1000.0
        assertTrue("Tashkent↔Samarkand ≈ 262km, got ${km}km", km in 250.0..280.0)
    }

    @Test
    fun nearbyPoint_under100m() {
        // ~0.0004° lat ≈ 44m
        val d = Haversine.distanceMeters(41.2995, 69.2401, 41.2999, 69.2401)
        assertTrue("Expected 30–100m, got ${d}m", d in 30.0..100.0)
    }

    @Test
    fun geofenceBoundary_100m() {
        // ~0.0009° lat ≈ 100m
        val d = Haversine.distanceMeters(41.2995, 69.2401, 41.3004, 69.2401)
        assertTrue("Expected near 100m boundary, got ${d}m", d in 90.0..120.0)
    }

    @Test
    fun antipodal_returnsHalfCircumference() {
        // (0°,0°) ↔ (0°,180°) ≈ half Earth circumference ≈ 20015 km
        val d = Haversine.distanceMeters(0.0, 0.0, 0.0, 180.0)
        val km = d / 1000.0
        val halfCircum = PI * 6371.0 // ≈ 20015 km
        assertTrue(
            "Expected ~${halfCircum}km, got ${km}km",
            abs(km - halfCircum) < 50
        )
    }
}
