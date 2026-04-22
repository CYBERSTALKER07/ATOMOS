package com.thelab.driver.ui.screens.offload

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.thelab.driver.BuildConfig
import com.thelab.driver.data.model.CompleteOrderRequest
import com.thelab.driver.data.remote.DriverApi
import com.thelab.driver.data.remote.DriverWebSocket
import com.thelab.driver.data.remote.TokenHolder
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class PaymentWaitingUiState(
    val orderId: String = "",
    val amount: Long = 0,
    val paymentSettled: Boolean = false,
    val isCompleting: Boolean = false,
    val completed: Boolean = false,
    val error: String? = null
)

@HiltViewModel
class PaymentWaitingViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val api: DriverApi,
    private val driverWS: DriverWebSocket
) : ViewModel() {

    private val orderId: String = savedStateHandle["orderId"] ?: ""
    private val amount: Long = savedStateHandle.get<Long>("amount") ?: 0L

    private val _state = MutableStateFlow(PaymentWaitingUiState(orderId = orderId, amount = amount))
    val state: StateFlow<PaymentWaitingUiState> = _state.asStateFlow()

    init {
        connectAndListen()
    }

    private fun connectAndListen() {
        val driverId = TokenHolder.userId ?: return
        val token = TokenHolder.token ?: return
        driverWS.connect(BuildConfig.API_BASE_URL, driverId, token)

        viewModelScope.launch {
            driverWS.messages
                .filter { it.type == "PAYMENT_SETTLED" && it.orderId == orderId }
                .collect {
                    _state.update { s -> s.copy(paymentSettled = true) }
                }
        }
    }

    fun completeOrder() {
        viewModelScope.launch {
            _state.update { it.copy(isCompleting = true, error = null) }
            try {
                api.completeOrder(CompleteOrderRequest(orderId = orderId))
                _state.update { it.copy(isCompleting = false, completed = true) }
            } catch (e: Exception) {
                _state.update { it.copy(isCompleting = false, error = e.message ?: "Failed to complete order") }
            }
        }
    }

    override fun onCleared() {
        driverWS.disconnect()
        super.onCleared()
    }
}
