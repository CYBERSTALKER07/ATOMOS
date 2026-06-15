package com.pegasusx.driver.data.remote

import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/** Refetch server-authoritative driver snapshots after transport reconnect. */
suspend fun reconcileDriverSession(api: DriverApi) {
    runCatching { api.getAssignedOrders() }
    val today = SimpleDateFormat("yyyy-MM-dd", Locale.US).format(Date())
    runCatching { api.getManifest(today) }
}
