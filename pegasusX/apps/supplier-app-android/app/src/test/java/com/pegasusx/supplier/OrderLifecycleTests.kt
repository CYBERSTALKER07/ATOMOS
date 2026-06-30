package com.pegasusx.supplier

import com.pegasusx.supplier.data.model.SupplierOrder
import com.pegasusx.supplier.data.model.SupplierOrdersResponse
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Test

class OrderLifecycleTests {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testSupplierOrdersResponseSerialization() {
        val rawJson = """
            {
                "orders": [
                    {
                        "order_id": "ord_001",
                        "retailer_id": "ret_100",
                        "warehouse_id": "wh_01",
                        "status": "PENDING",
                        "total_minor": 1500000,
                        "currency": "UZS",
                        "updated_at": "2026-06-30T10:00:00Z"
                    }
                ],
                "total": 1,
                "limit": 50,
                "offset": 0
            }
        """.trimIndent()

        val response = json.decodeFromString<SupplierOrdersResponse>(rawJson)
        assertNotNull(response.orders)
        assertEquals(1, response.orders.size)
        
        val order = response.orders.first()
        assertEquals("ord_001", order.orderId)
        assertEquals("ret_100", order.retailerId)
        assertEquals("PENDING", order.status)
        assertEquals(1500000L, order.totalMinor)
        assertEquals("UZS", order.currency)
    }
}
