package com.pegasus.retailer

import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.OrderStatus
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Test

class OrderLifecycleTests {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testOrderSerialization() {
        val rawJson = """
            {
                "order_id": "ord_001",
                "retailer_id": "ret_100",
                "supplier_id": "sup_200",
                "supplier_name": "Test Supplier",
                "state": "PENDING",
                "amount": 1500000,
                "currency": "UZS",
                "items": [
                    {
                        "line_item_id": "item_1",
                        "sku_id": "sku_1",
                        "sku_name": "Product A",
                        "quantity": 5,
                        "unit_price": 300000.0,
                        "total_price": 1500000.0
                    }
                ]
            }
        """.trimIndent()

        val order = json.decodeFromString<Order>(rawJson)
        assertNotNull(order)
        assertEquals("ord_001", order.id)
        assertEquals("ret_100", order.retailerId)
        assertEquals(OrderStatus.PENDING, order.status)
        assertEquals(1500000L, order.totalAmount)
        assertEquals("UZS", order.currency)
        
        assertEquals(1, order.items.size)
        val item = order.items.first()
        assertEquals("item_1", item.id)
        assertEquals("sku_1", item.productId)
        assertEquals(5, item.quantity)
    }
}
