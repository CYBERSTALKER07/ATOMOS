package com.thelab.retailer.data.api

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class RetailerWSMessageTest {

    private val json = Json { ignoreUnknownKeys = true }

    // ── Full deserialization ────────────────────────────────────────────────

    @Test
    fun `deserialize full payment_ready message`() {
        val raw = """
            {
                "type": "PAYMENT_READY",
                "order_id": "ORD-001",
                "invoice_id": "INV-001",
                "session_id": "SESS-001",
                "amount": 150000,
                "original_amount": 160000,
                "available_card_gateways": ["PAYME", "CLICK"],
                "message": "Payment ready",
                "delivery_token": "tok_abc",
                "payment_method": "CARD",
                "gateway": "PAYME",
                "driver_latitude": 41.2995,
                "driver_longitude": 69.2401,
                "supplier_id": "SUP-001",
                "supplier_name": "Test Supplier",
                "state": "ARRIVED",
                "timestamp": "2026-04-12T10:00:00Z"
            }
        """.trimIndent()
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertEquals("PAYMENT_READY", msg.type)
        assertEquals("ORD-001", msg.orderId)
        assertEquals("INV-001", msg.invoiceId)
        assertEquals("SESS-001", msg.sessionId)
        assertEquals(150000L, msg.amount)
        assertEquals(160000L, msg.originalAmount)
        assertEquals(listOf("PAYME", "CLICK"), msg.availableCardGateways)
        assertEquals("Payment ready", msg.message)
        assertEquals("tok_abc", msg.deliveryToken)
        assertEquals("PAYME", msg.gateway)
        assertEquals(41.2995, msg.driverLatitude!!, 0.001)
        assertEquals(69.2401, msg.driverLongitude!!, 0.001)
        assertEquals("SUP-001", msg.supplierId)
        assertEquals("ARRIVED", msg.state)
    }

    // ── Minimal deserialization (defaults) ──────────────────────────────────

    @Test
    fun `deserialize minimal message with only type`() {
        val raw = """{"type":"PING"}"""
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertEquals("PING", msg.type)
        assertEquals("", msg.orderId)
        assertEquals("", msg.invoiceId)
        assertEquals(0L, msg.amount)
        assertEquals(emptyList<String>(), msg.availableCardGateways)
        assertEquals(null, msg.driverLatitude)
        assertEquals(null, msg.driverLongitude)
    }

    // ── Type field values ───────────────────────────────────────────────────

    @Test
    fun `type field preserves exact value`() {
        val types = listOf(
            "PAYMENT_READY", "ORDER_UPDATE", "DRIVER_APPROACHING",
            "OFFLOAD_CONFIRMED", "PAYMENT_SUCCESS", "PAYMENT_FAILED"
        )
        for (typeName in types) {
            val raw = """{"type":"$typeName"}"""
            val msg = json.decodeFromString<RetailerWSMessage>(raw)
            assertEquals(typeName, msg.type)
        }
    }

    // ── Gateway list parsing ────────────────────────────────────────────────

    @Test
    fun `available_card_gateways parses multiple gateways`() {
        val raw = """{"type":"PAYMENT_READY","available_card_gateways":["PAYME","CLICK","UZCARD"]}"""
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertEquals(3, msg.availableCardGateways.size)
        assertTrue(msg.availableCardGateways.contains("PAYME"))
        assertTrue(msg.availableCardGateways.contains("CLICK"))
        assertTrue(msg.availableCardGateways.contains("UZCARD"))
    }

    @Test
    fun `available_card_gateways defaults to empty list`() {
        val raw = """{"type":"ORDER_UPDATE","order_id":"ORD-002"}"""
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertTrue(msg.availableCardGateways.isEmpty())
    }

    // ── Amount fields ───────────────────────────────────────────────────────

    @Test
    fun `amended order has different original and current amounts`() {
        val raw = """{"type":"PAYMENT_READY","amount":140000,"original_amount":150000}"""
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertEquals(140000L, msg.amount)
        assertEquals(150000L, msg.originalAmount)
        assertTrue(msg.amount < msg.originalAmount)
    }

    @Test
    fun `non-amended order has zero original amount by default`() {
        val raw = """{"type":"PAYMENT_READY","amount":150000}"""
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertEquals(150000L, msg.amount)
        assertEquals(0L, msg.originalAmount)
    }

    // ── Driver location ─────────────────────────────────────────────────────

    @Test
    fun `driver location null when not provided`() {
        val raw = """{"type":"ORDER_UPDATE","order_id":"ORD-003"}"""
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertEquals(null, msg.driverLatitude)
        assertEquals(null, msg.driverLongitude)
    }
}
