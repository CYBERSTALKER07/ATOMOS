package com.pegasusx.retailer.ui.navigation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.RetailerWSMessage
import com.pegasusx.retailer.data.api.RetailerWebSocket
import com.pegasusx.retailer.data.api.ShopClosedAlert
import com.pegasusx.retailer.data.api.reconcileRetailerSession
import com.pegasusx.retailer.util.RetailerIdempotencyKeys
import com.pegasusx.retailer.data.api.toShopClosedAlert
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.CardCheckoutRequest
import com.pegasusx.retailer.data.model.ConfirmCashRequest
import com.pegasusx.retailer.data.model.CashCheckoutRequest
import com.pegasusx.retailer.data.model.formatRetailerAmount
import com.pegasusx.retailer.data.model.selectableRetailerCatalogCodes
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.PendingPaymentSession
import com.pegasusx.retailer.services.PendingOrderSyncScheduler
import com.pegasusx.retailer.service.AutoUpdater
import com.pegasusx.retailer.service.EnterpriseUpdateConfig
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import android.content.Context
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import retrofit2.HttpException
import javax.inject.Inject

enum class NavigationLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class CardCheckoutResult(
    val paymentUrl: String? = null,
    val sessionId: String? = null,
    val attemptId: String? = null,
    val attemptNo: Int = 0,
)

data class NavigationUiState(
    val activeOrders: List<Order> = emptyList(),
    val approachingOrderIds: Set<String> = emptySet(),
    val userName: String = "",
    val companyName: String = "",
    val avatarInitial: String = "?",
    val paymentEvent: RetailerWSMessage? = null,
    val orderCompleted: Boolean = false,
    /** ADR-009: payment captured, waiting on OFD fiscal receipt. */
    val fiscalizing: Boolean = false,
    val paymentFailed: Boolean = false,
    val paymentFailureMessage: String = "",
    val isRefreshing: Boolean = false,
    val syncError: String? = null,
    val loadIssue: NavigationLoadIssue? = null,
    val shopClosedAlert: ShopClosedAlert? = null,
    val reconnectEpoch: Long = 0,
    val unreadNotificationCount: Int = 0,
    val clientPolicyMessage: String? = null,
    val clientPolicyForce: Boolean = false,
    val allowedCardGateways: List<String> = emptyList(),
) {
    val activeOrderCount: Int get() = activeOrders.size
    val floatingStatusText: String
        get() = activeOrders.firstOrNull()?.status?.displayName ?: ""
    val floatingTotalDisplay: String
        get() {
            val total = activeOrders.sumOf { it.totalAmount }
            val currency = activeOrders.firstOrNull()?.currency?.ifBlank { null }
                ?: com.pegasusx.retailer.data.model.sessionPackCurrency()
            return if (total > 0) formatRetailerAmount(total, currency) else ""
        }
    val floatingCountdownIso: String?
        get() = activeOrders.firstOrNull()?.estimatedDelivery
    val syncMessage: String?
        get() = when (loadIssue) {
            NavigationLoadIssue.RESTRICTED -> "Order feed access is restricted for this account"
            NavigationLoadIssue.OFFLINE -> "Offline mode active. Showing latest active orders"
            NavigationLoadIssue.ERROR -> "Order feed sync degraded. Retry is available"
            null -> null
        }
}

private fun PendingPaymentSession.toRetailerPaymentEvent(): RetailerWSMessage {
    val normalizedGateway = gateway.orEmpty().ifBlank { "GLOBAL_PAY" }
    return RetailerWSMessage(
        type = "PAYMENT_REQUIRED",
        orderId = orderId,
        invoiceId = invoiceId ?: "",
        sessionId = sessionId.orEmpty(),
        amount = lockedAmount,
        originalAmount = lockedAmount,
        availableCardGateways = if (normalizedGateway == "CASH") emptyList() else listOf(normalizedGateway),
        message = "Pending payment requires completion.",
        paymentMethod = if (normalizedGateway == "CASH") "CASH" else "CARD",
        gateway = normalizedGateway,
    )
}

@HiltViewModel
class NavigationViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
    private val retailerWebSocket: RetailerWebSocket,
    @ApplicationContext private val appContext: Context,
) : ViewModel() {

    private val _uiState = MutableStateFlow(NavigationUiState())
    val uiState: StateFlow<NavigationUiState> = _uiState.asStateFlow()

    private val autoUpdater = AutoUpdater(appContext.applicationContext)
    private var pendingManifest: AutoUpdater.Manifest? = null

    init {
        autoUpdater.register()
        val name = tokenManager.getUserName() ?: ""
        _uiState.update {
            it.copy(
                userName = name,
                companyName = "Pegasus",
                avatarInitial = name.firstOrNull()?.uppercase() ?: "?",
            )
        }
        loadActiveOrders()
        loadPendingPayments()
        loadPaymentCatalog()
        loadNotificationBadge()
        loadClientPolicy()
        connectWebSocket()
        observeTransportReconnect()
        PendingOrderSyncScheduler.enqueue(appContext)
    }

    private fun observeTransportReconnect() {
        viewModelScope.launch {
            retailerWebSocket.reconnects.collect {
                reconcileAfterReconnect()
            }
        }
    }

    fun reconcileAfterReconnect() {
        viewModelScope.launch {
            reconcileRetailerSession(api)
        }
        _uiState.update { it.copy(reconnectEpoch = System.currentTimeMillis()) }
        loadActiveOrders()
        loadPendingPayments(reconcile = true)
        loadClientPolicy()
        PendingOrderSyncScheduler.enqueue(appContext)
    }

    private fun loadPaymentCatalog() {
        viewModelScope.launch {
            try {
                val resp = api.getPaymentCatalog()
                _uiState.update {
                    it.copy(allowedCardGateways = selectableRetailerCatalogCodes(resp.catalog))
                }
            } catch (_: Exception) {
                _uiState.update { it.copy(allowedCardGateways = emptyList()) }
            }
        }
    }

    fun loadNotificationBadge() {
        viewModelScope.launch {
            try {
                val page = api.getNotifications(limit = 1, offset = 0)
                _uiState.update { it.copy(unreadNotificationCount = page.unreadCount) }
            } catch (_: Exception) {
                // Badge is best-effort; inbox screen loads full feed.
            }
        }
    }

    fun loadClientPolicy() {
        viewModelScope.launch {
            try {
                val policy = api.getClientPolicy(
                    role = EnterpriseUpdateConfig.POLICY_ROLE,
                    platform = "android",
                    version = com.pegasusx.retailer.BuildConfig.VERSION_NAME,
                    channel = EnterpriseUpdateConfig.CHANNEL,
                )
                val obj = policy.jsonObject
                val outdated = obj["outdated"]?.jsonPrimitive?.boolean == true
                val forceUpdate = obj["force_update"]?.jsonPrimitive?.boolean == true
                val updateDeferred = obj["update_deferred"]?.jsonPrimitive?.boolean == true
                val minimum = obj["minimum_version"]?.jsonPrimitive?.content
                val recommended = obj["recommended_version"]?.jsonPrimitive?.content
                val deferReason = obj["defer_reason"]?.jsonPrimitive?.content
                val updateUrl = obj["update_url"]?.jsonPrimitive?.content

                val state = autoUpdater.checkFromPolicyFields(
                    outdated = outdated,
                    forceUpdate = forceUpdate,
                    updateDeferred = updateDeferred,
                    minimumVersion = minimum,
                    recommendedVersion = recommended,
                    deferReason = deferReason,
                    updateUrl = updateUrl,
                    autoDownload = false,
                )
                pendingManifest = state.manifest
                _uiState.update {
                    it.copy(
                        clientPolicyMessage = state.message,
                        clientPolicyForce = state.force,
                    )
                }
            } catch (_: Exception) {
                // Policy fetch is optional on local/dev stacks.
            }
        }
    }

    fun onUpdateClick() {
        viewModelScope.launch {
            autoUpdater.startUpdate(pendingManifest)
        }
    }

    fun dismissClientPolicyBanner() {
        if (!_uiState.value.clientPolicyForce) {
            _uiState.update { it.copy(clientPolicyMessage = null) }
        }
    }

    fun retrySync() {
        loadActiveOrders()
        loadPendingPayments()
    }

    fun loadActiveOrders() {
        viewModelScope.launch {
            _uiState.update { it.copy(isRefreshing = true, syncError = null) }
            try {
                val rid = tokenManager.getUserId() ?: return@launch
                val orders = api.getOrders(rid)
                val active = orders.filter { it.status.isActive }
                _uiState.update {
                    it.copy(
                        activeOrders = active,
                        isRefreshing = false,
                        syncError = null,
                        loadIssue = null,
                    )
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isRefreshing = false,
                        syncError = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun clearPaymentEvent() {
        _uiState.update {
            it.copy(
                paymentEvent = null,
                orderCompleted = false,
                fiscalizing = false,
                paymentFailed = false,
                paymentFailureMessage = "",
            )
        }
        loadActiveOrders()
        loadPendingPayments()
    }

    fun clearShopClosedAlert() {
        _uiState.update { it.copy(shopClosedAlert = null) }
    }

    suspend fun respondToShopClosed(
        orderId: String,
        response: String,
        photoUrl: String? = null,
    ): Result<Unit> {
        return try {
            if (response == "AUTHORIZE_BYPASS" && photoUrl.isNullOrBlank()) {
                return Result.failure(
                    IllegalArgumentException("Doorway / drop-off photo is required to authorize bypass."),
                )
            }
            val body = mutableMapOf("order_id" to orderId, "response" to response)
            if (!photoUrl.isNullOrBlank()) {
                body["photo_url"] = photoUrl.trim()
            }
            api.shopClosedResponse(
                body,
                "shop-closed-response:$orderId:$response",
            )
            clearShopClosedAlert()
            loadActiveOrders()
            Result.success(Unit)
        } catch (e: Exception) {
            val msg = e.message.orEmpty()
            if (msg.contains("photo_url_required_for_bypass")) {
                Result.failure(
                    IllegalArgumentException("Doorway / drop-off photo is required to authorize bypass."),
                )
            } else {
                Result.failure(e)
            }
        }
    }

    fun loadPendingPayments(reconcile: Boolean = false) {
        viewModelScope.launch {
            try {
                val response = api.getPendingPayments()
                val pending = response.pendingPayments.firstOrNull()
                if (pending != null) {
                    _uiState.update {
                        it.copy(
                            paymentEvent = pending.toRetailerPaymentEvent(),
                            syncError = null,
                            loadIssue = null,
                        )
                    }
                } else if (reconcile && _uiState.value.paymentEvent != null) {
                    _uiState.update { it.copy(paymentEvent = null) }
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        syncError = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    suspend fun cashCheckout(orderId: String): Result<Unit> {
        return try {
            api.cashCheckout(CashCheckoutRequest(orderId = orderId), "retailer-cash-checkout:$orderId")
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun confirmCash(orderId: String): Result<Unit> {
        return try {
            api.confirmCash(ConfirmCashRequest(orderId = orderId), RetailerIdempotencyKeys.confirmCash(orderId))
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun cardCheckout(orderId: String, gateway: String): Result<CardCheckoutResult> {
        return try {
            val resp = api.cardCheckout(
                CardCheckoutRequest(orderId = orderId, gateway = gateway),
                "retailer-card-checkout:$orderId:$gateway",
            )
            val result = CardCheckoutResult(
                paymentUrl = resp.paymentUrl,
                sessionId = resp.sessionId,
                attemptId = resp.attemptId,
                attemptNo = resp.attemptNo ?: 0,
            )
            Result.success(result)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private fun connectWebSocket() {
        retailerWebSocket.connect()
        viewModelScope.launch {
            retailerWebSocket.events
                .filter {
                    it.type == "PAYMENT_REQUIRED" ||
                        it.type == "GLOBAL_PAYNT_REQUIRED" ||
                        it.type == "SETTLEMENT_REQUIRED"
                }
                .collect { msg ->
                    _uiState.update { it.copy(paymentEvent = msg) }
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "DELIVERY_SESSION_UPDATED" }
                .collect { msg ->
                    val current = _uiState.value.paymentEvent
                    if (current != null && current.orderId == msg.orderId) {
                        val updatedAmount = if (msg.adjustedAmount > 0) msg.adjustedAmount else current.amount
                        val updatedOriginalAmount = when {
                            msg.originalAmount > 0 -> msg.originalAmount
                            current.originalAmount > 0 -> current.originalAmount
                            else -> updatedAmount
                        }
                        val updatedState = if (msg.state.isNotBlank()) msg.state else current.state
                        _uiState.update {
                            it.copy(
                                paymentEvent = current.copy(
                                    amount = updatedAmount,
                                    originalAmount = updatedOriginalAmount,
                                    state = updatedState,
                                ),
                            )
                        }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter {
                    it.type == "ORDER_COMPLETED" ||
                        it.type == "ORDER_FINALIZED" ||
                        it.type == "FISCAL_RECEIPT_SUCCEEDED"
                }
                .collect { msg ->
                    val current = _uiState.value.paymentEvent
                    if (current != null && current.orderId == msg.orderId) {
                        _uiState.update { it.copy(orderCompleted = true, fiscalizing = false) }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter {
                    it.type == "FISCAL_RECEIPT_REQUESTED" ||
                        it.type == "PAYMENT_CLEARED" ||
                        (it.type == "ORDER_STATUS_CHANGED" && it.state.equals("FISCALIZING", true))
                }
                .collect { msg ->
                    val current = _uiState.value.paymentEvent
                    if (current != null && current.orderId == msg.orderId) {
                        _uiState.update { it.copy(fiscalizing = true, orderCompleted = false) }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "DRIVER_APPROACHING" }
                .collect { msg ->
                    if (msg.orderId.isNotBlank()) {
                        _uiState.update { it.copy(
                            approachingOrderIds = it.approachingOrderIds + msg.orderId
                        ) }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "PAYMENT_SETTLED" || it.type == "GLOBAL_PAYNT_SETTLED" }
                .collect { msg ->
                    // Money settled — still wait for fiscal SUCCESS before orderCompleted (ADR-009).
                    val current = _uiState.value.paymentEvent
                    if (current != null && current.orderId == msg.orderId) {
                        _uiState.update { it.copy(fiscalizing = true, orderCompleted = false) }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter {
                    it.type == "PAYMENT_FAILED" ||
                        it.type == "PAYMENT_EXPIRED" ||
                        it.type == "GLOBAL_PAYNT_FAILED" ||
                        it.type == "GLOBAL_PAYNT_EXPIRED"
                }
                .collect { msg ->
                    // If this failure/expiry matches the active payment event, signal failure
                    val current = _uiState.value.paymentEvent
                    if (current != null && current.orderId == msg.orderId) {
                        _uiState.update {
                            it.copy(
                                paymentFailed = true,
                                paymentFailureMessage = msg.message.ifBlank {
                                    if (msg.type == "PAYMENT_EXPIRED" || msg.type == "GLOBAL_PAYNT_EXPIRED") {
                                        "Payment session expired"
                                    } else {
                                        "Payment failed"
                                    }
                                },
                            )
                        }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter {
                    it.type == "ORDER_STATUS_CHANGED" ||
                        it.type == "ORDER_AMENDED" ||
                        it.type == "ORDER_REASSIGNED"
                }
                .collect { loadActiveOrders() }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "SHOP_CLOSED" || it.type == "SHOP_CLOSED_ALERT" }
                .collect { msg ->
                    msg.toShopClosedAlert()?.let { alert ->
                        _uiState.update { it.copy(shopClosedAlert = alert) }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter {
                    it.type == "SHOP_CLOSED_RESOLVED" ||
                        it.type == "SHOP_CLOSED_RESPONSE"
                }
                .collect { msg ->
                    val current = _uiState.value.shopClosedAlert
                    if (current != null && (msg.orderId.isBlank() || msg.orderId == current.orderId)) {
                        clearShopClosedAlert()
                    }
                    loadActiveOrders()
                }
        }
    }

    override fun onCleared() {
        autoUpdater.cleanup()
        retailerWebSocket.disconnect()
        super.onCleared()
    }

    private fun resolveLoadIssue(error: Exception): NavigationLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> NavigationLoadIssue.RESTRICTED
            error is IOException -> NavigationLoadIssue.OFFLINE
            else -> NavigationLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: NavigationLoadIssue): String {
        return when (issue) {
            NavigationLoadIssue.RESTRICTED -> "Order feed access is restricted for this account"
            NavigationLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            NavigationLoadIssue.ERROR -> error.message ?: "Order feed request failed"
        }
    }
}
