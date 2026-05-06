package com.pegasus.retailer.data.api

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class RetailerWSMessageTest {

    private val json = Json { ignoreUnknownKeys = true }

    // ── Full deserialization ────────────────────────────────────────────────

    @Test
    fun `deserialize full payment_required message`() {
        val raw = """
            {
                "type": "PAYMENT_REQUIRED",
                "order_id": "ORD-001",
                "invoice_id": "INV-001",
                "session_id": "SESS-001",
                "amount": 150000,
                "original_amount": 160000,
                "available_card_gateways": ["GLOBAL_PAY", "CASH"],
                "message": "Payment required",
                "delivery_token": "tok_abc",
                "payment_method": "CARD",
                "gateway": "GLOBAL_PAY",
                "driver_latitude": 41.2995,
                "driver_longitude": 69.2401,
                "supplier_id": "SUP-001",
                "supplier_name": "Test Supplier",
                "state": "ARRIVED",
                "timestamp": "2026-04-12T10:00:00Z"
            }
        """.trimIndent()
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertEquals("PAYMENT_REQUIRED", msg.type)
        assertEquals("ORD-001", msg.orderId)
        assertEquals("INV-001", msg.invoiceId)
        assertEquals("SESS-001", msg.sessionId)
        assertEquals(150000L, msg.amount)
        assertEquals(160000L, msg.originalAmount)
        assertEquals(listOf("GLOBAL_PAY", "CASH"), msg.availableCardGateways)
        assertEquals("Payment required", msg.message)
        assertEquals("tok_abc", msg.deliveryToken)
        assertEquals("GLOBAL_PAY", msg.gateway)
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
            "PAYMENT_REQUIRED", "PAYMENT_SETTLED", "PAYMENT_FAILED",
            "PAYMENT_EXPIRED", "DRIVER_APPROACHING", "ORDER_COMPLETED"
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
        val raw = """{"type":"PAYMENT_REQUIRED","available_card_gateways":["GLOBAL_PAY","CASH"]}"""
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertEquals(2, msg.availableCardGateways.size)
        assertTrue(msg.availableCardGateways.contains("GLOBAL_PAY"))
        assertTrue(msg.availableCardGateways.contains("CASH"))
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
        val raw = """{"type":"PAYMENT_REQUIRED","amount":140000,"original_amount":150000}"""
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertEquals(140000L, msg.amount)
        assertEquals(150000L, msg.originalAmount)
        assertTrue(msg.amount < msg.originalAmount)
    }

    @Test
    fun `non-amended order has zero original amount by default`() {
        val raw = """{"type":"PAYMENT_REQUIRED","amount":150000}"""
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

    @Test
    fun `supplier aliases remain available for completion events`() {
        val raw = """{"type":"ORDER_COMPLETED","order_id":"ORD-004","supplier_id":"SUP-777","supplier_name":"Supplier Seven"}"""
        val msg = json.decodeFromString<RetailerWSMessage>(raw)
        assertEquals("ORDER_COMPLETED", msg.type)
        assertEquals("SUP-777", msg.supplierId)
        assertEquals("Supplier Seven", msg.supplierName)
    }
}
