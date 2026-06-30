package com.pegasus.retailer

import com.pegasusx.retailer.data.api.RetailerWSMessage
import com.pegasusx.retailer.data.api.toShopClosedAlert
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Test

class WebSocketTests {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testRetailerWSMessageSerialization() {
        val rawJson = """
            {
                "type": "ORDER_UPDATED",
                "order_id": "ord_123",
                "state": "ACCEPTED",
                "timestamp": "2023-10-27T10:00:00Z"
            }
        """.trimIndent()

        val msg = json.decodeFromString<RetailerWSMessage>(rawJson)
        assertEquals("ORDER_UPDATED", msg.type)
        assertEquals("ord_123", msg.orderId)
        assertEquals("ACCEPTED", msg.state)
        assertEquals("2023-10-27T10:00:00Z", msg.timestamp)
    }

    @Test
    fun testShopClosedAlertConversion() {
        val msg = RetailerWSMessage(
            type = "SHOP_CLOSED_ALERT",
            orderId = "ord_456",
            driverName = "John Doe",
            attemptId = "attempt_1",
            options = listOf("CALL_ME", "OPEN_NOW")
        )
        val alert = msg.toShopClosedAlert()
        assertNotNull(alert)
        assertEquals("ord_456", alert?.orderId)
        assertEquals("John Doe", alert?.driverName)
        assertEquals("attempt_1", alert?.attemptId)
        assertEquals(2, alert?.options?.size)
    }

    @Test
    fun testInvalidShopClosedAlertConversion() {
        val msg = RetailerWSMessage(
            type = "ORDER_UPDATED",
            orderId = "ord_456"
        )
        val alert = msg.toShopClosedAlert()
        assertNull(alert)
    }
}
