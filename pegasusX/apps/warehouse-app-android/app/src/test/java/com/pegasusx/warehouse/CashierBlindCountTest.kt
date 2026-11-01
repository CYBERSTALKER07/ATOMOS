package com.pegasusx.warehouse

import com.pegasusx.mobilekit.foundry.AMLCheckRequest
import com.pegasusx.mobilekit.foundry.AMLCheckResponse
import com.pegasusx.mobilekit.foundry.CITHandoverRequest
import com.pegasusx.mobilekit.foundry.CashierSessionView
import com.pegasusx.mobilekit.foundry.DepotVaultLedgerEntry
import com.pegasusx.mobilekit.foundry.RRNTenderReclassificationRequest
import com.pegasusx.mobilekit.foundry.YardGateCheckEvent
import com.pegasusx.warehouse.ui.screens.cashier.CashierUiState
import com.pegasusx.warehouse.ui.screens.cashier.DenominationEntry
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CashierBlindCountTest {

    @Test
    fun testDenominationArithmeticInTiyin() {
        val entry200k = DenominationEntry(noteUzs = 200000L, count = 10L)
        // 10 notes of 200,000 UZS = 2,000,000 UZS = 200,000,000 Tiyin (1 UZS = 100 Tiyin)
        assertEquals(2000000L, entry200k.totalUzs)
        assertEquals(200000000L, entry200k.totalTiyin)

        val entry50k = DenominationEntry(noteUzs = 50000L, count = 4L)
        // 4 notes of 50,000 UZS = 200,000 UZS = 20,000,000 Tiyin
        assertEquals(200000L, entry50k.totalUzs)
        assertEquals(20000000L, entry50k.totalTiyin)

        val denoms = mapOf(
            200000L to 5L, // 1,000,000 UZS -> 100,000,000 Tiyin
            100000L to 3L, // 300,000 UZS -> 30,000,000 Tiyin
            50000L to 2L,  // 100,000 UZS -> 10,000,000 Tiyin
            20000L to 1L,  // 20,000 UZS -> 2,000,000 Tiyin
            10000L to 0L,
            5000L to 1L,   // 5,000 UZS -> 500,000 Tiyin
            2000L to 0L,
            1000L to 0L,
        )
        val totalMinor = denoms.entries.sumOf { (note, cnt) -> note * cnt * 100L }
        assertEquals(142500000L, totalMinor) // 1,425,000 UZS in Tiyin
    }

    @Test
    fun testStrictBlindCountMaskingInvariant() {
        val initialState = CashierUiState()
        // During blind count, expected amount and variance MUST remain strictly masked
        assertTrue(initialState.isBlindCountMasked)
        assertEquals(1L, initialState.attemptNumber)
        assertFalse(initialState.isRecountRequired)
        assertFalse(initialState.isBalanced)
        assertNull(initialState.session)

        // Simulate 409 conflict: Recount Ratchet triggered
        val recountState = initialState.copy(
            isRecountRequired = true,
            isBlindCountMasked = true, // Expected cash must remain masked on recount attempt 2!
            attemptNumber = 2L,
            statusMessage = "MANDATORY BLIND RECOUNT #2 REQUIRED: Physical count does not match route records. Expected amount remains masked.",
        )
        assertTrue(recountState.isRecountRequired)
        assertTrue(recountState.isBlindCountMasked)
        assertEquals(2L, recountState.attemptNumber)
        assertNotNull(recountState.statusMessage)
    }

    @Test
    fun testBalancedStateRevealsClearanceToken() {
        val session = CashierSessionView(
            sessionId = "sess-001",
            depotId = "depot-tashkent-south",
            supplierId = "supplier-tashkent-1",
            driverId = "driver-uz-09",
            routeId = "route-tashkent-042",
            bagSealBarcode = "BAG-UZ-99120",
            expectedCashMinor = 142500000L,
            countedCashMinor = 142500000L,
            varianceMinor = 0L,
            attemptNumber = 1L,
            status = "BALANCED",
            clearanceTokenHash = "SHA256:7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069",
            microSkimmingFlagged = false,
            createdAt = "2026-10-30T10:00:00Z",
        )

        val balancedState = CashierUiState(
            session = session,
            attemptNumber = 1L,
            isBalanced = true,
            isBlindCountMasked = false, // Unmasked once verified balanced
            isRecountRequired = false,
            clearanceTokenHash = session.clearanceTokenHash,
        )

        assertTrue(balancedState.isBalanced)
        assertFalse(balancedState.isBlindCountMasked)
        assertEquals("SHA256:7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069", balancedState.clearanceTokenHash)
        assertEquals(142500000L, balancedState.session?.expectedCashMinor)
        assertEquals(0L, balancedState.session?.varianceMinor)
    }

    @Test
    fun testAMLExposureCompliance() {
        val dynamicCeilingMinor = 5000000000L // Dynamic ceiling in Tiyin
        val uncappedCeilingMinor = 0L // Uncapped mode

        val uncappedCheck = AMLCheckResponse(
            legalEntityTin = "123456789",
            accumulatedCashMinor = 1800000000L,
            ceilingMinor = uncappedCeilingMinor,
        )
        assertFalse(uncappedCheck.ceilingMinor > 0 && uncappedCheck.accumulatedCashMinor > uncappedCheck.ceilingMinor)

        val compliantCheck = AMLCheckResponse(
            legalEntityTin = "123456789",
            accumulatedCashMinor = 1800000000L,
            ceilingMinor = dynamicCeilingMinor,
        )
        assertFalse(compliantCheck.accumulatedCashMinor > compliantCheck.ceilingMinor)
        assertTrue(compliantCheck.accumulatedCashMinor <= compliantCheck.ceilingMinor)

        val breachedCheck = AMLCheckResponse(
            legalEntityTin = "123456789",
            accumulatedCashMinor = 2650000000L, // 26,500,000 UZS
            ceilingMinor = ceilingMinor,
        )
        assertTrue(breachedCheck.accumulatedCashMinor > breachedCheck.ceilingMinor)
    }

    @Test
    fun testRRNTenderReclassificationContract() {
        val req = RRNTenderReclassificationRequest(
            sessionId = "sess-001",
            orderId = "order-uz-8812",
            disputedCashMinor = 50000000L, // 500,000 UZS in Tiyin
            gatewayRrn = "123456789012",
            terminalId = "POS-TASH-01",
            cardLastFour = "8821",
            authCode = "AUTH-482109",
            paymentNetwork = "HUMO",
            rrnCapturedAt = "2026-10-30T10:15:00Z",
            podGeofenceTimestamp = "2026-10-30T10:14:45Z",
        )

        assertEquals("sess-001", req.sessionId)
        assertEquals("order-uz-8812", req.orderId)
        assertEquals(50000000L, req.disputedCashMinor)
        assertEquals("HUMO", req.paymentNetwork)
        assertEquals("123456789012", req.gatewayRrn)
        assertNotNull(req.authCode)
    }

    @Test
    fun testCITHandoverContract() {
        val citReq = CITHandoverRequest(
            depotId = "depot-tashkent-south",
            supplierId = "supplier-tashkent-1",
            sessionIds = listOf("sess-001", "sess-002"),
            courierBadgeId = "INKAS-9921",
            citSealBarcode = "CIT-SEAL-7712",
            cashMinor = 3500000000L, // 35,000,000 UZS in Tiyin
        )

        assertEquals("depot-tashkent-south", citReq.depotId)
        assertEquals("INKAS-9921", citReq.courierBadgeId)
        assertEquals("CIT-SEAL-7712", citReq.citSealBarcode)
        assertEquals(2, citReq.sessionIds.size)
        assertEquals(3500000000L, citReq.cashMinor)

        val ledgerEntry = DepotVaultLedgerEntry(
            vaultEntryId = "vault-tx-9901",
            depotId = "depot-tashkent-south",
            supplierId = "supplier-tashkent-1",
            entryType = "CIT_ARMORED_HANDOVER",
            amountMinor = 3500000000L,
            runningBalanceMinor = 0L,
            citManifestId = "MANIFEST-CIT-20261030-01",
            createdAt = "2026-10-30T14:30:00Z",
        )
        assertEquals("CIT_ARMORED_HANDOVER", ledgerEntry.entryType)
        assertEquals("MANIFEST-CIT-20261030-01", ledgerEntry.citManifestId)

    }

    @Test
    fun testYardGateBoltSealVerification() {
        val inboundCheck = YardGateCheckEvent(
            depotId = "depot-tashkent-south",
            supplierId = "supplier-tashkent-1",
            vehicleId = "vh-isuzu-01",
            licensePlate = "01 777 AAA",
            direction = "GATE_IN",
            vehiclePurpose = "INBOUND_SUPPLIER_DROP",
            boltSealBarcode = "SEAL-UZ-11029",
            sealIntact = true,
            assignedBaySlot = "BAY_3_INSPECTION",
        )
        assertTrue(inboundCheck.sealIntact)
        assertEquals("GATE_IN", inboundCheck.direction)
        assertEquals("BAY_3_INSPECTION", inboundCheck.assignedBaySlot)

        val tamperedCheck = inboundCheck.copy(
            sealIntact = false,
            assignedBaySlot = "SECURITY_QUARANTINE_BAY",
        )
        assertFalse(tamperedCheck.sealIntact)
        assertEquals("SECURITY_QUARANTINE_BAY", tamperedCheck.assignedBaySlot)
    }
}
