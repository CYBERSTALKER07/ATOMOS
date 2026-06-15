package com.pegasusx.driver.util

import com.pegasusx.driver.data.remote.TokenHolder

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object DriverIdempotencyKeys {
    private fun driverId(): String = TokenHolder.userId?.takeIf { it.isNotBlank() } ?: "driver"

    fun deliver(orderId: String): String = "driver-deliver:${driverId()}:$orderId"

    fun offload(orderId: String): String = "driver-offload:${driverId()}:$orderId"

    fun complete(orderId: String): String = "driver-complete:${driverId()}:$orderId"

    fun collectCash(orderId: String): String = "driver-collect-cash:${driverId()}:$orderId"

    fun confirmPaymentBypass(orderId: String): String = "driver-confirm-payment-bypass:${driverId()}:$orderId"

    fun bypassOffload(orderId: String): String = "driver-bypass-offload:${driverId()}:$orderId"

    fun reportShopClosed(orderId: String): String = "driver-report-shop-closed:${driverId()}:$orderId"

    fun depart(truckId: String): String = "driver-depart:${driverId()}:$truckId"

    fun returnComplete(truckId: String): String = "driver-return-complete:${driverId()}:$truckId"

    fun syncBatch(orderSignatures: List<String>): String {
        val sorted = orderSignatures.map { it.trim() }.filter { it.isNotEmpty() }.sorted()
        return "driver-sync-batch:${driverId()}:${stableHash(sorted.joinToString(","))}"
    }

    fun markArrived(orderId: String): String = "driver-mark-arrived-$orderId"

    fun splitPayment(orderId: String, cashMinor: Long, cardMinor: Long): String =
        "driver-split-payment:${driverId()}:$orderId:$cashMinor:$cardMinor"

    fun creditDelivery(orderId: String): String = "driver-credit-delivery:${driverId()}:$orderId"

    fun missingItems(orderId: String): String = "driver-missing-items:${driverId()}:$orderId"

    fun reportDamage(orderId: String): String = "driver-report-damage:${driverId()}:$orderId"

    fun negotiate(orderId: String): String = "driver-negotiate:${driverId()}:$orderId"

    fun requestEarlyComplete(reason: String): String = "driver-request-early-complete:${driverId()}:${stableHash(reason)}"

    fun routeReorder(routeId: String, orderSequence: List<String>): String {
        val seq = orderSequence.map { it.trim() }.filter { it.isNotEmpty() }
        return "driver-route-reorder:${driverId()}:$routeId:${stableHash(seq.joinToString(","))}"
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
