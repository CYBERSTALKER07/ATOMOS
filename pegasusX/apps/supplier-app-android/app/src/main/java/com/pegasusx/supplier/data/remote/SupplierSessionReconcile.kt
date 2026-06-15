package com.pegasusx.supplier.data.remote

/** Refetch server-authoritative supplier snapshots after transport reconnect. */
suspend fun reconcileSupplierSession(api: SupplierApi) {
    runCatching { api.getDispatchPreview() }
    runCatching { api.getManifests() }
}
