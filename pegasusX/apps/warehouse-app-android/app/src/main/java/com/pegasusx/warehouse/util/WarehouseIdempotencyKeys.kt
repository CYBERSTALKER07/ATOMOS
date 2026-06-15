package com.pegasusx.warehouse.util

import com.pegasusx.warehouse.data.remote.TokenHolder

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object WarehouseIdempotencyKeys {
    private fun warehouseId(): String = TokenHolder.warehouseId?.takeIf { it.isNotBlank() } ?: "warehouse"

    private fun stableHash(input: String): String {
        var hash = 2166136261L
        for (ch in input) {
            hash = hash xor ch.code.toLong()
            hash = (hash * 16777619L) and 0xFFFFFFFFL
        }
        return hash.toString(36)
    }

    fun emergencyTransfer(volumeVu: Double, notes: String?): String =
        "warehouse-emergency-transfer:${warehouseId()}:$volumeVu:${stableHash(notes.orEmpty())}"

    fun forceReceive(volumeVu: Double, notes: String?, factoryId: String? = null): String =
        "warehouse-force-receive:${warehouseId()}:${factoryId.orEmpty()}:$volumeVu:${stableHash(notes.orEmpty())}"

    fun receiveTransfer(transferId: String): String = "warehouse-receive-transfer:$transferId"

    fun dispatchLockAcquire(entityType: String = "WAREHOUSE", entityId: String? = null): String {
        val wh = entityId?.takeIf { it.isNotBlank() } ?: warehouseId()
        return "warehouse-dispatch-lock-acquire:${warehouseId()}:$entityType:$wh"
    }

    fun dispatchLockRelease(lockId: String): String = "warehouse-dispatch-lock-release:$lockId"
}
