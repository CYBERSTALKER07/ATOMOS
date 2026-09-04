package com.pegasusx.driver.offline

import com.pegasusx.mobilekit.offline.OfflineEndpointCatalog
import com.pegasusx.mobilekit.offline.OfflineHttpSemantics

/**
 * Offline-eligible driver mutations and flush priority (lower = earlier).
 * Aligns with docs/big-platform-baseline/last-mile/4.1-offline-sync-protocol.md
 */
object DriverOfflineActionCatalog : OfflineEndpointCatalog {
    const val STATUS_PENDING = OfflineHttpSemantics.STATUS_PENDING
    const val STATUS_DEAD = OfflineHttpSemantics.STATUS_DEAD
    const val MAX_ATTEMPTS = OfflineHttpSemantics.MAX_ATTEMPTS_DEFAULT
    const val PROXIMITY_MAX_AGE_MS = 2 * 60 * 1000L

    const val ENDPOINT_PROXIMITY = "v1/delivery/proximity-unlock"
    const val ENDPOINT_SHOP_CLOSED = "v1/delivery/shop-closed"
    const val ENDPOINT_PARTIAL = "v1/delivery/partial-offload"
    const val ENDPOINT_DELIVER = "v1/order/deliver"
    const val ENDPOINT_COLLECT_CASH = "v1/order/collect-cash"
    const val ENDPOINT_CREDIT = "v1/delivery/credit-delivery"
    const val ENDPOINT_ARRIVE = "v1/delivery/arrive"
    const val ENDPOINT_OFFLOAD = "v1/order/confirm-offload"
    const val ENDPOINT_COMPLETE = "v1/order/complete"
    const val ENDPOINT_FISCAL_RETRY = "v1/order/{orderId}/fiscal/retry"
    const val ENDPOINT_SPLIT = "v1/delivery/split-payment"
    const val ENDPOINT_BYPASS_OFFLOAD = "v1/delivery/bypass-offload"
    const val ENDPOINT_PAYMENT_BYPASS = "v1/delivery/confirm-payment-bypass"
    const val ENDPOINT_DEPART = "v1/fleet/driver/depart"
    const val ENDPOINT_RETURN = "v1/fleet/driver/return-complete"
    const val ENDPOINT_CASH_RECON = "v1/driver/cash-reconciliations"
    const val ENDPOINT_ROUTE_REORDER = "v1/fleet/route/reorder"
    const val ENDPOINT_AVAILABILITY = "v1/driver/availability"

    override fun priorityFor(endpoint: String): Int = when (normalize(endpoint)) {
        ENDPOINT_PROXIMITY -> 10
        ENDPOINT_SHOP_CLOSED, ENDPOINT_PARTIAL, ENDPOINT_DELIVER -> 20
        ENDPOINT_COLLECT_CASH, ENDPOINT_CREDIT -> 30
        else -> 40
    }

    override fun isOfflineEligible(endpoint: String): Boolean {
        val ep = normalize(endpoint)
        return ep in setOf(
            ENDPOINT_PROXIMITY,
            ENDPOINT_SHOP_CLOSED,
            ENDPOINT_PARTIAL,
            ENDPOINT_DELIVER,
            ENDPOINT_COLLECT_CASH,
            ENDPOINT_CREDIT,
            ENDPOINT_ARRIVE,
            ENDPOINT_OFFLOAD,
            ENDPOINT_COMPLETE,
            ENDPOINT_SPLIT,
            ENDPOINT_BYPASS_OFFLOAD,
            ENDPOINT_PAYMENT_BYPASS,
            ENDPOINT_DEPART,
            ENDPOINT_RETURN,
            ENDPOINT_CASH_RECON,
            ENDPOINT_ROUTE_REORDER,
            ENDPOINT_AVAILABILITY,
        ) || ep.contains("/fiscal/retry")
    }

    override fun normalize(endpoint: String): String =
        OfflineHttpSemantics.normalizeEndpoint(endpoint)

    fun isRetryableHttp(code: Int): Boolean = OfflineHttpSemantics.isRetryableHttp(code)

    fun isSuccessHttp(code: Int): Boolean = OfflineHttpSemantics.isSuccessHttp(code)

    fun isNetworkEnqueueable(error: Throwable): Boolean {
        if (OfflineHttpSemantics.isNetworkEnqueueable(error)) return true
        return when (error) {
            is retrofit2.HttpException -> isRetryableHttp(error.code())
            else -> false
        }
    }
}
