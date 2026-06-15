package com.pegasusx.driver.ui.screens.offload

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.driver.data.model.CompleteOrderRequest
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.DriverWebSocket
import com.pegasusx.driver.data.remote.DRIVER_RECONNECT_RECOVERY_HINT
import com.pegasusx.driver.data.remote.reconcileDriverSession
import com.pegasusx.driver.util.DriverIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
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
        viewModelScope.launch {
            driverWS.onReconnect.collect {
                recoverInFlightMutation()
            }
        }
    }

    private suspend fun recoverInFlightMutation() {
        val hadInFlight = _state.value.isCompleting
        runCatching { reconcileDriverSession(api) }
        _state.update {
            it.copy(
                isCompleting = false,
                error = if (hadInFlight) DRIVER_RECONNECT_RECOVERY_HINT else it.error,
            )
        }
    }

    private fun connectAndListen() {
        viewModelScope.launch {
            driverWS.outdatedState.collect { outdated ->
                if (outdated == null) return@collect
                _state.update { it.copy(error = outdated.message) }
            }
        }

        viewModelScope.launch {
            driverWS.messages
                .collect { msg ->
                    when {
                        msg.type == "PAYMENT_SETTLED" && msg.orderId == orderId -> {
                            _state.update { s -> s.copy(paymentSettled = true) }
                        }
                    }
                }
        }
    }

    fun completeOrder() {
        viewModelScope.launch {
            _state.update { it.copy(isCompleting = true, error = null) }
            try {
                api.completeOrder(
                    request = CompleteOrderRequest(orderId = orderId),
                    idempotencyKey = DriverIdempotencyKeys.complete(orderId),
                )
                _state.update { it.copy(isCompleting = false, completed = true) }
            } catch (e: Exception) {
                _state.update { it.copy(isCompleting = false, error = e.message ?: "Failed to complete order") }
            }
        }
    }

    override fun onCleared() {
        super.onCleared()
    }
}
