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

    fun confirmPreorder(orderId: String): String = "retailer-confirm-preorder:$orderId"

    fun confirmAI(orderId: String): String = "retailer-confirm-ai:$orderId"

    fun acceptDeliveryProposal(orderId: String): String = "retailer-accept-delivery-proposal:$orderId"

    fun rejectDeliveryProposal(orderId: String, reason: String = ""): String =
        "retailer-reject-delivery-proposal:$orderId:${stableHash(reason)}"

    fun rejectPreorder(orderId: String, reason: String = ""): String =
        "retailer-reject-preorder:$orderId:${stableHash(reason)}"

    fun cancel(orderId: String): String = "retailer-cancel:$orderId"

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
