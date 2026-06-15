package com.pegasusx.warehouse.data.remote

/** Refetch server-authoritative warehouse snapshots after transport reconnect. */
suspend fun reconcileWarehouseSession(api: WarehouseApi) {
    runCatching { api.getDispatchPreview() }
    runCatching { api.getDispatchLocks() }
}
