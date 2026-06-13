package com.pegasusx.driver.data.telemetry

/**
 * Shared driver-map camera constants — keep aligned with iOS MapCameraConfig.
 */
object MapCameraConfig {
    const val IDLE_RECENTER_MS = 8_000L
    const val INTERPOLATION_FRAME_MS = 33L
    const val CAMERA_ANIMATION_MS = 500
    const val MIN_ZOOM = 14f
    const val MAX_ZOOM = 19f
    const val NAVIGATION_TILT = 60f
    const val SPEED_ZOOM_DIVISOR = 6f
    const val LOOK_AHEAD_SECONDS = 2.0
    const val MAX_LOOK_AHEAD_METERS = 200.0
}
