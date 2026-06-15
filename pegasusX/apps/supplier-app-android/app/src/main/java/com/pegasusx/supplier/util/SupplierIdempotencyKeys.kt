package com.pegasusx.supplier.util

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object SupplierIdempotencyKeys {
    fun shopClosedResolve(attemptId: String, action: String): String =
        "shop-closed-resolve:$attemptId:$action"

    fun approveEarlyComplete(driverId: String): String =
        "supplier-approve-early-complete:$driverId"

    fun negotiationResolve(proposalId: String, action: String): String =
        "supplier-negotiate-resolve:$proposalId:$action"
}

const val SUPPLIER_RECONNECT_RECOVERY_HINT =
    "Connection restored — verify status before retrying."
