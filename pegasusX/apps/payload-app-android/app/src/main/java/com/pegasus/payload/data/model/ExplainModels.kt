package com.pegasus.payload.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.jsonObject
import retrofit2.HttpException

@Serializable
data class StatusExplain(
    val code: String = "",
    val title: String = "",
    val summary: String = "",
    @SerialName("next_steps") val nextSteps: List<String> = emptyList(),
    @SerialName("deep_link") val deepLink: String? = null,
    val recoverable: Boolean = true,
)

@Serializable
data class HandoffCardMetadata(
    val kind: String = "",
    val title: String = "",
    val subtitle: String? = null,
    @SerialName("primary_cta") val primaryCta: String? = null,
    @SerialName("primary_link") val primaryLink: String? = null,
    @SerialName("entity_type") val entityType: String? = null,
    @SerialName("entity_id") val entityId: String? = null,
    val fields: Map<String, String>? = null,
)

data class ApiErrorPayload(
    val message: String,
    val explain: StatusExplain? = null,
)

private val explainJson = Json { ignoreUnknownKeys = true }

fun parseApiErrorPayload(error: Throwable): ApiErrorPayload {
    val raw = when (error) {
        is HttpException -> error.response()?.errorBody()?.string()
        else -> error.message
    }?.takeIf { it.isNotBlank() } ?: return ApiErrorPayload(error.message ?: "Request failed")
    return try {
        val obj = explainJson.parseToJsonElement(raw).jsonObject
        val explain = obj["explain"]?.let { explainJson.decodeFromJsonElement<StatusExplain>(it) }
        val message = obj["message"]?.toString()?.trim('"')
            ?: obj["detail"]?.toString()?.trim('"')
            ?: obj["error"]?.toString()?.trim('"')
            ?: explain?.summary
            ?: raw
        ApiErrorPayload(message = message, explain = explain)
    } catch (_: Exception) {
        ApiErrorPayload(message = raw)
    }
}
