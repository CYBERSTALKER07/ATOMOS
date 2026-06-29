package com.pegasus.payload.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class PulseEvent(
    val id: String,
    val kind: String,
    val title: String,
    val description: String? = null,
    @SerialName("occurred_at") val occurredAt: String,
    @SerialName("deep_link") val deepLink: String? = null,
    @SerialName("order_id") val orderId: String? = null,
    @SerialName("manifest_id") val manifestId: String? = null,
)

@Serializable
data class PulseResponse(
    val events: List<PulseEvent> = emptyList(),
    @SerialName("fetched_at") val fetchedAt: String = "",
    @SerialName("unread_count") val unreadCount: Int? = null,
)
