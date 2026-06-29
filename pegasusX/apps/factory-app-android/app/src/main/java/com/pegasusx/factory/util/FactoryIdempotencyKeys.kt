package com.pegasusx.factory.util

import com.pegasusx.factory.data.remote.TokenHolder

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object FactoryIdempotencyKeys {
    private fun factoryId(): String = TokenHolder.factoryId?.takeIf { it.isNotBlank() } ?: "factory"

    fun startLoading(manifestId: String): String = "factory-start-loading:$manifestId"

    fun seal(manifestId: String): String = "factory-manifest-seal:${factoryId()}:$manifestId"

    fun dispatch(manifestId: String): String = "factory-manifest-dispatch:${factoryId()}:$manifestId"

    fun complete(manifestId: String): String = "factory-manifest-complete:${factoryId()}:$manifestId"

    fun batchDispatch(transferIds: List<String>): String {
        val sorted = transferIds.map { it.trim() }.filter { it.isNotEmpty() }.sorted()
        return "factory-dispatch:${factoryId()}:${stableHash(sorted.joinToString(","))}"
    }

    fun rebalance(
        manifestId: String,
        transferId: String,
        toDriverId: String = "",
        toVehicle: String = "",
        targetManifestId: String = "",
    ): String {
        val fingerprint = listOf(toDriverId, toVehicle, targetManifestId).joinToString(":")
        return "factory-manifest-rebalance:$manifestId:$transferId:${stableHash(fingerprint)}"
    }

    fun cancelTransfer(manifestId: String, transferId: String): String =
        "factory-manifest-cancel-transfer:$manifestId:$transferId"

    fun cancelManifest(manifestId: String, reason: String = ""): String =
        "factory-manifest-cancel:$manifestId:${stableHash(reason)}"

    fun transferCreate(
        orderId: String,
        totalVu: Long,
        driverId: String = "",
        vehicleId: String = "",
    ): String {
        val fingerprint = "${orderId.trim()}:$totalVu:${driverId.trim()}:${vehicleId.trim()}"
        return "factory-transfer-create:${factoryId()}:${stableHash(fingerprint)}"
    }

    fun transferTransition(transferId: String, targetState: String): String =
        "factory-transfer-transition:$transferId:${targetState.trim().uppercase()}"

    fun supplyRequestTransition(requestId: String, action: String): String =
        "factory-supply-transition:$requestId:${action.trim().uppercase()}"

    fun supplyRequestAccept(requestId: String): String =
        "factory-supply-accept:$requestId"

    fun opsLocation(lat: Double, lng: Double, placeId: String? = null): String {
        val fingerprint = stableHash("${"%.6f".format(lat)}:${"%.6f".format(lng)}:${placeId.orEmpty()}")
        return "factory-ops-location:${factoryId()}:$fingerprint"
    }

    fun forLifecyclePath(manifestId: String, path: String): String = when (path) {
        "start-loading" -> startLoading(manifestId)
        "seal" -> seal(manifestId)
        "dispatch" -> dispatch(manifestId)
        "complete" -> complete(manifestId)
        else -> "factory-manifest-transition:${factoryId()}:$manifestId:$path"
    }

    private fun stableHash(input: String): String {
        var hash = 2166136261L
        for (c in input) {
            hash = hash xor c.code.toLong()
            hash = (hash * 16777619L) and 0xFFFFFFFFL
        }
        return hash.toString(36)
    }
}
