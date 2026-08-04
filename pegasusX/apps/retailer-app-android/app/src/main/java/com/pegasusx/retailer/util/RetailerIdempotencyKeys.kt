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

    fun setup(retailerId: String): String = "retailer-setup:$retailerId"

    fun editPreorder(orderId: String): String = "retailer-edit-preorder:$orderId"

    fun rejectAI(orderId: String, reason: String = ""): String =
        "retailer-reject-ai:$orderId:${stableHash(reason)}"

    fun supplierAdd(supplierId: String): String = "retailer-supplier-add:$supplierId"

    fun supplierRemove(supplierId: String): String = "retailer-supplier-remove:$supplierId"

    fun profileUpdate(retailerId: String, payload: Map<String, String>): String {
        val fingerprint = payload.entries
            .sortedBy { it.key }
            .joinToString("|") { "${it.key}=${it.value}" }
        return "retailer-profile-update:$retailerId:${stableHash(fingerprint)}"
    }

    /** Stable file-claim key — same body retries return the same claim_id (G11/G25). */
    fun claimFile(orderId: String, bodyFingerprint: String): String =
        "claim-file:$orderId:${stableHash(bodyFingerprint)}"

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
