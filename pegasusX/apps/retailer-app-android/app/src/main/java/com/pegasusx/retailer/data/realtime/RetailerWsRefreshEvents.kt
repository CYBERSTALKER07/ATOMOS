package com.pegasusx.retailer.data.realtime

/** Canonical retailer websocket refresh contract (mirrors @pegasusx/ws-refresh-contract). */
object RetailerWsRefreshEvents {
    val orderStatus: Set<String> = setOf(
        "ORDER_STATUS_CHANGED",
        "ORDER_ASSIGNED",
        "ORDER_REASSIGNED",
        "ORDER_AMENDED",
        "ORDER_FINALIZED",
        "ORDER_COMPLETED",
        "ORDER_DISPATCHED",
        "ORDER_DELAYED",
        "ORDER_REROUTED",
        "FISCAL_RECEIPT_REQUESTED",
        "FISCAL_RECEIPT_SUCCEEDED",
        "FISCAL_RECEIPT_FAILED",
        "ORDER_FORCE_COMPLETED",
    )

    val dispatch: Set<String> = orderStatus + setOf(
        "DISPATCH_COMMITTED",
        "DRIVER_APPROACHING",
        "DRIVER_ARRIVED",
        "SETTLEMENT_REQUIRED",
        "PAYMENT_REQUIRED",
        "PAYMENT_CLEARED",
        "PAYMENT_SETTLED",
        "PAYMENT_FAILED",
        "PAYMENT_EXPIRED",
        "DELIVERY_SESSION_UPDATED",
        "FISCAL_RECEIPT_REQUESTED",
        "FISCAL_RECEIPT_SUCCEEDED",
        "FISCAL_RECEIPT_FAILED",
        "ORDER_FORCE_COMPLETED",
    )

    fun shouldRefresh(eventType: String, allowed: Set<String> = dispatch): Boolean =
        allowed.contains(eventType)
}
