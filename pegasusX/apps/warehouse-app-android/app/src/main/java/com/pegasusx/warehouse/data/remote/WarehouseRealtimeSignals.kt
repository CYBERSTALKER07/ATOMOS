package com.pegasusx.warehouse.data.remote

import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import javax.inject.Inject
import javax.inject.Singleton

/** Cross-screen refresh tick driven by WebSocket operational events. */
@Singleton
class WarehouseRealtimeSignals @Inject constructor() {
    private val _refreshTick = MutableSharedFlow<Long>(extraBufferCapacity = 16)
    val refreshTick: SharedFlow<Long> = _refreshTick.asSharedFlow()

    fun bump() {
        _refreshTick.tryEmit(System.currentTimeMillis())
    }
}
