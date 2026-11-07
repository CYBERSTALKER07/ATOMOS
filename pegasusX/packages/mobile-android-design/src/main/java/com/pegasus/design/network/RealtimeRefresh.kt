package com.pegasus.design.network

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberUpdatedState
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.launch

/**
 * Collects WebSocket refresh ticks and invokes [onRefresh] without remounting the host.
 * Coalesces rapid bursts (e.g. simultaneous refreshTick + reconnectTick or Kafka/Redis
 * event storms) within a 120ms window so the screen only reloads once per burst.
 */
@Composable
fun RealtimeRefreshEffect(
    refreshTick: SharedFlow<Long>,
    reconnectTick: SharedFlow<Long>? = null,
    onRefresh: (silent: Boolean) -> Unit,
) {
    val currentOnRefresh by rememberUpdatedState(onRefresh)
    LaunchedEffect(refreshTick, reconnectTick) {
        var coalesceJob: Job? = null
        fun scheduleCoalescedRefresh() {
            coalesceJob?.cancel()
            coalesceJob = launch {
                delay(120L)
                currentOnRefresh(true)
            }
        }
        launch {
            refreshTick.collect { scheduleCoalescedRefresh() }
        }
        if (reconnectTick != null) {
            launch {
                reconnectTick.collect { scheduleCoalescedRefresh() }
            }
        }
    }
}

/** Show full-screen loading only on cold start when no cached data exists. */
fun showFullScreenLoading(loading: Boolean, hasData: Boolean): Boolean = loading && !hasData
