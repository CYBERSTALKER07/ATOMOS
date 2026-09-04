package com.pegasusx.supplier

import com.pegasus.design.ui.ORDER_STATUS_FUNNEL
import com.pegasus.design.ui.StatusStackMode
import com.pegasus.design.network.formatPackMoney
import com.pegasus.design.incrementOrderStatusCount
import com.pegasus.design.statusStackModel
import com.pegasusx.supplier.data.model.SupplierDashboard
import com.pegasusx.supplier.ui.viewmodel.OrderFilterTab
import com.pegasusx.supplier.ui.viewmodel.resolveSupplierOrdersQuery
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DashboardTests {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testDashboardSerialization() {
        val rawJson = """
            {
                "supplier_id": "sup_123",
                "is_configured": true,
                "inventory_skus": 150,
                "pending_orders": 25,
                "updated_at": "2026-06-30T10:00:00Z"
            }
        """.trimIndent()

        val response = json.decodeFromString<SupplierDashboard>(rawJson)
        assertEquals("sup_123", response.supplierId)
        assertTrue(response.isConfigured)
        assertEquals(150, response.inventorySKUs)
        assertEquals(25, response.pendingOrders)
        assertEquals("2026-06-30T10:00:00Z", response.updatedAt)
        assertTrue(response.ordersByStatus.isEmpty())
    }

    @Test
    fun testOrdersByStatusAndStackModes() {
        val rawJson = """
            {
                "supplier_id": "sup_123",
                "is_configured": true,
                "inventory_skus": 150,
                "pending_orders": 25,
                "updated_at": "2026-06-30T10:00:00Z",
                "orders_by_status": { "PENDING": 3, "COMPLETED": 1 }
            }
        """.trimIndent()

        val dash = json.decodeFromString<SupplierDashboard>(rawJson)
        assertEquals(3, dash.ordersByStatus["PENDING"])
        assertEquals(17, ORDER_STATUS_FUNNEL.size)

        val empty = statusStackModel(counts = null)
        assertEquals(StatusStackMode.Empty, empty.mode)
        assertTrue(empty.rows.isEmpty())

        val zero = statusStackModel(counts = emptyMap())
        assertEquals(StatusStackMode.Zero, zero.mode)
        assertEquals(17, zero.rows.size)

        val unavailable = statusStackModel(counts = dash.ordersByStatus, available = false)
        assertEquals(StatusStackMode.Unavailable, unavailable.mode)
        assertTrue(unavailable.rows.all { it.count == null })

        val live = statusStackModel(counts = dash.ordersByStatus)
        assertEquals(StatusStackMode.Live, live.mode)
        assertEquals(17, live.rows.size)
        assertEquals(4, live.total)
        assertEquals(0, live.rows.first { it.key == "LOADED" }.count)
    }

    @Test
    fun testFiscalFailedIncrementsChip() {
        val next = incrementOrderStatusCount(emptyMap(), "FISCAL_FAILED")
        assertEquals(1, next["FISCAL_FAILED"])
        assertEquals(17, next.size)
        assertEquals(1, incrementOrderStatusCount(next, "DISPATCHED")["LOADED"])
    }

    @Test
    fun testRevenueParsesWithoutInventingUzs() {
        val rawJson = """
            {
                "supplier_id": "sup_123",
                "is_configured": true,
                "inventory_skus": 150,
                "pending_orders": 25,
                "updated_at": "2026-06-30T10:00:00Z",
                "orders_by_status": { "FISCAL_FAILED": 1 },
                "today_revenue_minor": 1500
            }
        """.trimIndent()
        val dash = json.decodeFromString<SupplierDashboard>(rawJson)
        assertEquals(1500L, dash.todayRevenueMinor)
        assertEquals(1, dash.ordersByStatus["FISCAL_FAILED"])
        val money = formatPackMoney(dash.todayRevenueMinor, null)
        assertEquals("15", money)
        assertTrue(!money.contains("UZS"))
    }

    @Test
    fun testCommandChipUsesFunnelStatusNotCoarseTab() {
        val fiscal = resolveSupplierOrdersQuery("FISCAL_FAILED", OrderFilterTab.ACTIVE)
        assertEquals("FISCAL_FAILED", fiscal.status)
        assertEquals(null, fiscal.filter)
        val dispatched = resolveSupplierOrdersQuery("DISPATCHED", OrderFilterTab.ACTIVE)
        assertEquals("LOADED", dispatched.status)
        val coarse = resolveSupplierOrdersQuery(null, OrderFilterTab.ACTIVE)
        assertEquals("ACTIVE", coarse.filter)
        assertEquals(null, coarse.status)
    }

    @Test
    fun pulseFailureDoesNotTreatAsEmptyTimeline() {
        val src = java.io.File("src/main/java/com/pegasusx/supplier/ui/screens/dashboard/DashboardScreen.kt").readText()
        assertTrue(src.contains("PulseHonesty.FAILED"))
        assertTrue(src.contains("error = pulseError"))
        assertTrue(!src.contains("pulseEvents = emptyList()"))
    }
}
