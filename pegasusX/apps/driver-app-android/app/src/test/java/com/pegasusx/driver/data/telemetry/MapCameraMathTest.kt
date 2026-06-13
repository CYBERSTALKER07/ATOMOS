package com.pegasusx.driver.data.telemetry

import org.junit.Assert.assertEquals
import org.junit.Test

class MapCameraMathTest {
    @Test
    fun dynamicZoom_clampsBetweenMinAndMax() {
        assertEquals(MapCameraConfig.MAX_ZOOM, MapCameraMath.dynamicZoom(0f), 0.01f)
        assertEquals(18f, MapCameraMath.dynamicZoom(6f), 0.01f)
        assertEquals(MapCameraConfig.MIN_ZOOM, MapCameraMath.dynamicZoom(100f), 0.01f)
    }
}
