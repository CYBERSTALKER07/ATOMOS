package com.pegasusx.mobilekit.offline

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Durable offline mutation row (§8.8). Capture-time coordinates must be
 * stored at enqueue and replayed — never replaced with live GPS on flush.
 */
@Serializable
data class QueuedMutation(
    val id: String,
    val endpoint: String,
    val method: String = "POST",
    @SerialName("payload_json") val payloadJson: String,
    @SerialName("idempotency_key") val idempotencyKey: String,
    @SerialName("captured_lat") val capturedLat: Double? = null,
    @SerialName("captured_lng") val capturedLng: Double? = null,
    @SerialName("captured_at_ms") val capturedAtMs: Long? = null,
    @SerialName("client_timestamp_iso") val clientTimestampIso: String = "",
    @SerialName("attempt_count") val attemptCount: Int = 0,
    @SerialName("last_error") val lastError: String = "",
    val status: String = OfflineHttpSemantics.STATUS_PENDING,
    @SerialName("order_id") val orderId: String = "",
    val priority: Int = 40,
    @SerialName("created_at_ms") val createdAtMs: Long = System.currentTimeMillis(),
)
