package com.pegasusx.warehouse

import com.pegasusx.warehouse.data.foundry.AMLCheckResponse
import com.pegasusx.warehouse.data.foundry.CITHandoverRequest
import com.pegasusx.warehouse.data.foundry.CashierSessionView
import com.pegasusx.warehouse.data.foundry.DepotVaultLedgerEntry
import com.pegasusx.warehouse.data.foundry.RRNTenderReclassificationRequest
import com.pegasusx.warehouse.data.foundry.StartCashierSessionRequest
import com.pegasusx.warehouse.data.foundry.YardGateCheckEvent
import com.pegasusx.warehouse.data.model.Driver
import com.pegasusx.warehouse.data.model.DriverListResponse
import com.pegasusx.warehouse.data.model.Manifest
import com.pegasusx.warehouse.data.model.ManifestListResponse
import com.pegasusx.warehouse.data.model.WarehouseFleetLiveMapResponse
import com.pegasusx.warehouse.data.model.WarehouseFleetLiveRoute
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.screens.cashier.CashierUiState
import com.pegasusx.warehouse.ui.screens.cashier.CashierViewModel
import com.pegasusx.warehouse.ui.screens.cashier.DenominationEntry
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Response
import java.io.File
import java.lang.reflect.Proxy

@OptIn(ExperimentalCoroutinesApi::class)
class CashierBlindCountTest {

    private val testDispatcher = UnconfinedTestDispatcher()

    @Before
    fun setUp() {
        Dispatchers.setMain(testDispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private fun fakeApi(
        handler: (name: String, args: List<Any?>) -> Response<*>,
    ): Pair<WarehouseApi, MutableList<Pair<String, List<Any?>>>> {
        val calls = mutableListOf<Pair<String, List<Any?>>>()
        val api = Proxy.newProxyInstance(
            WarehouseApi::class.java.classLoader,
            arrayOf(WarehouseApi::class.java),
        ) { _, method, rawArgs ->
            val argsList = rawArgs?.dropLast(1).orEmpty()
            calls += method.name to argsList
            handler(method.name, argsList)
        } as WarehouseApi
        return api to calls
    }

    @Test
    fun testDenominationArithmeticToMinorUnits() {
        val entry200k = DenominationEntry(noteSom = 200000L, count = 10L)
        assertEquals(2000000L, entry200k.totalSom)
        assertEquals(200000000L, entry200k.totalMinor)

        val entry50k = DenominationEntry(noteSom = 50000L, count = 4L)
        assertEquals(200000L, entry50k.totalSom)
        assertEquals(20000000L, entry50k.totalMinor)

        val denoms = mapOf(
            200000L to 5L,
            100000L to 3L,
            50000L to 2L,
            20000L to 1L,
            10000L to 0L,
            5000L to 1L,
            2000L to 0L,
            1000L to 0L,
        )
        val totalMinor = denoms.entries.sumOf { (note, cnt) -> note * cnt * 100L }
        assertEquals(142500000L, totalMinor)
    }

    @Test
    fun testWholeSomFormattingWithoutMinorInflationOrNegativeZero() {
        assertEquals("1 425 000 so'm", CashierViewModel.formatWholeSom(142500000L))
        assertEquals("-25 000 so'm", CashierViewModel.formatWholeSom(-2500000L))
        assertEquals("0 so'm", CashierViewModel.formatWholeSom(-50L))
        assertEquals("200 000 so'm", CashierViewModel.formatWholeSomMajor(200000L))
        assertFalse(CashierViewModel.formatWholeSom(142500000L).contains("Tiyin", ignoreCase = true))
        assertFalse(CashierViewModel.formatWholeSom(142500000L).contains("UZS", ignoreCase = true))
    }

    @Test
    fun testCleanEmptyInitialDefaultsAndBlindMasking() {
        val initialState = CashierUiState()
        assertEquals("", initialState.depotId)
        assertEquals("", initialState.supplierId)
        assertEquals("", initialState.driverId)
        assertEquals("", initialState.routeId)
        assertEquals("", initialState.bagSealBarcode)
        assertEquals("", initialState.cellId)
        assertTrue(initialState.isBlindCountMasked)
        assertEquals(1L, initialState.attemptNumber)
        assertFalse(initialState.isRecountRequired)
        assertFalse(initialState.isBalanced)
        assertEquals(0L, initialState.countedCashMinor)
    }

    @Test
    fun testLoadSelectorsAndManualFallback() = runTest {
        val (liveApi, _) = fakeApi { method, _ ->
            when (method) {
                "getDrivers" -> Response.success(
                    DriverListResponse(listOf(Driver(driverId = "drv-live-1", name = "Jamshid R."))),
                )
                "getManifests" -> Response.success(
                    ManifestListResponse(listOf(Manifest(manifestId = "mf-live-2", driverName = "Sardor M.", vehicleLabel = "Van 7"))),
                )
                "getFleetLiveMap" -> Response.success(
                    WarehouseFleetLiveMapResponse(
                        routes = listOf(
                            WarehouseFleetLiveRoute(
                                manifestId = "mf-live-1",
                                routeId = "rt-live-1",
                                driverId = "drv-live-1",
                                driverName = "Jamshid R.",
                                manifestState = "OUT_FOR_DELIVERY",
                            ),
                        ),
                        warehouseId = "wh-live-1",
                        fetchedAt = "2026-11-12T10:00:00Z",
                    ),
                )
                else -> error("Unexpected API call: $method")
            }
        }

        val vm = CashierViewModel(liveApi)
        vm.loadSelectors()

        val state = vm.uiState.value
        assertFalse(state.selectorsLoading)
        assertFalse(state.manualEntryMode)
        assertEquals(1, state.driverOptions.size)
        assertEquals("Jamshid R. (drv-live-1)", state.driverOptions.first().label)
        assertEquals(listOf("rt-live-1", "mf-live-2"), state.routeOptions.map { it.routeId })

        vm.selectRoute("rt-live-1")
        assertEquals("rt-live-1", vm.uiState.value.routeId)
        assertEquals("drv-live-1", vm.uiState.value.driverId)

        val (emptyApi, _) = fakeApi { method, _ ->
            when (method) {
                "getDrivers" -> Response.success(DriverListResponse(emptyList()))
                "getManifests" -> Response.success(ManifestListResponse(emptyList()))
                "getFleetLiveMap" -> Response.success(WarehouseFleetLiveMapResponse())
                else -> error("Unexpected API call: $method")
            }
        }
        val emptyVm = CashierViewModel(emptyApi)
        emptyVm.loadSelectors()
        assertTrue(emptyVm.uiState.value.manualEntryMode)
    }

    @Test
    fun testStartSessionAndRecountMaskingWithMinorWirePayloads() = runTest {
        var countAttempt = 0
        val (api, calls) = fakeApi { method, _ ->
            when (method) {
                "startCashierSession" -> Response.success(
                    CashierSessionView(
                        sessionId = "sess-live-01",
                        depotId = "wh-live-01",
                        supplierId = "sup-live-01",
                        driverId = "drv-live-1",
                        routeId = "rt-live-1",
                        bagSealBarcode = "SEAL-LIVE-99",
                        expectedCashMinor = 142500000L,
                        countedCashMinor = 0L,
                        varianceMinor = 0L,
                        attemptNumber = 1L,
                        status = "AWAITING_BLIND_COUNT_1",
                        createdAt = "2026-11-12T10:00:00Z",
                    ),
                )
                "submitBlindCount" -> {
                    countAttempt++
                    if (countAttempt == 1) {
                        Response.error<CashierSessionView>(
                            409,
                            """{"error":"recount_required"}""".toResponseBody("application/json".toMediaType()),
                        )
                    } else {
                        Response.success(
                            CashierSessionView(
                                sessionId = "sess-live-01",
                                depotId = "wh-live-01",
                                supplierId = "sup-live-01",
                                driverId = "drv-live-1",
                                routeId = "rt-live-1",
                                bagSealBarcode = "SEAL-LIVE-99",
                                expectedCashMinor = 142500000L,
                                countedCashMinor = 142500000L,
                                varianceMinor = 0L,
                                attemptNumber = 2L,
                                status = "BALANCED",
                                clearanceTokenHash = "tok-hash-99",
                                createdAt = "2026-11-12T10:05:00Z",
                            ),
                        )
                    }
                }
                else -> error("Unexpected API call: $method")
            }
        }

        val vm = CashierViewModel(api)
        vm.startSession()
        assertEquals("Bag seal barcode is required", vm.uiState.value.errorMessage)

        vm.applyScope(cellId = "cell-live-1", depotId = "wh-live-01", supplierId = "sup-live-01")
        vm.updateDriverId("drv-live-1")
        vm.updateRouteId("rt-live-1")
        vm.updateBagSealBarcode("SEAL-LIVE-99")
        vm.startSession()

        val startReq = calls.first { it.first == "startCashierSession" }.second.first() as StartCashierSessionRequest
        assertEquals("cell-live-1", startReq.cellId)
        assertEquals(0L, startReq.expectedCashMinor)
        assertTrue(vm.uiState.value.isBlindCountMasked)

        vm.setDenominationCount(200000L, 5L)
        assertEquals(100000000L, vm.uiState.value.countedCashMinor)
        vm.submitBlindCount()

        assertTrue(vm.uiState.value.isRecountRequired)
        assertTrue(vm.uiState.value.isBlindCountMasked)
        assertEquals(2L, vm.uiState.value.attemptNumber)

        vm.setDenominationCount(200000L, 7L)
        vm.setDenominationCount(20000L, 1L)
        vm.setDenominationCount(5000L, 1L)
        assertEquals(142500000L, vm.uiState.value.countedCashMinor)
        vm.submitBlindCount()

        assertTrue(vm.uiState.value.isBalanced)
        assertFalse(vm.uiState.value.isBlindCountMasked)
        assertEquals("tok-hash-99", vm.uiState.value.clearanceTokenHash)
    }

    @Test
    fun testBalancedStateRevealsClearanceToken() {
        val session = CashierSessionView(
            sessionId = "sess-001",
            depotId = "depot-live-1",
            supplierId = "supplier-live-1",
            driverId = "driver-live-1",
            routeId = "route-live-1",
            bagSealBarcode = "SEAL-001",
            expectedCashMinor = 142500000L,
            countedCashMinor = 142500000L,
            varianceMinor = 0L,
            attemptNumber = 1L,
            status = "BALANCED",
            clearanceTokenHash = "sha256:9981abc...",
            createdAt = "2026-10-30T10:00:00Z",
        )

        val balancedState = CashierUiState(
            session = session,
            attemptNumber = 1L,
            isBalanced = true,
            isBlindCountMasked = false,
            clearanceTokenHash = session.clearanceTokenHash,
        )

        assertTrue(balancedState.isBalanced)
        assertFalse(balancedState.isBlindCountMasked)
        assertNotNull(balancedState.clearanceTokenHash)
        assertEquals(0L, balancedState.session?.varianceMinor)
    }

    @Test
    fun testAMLExposureAndContractsPreserveMinorUnits() {
        val dynamicCeilingMinor = 2500000000L
        val breachedCheck = AMLCheckResponse(
            legalEntityTin = "123456789",
            accumulatedCashMinor = 2650000000L,
            ceilingMinor = dynamicCeilingMinor,
        )
        assertTrue(breachedCheck.accumulatedCashMinor > breachedCheck.ceilingMinor)

        val req = RRNTenderReclassificationRequest(
            sessionId = "sess-001",
            orderId = "ord-001",
            disputedCashMinor = 50000000L,
            gatewayRrn = "999888777666",
            terminalId = "term-01",
            cardLastFour = "4411",
            authCode = "auth-1",
            paymentNetwork = "HUMO",
            rrnCapturedAt = "2026-10-30T10:15:00Z",
            podGeofenceTimestamp = "2026-10-30T10:10:00Z",
        )
        assertEquals(50000000L, req.disputedCashMinor)

        val citReq = CITHandoverRequest(
            depotId = "depot-live-1",
            supplierId = "supplier-live-1",
            sessionIds = listOf("sess-001", "sess-002"),
            courierBadgeId = "badge-01",
            citSealBarcode = "seal-01",
            cashMinor = 3500000000L,
        )
        assertEquals(3500000000L, citReq.cashMinor)

        val ledgerEntry = DepotVaultLedgerEntry(
            vaultEntryId = "vault-tx-9901",
            depotId = "depot-live-1",
            supplierId = "supplier-live-1",
            entryType = "CIT_ARMORED_HANDOVER",
            amountMinor = 3500000000L,
            runningBalanceMinor = 0L,
            citManifestId = "MANIFEST-CIT-01",
            createdAt = "2026-10-30T14:30:00Z",
        )
        assertEquals(0L, ledgerEntry.runningBalanceMinor)

        val inboundCheck = YardGateCheckEvent(
            depotId = "depot-live-1",
            supplierId = "supplier-live-1",
            vehicleId = "vh-01",
            licensePlate = "01 777 AAA",
            direction = "GATE_IN",
            vehiclePurpose = "SUPPLIER_INBOUND_RESTOCK",
            sealIntact = true,
        )
        assertTrue(inboundCheck.sealIntact)
    }

    @Test
    fun testCashierSourceContainsZeroTiyinOrFakeDemoDefaults() {
        val appDir = File(".").canonicalFile.let { if (it.name == "app") it else File(it, "app") }
        val screenFile = File(appDir, "src/main/java/com/pegasusx/warehouse/ui/screens/cashier/CashierBlindCountScreen.kt")
        val vmFile = File(appDir, "src/main/java/com/pegasusx/warehouse/ui/screens/cashier/CashierViewModel.kt")
        val screenText = screenFile.readText()
        val vmText = vmFile.readText()

        assertFalse(screenText.contains("Tiyin", ignoreCase = true))
        assertFalse(vmText.contains("Tiyin", ignoreCase = true))
        assertFalse(screenText.contains("UZS"))
        assertFalse(vmText.contains("UZS"))

        val forbidden = listOf(
            "driver-uz-09",
            "route-tashkent-042",
            "BAG-UZ-99120",
            "depot-tashkent-south",
            "supplier-tashkent-1",
            "cell-uz",
            "1425000000",
            "order-tashkent-701",
            "RRN-HUMO-991288",
            "INKAS-TASH-44",
            "CIT-SEAL-8812",
            "304899122",
            "sess-batch-01",
            "POS-TASH-01",
        )
        for (token in forbidden) {
            assertFalse("Forbidden token $token in CashierBlindCountScreen.kt", screenText.contains(token))
            assertFalse("Forbidden token $token in CashierViewModel.kt", vmText.contains(token))
        }
    }
}
