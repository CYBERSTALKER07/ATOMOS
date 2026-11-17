package com.pegasusx.retailer.ui.screens

import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.InMemoryRetailerPreferencesStore
import com.pegasusx.retailer.data.local.NoopRetailerRealtimeHub
import com.pegasusx.retailer.data.local.RetailerLanguage
import com.pegasusx.retailer.data.local.RetailerSessionContext
import com.pegasusx.retailer.data.local.RetailerWorkspaceMode
import com.pegasusx.retailer.data.model.ControlTowerPulse
import com.pegasusx.retailer.data.model.CreditProfile
import com.pegasusx.retailer.data.model.FastHomePendingProposal
import com.pegasusx.retailer.data.model.FastHomeProposalItem
import com.pegasusx.retailer.data.model.FastHomeTomorrowDelivery
import com.pegasusx.retailer.data.model.LoyaltyTierView
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.OrderLineItem
import com.pegasusx.retailer.data.model.OrderStatus
import com.pegasusx.retailer.data.model.RetailerAIPrediction
import com.pegasusx.retailer.data.model.RetailerAIPredictionsResponse
import com.pegasusx.retailer.data.model.RetailerFastHomeResponse
import com.pegasusx.retailer.data.model.TelegramLinkRequestResponse
import com.pegasusx.retailer.data.model.TelegramLinkStatusResponse
import com.pegasusx.retailer.ui.screens.dashboard.DashboardViewModel
import com.pegasusx.retailer.ui.screens.profile.ProfileViewModel
import com.pegasusx.retailer.util.FastHomeContractHelper
import com.pegasusx.retailer.util.OrderLineContractHelper
import java.lang.reflect.Proxy
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class FastHomeAndLocaleBehaviorTest {

    private val testDispatcher = StandardTestDispatcher()

    @Before
    fun setUp() {
        Dispatchers.setMain(testDispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private class FakeSessionContext(
        private val userId: String = "ret-42",
        private val userName: String = "Baraka Market",
        private val token: String? = null,
    ) : RetailerSessionContext {
        override fun getUserId(): String = userId
        override fun getUserName(): String = userName
        override fun getPreferredToken(): String? = token
    }

    private fun buildApiProxy(
        role: String = "BUYER",
        isOwner: Boolean = false,
        fastHomeProvider: (() -> RetailerFastHomeResponse)? = null,
        orders: List<Order> = emptyList(),
        predictions: List<RetailerAIPrediction> = emptyList(),
        creditProfile: CreditProfile = CreditProfile(),
        relationships: JsonElement = buildJsonArray { },
        onConfirmPreorder: (Map<String, Any>) -> Unit = {},
        onRejectPreorder: (Map<String, Any>) -> Unit = {},
        onEditPreorder: (Map<String, Any>) -> Unit = {},
        onConfirmAiOrder: (Map<String, Any>) -> Unit = {},
        onRejectAiOrder: (Map<String, Any>) -> Unit = {},
    ): PegasusApi {
        return Proxy.newProxyInstance(
            PegasusApi::class.java.classLoader,
            arrayOf(PegasusApi::class.java),
        ) { _, method, args ->
            when (method.name) {
                "getRetailerMe" -> buildJsonObject {
                    put("retailer_role", role)
                    put("is_owner", isOwner)
                }
                "getRetailerFastHome" -> fastHomeProvider?.invoke()
                    ?: error("404 fast-home not mocked")
                "getOrders" -> orders
                "getRetailerAIPredictions" -> RetailerAIPredictionsResponse(items = predictions)
                "getCatalogProducts" -> emptyList<Any>()
                "getRetailerPulse" -> com.pegasusx.retailer.data.model.PulseResponse()
                "getControlTowerPulse" -> ControlTowerPulse(retailerId = "ret-42", empty = false, source = "live")
                "getCreditProfile" -> creditProfile
                "getCreditRelationships" -> relationships
                "getRetailerProfile" -> mapOf("name" to "Baraka Market", "phone" to "+998901234567")
                "getPricingRules" -> buildJsonObject { put("status", "ok") }
                "getLoyaltyTier" -> LoyaltyTierView(enrolled = false)
                "getClientPolicy" -> buildJsonObject { put("outdated", false) }
                "getGlobalAutoOrder" -> com.pegasusx.retailer.data.model.EmpathySettings()
                "getTelegramLinkStatus" -> TelegramLinkStatusResponse(linked = false)
                "requestTelegramLink" -> TelegramLinkRequestResponse(
                    ok = true,
                    otpCode = "654321",
                    expiresAt = "2026-11-13T10:00:00Z",
                    botDeepLink = "https://t.me/PegasusBot?start=654321",
                )
                "confirmPreorder" -> {
                    @Suppress("UNCHECKED_CAST")
                    onConfirmPreorder(args?.firstOrNull() as? Map<String, Any> ?: emptyMap())
                    com.pegasusx.retailer.data.model.ApiResponse(status = "CONFIRMED")
                }
                "rejectPreorder" -> {
                    @Suppress("UNCHECKED_CAST")
                    onRejectPreorder(args?.firstOrNull() as? Map<String, Any> ?: emptyMap())
                    com.pegasusx.retailer.data.model.ApiResponse(status = "HELD")
                }
                "editPreorder" -> {
                    @Suppress("UNCHECKED_CAST")
                    onEditPreorder(args?.firstOrNull() as? Map<String, Any> ?: emptyMap())
                    com.pegasusx.retailer.data.model.ApiResponse(status = "EDITED")
                }
                "confirmAiOrder" -> {
                    @Suppress("UNCHECKED_CAST")
                    onConfirmAiOrder(args?.firstOrNull() as? Map<String, Any> ?: emptyMap())
                    com.pegasusx.retailer.data.model.ApiResponse(status = "CONFIRMED")
                }
                "rejectAiOrder" -> {
                    @Suppress("UNCHECKED_CAST")
                    onRejectAiOrder(args?.firstOrNull() as? Map<String, Any> ?: emptyMap())
                    com.pegasusx.retailer.data.model.ApiResponse(status = "REJECTED")
                }
                else -> {
                    when (method.returnType) {
                        List::class.java -> emptyList<Any>()
                        Map::class.java -> emptyMap<String, String>()
                        JsonElement::class.java, JsonObject::class.java -> buildJsonObject { }
                        else -> null
                    }
                }
            }
        } as PegasusApi
    }

    @Test
    fun cutoffCountdown_ignoresDeviceClockSkew_usingServerOffset() {
        // Server says it's 10:00:00Z and cutoff is 12:30:45Z -> remaining MUST be 02:30:45
        // even if device clock is drifted 3 hours ahead (13:00:00Z)!
        val serverNow = "2026-11-13T10:00:00Z"
        val cutoffAt = "2026-11-13T12:30:45Z"
        val skewedClientNowMs = FastHomeContractHelper.parseServerTimestampMillis("2026-11-13T13:00:00Z")!!

        val offsetMs = FastHomeContractHelper.computeServerClockOffsetMs(serverNow, skewedClientNowMs)
        val countdown = FastHomeContractHelper.computeCutoffCountdown(cutoffAt, offsetMs, skewedClientNowMs)

        assertTrue(countdown.valid)
        assertFalse(countdown.expired)
        assertEquals("02:30:45", countdown.formatted)

        // Advance skewed client clock by 3 hours -> past server cutoff (13:00Z >= 12:30:45Z)
        val laterClientMs = skewedClientNowMs + 3 * 3600_000L
        val expiredCountdown = FastHomeContractHelper.computeCutoffCountdown(cutoffAt, offsetMs, laterClientMs)
        assertTrue(expiredCountdown.valid)
        assertTrue(expiredCountdown.expired)
        assertEquals("00:00:00", expiredCountdown.formatted)
    }

    @Test
    fun dashboardViewModel_defaultsFastModeForBuyerAndFullErpForOwner_andPersistsSwitch() = runTest(testDispatcher) {
        val buyerPrefs = InMemoryRetailerPreferencesStore()
        val buyerVm = DashboardViewModel(
            api = buildApiProxy(
                role = "BUYER",
                isOwner = false,
                fastHomeProvider = {
                    RetailerFastHomeResponse(
                        serverNow = "2026-11-13T10:00:00Z",
                        cutoffAt = "2026-11-13T13:00:00Z",
                        tomorrowDelivery = FastHomeTomorrowDelivery(
                            orderId = "ord-del-1",
                            supplierName = "Sultona Beverages",
                            state = "DISPATCHED",
                            itemCount = 12,
                            amountMinor = 25_000_000L,
                            currency = "UZS",
                        ),
                        balanceOwedMinor = 45_000_000L,
                        overdueMinor = 10_000_000L,
                        creditHeadroomMinor = 55_000_000L,
                        pendingProposal = FastHomePendingProposal(
                            orderId = "prop-1",
                            source = "MANUAL_PREORDER",
                            supplierName = "Sultona Beverages",
                            state = "SCHEDULED",
                            amountMinor = 18_500_000L,
                            currency = "UZS",
                            items = listOf(
                                FastHomeProposalItem(
                                    productId = "sku-water",
                                    sku = "sku-water",
                                    productName = "Water 1.5L",
                                    quantity = 20,
                                    unitPriceMinor = 925_000L,
                                ),
                            ),
                        ),
                        pendingApprovalsCount = 2,
                    )
                },
            ),
            sessionContext = FakeSessionContext(),
            preferencesStore = buyerPrefs,
            realtimeHub = NoopRetailerRealtimeHub,
        )
        advanceUntilIdle()

        assertEquals(RetailerWorkspaceMode.FAST, buyerVm.uiState.value.workspaceMode)
        val fastHome = buyerVm.uiState.value.fastHome
        assertNotNull(fastHome)
        assertEquals("450 000 so'm", OrderLineContractHelper.formatWholeSom(fastHome!!.balanceOwedMinor, fastHome.currency))
        assertEquals(2, fastHome.pendingApprovalsCount)

        // Owner defaults to Full ERP, and can switch to Fast Mode persistently
        val ownerPrefs = InMemoryRetailerPreferencesStore()
        val ownerVm = DashboardViewModel(
            api = buildApiProxy(role = "OWNER", isOwner = true),
            sessionContext = FakeSessionContext(),
            preferencesStore = ownerPrefs,
            realtimeHub = NoopRetailerRealtimeHub,
        )
        advanceUntilIdle()
        assertEquals(RetailerWorkspaceMode.FULL, ownerVm.uiState.value.workspaceMode)

        ownerVm.switchWorkspaceMode(RetailerWorkspaceMode.FAST)
        assertEquals(RetailerWorkspaceMode.FAST, ownerVm.uiState.value.workspaceMode)
        assertEquals(RetailerWorkspaceMode.FAST, ownerPrefs.getWorkspaceMode())
    }

    @Test
    fun dashboardViewModel_fallsBackSafelyWhenFastHomeUnavailable_andDispatchesProposalActions() = runTest(testDispatcher) {
        var confirmedPreorderId: String? = null
        var heldPreorderId: String? = null
        val prefs = InMemoryRetailerPreferencesStore()
        val vm = DashboardViewModel(
            api = buildApiProxy(
                role = "MANAGER",
                fastHomeProvider = { error("404 endpoint not deployed") },
                orders = listOf(
                    Order(
                        id = "ord-preorder-9",
                        supplierId = "sup-1",
                        supplierName = "Atlas Dairy",
                        status = OrderStatus.SCHEDULED,
                        orderSource = "MANUAL_PREORDER",
                        confirmationStatus = "DRAFT",
                        totalAmount = 12_000_000L,
                        currency = "UZS",
                        proposedDeliveryDate = "2026-11-14T08:00:00Z",
                        autoConfirmAt = "2026-11-13T16:00:00Z",
                        items = listOf(
                            OrderLineItem(id = "li-1", productId = "sku-milk", productName = "Milk 1L", quantity = 10, unitPrice = 1_200_000),
                        ),
                    ),
                ),
                creditProfile = CreditProfile(
                    currentBalanceMinor = 80_000_000L,
                    availableCreditMinor = 120_000_000L,
                    delinquencyCount = 1L,
                ),
                onConfirmPreorder = { body -> confirmedPreorderId = body["order_id"] as? String },
                onRejectPreorder = { body -> heldPreorderId = body["order_id"] as? String },
            ),
            sessionContext = FakeSessionContext(),
            preferencesStore = prefs,
            realtimeHub = NoopRetailerRealtimeHub,
        )
        advanceUntilIdle()

        val synthesized = vm.uiState.value.fastHome
        assertNotNull(synthesized)
        assertTrue(synthesized!!.degradedFallback)
        assertEquals(80_000_000L, synthesized.balanceOwedMinor)
        assertEquals("ord-preorder-9", synthesized.pendingProposal?.orderId)
        assertEquals("ord-preorder-9", synthesized.tomorrowDelivery?.orderId)

        val proposal = synthesized.pendingProposal!!
        vm.confirmFastProposal(proposal)
        advanceUntilIdle()
        assertEquals("ord-preorder-9", confirmedPreorderId)

        vm.holdFastProposal(proposal)
        advanceUntilIdle()
        assertEquals("ord-preorder-9", heldPreorderId)
    }

    @Test
    fun profileViewModel_defaultsToUzbekAndSwitchesLanguagesAndGeneratesTelegramOtp() = runTest(testDispatcher) {
        val prefs = InMemoryRetailerPreferencesStore()
        val vm = ProfileViewModel(
            api = buildApiProxy(role = "CASHIER", isOwner = false),
            sessionContext = FakeSessionContext(),
            preferencesStore = prefs,
        )
        advanceUntilIdle()

        assertEquals(RetailerLanguage.UZ, vm.uiState.value.selectedLanguage)
        assertEquals(RetailerWorkspaceMode.FAST, vm.uiState.value.workspaceMode)

        vm.setLanguage(null, RetailerLanguage.RU)
        assertEquals(RetailerLanguage.RU, vm.uiState.value.selectedLanguage)
        assertEquals(RetailerLanguage.RU, prefs.getLanguage())

        vm.setLanguage(null, RetailerLanguage.EN)
        assertEquals(RetailerLanguage.EN, vm.uiState.value.selectedLanguage)
        assertEquals(RetailerLanguage.EN, prefs.getLanguage())

        vm.requestTelegramLink()
        advanceUntilIdle()
        assertEquals("654321", vm.uiState.value.telegramLinkRequest?.otpCode)
        assertEquals("https://t.me/PegasusBot?start=654321", vm.uiState.value.telegramLinkRequest?.botDeepLink)
    }
}
