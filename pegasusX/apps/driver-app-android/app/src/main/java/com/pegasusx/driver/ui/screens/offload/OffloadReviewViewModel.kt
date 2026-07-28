package com.pegasusx.driver.ui.screens.offload

import android.net.Uri
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.driver.data.model.ConfirmOffloadRequest
import com.pegasusx.driver.data.model.ConfirmOffloadResponse
import com.pegasusx.driver.data.model.MissingItemRequest
import com.pegasusx.driver.data.model.MissingItemsPayload
import com.pegasusx.driver.data.model.OrderLineItem
import com.pegasusx.driver.data.model.RejectionReason
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.DriverWebSocket
import com.pegasusx.driver.data.remote.DRIVER_RECONNECT_RECOVERY_HINT
import com.pegasusx.driver.data.remote.MediaUploadService
import com.pegasusx.driver.data.remote.reconcileDriverSession
import com.pegasusx.driver.util.DriverIdempotencyKeys
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
    val customReason: String = "",
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
    val offloadResult: ConfirmOffloadResponse? = null,
    val creditDeliveryRecorded: Boolean = false,
    val evidencePhotoUrl: String = "",
    val isUploadingPhoto: Boolean = false,
    val photoPreviewUri: String? = null,
) {
    val originalTotal: Long get() = audits.sumOf { it.item.lineTotal }
    val adjustedTotal: Long get() = audits.sumOf { it.acceptedTotal }
    val hasExclusions: Boolean get() = audits.any { it.excluded }
    val hasRejections: Boolean get() = audits.any { it.rejected > 0 }
    val needsPhotoProof: Boolean get() = audits.any {
        it.rejected > 0 && (it.reason == RejectionReason.DAMAGED || it.reason == RejectionReason.WRONG_ITEM)
    }
}

@HiltViewModel
class OffloadReviewViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val api: DriverApi,
    private val driverWebSocket: DriverWebSocket,
    private val mediaUpload: MediaUploadService,
) : ViewModel() {

    private val orderId: String = savedStateHandle["orderId"] ?: ""
    private val retailerName: String = savedStateHandle["retailerName"] ?: ""

    private val _state = MutableStateFlow(OffloadReviewUiState(orderId = orderId, retailerName = retailerName))
    val state: StateFlow<OffloadReviewUiState> = _state.asStateFlow()

    init {
        loadItems()
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
            val audits = order.items.map { OffloadLineAudit(item = it, accepted = it.quantity) }
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
            audits[index] = audits[index].copy(
                reason = reason,
                customReason = if (reason == RejectionReason.OTHER) audits[index].customReason else "",
            )
            current.copy(audits = audits)
        }
    }

    fun updateCustomReason(index: Int, value: String) {
        _state.update { current ->
            val audits = current.audits.toMutableList()
            audits[index] = audits[index].copy(customReason = value)
            current.copy(audits = audits)
        }
    }

    fun markCreditDelivery(photoProofUrl: String? = null) {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                val body = buildMap {
                    put("order_id", orderId)
                    photoProofUrl?.takeIf { it.isNotBlank() }?.let { put("photo_proof_url", it) }
                }
                api.markCreditDelivery(body, DriverIdempotencyKeys.creditDelivery(orderId))
                _state.update { it.copy(isSubmitting = false, creditDeliveryRecorded = true) }
            } catch (e: Exception) {
                _state.update {
                    it.copy(isSubmitting = false, error = e.message ?: "Credit delivery failed")
                }
            }
        }
    }

    fun uploadEvidencePhoto(uri: Uri) {
        viewModelScope.launch {
            _state.update {
                it.copy(isUploadingPhoto = true, error = null, photoPreviewUri = uri.toString())
            }
            try {
                val publicUrl = mediaUpload.uploadJpegUri(
                    uri = uri,
                    purpose = "driver_exception",
                    orderId = orderId,
                )
                _state.update {
                    it.copy(isUploadingPhoto = false, evidencePhotoUrl = publicUrl)
                }
            } catch (e: Exception) {
                _state.update {
                    it.copy(
                        isUploadingPhoto = false,
                        evidencePhotoUrl = "",
                        error = e.message ?: "Photo upload failed",
                    )
                }
            }
        }
    }

    fun confirmOffload() {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                val current = _state.value
                if (current.hasRejections) {
                    val rejectedAudits = current.audits.filter { it.rejected > 0 }
                    val missingOtherReason = rejectedAudits.firstOrNull {
                        it.reason == RejectionReason.OTHER && it.customReason.isBlank()
                    }
                    if (missingOtherReason != null) {
                        _state.update {
                            it.copy(
                                isSubmitting = false,
                                error = "Describe the issue for ${missingOtherReason.item.productName}",
                            )
                        }
                        return@launch
                    }
                    if (current.needsPhotoProof && current.evidencePhotoUrl.isBlank()) {
                        _state.update {
                            it.copy(
                                isSubmitting = false,
                                error = "Photo required for damaged or wrong-item rejections.",
                            )
                        }
                        return@launch
                    }
                    // Route OS&D through exception-report so claims + photo_url are enforced.
                    val missingItems = rejectedAudits.map { audit ->
                        val needsLinePhoto = audit.reason == RejectionReason.DAMAGED ||
                            audit.reason == RejectionReason.WRONG_ITEM
                        MissingItemRequest(
                            skuId = audit.item.productId,
                            missingQty = audit.rejected,
                            reason = audit.reason.name,
                            photoUrl = if (needsLinePhoto) current.evidencePhotoUrl else null,
                        )
                    }
                    api.reportMissingItems(
                        body = MissingItemsPayload(
                            orderId = orderId,
                            missingItems = missingItems,
                            photoUrl = current.evidencePhotoUrl.takeIf { it.isNotBlank() },
                            note = rejectedAudits
                                .mapNotNull { it.customReason.takeIf { r -> r.isNotBlank() } }
                                .joinToString("; ")
                                .ifBlank { null },
                        ),
                        idempotencyKey = DriverIdempotencyKeys.missingItems(orderId),
                    )
                }

                val response = api.confirmOffload(
                    request = ConfirmOffloadRequest(orderId = orderId),
                    idempotencyKey = DriverIdempotencyKeys.offload(orderId),
                )
                _state.update { it.copy(isSubmitting = false, offloadResult = response) }
            } catch (e: Exception) {
                _state.update { it.copy(isSubmitting = false, error = e.message ?: "Offload failed") }
            }
        }
    }
}
