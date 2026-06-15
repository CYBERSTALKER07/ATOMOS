package com.pegasus.payload.util

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object PayloadIdempotencyKeys {
    fun key(action: String, entityId: String): String = "payload-$action-$entityId"

    fun recommendReassign(orderId: String): String = key("recommend-reassign", orderId)

    fun fleetReassign(orderIds: List<String>): String =
        key("fleet-reassign", orderIds.map { it.trim() }.filter { it.isNotEmpty() }.sorted().joinToString(","))

    fun applyReassign(orderId: String, toDriverId: String): String =
        key("reassign-order", "$orderId-$toDriverId")
}
