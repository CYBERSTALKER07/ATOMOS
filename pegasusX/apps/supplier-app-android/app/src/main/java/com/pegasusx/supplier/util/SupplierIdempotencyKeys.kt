package com.pegasusx.supplier.util

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object SupplierIdempotencyKeys {
    fun shopClosedResolve(attemptId: String, action: String): String =
        "shop-closed-resolve:$attemptId:$action"

    fun approveEarlyComplete(driverId: String): String =
        "supplier-approve-early-complete:$driverId"

    fun negotiationResolve(proposalId: String, action: String): String =
        "supplier-negotiate-resolve:$proposalId:$action"

    fun resolveReturn(returnId: String, resolution: String): String =
        "supplier-resolve-return:$returnId:${resolution.uppercase()}"

    fun dispatch(
        supplierId: String,
        warehouseId: String,
        mode: String,
        routeFingerprint: String,
    ): String =
        "supplier-dispatch:$supplierId:$warehouseId:$mode:${stableHash(routeFingerprint)}"

    fun broadcast(scopeId: String, role: String, title: String, body: String): String =
        "supplier-broadcast:$scopeId:${stableHash("$role:$title:$body")}"

    fun paymentBypass(orderId: String, reason: String): String =
        "supplier-payment-bypass:$orderId:${stableHash(reason)}"

    /** FNV-1a 32-bit — matches api-client `stableHash`. */
    private fun stableHash(input: String): String {
        var hash = 2166136261L
        for (c in input) {
            hash = hash xor c.code.toLong()
            hash = (hash * 16777619L) and 0xFFFFFFFFL
        }
        return hash.toULong().toString(36)
    }
}

const val SUPPLIER_RECONNECT_RECOVERY_HINT =
    "Connection restored — verify status before retrying."
