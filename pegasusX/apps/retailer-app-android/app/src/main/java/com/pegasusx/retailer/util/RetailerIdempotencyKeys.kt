package com.pegasusx.retailer.util

import com.pegasusx.retailer.data.model.ProcurementOrderItem

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object RetailerIdempotencyKeys {
    fun orderCreate(items: List<ProcurementOrderItem>): String {
        val fingerprint = items
            .map { "${it.productId}:${it.quantity}" }
            .sorted()
            .joinToString("|")
        return "retailer-procurement:$fingerprint"
    }

    fun confirmCash(orderId: String): String = "retailer-confirm-cash:$orderId"

    fun cancel(orderId: String): String = "retailer-cancel:$orderId"
}
