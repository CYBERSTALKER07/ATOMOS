package com.pegasusx.driver.ui.screens.offload

import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.DriverWebSocket
import com.pegasusx.driver.data.remote.DRIVER_RECONNECT_RECOVERY_HINT
import com.pegasusx.driver.data.remote.reconcileDriverSession
import com.pegasusx.driver.offline.DriverOfflineActionCatalog
import com.pegasusx.driver.offline.DriverOfflineQueue
import com.pegasusx.driver.util.DriverIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class ShopClosedUiState(
    val isSubmitting: Boolean = false,
    val retailerResponse: String? = null,
    val escalated: Boolean = false,
    val bypassToken: String? = null,
    val showBypassInput: Boolean = false,
    val bypassConfirmed: Boolean = false,
    val queuedOffline: Boolean = false,
    val error: String? = null
)

@HiltViewModel
class ShopClosedWaitingViewModel @Inject constructor(
    private val api: DriverApi,
    private val webSocket: DriverWebSocket,
    private val offlineQueue: DriverOfflineQueue,
) : ViewModel() {

    private val _state = MutableStateFlow(ShopClosedUiState())
    val state: StateFlow<ShopClosedUiState> = _state.asStateFlow()

    init {
        viewModelScope.launch {
            webSocket.outdatedState.collect { outdated ->
                if (outdated == null) return@collect
                _state.update { it.copy(error = outdated.message) }
            }
        }

        // Listen for WebSocket messages related to shop-closed flow
        viewModelScope.launch {
            webSocket.messages.collect { msg ->
                when (msg.type) {
                    "SHOP_CLOSED_RESPONSE" -> {
                        _state.update {
                            it.copy(
                                retailerResponse = msg.response,
                                showBypassInput = msg.response == "CLOSED_TODAY"
                            )
                        }
                    }
                    "BYPASS_TOKEN_ISSUED" -> {
                        _state.update {
                            it.copy(
                                bypassToken = msg.bypassToken,
                                showBypassInput = true,
                                escalated = true
                            )
                        }
                    }
                    "SHOP_CLOSED_ESCALATED" -> {
                        _state.update { it.copy(escalated = true) }
                    }
                }
            }
        }

        viewModelScope.launch {
            webSocket.onReconnect.collect {
                recoverInFlightMutation()
            }
        }
    }

    private suspend fun recoverInFlightMutation() {
        val hadInFlight = _state.value.isSubmitting
        runCatching { reconcileDriverSession(api) }
        _state.update {
            it.copy(
                isSubmitting = false,
                error = if (hadInFlight) DRIVER_RECONNECT_RECOVERY_HINT else it.error,
            )
        }
    }

    fun reportShopClosed(
        orderId: String,
        reason: String = "CLOSED",
        latitude: Double? = null,
        longitude: Double? = null,
        photoUrl: String? = null,
    ) {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            val ts = offlineQueue.nowIso()
            val body = buildMap {
                put("order_id", orderId)
                put("reason", reason)
                put("client_timestamp", ts)
                latitude?.let { put("latitude", it.toString()) }
                longitude?.let { put("longitude", it.toString()) }
                photoUrl?.takeIf { it.isNotBlank() }?.let { put("photo_url", it) }
            }
            val key = DriverIdempotencyKeys.reportShopClosed(orderId)
            try {
                api.reportShopClosed(body, key)
                _state.update { it.copy(isSubmitting = false) }
            } catch (e: Exception) {
                Log.e("ShopClosed", "Failed to report: ${e.message}")
                if (DriverOfflineActionCatalog.isNetworkEnqueueable(e)) {
                    offlineQueue.enqueueMap(
                        endpoint = DriverOfflineActionCatalog.ENDPOINT_SHOP_CLOSED,
                        body = body,
                        idempotencyKey = key,
                        orderId = orderId,
                        clientTimestampIso = ts,
                    )
                    _state.update {
                        it.copy(
                            isSubmitting = false,
                            queuedOffline = true,
                            error = "Offline — shop-closed queued for sync",
                        )
                    }
                } else {
                    _state.update { it.copy(isSubmitting = false, error = e.message) }
                }
            }
        }
    }

    fun submitBypass(orderId: String, token: String) {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                api.bypassOffload(
                    mapOf("order_id" to orderId, "bypass_token" to token),
                    DriverIdempotencyKeys.bypassOffload(orderId),
                )
                _state.update { it.copy(isSubmitting = false, bypassConfirmed = true) }
            } catch (e: Exception) {
                Log.e("ShopClosed", "Bypass failed: ${e.message}")
                _state.update { it.copy(isSubmitting = false, error = e.message) }
            }
        }
    }
}
