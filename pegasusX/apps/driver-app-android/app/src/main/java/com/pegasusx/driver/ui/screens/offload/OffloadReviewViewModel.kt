package com.pegasusx.driver.ui.screens.offload

import android.net.Uri
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.driver.data.model.ConfirmOffloadRequest
import com.pegasusx.driver.data.model.ConfirmOffloadResponse
import com.pegasusx.driver.data.model.DeliveryScanQRRequest
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
import java.net.URLDecoder
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
    /** Settlement proximity: payment modes locked until unlock. */
    val proximityUnlocked: Boolean = false,
    val proximityMethod: String? = null,
    val partialOffloadRecorded: Boolean = false,
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
    private val scannedToken: String = (savedStateHandle.get<String>("scannedToken") ?: "")
        .let { raw ->
            runCatching { URLDecoder.decode(raw, "UTF-8") }.getOrDefault(raw)
        }

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

    fun markCreditDelivery(photoProofUrl: String? = null, forceBypassToken: String? = null) {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                if (!_state.value.proximityUnlocked && forceBypassToken.isNullOrBlank()) {
                    _state.update {
                        it.copy(
                            isSubmitting = false,
                            error = "Proximity locked — unlock settlement at the stop before credit leave.",
                        )
                    }
                    return@launch
                }
                val body = buildMap {
                    put("order_id", orderId)
                    photoProofUrl?.takeIf { it.isNotBlank() }?.let { put("photo_proof_url", it) }
                    forceBypassToken?.takeIf { it.isNotBlank() }?.let { put("force_bypass_token", it) }
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

    /** Unlock cash/credit when GPS is at the stop (≤100 m or H3). */
    fun unlockProximity(latitude: Double, longitude: Double) {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                val resp = api.proximityUnlock(
                    mapOf(
                        "order_id" to orderId,
                        "latitude" to latitude,
                        "longitude" to longitude,
                    ),
                    DriverIdempotencyKeys.proximityUnlock(orderId),
                )
                val unlocked = (resp["proximity_unlocked"] as? Boolean) == true
                    || resp["proximity_unlocked"]?.toString() == "true"
                val method = resp["proximity_method"]?.toString()
                _state.update {
                    it.copy(
                        isSubmitting = false,
                        proximityUnlocked = unlocked,
                        proximityMethod = method,
                        error = if (unlocked) null else (resp["message"]?.toString() ?: "Proximity still locked"),
                    )
                }
            } catch (e: Exception) {
                _state.update {
                    it.copy(isSubmitting = false, error = e.message ?: "Proximity unlock failed")
                }
            }
        }
    }

    /** Submit line-level partial offload (delivered + remaining = qty). */
    fun submitPartialOffload() {
        viewModelScope.launch {
            _state.update { it.copy(isSubmitting = true, error = null) }
            try {
                val current = _state.value
                val lines = current.audits.map { audit ->
                    mapOf(
                        "sku" to audit.item.productId,
                        "delivered_qty" to audit.accepted.toLong(),
                        "remaining_qty" to audit.rejected.toLong(),
                        "reason" to when (audit.reason) {
                            RejectionReason.DAMAGED -> "DAMAGED"
                            RejectionReason.MISSING -> "MISSING"
                            RejectionReason.WRONG_ITEM -> "OTHER"
                            RejectionReason.OTHER -> "OTHER"
                        },
                    )
                }
                val fingerprint = lines.joinToString("|") {
                    "${it["sku"]}:${it["delivered_qty"]}:${it["remaining_qty"]}"
                }
                api.partialOffload(
                    mapOf(
                        "order_id" to orderId,
                        "lines" to lines,
                    ),
                    DriverIdempotencyKeys.partialOffload(orderId, fingerprint),
                )
                _state.update {
                    it.copy(isSubmitting = false, partialOffloadRecorded = true)
                }
            } catch (e: Exception) {
                _state.update {
                    it.copy(isSubmitting = false, error = e.message ?: "Partial offload failed")
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
                    // Prefer partial-offload contract (qty math + return path); keep exception-report for photo OS&D.
                    val lines = current.audits.map { audit ->
                        mapOf(
                            "sku" to audit.item.productId,
                            "delivered_qty" to audit.accepted.toLong(),
                            "remaining_qty" to audit.rejected.toLong(),
                            "reason" to when (audit.reason) {
                                RejectionReason.DAMAGED -> "DAMAGED"
                                RejectionReason.MISSING -> "MISSING"
                                RejectionReason.WRONG_ITEM -> "OTHER"
                                RejectionReason.OTHER -> "OTHER"
                            },
                        )
                    }
                    val fingerprint = lines.joinToString("|") {
                        "${it["sku"]}:${it["delivered_qty"]}:${it["remaining_qty"]}"
                    }
                    api.partialOffload(
                        mapOf("order_id" to orderId, "lines" to lines),
                        DriverIdempotencyKeys.partialOffload(orderId, fingerprint),
                    )
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
                    if (current.needsPhotoProof) {
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
                }

                // Canonical ARRIVED → AWAITING_PAYMENT via scan-qr (validate-qr already done).
                val response = if (scannedToken.isNotBlank()) {
                    val scan = api.scanDeliveryQR(
                        request = DeliveryScanQRRequest(orderId = orderId, qrToken = scannedToken),
                        idempotencyKey = DriverIdempotencyKeys.offload(orderId),
                    )
                    ConfirmOffloadResponse(
                        orderId = scan.orderId.ifBlank { orderId },
                        state = scan.state,
                        paymentMethod = "",
                        amount = current.adjustedTotal,
                        message = "Collect payment",
                    )
                } else {
                    api.confirmOffload(
                        request = ConfirmOffloadRequest(orderId = orderId),
                        idempotencyKey = DriverIdempotencyKeys.offload(orderId),
                    )
                }
                _state.update { it.copy(isSubmitting = false, offloadResult = response) }
            } catch (e: Exception) {
                _state.update { it.copy(isSubmitting = false, error = e.message ?: "Offload failed") }
            }
        }
    }
}
