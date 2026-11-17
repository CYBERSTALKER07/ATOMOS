package com.pegasusx.retailer.ui.screens

import androidx.lifecycle.ViewModelStore
import com.pegasusx.retailer.data.local.PendingOrderEntity
import com.pegasusx.retailer.data.model.ApprovalDecisionResponse
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.data.model.PSPListing
import com.pegasusx.retailer.data.model.ProblemDetail
import com.pegasusx.retailer.data.model.ProblemDetailException
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.RetailerApprovalItem
import com.pegasusx.retailer.data.model.RetailerApprovalsListResponse
import com.pegasusx.retailer.data.model.RetailerPaymentCatalogResponse
import com.pegasusx.retailer.data.model.SellerDailyUsageDTO
import com.pegasusx.retailer.data.model.SellerMandateDTO
import com.pegasusx.retailer.data.model.SellerMandateResponse
import com.pegasusx.retailer.data.model.UnifiedCheckoutResponse
import com.pegasusx.retailer.data.model.Variant
import com.pegasusx.retailer.testsupport.FakePendingOrderDao
import com.pegasusx.retailer.testsupport.FakeRealtimeEventSource
import com.pegasusx.retailer.testsupport.FakeSyncEnqueuer
import com.pegasusx.retailer.testsupport.FakeUserIdProvider
import com.pegasusx.retailer.testsupport.RecordingPegasusApiConfig
import com.pegasusx.retailer.testsupport.httpJsonException
import com.pegasusx.retailer.ui.screens.cart.CartViewModel
import com.pegasusx.retailer.ui.screens.procurement.ProcurementViewModel
import com.pegasusx.retailer.ui.screens.settings.OwnerGovernanceViewModel
import com.pegasusx.retailer.util.OrderLineContractHelper
import com.pegasusx.retailer.util.RetailerW2ContractHelper
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class RetailerW2Tests {

    private val testDispatcher = StandardTestDispatcher()
    private val store = ViewModelStore()

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
    fun claimMoneyAndErrors_formatWholeSomAndMapReusedPhotoRefused() {
        assertEquals("15 000 so'm", OrderLineContractHelper.formatWholeSom(1_500_000L, "UZS"))
        assertEquals("150 so'm", OrderLineContractHelper.formatWholeSom(15_000L, "UZS"))
        assertEquals(250_000_000L, RetailerW2ContractHelper.somInputToMinor("2 500 000"))
        assertEquals("15000", RetailerW2ContractHelper.minorToSomInput(1_500_000L))
        assertNull(RetailerW2ContractHelper.somInputToMinor("-100"))

        val reusedEx = httpJsonException(
            422,
            """{"error":"reused_photo_evidence_refused","message":"duplicate sha256"}""",
        )
        val msg = RetailerW2ContractHelper.formatClaimErrorMessage(reusedEx)
        assertTrue(msg.contains("already used", ignoreCase = true))

        val requiredEx = ProblemDetailException(
            ProblemDetail(status = 400, code = "photo_evidence_required", detail = "photo required"),
        )
        assertTrue(
            RetailerW2ContractHelper.formatClaimErrorMessage(requiredEx)
                .contains("Photo proof is required", ignoreCase = true),
        )
    }

    @Test
    fun cartViewModel_handles409OwnerApprovalRequired_andOneTapCashFallback() = runTest(testDispatcher) {
        val dao = FakePendingOrderDao()
        val apiCfg = RecordingPegasusApiConfig().apply {
            paymentCatalog = RetailerPaymentCatalogResponse(
                currencyCode = "UZS",
                catalog = listOf(
                    PSPListing(code = "GLOBAL_PAY", status = "ACTIVE", selectable = true),
                    PSPListing(code = "CASH", status = "ACTIVE", selectable = true),
                ),
            )
            onUnifiedCheckout = { req, _ ->
                if (req.paymentGateway != "CASH" && req.paymentMethod != "CASH") {
                    throw httpJsonException(
                        409,
                        """
                        {
                          "code": "owner_approval_required",
                          "request_id": "apr-101",
                          "status": "PENDING_OWNER",
                          "tripped_rule": "DAILY_CREDIT_CAP_EXCEEDED",
                          "amount_minor": 1500000,
                          "threshold_minor": 1000000,
                          "remaining_cap_minor": 200000,
                          "message": "Order exceeds daily credit limit.",
                          "fallback_payment_methods": ["CASH", "CARD"]
                        }
                        """.trimIndent(),
                    )
                }
                UnifiedCheckoutResponse(status = "CONFIRMED", invoiceId = "inv-cash-1", total = 1_500_000L)
            }
        }
        val vm = CartViewModel(
            api = apiCfg.build(),
            tokenManager = FakeUserIdProvider("ret-42"),
            retailerWebSocket = FakeRealtimeEventSource(),
            pendingOrderDao = dao,
            syncEnqueuer = FakeSyncEnqueuer(),
        )
        store.put("cart", vm)
        advanceUntilIdle()

        val variant = Variant(id = "sku-1", size = "1L", pack = "Box", packCount = 1, weightPerUnit = "1kg", price = 1_500_000L)
        val product = Product(id = "p-1", name = "Milk", supplierId = "sup-1", variants = listOf(variant))
        vm.addToCart(product, variant)
        vm.processPayment(skipBackorderConfirm = true)
        advanceUntilIdle()

        val approval = vm.uiState.value.ownerApprovalRequired
        assertNotNull(approval)
        assertEquals("apr-101", approval!!.requestId)
        assertEquals("15 000 so'm", approval.formattedAmount)
        assertTrue(approval.allowsCashFallback)
        assertTrue(approval.allowsCardFallback)
        assertEquals(PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL, dao.rows.single().status)

        vm.retryCheckoutWithFallbackMethod("CASH")
        advanceUntilIdle()

        assertEquals(2, apiCfg.unifiedCalls.size)
        val (firstReq, firstKey) = apiCfg.unifiedCalls[0]
        val (secondReq, secondKey) = apiCfg.unifiedCalls[1]
        assertEquals("GLOBAL_PAY", firstReq.paymentGateway)
        assertEquals("CASH", secondReq.paymentGateway)
        assertEquals("CASH", secondReq.paymentMethod)
        assertNotEquals(firstKey, secondKey)
        assertTrue(dao.rows.isEmpty())
        assertNull(vm.uiState.value.ownerApprovalRequired)
    }

    @Test
    fun procurementViewModel_handles409OwnerApprovalRequired_andCardFallback() = runTest(testDispatcher) {
        val dao = FakePendingOrderDao()
        val forecast = DemandForecast(
            id = "fc-1",
            productId = "sku-99",
            productName = "Sugar",
            predictedQuantity = 4,
            confidence = 0.9,
        )
        val apiCfg = RecordingPegasusApiConfig().apply {
            catalogProducts = listOf(
                Product(
                    id = "sku-99",
                    name = "Sugar",
                    supplierId = "sup-1",
                    variants = listOf(
                        Variant(id = "sku-99", size = "1kg", pack = "Bag", packCount = 1, weightPerUnit = "1kg", price = 225_000L),
                    ),
                ),
            )
            onCreateOrder = { req, _ ->
                if (req.paymentMethod != "CARD") {
                    throw httpJsonException(
                        409,
                        """
                        {
                          "error": "owner_approval_required",
                          "request_id": "apr-202",
                          "tripped_rule": "MAX_SINGLE_ORDER_EXCEEDED",
                          "amount_minor": 900000,
                          "fallback_payment_methods": ["CASH", "CARD"]
                        }
                        """.trimIndent(),
                    )
                }
                com.pegasusx.retailer.data.model.ProcurementOrderResponse(status = "CREATED", orderId = "ord-card-2")
            }
        }
        val vm = ProcurementViewModel(
            api = apiCfg.build(),
            tokenManager = FakeUserIdProvider("ret-42"),
            pendingOrderDao = dao,
            syncEnqueuer = FakeSyncEnqueuer(),
        )
        store.put("proc", vm)
        advanceUntilIdle()

        vm.toggleSelection(forecast)
        vm.createOrder()
        advanceUntilIdle()

        val approval = vm.uiState.value.ownerApprovalRequired
        assertNotNull(approval)
        assertEquals("9 000 so'm", approval!!.formattedAmount)
        assertEquals(PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL, dao.rows.single().status)

        vm.retryWithFallbackPaymentMethod("CARD")
        advanceUntilIdle()

        assertEquals(2, apiCfg.procurementCalls.size)
        assertEquals("CARD", apiCfg.procurementCalls[1].second.paymentMethod)
        assertEquals("GLOBAL_PAY", apiCfg.procurementCalls[1].second.paymentGateway)
        assertNotEquals(apiCfg.procurementCalls[0].third, apiCfg.procurementCalls[1].third)
        assertTrue(dao.rows.isEmpty())
        assertNull(vm.uiState.value.ownerApprovalRequired)
        assertTrue(vm.uiState.value.showSuccess)
    }

    @Test
    fun ownerGovernanceViewModel_gatesNonOwner_approvesAndSavesMandateInWholeSom() = runTest(testDispatcher) {
        val nonOwnerApi = RecordingPegasusApiConfig().apply {
            onGetRetailerMe = {
                JsonObject(mapOf("retailer_role" to JsonPrimitive("SELLER"), "is_owner" to JsonPrimitive(false)))
            }
        }
        val nonOwnerVm = OwnerGovernanceViewModel(nonOwnerApi.build())
        store.put("gov-non-owner", nonOwnerVm)
        nonOwnerVm.loadRoleAndApprovals()
        advanceUntilIdle()
        assertFalse(nonOwnerVm.uiState.value.isOwner)
        assertTrue(nonOwnerVm.uiState.value.approvals.isEmpty())

        var approvedId = ""
        val ownerApi = RecordingPegasusApiConfig().apply {
            onGetRetailerMe = {
                JsonObject(mapOf("retailer_role" to JsonPrimitive("OWNER"), "is_owner" to JsonPrimitive(true)))
            }
            onGetRetailerApprovals = { _, _, _ ->
                RetailerApprovalsListResponse(
                    retailerId = "ret-1",
                    items = listOf(
                        RetailerApprovalItem(
                            requestId = "req-1",
                            requestedByName = "Seller Ali",
                            amountMinor = 2_500_000L,
                            trippedRule = "DAILY_CREDIT_CAP_EXCEEDED",
                            status = "PENDING_OWNER",
                        ),
                    ),
                )
            }
            onApproveRetailerRequest = { id, _ ->
                approvedId = id
                ApprovalDecisionResponse(requestId = id, status = "APPROVED", orderId = "ord-9")
            }
            onGetSellerMandate = { userId ->
                SellerMandateResponse(
                    userId = userId,
                    retailerRole = "SELLER",
                    mandate = SellerMandateDTO(
                        canReceiveGoods = true,
                        canPayCash = false,
                        canUseCredit = true,
                        dailyCreditCapMinor = 1_500_000L,
                        dailyCashCapMinor = 500_000L,
                        maxSingleOrderMinor = 250_000L,
                    ),
                    dailyUsage = SellerDailyUsageDTO(spentCreditMinor = 300_000L, orderCount = 2),
                )
            }
        }
        val ownerVm = OwnerGovernanceViewModel(ownerApi.build())
        store.put("gov-owner", ownerVm)
        ownerVm.loadRoleAndApprovals()
        advanceUntilIdle()

        assertTrue(ownerVm.uiState.value.isOwner)
        assertEquals(1, ownerVm.uiState.value.approvals.size)

        ownerVm.approveRequest("req-1", "Approved by owner")
        advanceUntilIdle()
        assertEquals("req-1", approvedId)

        ownerVm.openMandateEditor("usr-seller-7", "Seller Ali")
        advanceUntilIdle()
        val form = ownerVm.uiState.value.mandateForm!!
        assertEquals("15000", form.dailyCreditCapSom)
        assertEquals("5000", form.dailyCashCapSom)
        assertEquals("2500", form.maxSingleOrderSom)

        ownerVm.updateMandateForm {
            it.copy(
                canPayCash = true,
                dailyCreditCapSom = "30000",
                dailyCashCapSom = "10000",
                maxSingleOrderSom = "5000",
            )
        }
        ownerVm.saveMandate()
        advanceUntilIdle()

        val (savedUserId, body) = ownerApi.mandateUpdates.single()
        assertEquals("usr-seller-7", savedUserId)
        assertEquals(true, body.canPayCash)
        assertEquals(3_000_000L, body.dailyCreditCapMinor)
        assertEquals(1_000_000L, body.dailyCashCapMinor)
        assertEquals(500_000L, body.maxSingleOrderMinor)
        assertNull(ownerVm.uiState.value.mandateForm)
    }
}
