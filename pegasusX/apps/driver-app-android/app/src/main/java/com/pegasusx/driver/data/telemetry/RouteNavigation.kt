package com.pegasusx.driver.data.telemetry

import com.pegasusx.driver.data.model.RouteStep
import com.pegasusx.driver.util.Haversine
import kotlin.math.roundToInt

const val NAVIGATION_STEP_PASSED_METERS = 35.0

data class NavigationCue(
    val instruction: String,
    val distanceM: Double,
    val maneuver: String? = null,
)

fun advanceNavigationStepIndex(
    currentIndex: Int,
    steps: List<RouteStep>,
    lat: Double,
    lng: Double,
): Int {
    if (steps.isEmpty()) {
        return 0
    }
    var lastPassedIndex = -1
    for (index in steps.indices) {
        val step = steps[index]
        val distance = Haversine.distanceMeters(lat, lng, step.lat, step.lng)
        if (distance <= NAVIGATION_STEP_PASSED_METERS) {
            lastPassedIndex = index
        }
    }
    if (lastPassedIndex < 0) {
        return currentIndex.coerceIn(0, steps.lastIndex)
    }
    return (lastPassedIndex + 1).coerceAtMost(steps.lastIndex)
}

fun resolveNavigationCue(
    steps: List<RouteStep>,
    stepIndex: Int,
    lat: Double,
    lng: Double,
): NavigationCue? {
    val step = steps.getOrNull(stepIndex) ?: return null
    val distance = Haversine.distanceMeters(lat, lng, step.lat, step.lng)
    return NavigationCue(
        instruction = step.instruction,
        distanceM = distance,
        maneuver = step.maneuver,
    )
}

fun formatNavigationDistance(meters: Double): String {
    return if (meters < 1000) {
        "${meters.roundToInt()} m"
    } else {
        String.format("%.1f km", meters / 1000.0)
    }
}
