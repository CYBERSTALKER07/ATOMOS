package com.pegasusx.supplier.data.remote

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.launch
import java.util.concurrent.atomic.AtomicLong
import javax.inject.Inject
import javax.inject.Singleton

/** Cross-screen refresh tick driven by WebSocket operational events. */
@Singleton
class SupplierRealtimeSignals @Inject constructor() {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val lastBumpMs = AtomicLong(0L)
    @Volatile private var trailingJob: Job? = null
    private val coalesceWindowMs = 200L

    private val _refreshTick = MutableSharedFlow<Long>(extraBufferCapacity = 16)
    val refreshTick: SharedFlow<Long> = _refreshTick.asSharedFlow()

    private val _reconnectTick = MutableSharedFlow<Long>(extraBufferCapacity = 4)
    val reconnectTick: SharedFlow<Long> = _reconnectTick.asSharedFlow()

    @Synchronized
    fun bump() {
        val now = System.currentTimeMillis()
        val elapsed = now - lastBumpMs.get()
        if (elapsed >= coalesceWindowMs) {
            trailingJob?.cancel()
            trailingJob = null
            lastBumpMs.set(now)
            _refreshTick.tryEmit(now)
            return
        }
        if (trailingJob?.isActive == true) return
        val remaining = (coalesceWindowMs - elapsed).coerceAtLeast(40L)
        trailingJob = scope.launch {
            delay(remaining)
            val tick = System.currentTimeMillis()
            lastBumpMs.set(tick)
            _refreshTick.tryEmit(tick)
        }
    }

    fun bumpReconnect() {
        val now = System.currentTimeMillis()
        _reconnectTick.tryEmit(now)
        bump()
    }
}
