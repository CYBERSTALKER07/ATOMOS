package com.thelab.driver.ui.screens.offload

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.thelab.driver.data.model.AmendItemPayload
import com.thelab.driver.data.model.AmendOrderRequest
import com.thelab.driver.data.model.ConfirmOffloadRequest
import com.thelab.driver.data.model.ConfirmOffloadResponse
import com.thelab.driver.data.model.OrderLineItem
import com.thelab.driver.data.model.RejectionReason
import com.thelab.driver.data.remote.DriverApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class OffloadLineAudit(
    val item: OrderLineItem,
    val accepted: Int,
    val rejected: Int = 0,
    val reason: RejectionReason = RejectionReason.DAMAGED,
    val excluded: Boolean = false
) {
    val acceptedTotal: Long get() = item.unitPrice * accepted
    val isFullyRejected: Boolean get() = rejected == item.quantity
    val isPartiallyRejected: Boolean get() = rejected > 0 && rejected < item.quantity
}

data class OffloadReviewUiState(
    val orderId: String = "",
    val retailerName: String = "",
    val audits: List<OffloadLineAudit> = emptyList(),
    val isSubmitting: Boolean = false,
    val error: String? = null,
    val offloadResult: ConfirmOffloadResponse? = null
) {
    val originalTotal: Long get() = audits.sumOf { it.item.lineTotal }
    val adjustedTotal: Long get() = audits.sumOf { it.acceptedTotal }
    val hasExclusions: Boolean get() = audits.any { it.excluded }
}

@HiltViewModel
class OffloadReviewViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val api: DriverApi
) : ViewModel() {

    private val orderId: String = savedStateHandle["orderId"] ?: ""
    private val retailerName: String = savedStateHandle["retailerName"] ?: ""

    private val _state = MutableStateFlow(OffloadReviewUiState(orderId = orderId, retailerName = retailerName))
    val state: StateFlow<OffloadReviewUiState> = _state.asStateFlow()

    init {
        loadItems()
    }

    private fun loadItems() {
        viewModelScope.launch {
            try {
                val order = api.getOrder(orderId)
                val audits = order.items.map { OffloadLineAudit(item = it, accepted = it.quantity) }
                _state.update {
                    it.copy(
                        retailerName = order.retailerName.ifBlank { retailerName },
                        audits = audits
                    )
                }
            } catch (e: Exception) {
                _state.update { it.copy(error = e.message ?: "Failed to load order") }
            }
        }
    }

    fun updateRejectedQty(index: Int, delta: Int) {
        _state.update { current ->
            val audits = current.audits.toMutableList()
            val audit = audits[index]
            val newRejected = (audit.rejected + delta).coerceIn(0, audit.item.quantity)
            val newAccepted = audit.item.quantity - newRejected
            audits[index] = audit.copy(
                rejected = newRejected,
                accepted = newAccepted,
                excluded = newRejected == audit.item.quantity
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

    fun confirmOffload() {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                // If items were excluded, amend first
                val current = _state.value
                if (current.hasExclusions) {
                    val amendPayload = AmendOrderRequest(
                        orderId = orderId,
                        items = current.audits.filter { it.rejected > 0 }.map { audit ->
                            AmendItemPayload(
                                productId = audit.item.productId,
                                acceptedQty = audit.accepted,
                                rejectedQty = audit.rejected,
                                reason = audit.reason.name
                            )
                        }
                    )
                    val amendResp = api.amendOrder(amendPayload)
                    if (!amendResp.success) {
                        _state.update { it.copy(isSubmitting = false, error = amendResp.message) }
                        return@launch
                    }
                }

                // Now confirm offload
                val response = api.confirmOffload(ConfirmOffloadRequest(orderId = orderId))
                _state.update { it.copy(isSubmitting = false, offloadResult = response) }
            } catch (e: Exception) {
                _state.update { it.copy(isSubmitting = false, error = e.message ?: "Offload failed") }
            }
        }
    }
}
