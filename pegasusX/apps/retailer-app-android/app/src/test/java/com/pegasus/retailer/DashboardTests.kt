package com.pegasus.retailer

import com.pegasus.design.ORDER_STATUS_FUNNEL
import com.pegasus.design.statusStackModel
import com.pegasusx.retailer.data.model.ControlTowerPulse
import com.pegasusx.retailer.ui.retailerOrderMatchesCommand
import com.pegasusx.retailer.data.model.RetailerDetailedAnalytics
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class DashboardTests {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testControlTowerPulseFiscalFailedFacet() {
        val raw = """
            {
              "source": "spanner",
              "empty": false,
              "open_orders": 1,
              "loyalty": { "enrolled": false },
              "orders_by_status": { "FISCAL_FAILED": 1, "COMPLETED": 1 },
              "orders_by_supplier": [
                { "supplier_id": "sup-a", "orders_by_status": { "FISCAL_FAILED": 1 } },
                { "supplier_id": "sup-b", "orders_by_status": { "COMPLETED": 1 } }
              ]
            }
        """.trimIndent()
        val pulse = json.decodeFromString(ControlTowerPulse.serializer(), raw)
        assertEquals("spanner", pulse.source)
        assertFalse(pulse.empty)
        assertEquals(1, pulse.ordersByStatus["FISCAL_FAILED"])
        assertEquals(1, pulse.ordersBySupplier.first { it.supplierId == "sup-a" }.ordersByStatus["FISCAL_FAILED"])
        assertEquals(1, pulse.ordersBySupplier.first { it.supplierId == "sup-b" }.ordersByStatus["COMPLETED"])
        assertFalse(pulse.loyalty.enrolled)
        assertTrue(ORDER_STATUS_FUNNEL.contains("FISCAL_FAILED"))
        assertEquals(17, ORDER_STATUS_FUNNEL.size)
        assertEquals(17, statusStackModel(ORDER_STATUS_FUNNEL, pulse.ordersByStatus).rows.size)
    }

    @Test
    fun testRetailerDetailedAnalyticsSerialization() {
        val rawJson = """
            {
                "total_spent": 1500000,
                "total_orders": 15,
                "avg_order_value": 100000,
                "orders_by_state": [
                    {
                        "state": "PENDING",
                        "count": 2
                    }
                ]
            }
        """.trimIndent()

        val analytics = json.decodeFromString<RetailerDetailedAnalytics>(rawJson)
        assertEquals(1500000L, analytics.totalSpent)
        assertEquals(15L, analytics.totalOrders)
        assertEquals(100000L, analytics.avgOrderValue)
        assertEquals(1, analytics.ordersByState.size)
        assertEquals("PENDING", analytics.ordersByState.first().state)
        assertEquals(2L, analytics.ordersByState.first().count)
    }

    @Test
    fun testCommandChipMatchesCanonicalStatusAndSupplier() {
        assertTrue(retailerOrderMatchesCommand("FISCAL_FAILED", "sup-a", "FISCAL_FAILED", null))
        assertTrue(retailerOrderMatchesCommand("DISPATCHED", "sup-a", "LOADED", "sup-a"))
        assertFalse(retailerOrderMatchesCommand("FISCAL_FAILED", "sup-a", "FISCAL_FAILED", "sup-b"))
    }

    @Test
    fun pulseFailureDoesNotTreatAsEmptyTimeline() {
        val src = java.io.File("src/main/java/com/pegasusx/retailer/ui/screens/dashboard/DashboardViewModel.kt").readText()
        assertTrue(src.contains("PulseHonesty.FAILED"))
        assertTrue(!src.contains("pulseEvents = emptyList()"))
        val screen = java.io.File("src/main/java/com/pegasusx/retailer/ui/screens/dashboard/DashboardScreen.kt").readText()
        assertTrue(screen.contains("error = uiState.pulseError"))
    }

    @Test
    fun commandPulseFailureDoesNotTreatAsEmpty() {
        val src = java.io.File("src/main/java/com/pegasusx/retailer/ui/screens/dashboard/DashboardViewModel.kt").readText()
        assertTrue(src.contains("PulseHonesty.applyObject"))
        assertTrue(src.contains("commandPulseError = applied.error"))
        val screen = java.io.File("src/main/java/com/pegasusx/retailer/ui/screens/dashboard/DashboardScreen.kt").readText()
        assertTrue(screen.contains("commandPulseError"))
        assertTrue(screen.contains("commandError"))
        val tower = java.io.File("src/main/java/com/pegasusx/retailer/ui/controltower/ControlTowerScreen.kt").readText()
        assertTrue(tower.contains("PulseHonesty.COMMAND_FAILED"))
        assertTrue(screen.contains("commandPulseError.isNullOrBlank()"))
        assertTrue(screen.contains("DashboardOverviewCard"))
    }

    @Test
    fun reportsFailureDoesNotTreatAsZeroDigest() {
        val src = java.io.File("src/main/java/com/pegasusx/retailer/ui/screens/settings/ReportsScreen.kt").readText()
        assertTrue(src.contains("reports_failed"))
        assertTrue(src.contains("summaryReady && loadError == null"))
        assertTrue(src.contains("loadError = \"reports_failed\""))
    }

    @Test
    fun hqFailureDoesNotTreatAsEmptyRows() {
        val src = java.io.File("src/main/java/com/pegasusx/retailer/ui/screens/hq/HqScreen.kt").readText()
        assertTrue(src.contains("hq_failed"))
        assertTrue(src.contains("summaryReady && loadError == null"))
        assertTrue(src.contains("loadError = \"hq_failed\""))
    }

    @Test
    fun localSkusFailureDoesNotTreatAsEmptyCatalog() {
        val src = java.io.File("src/main/java/com/pegasusx/retailer/ui/screens/settings/LocalSkusScreen.kt").readText()
        assertTrue(src.contains("local_skus_failed"))
        assertTrue(src.contains("summaryReady && loadError == null"))
        assertTrue(src.contains("loadError = \"local_skus_failed\""))
    }

    @Test
    fun sectionSkuPutFailureDoesNotTreatAsSaved() {
        val src = java.io.File("src/main/java/com/pegasusx/retailer/ui/screens/settings/SectionsScreen.kt").readText()
        assertTrue(src.contains("section_skus_failed"))
        assertTrue(src.contains("saveError = \"section_skus_failed\""))
        assertTrue(src.contains("saveError == null"))
    }
}
