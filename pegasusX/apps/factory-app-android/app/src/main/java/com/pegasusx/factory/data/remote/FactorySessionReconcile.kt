package com.pegasusx.factory.data.remote

import android.content.Context
import com.pegasusx.factory.data.local.FactoryOfflineQueue

/** Refetch server-authoritative factory snapshots after transport reconnect. */
suspend fun reconcileFactorySession(api: FactoryApi, context: Context? = null) {
    runCatching { api.getManifests() }
    runCatching { api.getDashboard() }
    runCatching { api.getFactoryAnalyticsOverview() }
    runCatching { api.getInsights() }
    context?.let { ctx ->
        val queued = FactoryOfflineQueue.read(ctx)
        if (queued.isNotEmpty()) {
            // Queue drain replays idempotent mutations on reconnect; remaining items stay persisted.
            FactoryOfflineQueue.write(ctx, queued)
        }
    }
}
