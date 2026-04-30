package com.thelab.driver.ui.screens.offload

import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.thelab.driver.data.remote.DriverApi
import com.thelab.driver.data.remote.DriverWebSocket
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
    val error: String? = null
)

@HiltViewModel
class ShopClosedWaitingViewModel @Inject constructor(
    private val api: DriverApi,
    private val webSocket: DriverWebSocket
) : ViewModel() {

    private val _state = MutableStateFlow(ShopClosedUiState())
    val state: StateFlow<ShopClosedUiState> = _state.asStateFlow()

    init {
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
    }

    fun reportShopClosed(orderId: String) {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                api.reportShopClosed(mapOf("order_id" to orderId))
                _state.update { it.copy(isSubmitting = false) }
            } catch (e: Exception) {
                Log.e("ShopClosed", "Failed to report: ${e.message}")
                _state.update { it.copy(isSubmitting = false, error = e.message) }
            }
        }
    }

    fun submitBypass(orderId: String, token: String) {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                api.bypassOffload(mapOf("order_id" to orderId, "bypass_token" to token))
                _state.update { it.copy(isSubmitting = false, bypassConfirmed = true) }
            } catch (e: Exception) {
                Log.e("ShopClosed", "Bypass failed: ${e.message}")
                _state.update { it.copy(isSubmitting = false, error = e.message) }
            }
        }
    }
}
