package com.pegasusx.factory.data.remote

/** Refetch server-authoritative factory snapshots after transport reconnect. */
suspend fun reconcileFactorySession(api: FactoryApi) {
    runCatching { api.getManifests() }
}
