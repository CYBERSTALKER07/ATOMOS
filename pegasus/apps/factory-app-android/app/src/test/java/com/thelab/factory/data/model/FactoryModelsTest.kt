package com.thelab.factory.data.model

import kotlinx.serialization.json.Json
import org.junit.Assert.*
import org.junit.Test

/**
 * Factory Models — Serialization, Defaults, Computed Properties
 */
class FactoryModelsTest {

    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    // ── Auth ──

    @Test
    fun `LoginRequest serializes phone and password`() {
        val req = LoginRequest(phone = "+998901234567", password = "TestPass123!")
        val encoded = json.encodeToString(LoginRequest.serializer(), req)
        assertTrue(encoded.contains("+998901234567"))
        assertTrue(encoded.contains("TestPass123!"))
    }

    @Test
    fun `AuthResponse deserializes snake_case fields`() {
        val raw = """
            {"token":"jwt-abc","refresh_token":"rt-xyz","factory_id":"fac-1","factory_name":"Test Factory"}
        """.trimIndent()
        val auth = json.decodeFromString(AuthResponse.serializer(), raw)
        assertEquals("jwt-abc", auth.token)
        assertEquals("rt-xyz", auth.refreshToken)
        assertEquals("fac-1", auth.factoryId)
        assertEquals("Test Factory", auth.factoryName)
    }

    // ── Dashboard ──

    @Test
    fun `DashboardStats defaults to zero`() {
        val raw = """{}"""
        val stats = json.decodeFromString(DashboardStats.serializer(), raw)
        assertEquals(0, stats.pendingTransfers)
        assertEquals(0, stats.loadingTransfers)
        assertEquals(0, stats.activeManifests)
        assertEquals(0, stats.dispatchedToday)
        assertEquals(0, stats.vehiclesTotal)
        assertEquals(0, stats.vehiclesAvailable)
        assertEquals(0, stats.staffOnShift)
        assertEquals(0, stats.criticalInsights)
    }

    @Test
    fun `DashboardStats deserializes full payload`() {
        val raw = """
            {
                "pending_transfers":5,"loading_transfers":3,"active_manifests":2,
                "dispatched_today":10,"vehicles_total":15,"vehicles_available":8,
                "staff_on_shift":12,"critical_insights":1
            }
        """.trimIndent()
        val stats = json.decodeFromString(DashboardStats.serializer(), raw)
        assertEquals(5, stats.pendingTransfers)
        assertEquals(3, stats.loadingTransfers)
        assertEquals(2, stats.activeManifests)
        assertEquals(10, stats.dispatchedToday)
        assertEquals(15, stats.vehiclesTotal)
        assertEquals(8, stats.vehiclesAvailable)
        assertEquals(12, stats.staffOnShift)
        assertEquals(1, stats.criticalInsights)
    }

    // ── Transfer ──

    @Test
    fun `Transfer defaults missing fields to empty`() {
        val raw = """{"id":"t-1"}"""
        val t = json.decodeFromString(Transfer.serializer(), raw)
        assertEquals("t-1", t.id)
        assertEquals("", t.factoryId)
        assertEquals("", t.warehouseId)
        assertEquals("", t.warehouseName)
        assertEquals("", t.state)
        assertEquals("", t.priority)
        assertEquals(0, t.totalItems)
        assertEquals(0.0, t.totalVolumeL, 0.001)
        assertEquals("", t.notes)
        assertTrue(t.items.isEmpty())
    }

    @Test
    fun `Transfer deserializes full payload with items`() {
        val raw = """
            {
                "id":"t-1","factory_id":"f-1","warehouse_id":"wh-1",
                "warehouse_name":"Central WH","state":"LOADING",
                "priority":"HIGH","total_items":100,"total_volume_l":250.5,
                "notes":"Urgent","created_at":"2026-01-01","updated_at":"2026-01-02",
                "items":[
                    {"id":"ti-1","product_id":"p-1","product_name":"Milk","quantity":50,"quantity_available":45,"unit_volume_l":1.2}
                ]
            }
        """.trimIndent()
        val t = json.decodeFromString(Transfer.serializer(), raw)
        assertEquals("LOADING", t.state)
        assertEquals("HIGH", t.priority)
        assertEquals(100, t.totalItems)
        assertEquals(250.5, t.totalVolumeL, 0.001)
        assertEquals(1, t.items.size)
        assertEquals("Milk", t.items[0].productName)
        assertEquals(50, t.items[0].quantity)
        assertEquals(1.2, t.items[0].unitVolumeL, 0.001)
    }

    @Test
    fun `TransferListResponse empty transfers`() {
        val raw = """{"transfers":[],"total":0}"""
        val res = json.decodeFromString(TransferListResponse.serializer(), raw)
        assertTrue(res.transfers.isEmpty())
        assertEquals(0, res.total)
    }

    // ── Transfer state machine values ──

    @Test
    fun `TransitionRequest serializes target_state`() {
        val req = TransitionRequest(targetState = "LOADING")
        val encoded = json.encodeToString(TransitionRequest.serializer(), req)
        assertTrue(encoded.contains("\"target_state\":\"LOADING\""))
    }

    // ── Vehicle ──

    @Test
    fun `Vehicle deserializes with defaults`() {
        val raw = """{"id":"v-1"}"""
        val v = json.decodeFromString(Vehicle.serializer(), raw)
        assertEquals("v-1", v.id)
        assertEquals("", v.plateNumber)
        assertEquals("", v.driverName)
        assertEquals("", v.status)
        assertEquals(0.0, v.capacityKg, 0.001)
        assertEquals(0.0, v.capacityL, 0.001)
    }

    @Test
    fun `Vehicle full deserialization`() {
        val raw = """
            {"id":"v-1","plate_number":"01A123AB","driver_name":"Ali",
             "status":"AVAILABLE","capacity_kg":5000.0,"capacity_l":12000.0,"current_route":"r-1"}
        """.trimIndent()
        val v = json.decodeFromString(Vehicle.serializer(), raw)
        assertEquals("01A123AB", v.plateNumber)
        assertEquals("Ali", v.driverName)
        assertEquals("AVAILABLE", v.status)
        assertEquals(5000.0, v.capacityKg, 0.001)
        assertEquals(12000.0, v.capacityL, 0.001)
        assertEquals("r-1", v.currentRoute)
    }

    // ── Staff ──

    @Test
    fun `StaffMember defaults`() {
        val raw = """{"id":"s-1"}"""
        val s = json.decodeFromString(StaffMember.serializer(), raw)
        assertEquals("s-1", s.id)
        assertEquals("", s.name)
        assertEquals("", s.phone)
        assertEquals("", s.role)
        assertEquals("", s.status)
        assertEquals("", s.joinedAt)
    }

    // ── Insight ──

    @Test
    fun `Insight urgency field preserved`() {
        val raw = """
            {"id":"i-1","warehouse_id":"wh-1","warehouse_name":"Central",
             "product_id":"p-1","product_name":"Rice","urgency":"CRITICAL",
             "current_stock":10,"avg_daily_velocity":25.0,
             "days_until_stockout":0,"reorder_quantity":500,"status":"OPEN"}
        """.trimIndent()
        val i = json.decodeFromString(Insight.serializer(), raw)
        assertEquals("CRITICAL", i.urgency)
        assertEquals(10, i.currentStock)
        assertEquals(25.0, i.avgDailyVelocity, 0.001)
        assertEquals(0, i.daysUntilStockout)
        assertEquals(500, i.reorderQuantity)
    }

    // ── Dispatch ──

    @Test
    fun `DispatchRequest serializes transfer_ids`() {
        val req = DispatchRequest(transferIds = listOf("t-1", "t-2", "t-3"))
        val encoded = json.encodeToString(DispatchRequest.serializer(), req)
        assertTrue(encoded.contains("\"transfer_ids\""))
        assertTrue(encoded.contains("t-1"))
        assertTrue(encoded.contains("t-3"))
    }

    @Test
    fun `DispatchResponse deserializes correctly`() {
        val raw = """{"manifest_id":"m-1","truck_plate":"01A999AB","stop_count":5}"""
        val res = json.decodeFromString(DispatchResponse.serializer(), raw)
        assertEquals("m-1", res.manifestId)
        assertEquals("01A999AB", res.truckPlate)
        assertEquals(5, res.stopCount)
    }
}
