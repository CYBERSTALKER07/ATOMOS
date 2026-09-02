package com.pegasusx.warehouse

import com.pegasus.design.ui.ORDER_STATUS_FUNNEL
import com.pegasus.design.ui.TRUCK_DUTY_STATUSES
import com.pegasus.design.incrementOrderStatusCount
import com.pegasus.design.statusStackModel
import com.pegasusx.warehouse.data.model.DashboardData
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DashboardTests {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testFullStacksAndDemandSource() {
        val raw = """
            {
              "pending_dispatch": 2,
              "orders_by_status": { "PENDING": 2, "FISCAL_FAILED": 1 },
              "truck_duty": {
                "AVAILABLE": 1,
                "OFF_SHIFT": 1,
                "RETURNING_TO_WAREHOUSE": 1,
                "UNASSIGNED": 1,
                "VEHICLE_INACTIVE": 1
              },
              "hold_reasons": [{ "code": "MAINTENANCE", "count": 1 }],
              "demand_source": "empty",
              "history_available": false
            }
        """.trimIndent()
        val dash = json.decodeFromString<DashboardData>(raw)
        assertEquals(2, dash.ordersByStatus["PENDING"])
        assertEquals(1, dash.ordersByStatus["FISCAL_FAILED"])
        assertEquals(17, ORDER_STATUS_FUNNEL.size)
        assertTrue(TRUCK_DUTY_STATUSES.containsAll(listOf("OFF_SHIFT", "RETURNING_TO_WAREHOUSE", "UNASSIGNED", "VEHICLE_INACTIVE")))
        assertEquals(1, dash.truckDuty["VEHICLE_INACTIVE"])
        assertEquals("MAINTENANCE", dash.holdReasons.first().code)
        assertEquals("empty", dash.demandSource)
        assertEquals(false, dash.historyAvailable)
        val next = incrementOrderStatusCount(dash.ordersByStatus, "FISCAL_FAILED")
        assertEquals(2, next["FISCAL_FAILED"])
        val stack = statusStackModel(TRUCK_DUTY_STATUSES, dash.truckDuty)
        assertEquals(8, stack.rows.size)
    }

    @Test
    fun pulseFailureDoesNotTreatAsEmptyTimeline() {
        val src = java.io.File("src/main/java/com/pegasusx/warehouse/ui/screens/dispatch/DispatchScreen.kt").readText()
        assertTrue(src.contains("PulseHonesty.FAILED"))
        assertTrue(src.contains("error = handoffError"))
        assertTrue(!src.contains("handoffEvents = emptyList()"))
    }
}
