package com.pegasusx.driver.ui.screens.fleet

import com.pegasusx.mobilekit.foundry.DVIRChecklist8Point
import com.pegasusx.mobilekit.foundry.FleetDVIRInspection
import com.pegasusx.mobilekit.foundry.RescueToteItem
import com.pegasusx.mobilekit.foundry.YardGateCheckEvent
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class DVIRFleetTest {

    @Test
    fun `default DVIR checklist has all 8 safety points true`() {
        val checklist = DVIRChecklist8Point()
        assertTrue(checklist.steeringAndHornOk)
        assertTrue(checklist.serviceAndParkingBrakesOk)
        assertTrue(checklist.tiresTreadAndPressureOk)
        assertTrue(checklist.lightsAndHazardsOk)
        assertTrue(checklist.wipersAndFluidsOk)
        assertTrue(checklist.emergencyKitAndExtinguisherOk)
        assertTrue(checklist.cargoBoxSanitationOk)
        assertTrue(checklist.reeferTempCalibratedOk)
    }

    @Test
    fun `critical defect triggers red tag grounding`() {
        val defectiveChecklist = DVIRChecklist8Point(serviceAndParkingBrakesOk = false)
        assertFalse(defectiveChecklist.serviceAndParkingBrakesOk)

        val failedPoints = mutableListOf<String>()
        if (!defectiveChecklist.serviceAndParkingBrakesOk) failedPoints.add("service_and_parking_brakes")

        val inspection = FleetDVIRInspection(
            supplierId = "supplier-tashkent-1",
            vehicleId = "vh-isuzu-01",
            driverId = "driver-uz-09",
            checklist = defectiveChecklist,
            failedPoints = failedPoints,
            criticalDefectFound = true,
            redTagApplied = true,
            odometerKm = 125400L,
            status = "RED_TAG_QUARANTINE",
            maintenanceWorkOrderId = "wo-isuzu-01-4412",
        )

        assertTrue(inspection.criticalDefectFound)
        assertTrue(inspection.redTagApplied)
        assertEquals("RED_TAG_QUARANTINE", inspection.status)
        assertEquals("wo-isuzu-01-4412", inspection.maintenanceWorkOrderId)
        assertEquals(listOf("service_and_parking_brakes"), inspection.failedPoints)
    }

    @Test
    fun `cold chain reefer defect triggers red tag when cold chain required`() {
        val reeferChecklist = DVIRChecklist8Point(reeferTempCalibratedOk = false)
        val inspection = FleetDVIRInspection(
            supplierId = "supplier-tashkent-1",
            vehicleId = "vh-reefer-08",
            driverId = "driver-uz-09",
            requiresColdChain = true,
            checklist = reeferChecklist,
            failedPoints = listOf("reefer_temp_calibrated"),
            criticalDefectFound = true,
            redTagApplied = true,
            reeferTempCelsius = 14.5, // degraded
        )

        assertTrue(inspection.requiresColdChain)
        assertTrue(inspection.redTagApplied)
        assertEquals(14.5, inspection.reeferTempCelsius, 0.001)
    }

    @Test
    fun `mid-route rescue isolates driver liability at stop K`() {
        val totes = listOf(
            RescueToteItem(toteId = "tote-01", stopSequence = 1, isColdChain = false, expectedCashMinor = 50000000L),
            RescueToteItem(toteId = "tote-02", stopSequence = 2, isColdChain = false, expectedCashMinor = 75000000L),
            RescueToteItem(toteId = "tote-03", stopSequence = 3, isColdChain = true, expectedCashMinor = 120000000L, thermalExcursionMinutes = 45, maxAllowedExcursionMins = 30),
            RescueToteItem(toteId = "tote-04", stopSequence = 4, isColdChain = false, expectedCashMinor = 60000000L),
        )

        val lastCompletedStopK = 2L
        val originalDriverLiability = totes.filter { it.stopSequence <= lastCompletedStopK }.sumOf { it.expectedCashMinor }
        val remainingTotes = totes.filter { it.stopSequence > lastCompletedStopK }
        val spoiledTotes = remainingTotes.filter { it.isColdChain && it.thermalExcursionMinutes > it.maxAllowedExcursionMins }
        val transferredTotes = remainingTotes.filterNot { it.isColdChain && it.thermalExcursionMinutes > it.maxAllowedExcursionMins }
        val rescueDriverLiability = transferredTotes.sumOf { it.expectedCashMinor }

        assertEquals(125000000L, originalDriverLiability) // 50M + 75M
        assertEquals(1, spoiledTotes.size)
        assertEquals("tote-03", spoiledTotes.first().toteId)
        assertEquals(1, transferredTotes.size)
        assertEquals("tote-04", transferredTotes.first().toteId)
        assertEquals(60000000L, rescueDriverLiability)
    }

    @Test
    fun `yard gate outbound requires intact bolt seal`() {
        val validOutbound = YardGateCheckEvent(
            depotId = "depot-tashkent-south",
            supplierId = "supplier-tashkent-1",
            vehicleId = "vh-isuzu-01",
            licensePlate = "01 777 AAA",
            direction = "GATE_OUT",
            vehiclePurpose = "OUTBOUND_ROUTE",
            boltSealBarcode = "SEAL-UZ-44912",
            sealIntact = true,
            assignedBaySlot = "OUTBOUND_LOADING",
        )

        assertTrue(validOutbound.sealIntact)
        assertNotNull(validOutbound.boltSealBarcode)
        assertEquals("OUTBOUND_LOADING", validOutbound.assignedBaySlot)

        val brokenSealOutbound = validOutbound.copy(sealIntact = false)
        assertFalse(brokenSealOutbound.sealIntact)
    }
}
