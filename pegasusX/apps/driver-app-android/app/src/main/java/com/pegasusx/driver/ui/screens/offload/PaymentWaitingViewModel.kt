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
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
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

    private val _cashCollectionRequired = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val cashCollectionRequired: SharedFlow<Unit> = _cashCollectionRequired.asSharedFlow()

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
            driverWS.messages.collect { msg ->
                if (msg.orderId != null && msg.orderId != orderId) return@collect
                when (msg.type) {
                    "ORDER_COMPLETED", "ORDER_FINALIZED" -> {
                        _state.update { it.copy(paymentSettled = true, completed = true) }
                    }
                    "PAYMENT_SETTLED", "PAYMENT_CLEARED" -> {
                        _state.update { it.copy(paymentSettled = true) }
                        finalizeDelivery(autoTriggered = true)
                    }
                    "ORDER_STATUS_CHANGED", "PAYMENT_REQUIRED" -> {
                        val status = (msg.status ?: msg.state).orEmpty()
                        if (status.equals("PENDING_CASH_COLLECTION", ignoreCase = true)) {
                            _cashCollectionRequired.emit(Unit)
                        }
                    }
                }
            }
        }
    }

    private fun finalizeDelivery(autoTriggered: Boolean = false) {
        if (_state.value.completed || _state.value.isCompleting) return
        viewModelScope.launch {
            _state.update { it.copy(isCompleting = true, error = null) }
            try {
                api.completeOrder(
                    request = CompleteOrderRequest(orderId = orderId),
                    idempotencyKey = DriverIdempotencyKeys.complete(orderId),
                )
                _state.update { it.copy(isCompleting = false, completed = true) }
            } catch (e: Exception) {
                val message = e.message.orEmpty()
                val alreadyDone = message.contains("COMPLETED", ignoreCase = true) ||
                    message.contains("invalid_status", ignoreCase = true)
                if (alreadyDone) {
                    _state.update { it.copy(isCompleting = false, completed = true) }
                } else {
                    _state.update {
                        it.copy(
                            isCompleting = false,
                            error = if (autoTriggered) {
                                message.ifBlank { "Payment received but delivery could not be finalized. Tap retry." }
                            } else {
                                message.ifBlank { "Failed to complete order" }
                            },
                        )
                    }
                }
            }
        }
    }

    fun completeOrder() {
        finalizeDelivery(autoTriggered = false)
    }

    override fun onCleared() {
        super.onCleared()
    }
}
