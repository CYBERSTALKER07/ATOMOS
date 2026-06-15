package com.pegasusx.retailer.data.api

/** Refetch server-authoritative retailer snapshots after transport reconnect. */
suspend fun reconcileRetailerSession(api: PegasusApi) {
    runCatching { api.getActiveFulfillments() }
    runCatching { api.getPendingPayments() }
    runCatching { api.getTrackingOrders() }
}
