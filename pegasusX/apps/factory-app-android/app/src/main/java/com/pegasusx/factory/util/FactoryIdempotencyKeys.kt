package com.pegasusx.factory.util

import com.pegasusx.factory.data.remote.TokenHolder

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object FactoryIdempotencyKeys {
    private fun factoryId(): String = TokenHolder.factoryId?.takeIf { it.isNotBlank() } ?: "factory"

    fun startLoading(manifestId: String): String = "factory-start-loading:$manifestId"

    fun seal(manifestId: String): String = "factory-manifest-seal:${factoryId()}:$manifestId"

    fun dispatch(manifestId: String): String = "factory-manifest-dispatch:${factoryId()}:$manifestId"

    fun complete(manifestId: String): String = "factory-manifest-complete:${factoryId()}:$manifestId"

    fun forLifecyclePath(manifestId: String, path: String): String = when (path) {
        "start-loading" -> startLoading(manifestId)
        "seal" -> seal(manifestId)
        "dispatch" -> dispatch(manifestId)
        "complete" -> complete(manifestId)
        else -> "factory-manifest-transition:${factoryId()}:$manifestId:$path"
    }
}
