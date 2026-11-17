package com.pegasusx.retailer.ui.screens

import androidx.lifecycle.ViewModelStore
import com.pegasusx.retailer.data.local.PendingOrderEntity
import com.pegasusx.retailer.data.model.CheckoutLineItem
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.data.model.ProblemDetail
import com.pegasusx.retailer.data.model.ProblemDetailException
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.ProcurementOrderRequest
import com.pegasusx.retailer.data.model.UnifiedCheckoutRequest
import com.pegasusx.retailer.data.model.Variant
import com.pegasusx.retailer.testsupport.FakePendingOrderDao
import com.pegasusx.retailer.testsupport.FakeRealtimeEventSource
import com.pegasusx.retailer.testsupport.FakeSyncEnqueuer
import com.pegasusx.retailer.testsupport.FakeUserIdProvider
import com.pegasusx.retailer.testsupport.RecordingPegasusApiConfig
import com.pegasusx.retailer.testsupport.httpJsonException
import com.pegasusx.retailer.ui.screens.cart.CartViewModel
import com.pegasusx.retailer.ui.screens.procurement.ProcurementLoadIssue
import com.pegasusx.retailer.ui.screens.procurement.ProcurementViewModel
import com.pegasusx.retailer.util.OrderLineContractHelper
import com.pegasusx.retailer.util.PackSnapMode
import java.io.IOException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class CartAndProcurementBehaviorTest {

    private val testDispatcher = StandardTestDispatcher()
    private val store = ViewModelStore()
    private val json = OrderLineContractHelper.lenientJson

    @Before
    fun setUp() {
        Dispatchers.setMain(testDispatcher)
    }

    @After
    fun tearDown() {
        store.clear()
        Dispatchers.resetMain()
    }

    @Test
    fun cartViewModel_snapMustHaveSubstitutesAutoConfirm_andOfflinePreviewQueueFallback() = runTest(testDispatcher) {
        val dao = FakePendingOrderDao()
        val enqueuer = FakeSyncEnqueuer()
        val apiCfg = RecordingPegasusApiConfig().apply {
            onCheckoutPreview = { throw IOException("Network offline") }
        }
        val vm = CartViewModel(
            api = apiCfg.build(),
            tokenManager = FakeUserIdProvider("ret-42"),
            retailerWebSocket = FakeRealtimeEventSource(),
            pendingOrderDao = dao,
            syncEnqueuer = enqueuer,
        )
        store.put("cart", vm)
        advanceUntilIdle()

        val variant = Variant(
            id = "sku-pack-6",
            size = "0.5L",
            pack = "Pack 6",
            packCount = 6,
            weightPerUnit = "500ml",
            price = 20_000L,
        )
        val product = Product(id = "prod-1", name = "Sparkling Water", supplierId = "sup-1", variants = listOf(variant))

        vm.addToCart(product, variant)
        val itemId = "${product.id}_${variant.id}"
        vm.updateQuantity(itemId, 10)
        assertTrue(vm.needsPackSnap(vm.uiState.value.items.single()))

        vm.snapItemToPackMultiple(itemId, PackSnapMode.NEAREST)
        assertEquals(12, vm.uiState.value.items.single().quantity)
        assertFalse(vm.needsPackSnap(vm.uiState.value.items.single()))

        vm.toggleMustHave(itemId)
        vm.toggleAllowSubstitutes(itemId)
        vm.setAutoConfirmOptIn(true)
        advanceUntilIdle()

        vm.processPayment()
        advanceUntilIdle()

        assertEquals(1, dao.rows.size)
        assertEquals(1, enqueuer.enqueueCount)
        val queued = json.decodeFromString(UnifiedCheckoutRequest.serializer(), dao.rows.single().payloadJson)
        assertTrue(queued.autoConfirmOptIn)
        assertEquals(12, queued.items.single().quantity)
        assertTrue(queued.items.single().mustHave)
        assertTrue(queued.items.single().allowSubstitutes)
        assertNotNull(vm.uiState.value.pendingQueueNotice)
    }

    @Test
    fun cartViewModel_parsesPackMultipleRequired_andAppliesOneTapFix() = runTest(testDispatcher) {
        val dao = FakePendingOrderDao()
        val enqueuer = FakeSyncEnqueuer()
        val apiCfg = RecordingPegasusApiConfig().apply {
            onUnifiedCheckout = { _, _ ->
                throw httpJsonException(
                    422,
                    """
                    {
                      "code": "PACK_MULTIPLE_REQUIRED",
                      "message": "Quantity must be a multiple of 12",
                      "sku_id": "sku-cola-12",
                      "pack_size": 12,
                      "requested": 10,
                      "suggested_floor": 0,
                      "suggested_ceil": 12,
                      "suggested_nearest": 12
                    }
                    """.trimIndent(),
                )
            }
        }
        val vm = CartViewModel(
            api = apiCfg.build(),
            tokenManager = FakeUserIdProvider("ret-42"),
            retailerWebSocket = FakeRealtimeEventSource(),
            pendingOrderDao = dao,
            syncEnqueuer = enqueuer,
        )
        store.put("cart-pack", vm)
        advanceUntilIdle()

        val variant = Variant(
            id = "sku-cola-12",
            size = "0.5L",
            pack = "Case 12",
            packCount = 12,
            weightPerUnit = "500ml",
            price = 15_000L,
        )
        val product = Product(id = "prod-cola", name = "Cola", supplierId = "sup-1", variants = listOf(variant))
        vm.addToCart(product, variant)
        vm.updateQuantity("${product.id}_${variant.id}", 10)
        advanceUntilIdle()

        vm.processPayment()
        advanceUntilIdle()

        assertTrue(dao.rows.isEmpty())
        assertEquals(0, enqueuer.enqueueCount)
        assertEquals(1, vm.uiState.value.packSuggestions.size)
        val fix = vm.uiState.value.packSuggestions.single()
        assertEquals(12, fix.suggestedNearest)

        vm.applyPackSuggestion(fix.skuId, fix.suggestedNearest)
        assertEquals(12, vm.uiState.value.items.single().quantity)
        assertTrue(vm.uiState.value.packSuggestions.isEmpty())
    }

    @Test
    fun cartViewModel_initFlushPreserves409OwnerApprovalRequired_andSchedulesRetry() = runTest(testDispatcher) {
        val dao = FakePendingOrderDao().apply {
            val req = UnifiedCheckoutRequest(
                retailerId = "ret-42",
                paymentGateway = "CASH",
                items = listOf(CheckoutLineItem(skuId = "sku-1", quantity = 6, unitPrice = 10_000L)),
            )
            rows.add(
                PendingOrderEntity(
                    id = 9L,
                    endpoint = "/v1/checkout/unified",
                    payloadJson = json.encodeToString(UnifiedCheckoutRequest.serializer(), req),
                    idempotencyKey = "idem-owner-9",
                ),
            )
        }
        val enqueuer = FakeSyncEnqueuer()
        val apiCfg = RecordingPegasusApiConfig().apply {
            onUnifiedCheckout = { _, _ ->
                throw ProblemDetailException(
                    ProblemDetail(
                        status = 409,
                        code = "owner_approval_required",
                        detail = "Waiting for owner approval",
                    ),
                )
            }
        }
        val vm = CartViewModel(
            api = apiCfg.build(),
            tokenManager = FakeUserIdProvider("ret-42"),
            retailerWebSocket = FakeRealtimeEventSource(),
            pendingOrderDao = dao,
            syncEnqueuer = enqueuer,
        )
        store.put("cart-owner", vm)
        advanceUntilIdle()

        assertEquals(1, dao.rows.size)
        assertTrue(dao.deletedIds.isEmpty())
        assertEquals(PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL, dao.rows.single().status)
        assertEquals(1, enqueuer.enqueueCount)
        assertEquals(PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL, vm.uiState.value.pendingQueueNotice?.status)
    }

    @Test
    fun procurementViewModel_queuesOfflineAndAppliesPackMultipleFixAndLineFlags() = runTest(testDispatcher) {
        val packVariant = Variant(
            id = "v-proc",
            size = "1kg",
            pack = "Pack 6",
            packCount = 6,
            weightPerUnit = "1kg",
            price = 30_000L,
        )
        val prod = Product(id = "prod-proc-1", name = "Flour 1kg", supplierId = "sup-2", variants = listOf(packVariant))
        val dao = FakePendingOrderDao()
        val enqueuer = FakeSyncEnqueuer()
        var mode = "PACK_ERR"
        val apiCfg = RecordingPegasusApiConfig().apply {
            catalogProducts = listOf(prod)
            onCreateOrder = { _, _ ->
                when (mode) {
                    "PACK_ERR" -> throw ProblemDetailException(
                        ProblemDetail(
                            status = 422,
                            code = "PACK_MULTIPLE_REQUIRED",
                            skuId = "prod-proc-1",
                            packSize = 6,
                            requested = 8,
                            suggestedFloor = 6,
                            suggestedCeil = 12,
                            suggestedNearest = 6,
                        ),
                    )
                    "OFFLINE" -> throw IOException("Socket closed")
                    else -> com.pegasusx.retailer.data.model.ProcurementOrderResponse(
                        status = "CREATED",
                        orderId = "ord-1",
                    )
                }
            }
        }
        val vm = ProcurementViewModel(
            api = apiCfg.build(),
            tokenManager = FakeUserIdProvider("ret-7"),
            pendingOrderDao = dao,
            syncEnqueuer = enqueuer,
        )
        store.put("proc", vm)
        advanceUntilIdle()

        val forecast = DemandForecast(
            id = "fc-1",
            productId = "prod-proc-1",
            productName = "Flour 1kg",
            predictedQuantity = 8,
        )
        vm.toggleSelection(forecast)
        vm.toggleMustHave(forecast.id)
        vm.toggleAllowSubstitutes(forecast.id)
        vm.setAutoConfirmOptIn(true)

        vm.createOrder()
        advanceUntilIdle()

        assertTrue(dao.rows.isEmpty())
        assertEquals(1, vm.uiState.value.packSuggestions.size)

        vm.applyPackSuggestion("prod-proc-1", 6)
        assertTrue(vm.uiState.value.packSuggestions.isEmpty())
        assertEquals(6, vm.uiState.value.quantities[forecast.id])

        mode = "OFFLINE"
        vm.createOrder()
        advanceUntilIdle()

        assertEquals(1, dao.rows.size)
        assertEquals("/v1/order/create", dao.rows.single().endpoint)
        assertEquals(1, enqueuer.enqueueCount)
        assertEquals(ProcurementLoadIssue.OFFLINE, vm.uiState.value.loadIssue)
        val queuedProc = json.decodeFromString(ProcurementOrderRequest.serializer(), dao.rows.single().payloadJson)
        assertTrue(queuedProc.autoConfirmOptIn)
        assertTrue(queuedProc.items.single().mustHave)
        assertTrue(queuedProc.items.single().allowSubstitutes)
        assertEquals(6, queuedProc.items.single().quantity)
    }

    @Test
    fun cartViewModel_preservesQueuedCheckoutOn401And429_andSnapsPackMultiples() = runTest(testDispatcher) {
        val packVariant = Variant(
            id = "v-snap",
            size = "1L",
            pack = "Case 6",
            packCount = 6,
            weightPerUnit = "1L",
            price = 10_000L,
        )
        val product = Product(
            id = "prod-snap-1",
            name = "Juice 1L",
            supplierId = "sup-snap",
            variants = listOf(packVariant),
        )
        var statusToThrow = 401
        val dao = FakePendingOrderDao()
        val enqueuer = FakeSyncEnqueuer()
        val apiCfg = RecordingPegasusApiConfig().apply {
            catalogProducts = listOf(product)
            onUnifiedCheckout = { _, _ ->
                if (statusToThrow == 401) {
                    throw ProblemDetailException(ProblemDetail(status = 401, code = "unauthorized", detail = "Token expired"))
                }
                throw ProblemDetailException(ProblemDetail(status = 429, code = "rate_limit_exceeded", detail = "Too many requests"))
            }
        }
        val vm = CartViewModel(
            api = apiCfg.build(),
            tokenManager = FakeUserIdProvider("ret-resilience"),
            retailerWebSocket = FakeRealtimeEventSource(),
            pendingOrderDao = dao,
            syncEnqueuer = enqueuer,
        )
        store.put("cart-resilience", vm)
        advanceUntilIdle()

        // Add to cart -> set non-multiple qty=8 -> snapToPackMultiple on packCount=6 -> snaps to 6!
        vm.addToCart(product, packVariant)
        val itemId = vm.uiState.value.items.single().id
        vm.updateQuantity(itemId, 8)
        assertEquals(8, vm.uiState.value.items.single().quantity)
        vm.snapItemToPackMultiple(itemId)
        assertEquals(6, vm.uiState.value.items.single().quantity)

        // 401 preserves queued checkout cleanly with AUTH_REQUIRED
        vm.showCheckout()
        vm.processPayment()
        advanceUntilIdle()

        assertEquals(1, dao.rows.size)
        assertEquals("AUTH_REQUIRED", dao.rows.single().status)
        assertTrue(vm.uiState.value.pendingQueueNotice?.keptInQueue == true)

        // 429 parks in queue as RATE_LIMITED
        statusToThrow = 429
        vm.processPayment()
        advanceUntilIdle()

        assertEquals(2, dao.rows.size)
        assertEquals(PendingOrderEntity.STATUS_RATE_LIMITED, dao.rows.last().status)
        assertTrue(dao.deletedIds.isEmpty())
    }
}
