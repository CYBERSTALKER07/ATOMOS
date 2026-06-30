package com.pegasusx.supplier

import com.pegasusx.supplier.data.model.SupplierDashboard
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
    }
}
