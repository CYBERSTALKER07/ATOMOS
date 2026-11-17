package com.pegasusx.supplier.data.remote

import com.pegasusx.supplier.data.model.MigrationApplyProgressEvent
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
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

    private val _migrationProgressBySession =
        MutableStateFlow<Map<String, MigrationApplyProgressEvent>>(emptyMap())
    val migrationProgressBySession: StateFlow<Map<String, MigrationApplyProgressEvent>> =
        _migrationProgressBySession.asStateFlow()

    fun updateMigrationProgress(event: MigrationApplyProgressEvent) {
        _migrationProgressBySession.update { current ->
            current + (event.sessionId to event)
        }
    }

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
