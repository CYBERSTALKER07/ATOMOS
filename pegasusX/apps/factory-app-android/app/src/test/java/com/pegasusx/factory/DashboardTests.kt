package com.pegasusx.factory

import com.pegasus.design.FACTORY_TRANSFER_STATES
import com.pegasus.design.MANIFEST_STATES
import com.pegasus.design.statusStackModel
import com.pegasusx.factory.data.model.DashboardStats
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class DashboardTests {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testFactoryCommandStacksAndSource() {
        val raw = """
            {
              "source": "spanner",
              "plane": "factory_trucks",
              "pending_transfers": 1,
              "transfers_by_state": { "CREATED": 1, "CANCELLED": 1 },
              "manifests_by_state": { "LOADING": 1, "SEALED": 1 },
              "vehicles_by_state": { "READY": 1, "UNAVAILABLE": 1 },
              "driver_duty": { "ON_SHIFT": 1, "OFF_SHIFT": 1 },
              "qc_available": true,
              "qc_by_result": { "FAIL": 1 }
            }
        """.trimIndent()
        val dash = json.decodeFromString(DashboardStats.serializer(), raw)
        assertEquals("spanner", dash.source)
        assertEquals("factory_trucks", dash.plane)
        assertEquals(1, dash.transfersByState["CREATED"])
        assertEquals(1, dash.manifestsByState["SEALED"])
        assertEquals(1, dash.vehiclesByState["UNAVAILABLE"])
        assertEquals(1, dash.driverDuty["OFF_SHIFT"])
        assertTrue(FACTORY_TRANSFER_STATES.containsAll(listOf("CREATED", "PENDING", "DISPATCHED", "CANCELLED")))
        assertTrue(MANIFEST_STATES.contains("DRAFT"))
        assertEquals(FACTORY_TRANSFER_STATES.size, statusStackModel(FACTORY_TRANSFER_STATES, dash.transfersByState).rows.size)
        assertFalse(FACTORY_TRANSFER_STATES.contains("FISCAL_FAILED"))
    }

    @Test
    fun pulseFailureDoesNotTreatAsEmptyTimeline() {
        val src = java.io.File("src/main/java/com/pegasusx/factory/ui/screens/loadingbay/LoadingBayScreen.kt").readText()
        assertTrue(src.contains("PulseHonesty.FAILED"))
        assertTrue(src.contains("handoffError = handoffError"))
        assertTrue(!src.contains("handoffEvents = emptyList()"))
    }
}
