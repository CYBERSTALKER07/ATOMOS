package com.thelab.driver.data.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class DriverModelContractTest {

    // ── RejectionReason ──────────────────────────────────────────────────────

    @Test
    fun rejectionReason_has4Values() {
        assertEquals(4, RejectionReason.values().size)
    }

    @Test
    fun rejectionReason_damagedLabel() {
        assertEquals("Damaged", RejectionReason.DAMAGED.label)
    }

    @Test
    fun rejectionReason_missingLabel() {
        assertEquals("Missing", RejectionReason.MISSING.label)
    }

    @Test
    fun rejectionReason_wrongItemLabel() {
        assertEquals("Wrong Item", RejectionReason.WRONG_ITEM.label)
    }

    @Test
    fun rejectionReason_otherLabel() {
        assertEquals("Other", RejectionReason.OTHER.label)
    }

    @Test
    fun rejectionReason_allLabelsNonEmpty() {
        for (reason in RejectionReason.values()) {
            assertTrue("Label for ${reason.name} should not be empty", reason.label.isNotEmpty())
        }
    }

    // ── OrderLineItem ────────────────────────────────────────────────────────

    @Test
    fun orderLineItem_computedTotal() {
        val item = OrderLineItem(
            productId = "PROD-001",
            productName = "Water 1.5L",
            quantity = 10,
            unitPrice = 5000L,
            lineTotal = 50000L
        )
        assertEquals(50000L, item.lineTotal)
        assertEquals(10, item.quantity)
    }

    // ── RouteManifest ────────────────────────────────────────────────────────

    @Test
    fun routeManifest_totalStops_matchesOrders() {
        val manifest = RouteManifest(
            driverId = "DRV-001",
            date = "2026-04-12",
            orders = listOf(
                makeOrder("ORD-1"), makeOrder("ORD-2"), makeOrder("ORD-3")
            ),
            totalStops = 3,
            estimatedDistanceKm = 15.5
        )
        assertEquals(3, manifest.totalStops)
        assertEquals(manifest.orders.size, manifest.totalStops)
    }

    @Test
    fun routeManifest_nullDistance() {
        val manifest = RouteManifest(
            driverId = "DRV-001",
            date = "2026-04-12",
            orders = emptyList(),
            totalStops = 0,
            estimatedDistanceKm = null
        )
        assertEquals(null, manifest.estimatedDistanceKm)
    }

    // ── DeliverySubmitRequest ────────────────────────────────────────────────

    @Test
    fun deliverySubmitRequest_fields() {
        val req = DeliverySubmitRequest(
            orderId = "ORD-001",
            qrToken = "tok_abc",
            latitude = 41.2995,
            longitude = 69.2401
        )
        assertEquals("ORD-001", req.orderId)
        assertEquals("tok_abc", req.qrToken)
        assertEquals(41.2995, req.latitude, 0.001)
    }

    // ── AmendItemPayload ─────────────────────────────────────────────────────

    @Test
    fun amendItemPayload_damagedReason() {
        val payload = AmendItemPayload(
            productId = "PROD-001",
            acceptedQty = 8,
            rejectedQty = 2,
            reason = RejectionReason.DAMAGED.name
        )
        assertEquals("DAMAGED", payload.reason)
        assertEquals(2, payload.rejectedQty)
        assertEquals(8, payload.acceptedQty)
    }

    @Test
    fun amendItemPayload_totalPreserved() {
        val payload = AmendItemPayload(
            productId = "PROD-001",
            acceptedQty = 7,
            rejectedQty = 3,
            reason = "MISSING"
        )
        assertEquals(10, payload.acceptedQty + payload.rejectedQty)
    }

    // ── TelemetryPayload ─────────────────────────────────────────────────────

    @Test
    fun telemetryPayload_defaultSpeedBearing() {
        val payload = TelemetryPayload(
            driverId = "DRV-001",
            latitude = 41.2995,
            longitude = 69.2401,
            timestamp = System.currentTimeMillis()
        )
        assertEquals(0f, payload.speed, 0.001f)
        assertEquals(0f, payload.bearing, 0.001f)
    }

    // ── CollectCashResponse ──────────────────────────────────────────────────

    @Test
    fun collectCashResponse_defaults() {
        val resp = CollectCashResponse(orderId = "ORD-001")
        assertEquals("", resp.state)
        assertEquals(0L, resp.amount)
        assertEquals(0.0, resp.distanceM, 0.001)
        assertEquals("", resp.message)
    }

    // ── Helper ───────────────────────────────────────────────────────────────

    private fun makeOrder(id: String) = Order(
        id = id,
        retailerId = "RET-001",
        retailerName = "Test Shop",
        state = OrderState.PENDING,
        totalAmount = 100000L,
        deliveryAddress = "Test St",
        createdAt = "2026-04-12T10:00:00Z",
        updatedAt = "2026-04-12T10:00:00Z"
    )
}
