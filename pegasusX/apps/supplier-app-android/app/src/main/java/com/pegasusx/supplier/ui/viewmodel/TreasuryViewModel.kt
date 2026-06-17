package com.pegasusx.supplier.ui.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.supplier.data.model.SupplierEarnings
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import java.util.UUID

data class TreasuryHubUiState(
    val earnings: SupplierEarnings? = null,
    val ledgerEntryCount: Int = 0,
    val mismatchCount: Int = 0,
    val loading: Boolean = true,
    val error: String? = null,
)

data class ChargebacksUiState(
    val orderId: String = "",
    val retailerId: String = "",
    val gateway: String = "ADYEN",
    val amount: String = "",
    val currency: String = "UZS",
    val sessionId: String = "",
    val loading: Boolean = false,
    val message: String? = null,
    val error: String? = null,
)

@HiltViewModel
class TreasuryViewModel @Inject constructor(
    private val api: SupplierApi,
    private val ops: SupplierOperationsRepository,
) : ViewModel() {
    private val _hubState = MutableStateFlow(TreasuryHubUiState())
    val hubState: StateFlow<TreasuryHubUiState> = _hubState.asStateFlow()

    private val _chargebacksState = MutableStateFlow(ChargebacksUiState())
    val chargebacksState: StateFlow<ChargebacksUiState> = _chargebacksState.asStateFlow()

    fun loadHub() {
        viewModelScope.launch {
            _hubState.update { it.copy(loading = true, error = null) }
            try {
                val earningsResp = api.getEarnings()
                val ledgerResp = ops.getPaymentLedger()
                val reconResp = ops.getPaymentReconciliationMismatches()
                _hubState.update {
                    it.copy(
                        earnings = if (earningsResp.isSuccessful) earningsResp.body() else null,
                        ledgerEntryCount = ledgerResp.body()?.items?.size ?: 0,
                        mismatchCount = reconResp.body()?.items?.size ?: 0,
                        loading = false,
                        error = if (!earningsResp.isSuccessful) "Failed to load treasury KPIs" else null,
                    )
                }
            } catch (e: Exception) {
                _hubState.update { it.copy(loading = false, error = e.message) }
            }
        }
    }

    fun updateChargebacks(block: ChargebacksUiState.() -> ChargebacksUiState) {
        _chargebacksState.update(block)
    }

    fun recordChargeback() {
        val state = _chargebacksState.value
        val amount = state.amount.toLongOrNull() ?: return
        viewModelScope.launch {
            _chargebacksState.update { it.copy(loading = true, error = null, message = null) }
            try {
                val body = buildJsonObject {
                    put("order_id", JsonPrimitive(state.orderId.trim()))
                    put("retailer_id", JsonPrimitive(state.retailerId.trim()))
                    put("gateway", JsonPrimitive(state.gateway.trim()))
                    put("amount", JsonPrimitive(amount))
                    put("currency", JsonPrimitive(state.currency.trim()))
                }
                val resp = ops.recordChargeback(body, UUID.randomUUID().toString())
                if (resp.isSuccessful) {
                    _chargebacksState.update { it.copy(message = "Chargeback recorded") }
                } else {
                    _chargebacksState.update { it.copy(error = "Failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _chargebacksState.update { it.copy(error = e.message) }
            } finally {
                _chargebacksState.update { it.copy(loading = false) }
            }
        }
    }

    fun recordReversal() {
        val sessionId = _chargebacksState.value.sessionId.trim()
        if (sessionId.isBlank()) return
        viewModelScope.launch {
            _chargebacksState.update { it.copy(loading = true, error = null, message = null) }
            try {
                val body = buildJsonObject { put("session_id", JsonPrimitive(sessionId)) }
                val resp = ops.recordChargebackReversal(body, UUID.randomUUID().toString())
                if (resp.isSuccessful) {
                    _chargebacksState.update { it.copy(message = "Reversal recorded") }
                } else {
                    _chargebacksState.update { it.copy(error = "Failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _chargebacksState.update { it.copy(error = e.message) }
            } finally {
                _chargebacksState.update { it.copy(loading = false) }
            }
        }
    }
}
