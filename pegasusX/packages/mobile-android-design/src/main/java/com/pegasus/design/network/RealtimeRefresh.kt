package com.pegasus.design.network

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import kotlinx.coroutines.flow.SharedFlow

/**
 * Collects WebSocket refresh ticks and invokes [onRefresh] without remounting the host.
 * Pass `silent = true` via [onRefresh] to keep stale data visible during background reload.
 */
@Composable
fun RealtimeRefreshEffect(
    refreshTick: SharedFlow<Long>,
    reconnectTick: SharedFlow<Long>? = null,
    onRefresh: (silent: Boolean) -> Unit,
) {
    LaunchedEffect(refreshTick) {
        refreshTick.collect { onRefresh(true) }
    }
    if (reconnectTick != null) {
        LaunchedEffect(reconnectTick) {
            reconnectTick.collect { onRefresh(true) }
        }
    }
}

/** Show full-screen loading only on cold start when no cached data exists. */
fun showFullScreenLoading(loading: Boolean, hasData: Boolean): Boolean = loading && !hasData
