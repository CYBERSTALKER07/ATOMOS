package com.pegasusx.warehouse

import com.pegasusx.warehouse.data.model.CycleCount
import com.pegasusx.warehouse.data.model.CycleCountCreateRequest
import com.pegasusx.warehouse.data.model.CycleCountListResponse
import com.pegasusx.warehouse.data.model.CycleCountSubmitRequest
import com.pegasusx.warehouse.data.model.InventoryAdjustment
import com.pegasusx.warehouse.data.model.InventoryAdjustmentListResponse
import com.pegasusx.warehouse.data.model.PickTask
import com.pegasusx.warehouse.data.model.PickTaskConfirmRequest
import com.pegasusx.warehouse.data.model.PickWave
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.ui.screens.transfers.TransferActionsOperationsController
import com.pegasusx.warehouse.ui.screens.transfers.TransferWmsHelpers
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import java.lang.reflect.Proxy
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import retrofit2.Response

class WarehousePickWaveCycleCountTest {

    private val uuidRegex = Regex(".*:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

    private data class RecordedCall(
        val method: String,
        val args: List<Any?>,
    )

    private fun fakeApi(
        handler: (name: String, args: List<Any?>) -> Response<*>,
    ): Pair<WarehouseApi, MutableList<RecordedCall>> {
        val calls = mutableListOf<RecordedCall>()
        val api = Proxy.newProxyInstance(
            WarehouseApi::class.java.classLoader,
            arrayOf(WarehouseApi::class.java),
        ) { _, method, rawArgs ->
            val argsList = rawArgs?.dropLast(1).orEmpty() // drop Kotlin Continuation
            calls += RecordedCall(method.name, argsList)
            handler(method.name, argsList)
        } as WarehouseApi
        return api to calls
    }

    @Test
    fun idempotencyKeys_useUuidInsteadOfTimestamps() {
        val putaway1 = WarehouseIdempotencyKeys.putawayLot("prod-1", "loc-1")
        val putaway2 = WarehouseIdempotencyKeys.putawayLot("prod-1", "loc-1")
        assertNotEquals(putaway1, putaway2)
        assertTrue(uuidRegex.matches(putaway1))

        assertTrue(uuidRegex.matches(WarehouseIdempotencyKeys.createPickWave("man-1")))
        assertTrue(uuidRegex.matches(WarehouseIdempotencyKeys.confirmPickTask("wave-1", "task-1")))
        assertTrue(uuidRegex.matches(WarehouseIdempotencyKeys.waivePickShorts("wave-1")))
        assertTrue(uuidRegex.matches(WarehouseIdempotencyKeys.createCycleCount("loc-1", "prod-1")))
        assertTrue(uuidRegex.matches(WarehouseIdempotencyKeys.submitCycleCount("cc-1")))
        assertTrue(uuidRegex.matches(WarehouseIdempotencyKeys.approveInventoryAdjustment("adj-1")))
        assertTrue(uuidRegex.matches(WarehouseIdempotencyKeys.rejectInventoryAdjustment("adj-1")))
        assertTrue(uuidRegex.matches(WarehouseIdempotencyKeys.createBin("Z1")))
    }

    @Test
    fun pickTaskConfirmRequest_serializesExplicitZeroDistinctFromNull() {
        val json = Json { encodeDefaults = true; ignoreUnknownKeys = true }
        assertEquals("{\"quantity_picked\":0}", json.encodeToString(PickTaskConfirmRequest(quantityPicked = 0L)))
        assertEquals("{\"quantity_picked\":null}", json.encodeToString(PickTaskConfirmRequest(quantityPicked = null)))
    }

    @Test
    fun confirmPickWithZero_sendsExplicitZeroAndAllowsWaivingShorts() = runTest {
        val task = PickTask(
            taskId = "task-1",
            orderId = "ord-1",
            productId = "SKU-1",
            lotId = "lot-1",
            locationId = "A-01",
            quantityRequested = 5L,
            quantityPicked = 0L,
            status = "PENDING",
            pickSequence = 1L,
        )
        var waveState = PickWave(
            waveId = "wave-1",
            warehouseId = "wh-1",
            manifestId = "man-1",
            status = "OPEN",
            tasks = listOf(task),
        )

        val (api, calls) = fakeApi { name, args ->
            when (name) {
                "createPickWave" -> Response.success(waveState)
                "confirmPickTask" -> {
                    val body = args[2] as PickTaskConfirmRequest
                    val picked = body.quantityPicked ?: task.quantityRequested
                    waveState = waveState.copy(
                        status = if (picked < task.quantityRequested) "IN_PROGRESS" else "READY_TO_SEAL",
                        tasks = listOf(
                            task.copy(
                                quantityPicked = picked,
                                status = if (picked < task.quantityRequested) "SHORT" else "CONFIRMED",
                            ),
                        ),
                    )
                    Response.success(waveState)
                }
                "waivePickShorts" -> {
                    waveState = waveState.copy(
                        status = "READY_TO_SEAL",
                        tasks = waveState.tasks.map {
                            if (it.status == "SHORT") it.copy(status = "SHORT_WAIVED") else it
                        },
                    )
                    Response.success(waveState)
                }
                else -> error("Unexpected API call: $name")
            }
        }

        val controller = TransferActionsOperationsController(WarehouseOperationsRepository(api))
        val created = controller.createPickWave("man-1")
        assertTrue(created.isSuccess)
        assertEquals("5", controller.state.pickQtyText)

        // Blank input must fail validation cleanly and never coerce to 0 or requested.
        val blankAttempt = controller.confirmPickWithInput(task, "   ")
        assertTrue(blankAttempt.isFailure)
        assertFalse(calls.any { it.method == "confirmPickTask" })

        // Explicit "0" must send quantityPicked = 0L (SHORT), never requested (5L).
        val zeroResult = controller.confirmPickWithInput(task, "0")
        assertTrue(zeroResult.isSuccess)
        val confirmCall = calls.single { it.method == "confirmPickTask" }
        val sentBody = confirmCall.args[2] as PickTaskConfirmRequest
        assertEquals(0L, sentBody.quantityPicked)
        assertTrue(uuidRegex.matches(confirmCall.args[3] as String))
        assertEquals("SHORT", controller.state.activeWave?.tasks?.single()?.status)
        assertTrue(TransferWmsHelpers.hasShortTasks(controller.state.activeWave))

        // Waiving shorts sends POST waive-shorts with UUID key and transitions SHORT -> SHORT_WAIVED.
        val waiveResult = controller.waivePickShorts()
        assertTrue(waiveResult.isSuccess)
        val waiveCall = calls.single { it.method == "waivePickShorts" }
        assertEquals("wave-1", waiveCall.args[0])
        assertTrue(uuidRegex.matches(waiveCall.args[1] as String))
        assertEquals("READY_TO_SEAL", controller.state.activeWave?.status)
        assertEquals("SHORT_WAIVED", controller.state.activeWave?.tasks?.single()?.status)
        assertFalse(TransferWmsHelpers.hasShortTasks(controller.state.activeWave))
    }

    @Test
    fun cycleCount_submitsActualUserEnteredCountIncludingZeroAndHandlesAdjustments() = runTest {
        var counts = mutableListOf(
            CycleCount(
                countId = "cc-1",
                locationId = "BIN-1",
                productId = "SKU-MILK",
                expectedQty = 12L,
                countedQty = null,
                status = "OPEN",
            ),
        )
        val adjustments = mutableListOf<InventoryAdjustment>()

        val (api, calls) = fakeApi { name, args ->
            when (name) {
                "listCycleCounts" -> Response.success(
                    CycleCountListResponse(counts = counts.toList(), cycleCountsEnabled = true),
                )
                "listInventoryAdjustments" -> Response.success(
                    InventoryAdjustmentListResponse(adjustments = adjustments.toList(), cycleCountsEnabled = true),
                )
                "submitCycleCount" -> {
                    val countId = args[0] as String
                    val body = args[1] as CycleCountSubmitRequest
                    val submitted = CycleCount(
                        countId = countId,
                        locationId = "BIN-1",
                        productId = "SKU-MILK",
                        expectedQty = 12L,
                        countedQty = body.countedQty,
                        varianceQty = body.countedQty - 12L,
                        reasonCode = body.reasonCode,
                        status = "ADJUSTMENT_PENDING",
                    )
                    counts = mutableListOf(submitted)
                    adjustments += InventoryAdjustment(
                        adjustmentId = "adj-1",
                        warehouseId = "wh-1",
                        productId = "SKU-MILK",
                        countId = countId,
                        deltaQty = body.countedQty - 12L,
                        reasonCode = body.reasonCode,
                        status = "PENDING",
                    )
                    adjustments += InventoryAdjustment(
                        adjustmentId = "adj-2",
                        warehouseId = "wh-1",
                        productId = "SKU-JUICE",
                        deltaQty = 3L,
                        status = "PENDING",
                    )
                    Response.success(submitted)
                }
                "approveInventoryAdjustment" -> {
                    val adjId = args[0] as String
                    val idx = adjustments.indexOfFirst { it.adjustmentId == adjId }
                    val updated = adjustments[idx].copy(status = "APPROVED")
                    adjustments[idx] = updated
                    Response.success(updated)
                }
                "rejectInventoryAdjustment" -> {
                    val adjId = args[0] as String
                    val idx = adjustments.indexOfFirst { it.adjustmentId == adjId }
                    val updated = adjustments[idx].copy(status = "REJECTED")
                    adjustments[idx] = updated
                    Response.success(updated)
                }
                "createCycleCount" -> {
                    val req = args[0] as CycleCountCreateRequest
                    val created = CycleCount(
                        countId = "cc-new",
                        locationId = req.locationId,
                        productId = req.productId,
                        expectedQty = req.expectedQty ?: 0L,
                        status = "OPEN",
                    )
                    counts.add(0, created)
                    Response.success(created)
                }
                else -> error("Unexpected API call: $name")
            }
        }

        val controller = TransferActionsOperationsController(WarehouseOperationsRepository(api))
        assertTrue(controller.refreshCycleCountsAndAdjustments().isSuccess)
        assertEquals("cc-1", controller.state.selectedCountId)
        assertEquals("12", controller.state.countedQtyText)

        // Blank counted quantity must fail and never fall back to expectedQty or 0.
        assertTrue(controller.submitCycleCount("cc-1", "", "LOST").isFailure)
        assertTrue(controller.submitCycleCount("cc-1", "-1", "LOST").isFailure)
        assertFalse(calls.any { it.method == "submitCycleCount" })

        // Explicit 0 counted qty submits countedQty = 0L (NEVER expectedQty = 12L).
        val submitZero = controller.submitCycleCount("cc-1", "0", "DAMAGED")
        assertTrue(submitZero.isSuccess)
        val submitCall = calls.single { it.method == "submitCycleCount" }
        val body = submitCall.args[1] as CycleCountSubmitRequest
        assertEquals(0L, body.countedQty)
        assertEquals("DAMAGED", body.reasonCode)
        assertTrue(uuidRegex.matches(submitCall.args[2] as String))

        val refreshedCount = controller.state.cycleCounts.single()
        assertEquals(0L, refreshedCount.countedQty)
        assertEquals(-12L, TransferWmsHelpers.computeVariance(refreshedCount))
        assertEquals(2, controller.state.adjustments.size)

        // Approve adj-1 and reject adj-2
        assertTrue(controller.approveAdjustment("adj-1").isSuccess)
        assertTrue(controller.rejectAdjustment("adj-2").isSuccess)

        val approveCall = calls.single { it.method == "approveInventoryAdjustment" }
        val rejectCall = calls.single { it.method == "rejectInventoryAdjustment" }
        assertTrue(uuidRegex.matches(approveCall.args[1] as String))
        assertTrue(uuidRegex.matches(rejectCall.args[1] as String))

        assertEquals("APPROVED", controller.state.adjustments.first { it.adjustmentId == "adj-1" }.status)
        assertEquals("REJECTED", controller.state.adjustments.first { it.adjustmentId == "adj-2" }.status)
        assertNull(TransferWmsHelpers.selectedOpenCount(controller.state))
    }

    @Test
    fun migratedColdLots_warnStagingAndZoneMismatchAndPreSealReadiness() = runTest {
        val binsResp = com.pegasusx.warehouse.data.model.WarehouseBinListResponse(
            lotsEnabled = true,
            bins = listOf(
                com.pegasusx.warehouse.data.model.WarehouseBinLocation(
                    warehouseId = "wh-1",
                    locationId = "recv-default",
                    zone = "RECV",
                    locationType = "STAGE",
                ),
                com.pegasusx.warehouse.data.model.WarehouseBinLocation(
                    warehouseId = "wh-1",
                    locationId = "A-01",
                    zone = "AMBIENT",
                    locationType = "PICK",
                ),
                com.pegasusx.warehouse.data.model.WarehouseBinLocation(
                    warehouseId = "wh-1",
                    locationId = "CHILL-01",
                    zone = "CHILL",
                    locationType = "PICK",
                ),
            ),
        )
        val migratedLot = com.pegasusx.warehouse.data.model.StockLot(
            lotId = "lot-cold-1",
            supplierId = "sup-1",
            warehouseId = "wh-1",
            productId = "sku-milk",
            locationId = "recv-default",
            lotCode = "UZ-01",
            expiryDate = "2026-11-20",
            quantityOnHand = 20L,
            status = "AVAILABLE",
            productName = "Kefir 1L",
            handlingClass = "CHILLED",
            requiresColdChain = true,
            requiredZone = "CHILL",
            coldLotWarning = "cold_lot_migrated_without_live_baseline",
        )
        val coldWave = PickWave(
            waveId = "wave-cold",
            warehouseId = "wh-1",
            manifestId = "man-cold-1",
            status = "READY_TO_SEAL",
            coldLotMigratedWithoutLiveBaseline = true,
            coldChainBaselineMissing = true,
            tasks = listOf(
                PickTask(
                    taskId = "t-1",
                    orderId = "o-1",
                    productId = "sku-milk",
                    lotId = "lot-cold-1",
                    locationId = "recv-default",
                    quantityRequested = 2L,
                    status = "CONFIRMED",
                ),
            ),
        )
        val (api, _) = fakeApi { name, _ ->
            when (name) {
                "listBins" -> Response.success(binsResp)
                "listLots" -> Response.success(
                    com.pegasusx.warehouse.data.model.StockLotListResponse(
                        lots = listOf(migratedLot),
                        lotsEnabled = true,
                        coldLotWarningCount = 1L,
                        migrationStagingCount = 1L,
                    ),
                )
                "listPickWaves" -> Response.success(
                    com.pegasusx.warehouse.data.model.PickWaveListResponse(
                        waves = listOf(coldWave),
                        pickWavesEnabled = true,
                    ),
                )
                "getPickWave" -> Response.success(coldWave)
                "getManifestTemperature" -> Response.success(
                    com.pegasusx.warehouse.data.model.ManifestTemperatureResponse(
                        manifestId = "man-cold-1",
                        readings = emptyList(),
                    ),
                )
                else -> error("Unexpected API call: $name")
            }
        }

        val controller = TransferActionsOperationsController(WarehouseOperationsRepository(api))
        assertTrue(controller.refreshBinsAndLots().isSuccess)
        assertEquals(1, TransferWmsHelpers.coldWarningLots(controller.state.lots, controller.state.bins).size)
        assertEquals(1, TransferWmsHelpers.stagingPutawayLots(controller.state.lots, controller.state.bins).size)
        assertEquals(
            "CHILL",
            TransferWmsHelpers.putawayZoneMismatch("sku-milk", "A-01", controller.state.lots, controller.state.bins),
        )
        assertEquals("CHILL-01", TransferWmsHelpers.recommendedBinForLot(migratedLot, controller.state.bins))
        assertNull(TransferWmsHelpers.putawayZoneMismatch("sku-milk", "CHILL-01", controller.state.lots, controller.state.bins))

        assertTrue(controller.loadPickWaves("man-cold-1").isSuccess)
        controller.refreshManifestTempReadiness("man-cold-1")
        assertTrue(controller.state.manifestTempMissing)
        assertTrue(TransferWmsHelpers.isWavePreSealColdWarning(controller.state))
    }
}
