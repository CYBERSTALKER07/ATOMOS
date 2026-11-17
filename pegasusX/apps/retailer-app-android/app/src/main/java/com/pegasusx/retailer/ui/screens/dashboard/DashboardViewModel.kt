package com.pegasusx.retailer.ui.screens.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.BuildConfig
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.RetailerPreferencesStore
import com.pegasusx.retailer.data.local.RetailerRealtimeHub
import com.pegasusx.retailer.data.local.RetailerSessionContext
import com.pegasusx.retailer.data.local.RetailerWorkspaceMode
import com.pegasusx.retailer.data.model.ControlTowerPulse
import com.pegasusx.retailer.data.model.FastHomePendingProposal
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.PulseEvent
import com.pegasusx.retailer.data.model.RetailerAIPrediction
import com.pegasusx.retailer.data.model.RetailerFastHomeResponse
import com.pegasusx.retailer.ui.design.MarketPack
import com.pegasusx.retailer.ui.design.MarketPackBinder
import com.pegasusx.retailer.ui.design.PulseHonesty
import com.pegasusx.retailer.util.CutoffCountdownState
import com.pegasusx.retailer.util.FastHomeContractHelper
import com.pegasusx.retailer.util.RetailerIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import retrofit2.HttpException
import javax.inject.Inject

enum class DashboardLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class DashboardUiState(
    val isLoading: Boolean = false,
    val activeOrders: List<Order> = emptyList(),
    val predictions: List<RetailerAIPrediction> = emptyList(),
    val recentProducts: List<Product> = emptyList(),
    val pulseEvents: List<PulseEvent> = emptyList(),
    val pulseLoading: Boolean = false,
    val pulseError: String? = null,
    val commandPulse: ControlTowerPulse? = null,
    val commandPulseError: String? = null,
    val error: String? = null,
    val loadIssue: DashboardLoadIssue? = null,
    val orderActionPending: Boolean = false,
    val pack: MarketPack? = null,
    val workspaceMode: RetailerWorkspaceMode = RetailerWorkspaceMode.FAST,
    val fastHome: RetailerFastHomeResponse? = null,
    val serverClockOffsetMs: Long = 0L,
    val countdown: CutoffCountdownState = CutoffCountdownState(),
    val actionFeedback: String? = null,
    val retailerRole: String = "",
    val isOwner: Boolean = false,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            DashboardLoadIssue.RESTRICTED -> "Dashboard access is restricted for this account"
            DashboardLoadIssue.OFFLINE -> "Offline mode active. Showing latest dashboard data"
            DashboardLoadIssue.ERROR -> "Dashboard sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class DashboardViewModel @Inject constructor(
    private val api: PegasusApi,
    private val sessionContext: RetailerSessionContext,
    private val preferencesStore: RetailerPreferencesStore,
    private val realtimeHub: RetailerRealtimeHub,
) : ViewModel() {

    private val _uiState = MutableStateFlow(
        DashboardUiState(
            workspaceMode = preferencesStore.getWorkspaceMode() ?: RetailerWorkspaceMode.FAST,
        ),
    )
    val uiState: StateFlow<DashboardUiState> = _uiState.asStateFlow()

    private val retailerId: String get() = sessionContext.getUserId() ?: ""

    init {
        refresh()
        realtimeHub.connect()
        viewModelScope.launch {
            preferencesStore.workspaceModeFlow.collect { stored ->
                if (stored != null) {
                    _uiState.update { it.copy(workspaceMode = stored) }
                }
            }
        }
        viewModelScope.launch {
            realtimeHub.promotionChangedEvents.collect { refresh() }
        }
        viewModelScope.launch {
            realtimeHub.reconnectEvents.collect {
                if (_uiState.value.orderActionPending) {
                    _uiState.update {
                        it.copy(
                            orderActionPending = false,
                            error = ORDER_RECONNECT_RECOVERY_HINT,
                            loadIssue = null,
                        )
                    }
                }
                refresh()
            }
        }
    }

    fun switchWorkspaceMode(mode: RetailerWorkspaceMode) {
        preferencesStore.setWorkspaceMode(mode)
        _uiState.update { it.copy(workspaceMode = mode) }
    }

    fun tickCountdown(nowMs: Long = System.currentTimeMillis()) {
        val state = _uiState.value
        val next = FastHomeContractHelper.computeCutoffCountdown(
            cutoffAtIso = state.fastHome?.cutoffAt,
            serverClockOffsetMs = state.serverClockOffsetMs,
            clientNowMs = nowMs,
        )
        if (next != state.countdown) {
            _uiState.update { it.copy(countdown = next) }
        }
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, pulseLoading = true, pulseError = null, error = null) }
            val token = sessionContext.getPreferredToken().orEmpty()
            val pack = if (token.isNotBlank()) {
                kotlinx.coroutines.withContext(kotlinx.coroutines.Dispatchers.IO) {
                    MarketPackBinder.fetch(BuildConfig.BASE_URL, token)?.pack
                }
            } else {
                null
            }
            _uiState.update { it.copy(pack = pack) }

            var nextIssue: DashboardLoadIssue? = null
            var nextError: String? = null
            var allOrders: List<Order> = emptyList()
            var forecasts: List<RetailerAIPrediction> = emptyList()

            try {
                val me = api.getRetailerMe().jsonObject
                val role = me["retailer_role"]?.jsonPrimitive?.contentOrNull?.trim()?.uppercase().orEmpty()
                val ownerFlag = me["is_owner"]?.jsonPrimitive?.booleanOrNull == true || role == "OWNER"
                val effectiveMode = FastHomeContractHelper.resolveEffectiveWorkspaceMode(
                    persistedMode = preferencesStore.getWorkspaceMode(),
                    retailerRole = role,
                    isOwner = ownerFlag,
                )
                _uiState.update {
                    it.copy(
                        retailerRole = role,
                        isOwner = ownerFlag,
                        workspaceMode = effectiveMode,
                    )
                }
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                // Keep cached or default workspace mode when /v1/retailer/me is unavailable.
            }

            try {
                allOrders = api.getOrders(retailerId)
                val active = allOrders.filter { it.status.isActive }
                _uiState.update { it.copy(activeOrders = active) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                if (nextIssue == null) {
                    nextIssue = resolveLoadIssue(e)
                    nextError = resolveErrorMessage(e, nextIssue)
                }
            }

            try {
                forecasts = api.getRetailerAIPredictions().items
                _uiState.update { it.copy(predictions = forecasts) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                if (nextIssue == null) {
                    nextIssue = resolveLoadIssue(e)
                    nextError = resolveErrorMessage(e, nextIssue)
                }
            }

            try {
                val products = api.getCatalogProducts(retailerId = retailerId.takeIf { it.isNotBlank() })
                _uiState.update { it.copy(recentProducts = products.take(6)) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                if (nextIssue == null) {
                    nextIssue = resolveLoadIssue(e)
                    nextError = resolveErrorMessage(e, nextIssue)
                }
            }

            try {
                val pulse = api.getRetailerPulse()
                _uiState.update { it.copy(pulseEvents = pulse.events, pulseLoading = false, pulseError = null) }
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                _uiState.update { it.copy(pulseLoading = false, pulseError = PulseHonesty.FAILED) }
            }

            var latestCommandPulse: ControlTowerPulse? = _uiState.value.commandPulse
            try {
                val command = api.getControlTowerPulse()
                val applied = PulseHonesty.applyObject(
                    ok = true,
                    incoming = command,
                    previous = _uiState.value.commandPulse,
                )
                latestCommandPulse = applied.value
                _uiState.update {
                    it.copy(commandPulse = applied.value, commandPulseError = applied.error)
                }
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                val applied = PulseHonesty.applyObject(
                    ok = false,
                    incoming = null,
                    previous = _uiState.value.commandPulse,
                )
                latestCommandPulse = applied.value
                _uiState.update {
                    it.copy(commandPulse = applied.value, commandPulseError = applied.error)
                }
            }

            val bffFastHome = try {
                api.getRetailerFastHome()
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                null
            }

            val resolvedFastHome = bffFastHome ?: run {
                val creditProfile = try {
                    api.getCreditProfile()
                } catch (e: CancellationException) {
                    throw e
                } catch (_: Exception) {
                    null
                }
                val relationshipsJson = try {
                    api.getCreditRelationships()
                } catch (e: CancellationException) {
                    throw e
                } catch (_: Exception) {
                    null
                }
                FastHomeContractHelper.buildFastHomeFallback(
                    orders = allOrders,
                    predictions = forecasts,
                    creditProfile = creditProfile,
                    creditRelationshipsJson = relationshipsJson,
                    pulse = latestCommandPulse,
                    locale = preferencesStore.getLanguage(),
                )
            }

            val offsetMs = FastHomeContractHelper.computeServerClockOffsetMs(resolvedFastHome.serverNow)
            val countdown = FastHomeContractHelper.computeCutoffCountdown(
                cutoffAtIso = resolvedFastHome.cutoffAt,
                serverClockOffsetMs = offsetMs,
            )

            _uiState.update {
                it.copy(
                    isLoading = false,
                    loadIssue = nextIssue,
                    error = nextError,
                    fastHome = resolvedFastHome,
                    serverClockOffsetMs = offsetMs,
                    countdown = countdown,
                )
            }
        }
    }

    fun confirmFastProposal(proposal: FastHomePendingProposal) {
        if (proposal.source.equals("AI_PREDICTED", ignoreCase = true)) {
            confirmAiOrder(proposal.orderId)
        } else {
            confirmPreorder(proposal.orderId)
        }
        _uiState.update { it.copy(actionFeedback = null) }
    }

    fun holdFastProposal(proposal: FastHomePendingProposal, reason: String = "Held by retailer") {
        if (proposal.source.equals("AI_PREDICTED", ignoreCase = true)) {
            rejectAiOrder(proposal.orderId, reason)
            return
        }
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null, actionFeedback = null) }
            try {
                api.rejectPreorder(
                    body = mapOf("order_id" to proposal.orderId, "reason" to reason),
                    idempotencyKey = RetailerIdempotencyKeys.rejectPreorder(proposal.orderId, reason),
                )
                refresh()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                _uiState.update { it.copy(orderActionPending = false) }
            }
        }
    }

    fun editFastProposal(proposal: FastHomePendingProposal, updatedQuantities: Map<String, Int>) {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null, actionFeedback = null) }
            try {
                val lineItems = proposal.items.mapNotNull { item ->
                    val key = item.sku.ifBlank { item.productId }
                    val qty = (updatedQuantities[key] ?: item.quantity).coerceAtLeast(0)
                    if (qty <= 0) return@mapNotNull null
                    mapOf(
                        "sku" to key,
                        "name" to item.productName.ifBlank { key },
                        "quantity" to qty,
                        "unit_price_minor" to item.unitPriceMinor,
                    )
                }
                api.editPreorder(
                    body = mapOf(
                        "order_id" to proposal.orderId,
                        "line_items" to lineItems,
                        "requested_delivery_date" to (proposal.proposedDeliveryDate ?: ""),
                    ),
                    idempotencyKey = RetailerIdempotencyKeys.editPreorder(proposal.orderId),
                )
                refresh()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                _uiState.update { it.copy(orderActionPending = false) }
            }
        }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null, loadIssue = null) }
    }

    fun confirmAiOrder(orderId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                api.confirmAiOrder(
                    body = mapOf("order_id" to orderId),
                    idempotencyKey = RetailerIdempotencyKeys.confirmAI(orderId),
                )
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                _uiState.update { it.copy(orderActionPending = false) }
            }
        }
    }

    fun rejectAiOrder(orderId: String, reason: String = "Retailer rejected") {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                api.rejectAiOrder(
                    body = mapOf("order_id" to orderId, "reason" to reason),
                    idempotencyKey = RetailerIdempotencyKeys.rejectAI(orderId, reason),
                )
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                _uiState.update { it.copy(orderActionPending = false) }
            }
        }
    }

    fun confirmPreorder(orderId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                api.confirmPreorder(
                    body = mapOf("order_id" to orderId),
                    idempotencyKey = "retailer-confirm-preorder:$orderId",
                )
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                _uiState.update { it.copy(orderActionPending = false) }
            }
        }
    }

    fun editPreorder(order: Order, requestedDeliveryDate: String? = null) {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                val deliveryDate = requestedDeliveryDate
                    ?: order.deliverBefore
                    ?: order.autoConfirmAt
                    ?: ""
                val lineItems = order.items.map { item ->
                    mapOf(
                        "sku" to item.productId.ifBlank { item.id },
                        "name" to item.productName,
                        "quantity" to item.quantity,
                        "unit_price_minor" to item.unitPrice.toLong(),
                    )
                }
                api.editPreorder(
                    body = mapOf(
                        "order_id" to order.id,
                        "line_items" to lineItems,
                        "requested_delivery_date" to deliveryDate,
                    ),
                    idempotencyKey = RetailerIdempotencyKeys.editPreorder(order.id),
                )
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                _uiState.update { it.copy(orderActionPending = false) }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): DashboardLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> DashboardLoadIssue.RESTRICTED
            error is IOException -> DashboardLoadIssue.OFFLINE
            else -> DashboardLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: DashboardLoadIssue): String {
        return when (issue) {
            DashboardLoadIssue.RESTRICTED -> "Dashboard access is restricted for this account"
            DashboardLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            DashboardLoadIssue.ERROR -> error.message ?: "Dashboard request failed"
        }
    }

    private companion object {
        const val ORDER_RECONNECT_RECOVERY_HINT =
            "Connection restored — verify order status before retrying."
    }
}
