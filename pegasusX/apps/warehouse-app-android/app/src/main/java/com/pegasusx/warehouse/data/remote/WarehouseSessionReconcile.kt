package com.pegasusx.warehouse.data.remote

import android.content.Context
import com.pegasusx.warehouse.data.local.WarehouseOfflineQueue

/** Refetch server-authoritative warehouse snapshots after transport reconnect. */
suspend fun reconcileWarehouseSession(api: WarehouseApi, context: Context? = null) {
    runCatching { api.getDispatchPreview() }
    runCatching { api.getDispatchLocks() }
    runCatching { api.getDemandForecast() }
    runCatching { api.getReplenishmentInsights() }
    runCatching {
        val tomorrow = java.time.LocalDate.now().plusDays(1).toString()
        api.getOpsBoard(tomorrow)
    }
    context?.let { ctx ->
        val queued = WarehouseOfflineQueue.read(ctx)
        if (queued.isNotEmpty()) {
            WarehouseOfflineQueue.write(ctx, queued)
        }
    }
}
