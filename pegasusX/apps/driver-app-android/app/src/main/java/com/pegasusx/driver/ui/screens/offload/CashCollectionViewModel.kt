package com.pegasusx.driver.ui.screens.offload

import android.annotation.SuppressLint
import android.app.Application
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.pegasusx.driver.data.model.CollectCashRequest
import com.pegasusx.driver.data.model.SplitPaymentPayload
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
import kotlinx.coroutines.tasks.await
import javax.inject.Inject

/** ADR-009 cash path phases for driver UI. */
enum class CashFiscalPhase {
    COLLECT,
    FISCALIZING,
    FISCAL_FAILED,
    DONE,
}

data class CashCollectionUiState(
    val orderId: String = "",
    val amount: Long = 0,
    /** Cash actually taken (Tiyin). Editable; defaults to order total. */
    val amountReceivedMinor: Long = 0,
    val amountReceivedInput: String = "",
    val cashReceived: Boolean = false,
    val showConfirmDialog: Boolean = false,
    val isCompleting: Boolean = false,
    val completed: Boolean = false,
    val phase: CashFiscalPhase = CashFiscalPhase.COLLECT,
    val attemptId: String = "",
    val fiscalStatus: String = "",
    val splitPaymentRecorded: Boolean = false,
    val shortfallMinor: Long = 0,
    val overageMinor: Long = 0,
    val error: String? = null,
    val distanceM: Double? = null,
    val locationAvailable: Boolean = true
) {
    val varianceMinor: Long get() = amountReceivedMinor - amount
}

@HiltViewModel
class CashCollectionViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val api: DriverApi,
    private val app: Application,
    private val driverWebSocket: DriverWebSocket,
) : ViewModel() {

    private val orderId: String = savedStateHandle["orderId"] ?: ""
    private val amount: Long = savedStateHandle.get<Long>("amount") ?: 0L

    private val _state = MutableStateFlow(
        CashCollectionUiState(
            orderId = orderId,
            amount = amount,
            amountReceivedMinor = amount,
            amountReceivedInput = if (amount > 0) amount.toString() else "",
        )
    )
    val state: StateFlow<CashCollectionUiState> = _state.asStateFlow()

    fun onAmountReceivedChanged(raw: String) {
        val digits = raw.filter { it.isDigit() }.take(15)
        val parsed = digits.toLongOrNull() ?: 0L
        _state.update {
            it.copy(
                amountReceivedInput = digits,
                amountReceivedMinor = parsed,
                shortfallMinor = if (it.amount > parsed) it.amount - parsed else 0L,
                overageMinor = if (parsed > it.amount) parsed - it.amount else 0L,
                error = null,
            )
        }
    }

    private val fusedClient = LocationServices.getFusedLocationProviderClient(app)

    init {
        viewModelScope.launch {
            driverWebSocket.onReconnect.collect {
                recoverInFlightMutation()
            }
        }
        viewModelScope.launch {
            driverWebSocket.messages.collect { msg ->
                if (msg.orderId != null && msg.orderId != orderId) return@collect
                when (msg.type) {
                    "ORDER_COMPLETED", "ORDER_FINALIZED", "FISCAL_RECEIPT_SUCCEEDED" -> {
                        _state.update {
                            it.copy(
                                phase = CashFiscalPhase.DONE,
                                completed = true,
                                isCompleting = false,
                                fiscalStatus = "SUCCESS",
                                error = null,
                            )
                        }
                    }
                    "FISCAL_RECEIPT_FAILED", "ORDER_STATUS_CHANGED" -> {
                        val status = (msg.status ?: msg.state).orEmpty().uppercase()
                        if (status == "FISCAL_FAILED" || msg.type == "FISCAL_RECEIPT_FAILED") {
                            _state.update {
                                it.copy(
                                    phase = CashFiscalPhase.FISCAL_FAILED,
                                    isCompleting = false,
                                    fiscalStatus = "FAILED",
                                    error = "Fiscal receipt failed. Retry or call supervisor.",
                                )
                            }
                        } else if (status == "COMPLETED") {
                            _state.update {
                                it.copy(phase = CashFiscalPhase.DONE, completed = true, isCompleting = false)
                            }
                        } else if (status == "FISCALIZING") {
                            _state.update {
                                it.copy(phase = CashFiscalPhase.FISCALIZING, isCompleting = false)
                            }
                        }
                    }
                }
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

    fun acknowledgeCashReceived() {
        _state.update { it.copy(cashReceived = true, showConfirmDialog = true, error = null) }
    }

    fun dismissConfirmDialog() {
        _state.update { it.copy(showConfirmDialog = false) }
    }

    fun recordSplitPayment(cashMinor: Long? = null, cardMinor: Long? = null, currency: String? = null) {
        viewModelScope.launch {
            _state.update { it.copy(isCompleting = true, error = null) }
            try {
                val total = _state.value.amount
                val cash = cashMinor ?: total / 2
                val card = cardMinor ?: (total - cash)
                if (cash + card <= 0) {
                    _state.update {
                        it.copy(isCompleting = false, error = "Split amounts must be greater than zero")
                    }
                    return@launch
                }
                api.splitPayment(
                    SplitPaymentPayload(
                        orderId = orderId,
                        cashMinor = cash,
                        cardMinor = card,
                        currency = currency,
                    ),
                    DriverIdempotencyKeys.splitPayment(orderId, cash, card),
                )
                _state.update { it.copy(isCompleting = false, splitPaymentRecorded = true) }
            } catch (e: Exception) {
                _state.update {
                    it.copy(isCompleting = false, error = e.message ?: "Split payment failed")
                }
            }
        }
    }

    @SuppressLint("MissingPermission")
    fun collectCash() {
        viewModelScope.launch {
            _state.update { it.copy(isCompleting = true, error = null) }
            try {
                val cts = CancellationTokenSource()
                val location = fusedClient.getCurrentLocation(
                    Priority.PRIORITY_HIGH_ACCURACY, cts.token
                ).await()

                if (location == null) {
                    _state.update {
                        it.copy(
                            isCompleting = false,
                            locationAvailable = false,
                            error = "Unable to get GPS location. Move to an open area and try again."
                        )
                    }
                    return@launch
                }

                val received = _state.value.amountReceivedMinor
                if (received < 0L) {
                    _state.update {
                        it.copy(isCompleting = false, error = "Amount received cannot be negative.")
                    }
                    return@launch
                }
                val resp = api.collectCash(
                    request = CollectCashRequest(
                        orderId = orderId,
                        latitude = location.latitude,
                        longitude = location.longitude,
                        amountReceivedMinor = received,
                        note = when {
                            received < amount -> "shortfall"
                            received > amount -> "overage"
                            else -> null
                        },
                    ),
                    idempotencyKey = DriverIdempotencyKeys.collectCash(orderId),
                )
                val nextPhase = when (resp.state.uppercase()) {
                    "COMPLETED" -> CashFiscalPhase.DONE
                    "FISCAL_FAILED" -> CashFiscalPhase.FISCAL_FAILED
                    else -> CashFiscalPhase.FISCALIZING
                }
                _state.update {
                    it.copy(
                        isCompleting = false,
                        completed = nextPhase == CashFiscalPhase.DONE,
                        phase = nextPhase,
                        distanceM = resp.distanceM,
                        attemptId = resp.attemptId,
                        shortfallMinor = resp.shortfallMinor,
                        overageMinor = resp.overageMinor,
                        fiscalStatus = resp.fiscalStatus.ifBlank {
                            if (nextPhase == CashFiscalPhase.FISCALIZING) "PENDING" else resp.state
                        },
                        error = if (nextPhase == CashFiscalPhase.FISCAL_FAILED) {
                            "Fiscal receipt failed. Retry or call supervisor."
                        } else null,
                    )
                }
            } catch (e: Exception) {
                val msg = e.message ?: "Failed to collect cash"
                _state.update { it.copy(isCompleting = false, error = msg) }
            }
        }
    }

    fun retryFiscal() {
        viewModelScope.launch {
            _state.update { it.copy(isCompleting = true, error = null) }
            try {
                val resp = api.retryFiscal(
                    orderId = orderId,
                    idempotencyKey = DriverIdempotencyKeys.fiscalRetry(orderId),
                )
                _state.update {
                    it.copy(
                        isCompleting = false,
                        phase = CashFiscalPhase.FISCALIZING,
                        attemptId = resp.attemptId.ifBlank { it.attemptId },
                        fiscalStatus = "PENDING",
                        error = null,
                    )
                }
            } catch (e: Exception) {
                _state.update {
                    it.copy(
                        isCompleting = false,
                        phase = CashFiscalPhase.FISCAL_FAILED,
                        error = e.message ?: "Fiscal retry failed",
                    )
                }
            }
        }
    }
}
