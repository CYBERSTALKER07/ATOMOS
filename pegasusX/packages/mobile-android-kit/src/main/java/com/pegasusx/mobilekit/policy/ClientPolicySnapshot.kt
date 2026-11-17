package com.pegasusx.mobilekit.policy

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Minimal client-policy snapshot shared by enterprise updater banners. */
@Serializable
data class ClientPolicySnapshot(
    val role: String = "",
    val platform: String = "",
    val channel: String = "",
    @SerialName("client_version") val clientVersion: String = "",
    @SerialName("minimum_version") val minimumVersion: String = "",
    @SerialName("recommended_version") val recommendedVersion: String = "",
    @SerialName("force_update") val forceUpdate: Boolean = false,
    @SerialName("update_url") val updateUrl: String? = null,
    @SerialName("update_deferred") val updateDeferred: Boolean = false,
    @SerialName("defer_reason") val deferReason: String? = null,
    val outdated: Boolean = false,
)
