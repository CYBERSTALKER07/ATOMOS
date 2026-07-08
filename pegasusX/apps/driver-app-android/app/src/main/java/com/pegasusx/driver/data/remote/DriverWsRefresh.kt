package com.pegasusx.driver.data.remote

/** Returns true when a driver websocket envelope should trigger a manifest refresh. */
fun shouldRefreshManifestOnWsEvent(eventType: String): Boolean {
    return when (eventType) {
        "ORDER_ASSIGNED",
        "ORDER_REASSIGNED",
        "ORDER_STATUS_CHANGED",
        "ORDER_FINALIZED",
        "ROUTE_CREATED",
        "ROUTE_REORDERED",
        "PAYMENT_REQUIRED",
        "PAYMENT_CLEARED",
        "MANIFEST_DISPATCHED",
        "MANIFEST_COMPLETED",
        "SHOP_CLOSED_RESOLVED",
        "NEGOTIATION_RESOLVED",
        "DELIVERY_SESSION_UPDATED",
        "REASSIGN_HANDSHAKE_COMPLETED",
        "DRIVER_AVAILABILITY_CHANGED" -> true
        else -> false
    }
}
