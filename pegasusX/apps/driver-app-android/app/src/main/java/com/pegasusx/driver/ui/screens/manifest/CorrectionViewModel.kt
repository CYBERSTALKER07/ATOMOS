package com.pegasusx.driver.ui.screens.manifest

import android.annotation.SuppressLint
import android.app.Application
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.pegasusx.driver.data.model.AmendItemPayload
import com.pegasusx.driver.data.model.AmendOrderRequest
import com.pegasusx.driver.data.model.OrderLineItem
import com.pegasusx.driver.data.model.RejectionReason
import com.pegasusx.driver.data.model.UpdateOrderDuringDeliveryRequest
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.DriverWebSocket
import com.pegasusx.driver.data.remote.DRIVER_RECONNECT_RECOVERY_HINT
import com.pegasusx.driver.data.remote.reconcileDriverSession
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
import javax.inject.Inject

// ── Per-item audit state ─────────────────────────────────────────────────────

data class LineItemAudit(
    val item: OrderLineItem,
    val acceptedQty: Int,
    val rejectedQty: Int = 0,
    val reason: RejectionReason = RejectionReason.DAMAGED,
    val isModified: Boolean = false
) {
    val acceptedTotal: Long get() = item.unitPrice * acceptedQty
}

// ── Screen-level UI state ────────────────────────────────────────────────────

data class CorrectionUiState(
    val orderId: String = "",
    val retailerName: String = "",
    val audits: List<LineItemAudit> = emptyList(),
    val isLoading: Boolean = true,
    val isSubmitting: Boolean = false,
    val error: String? = null,
    val submitSuccess: Boolean = false
) {
    val originalTotal: Long get() = audits.sumOf { it.item.lineTotal }
    val adjustedTotal: Long get() = audits.sumOf { it.acceptedTotal }
    val refundDelta: Long get() = originalTotal - adjustedTotal
    val modifiedCount: Int get() = audits.count { it.isModified }
    val hasModifications: Boolean get() = modifiedCount > 0
}

// ── ViewModel ────────────────────────────────────────────────────────────────

@HiltViewModel
class CorrectionViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val app: Application,
    private val api: DriverApi,
    private val driverWebSocket: DriverWebSocket,
) : ViewModel() {

    private val orderId: String = savedStateHandle["orderId"] ?: ""
    private val retailerName: String = savedStateHandle["retailerName"] ?: ""

    private val _state = MutableStateFlow(CorrectionUiState(orderId = orderId, retailerName = retailerName))
    val state: StateFlow<CorrectionUiState> = _state.asStateFlow()

    private val fusedClient = LocationServices.getFusedLocationProviderClient(app)

    // Which item index is open in the bottom sheet (-1 = none)
    private val _editingIndex = MutableStateFlow(-1)
    val editingIndex: StateFlow<Int> = _editingIndex.asStateFlow()

    init {
        loadOrderItems()
        viewModelScope.launch {
            driverWebSocket.onReconnect.collect {
                recoverInFlightMutation()
            }
        }
    }

    private suspend fun recoverInFlightMutation() {
        val hadInFlight = _state.value.isSubmitting
        runCatching { reconcileDriverSession(api) }
        runCatching { api.getOrder(orderId) }.onSuccess { order ->
            val audits = order.items.map { LineItemAudit(item = it, acceptedQty = it.quantity) }
            _state.update {
                it.copy(
                    retailerName = order.retailerName.ifBlank { retailerName },
                    audits = audits,
                )
            }
        }
        _state.update {
            it.copy(
                isSubmitting = false,
                error = if (hadInFlight) DRIVER_RECONNECT_RECOVERY_HINT else it.error,
            )
        }
    }

    private fun loadOrderItems() {
        viewModelScope.launch {
            _state.update { it.copy(isLoading = true, error = null) }
            try {
                val order = api.getOrder(orderId)
                val audits = order.items.map { item ->
                    LineItemAudit(item = item, acceptedQty = item.quantity)
                }
                _state.update {
                    it.copy(
                        isLoading = false,
                        retailerName = order.retailerName.ifBlank { retailerName },
                        audits = audits
                    )
                }
            } catch (e: Exception) {
                _state.update { it.copy(isLoading = false, error = e.message ?: "Failed to load manifest") }
            }
        }
    }

    fun openEditor(index: Int) {
        _editingIndex.value = index
    }

    fun closeEditor() {
        _editingIndex.value = -1
    }

    fun updateAcceptedQty(index: Int, newAccepted: Int) {
        _state.update { current ->
            val audits = current.audits.toMutableList()
            val audit = audits[index]
            val clamped = newAccepted.coerceIn(0, audit.item.quantity)
            audits[index] = audit.copy(
                acceptedQty = clamped,
                rejectedQty = audit.item.quantity - clamped,
                isModified = clamped != audit.item.quantity
            )
            current.copy(audits = audits)
        }
    }

    fun updateReason(index: Int, reason: RejectionReason) {
        _state.update { current ->
            val audits = current.audits.toMutableList()
            audits[index] = audits[index].copy(reason = reason)
            current.copy(audits = audits)
        }
    }

    @SuppressLint("MissingPermission")
    fun submitAmendment() {
        val current = _state.value
        if (!current.hasModifications) return

        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                val location = fusedClient.getCurrentLocation(
                    Priority.PRIORITY_HIGH_ACCURACY,
                    CancellationTokenSource().token,
                ).await()
                if (location != null) {
                    val edge = api.updateOrderDuringDelivery(
                        UpdateOrderDuringDeliveryRequest(
                            orderId = orderId,
                            latitude = location.latitude,
                            longitude = location.longitude,
                        )
                    )
                    if (!edge.success) {
                        _state.update {
                            it.copy(isSubmitting = false, error = edge.message.ifBlank { "In-delivery update rejected" })
                        }
                        return@launch
                    }
                }

                val payload = AmendOrderRequest(
                    orderId = orderId,
                    items = current.audits
                        .filter { it.isModified }
                        .map { audit ->
                            AmendItemPayload(
                                productId = audit.item.productId,
                                acceptedQty = audit.acceptedQty,
                                rejectedQty = audit.rejectedQty,
                                reason = audit.reason.name
                            )
                        }
                )
                val response = api.amendOrder(payload)
                if (response.success) {
                    _state.update { it.copy(isSubmitting = false, submitSuccess = true) }
                } else {
                    _state.update { it.copy(isSubmitting = false, error = response.message) }
                }
            } catch (e: Exception) {
                _state.update { it.copy(isSubmitting = false, error = e.message ?: "Amendment failed") }
            }
        }
    }

    fun submitWithoutModification() {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                api.transitionState(orderId, mapOf("state" to "COMPLETED"))
                _state.update { it.copy(isSubmitting = false, submitSuccess = true) }
            } catch (e: Exception) {
                _state.update { it.copy(isSubmitting = false, error = e.message ?: "Failed to complete delivery") }
            }
        }
    }
}
