package com.pegasus.payload.ui.inbound

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.payload.data.model.QueuedAction
import com.pegasus.payload.data.remote.PayloadApi
import com.pegasus.payload.data.repository.PayloadRepository
import com.pegasus.payload.services.PayloadWebSocket
import com.pegasus.payload.util.PayloadIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import javax.inject.Inject

@Serializable
private data class InboundScanBody(
    val barcode: String,
    val qty: Int = 1,
    @SerialName("session_id") val sessionId: String,
)

@HiltViewModel
class InboundReturnsViewModel @Inject constructor(
    val api: PayloadApi,
    private val repository: PayloadRepository,
    webSocket: PayloadWebSocket,
) : ViewModel() {
    private val json = Json { ignoreUnknownKeys = true }

    val online: StateFlow<Boolean> = webSocket.online.stateIn(
        viewModelScope,
        SharingStarted.WhileSubscribed(5_000),
        true,
    )

    fun queuedScanCount(): Int =
        repository.readQueue().count { it.endpoint.contains("returns/inbound/scan") }

    fun enqueueOfflineScan(barcode: String, sessionId: String?) {
        val body = json.encodeToString(
            InboundScanBody(barcode = barcode, sessionId = sessionId.orEmpty()),
        )
        repository.enqueue(
            QueuedAction(
                id = PayloadIdempotencyKeys.key("inbound-scan", "${barcode}-${sessionId.orEmpty()}"),
                endpoint = "/v1/returns/inbound/scan",
                method = "POST",
                body = body,
                createdAt = System.currentTimeMillis(),
            ),
        )
    }
}
