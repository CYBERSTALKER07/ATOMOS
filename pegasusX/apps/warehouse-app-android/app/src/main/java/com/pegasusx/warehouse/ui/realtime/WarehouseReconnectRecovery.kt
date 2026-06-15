package com.pegasusx.warehouse.ui.realtime

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import kotlinx.coroutines.flow.collectLatest

const val WAREHOUSE_RECONNECT_RECOVERY_HINT =
    "Connection restored — verify status before retrying."

@Composable
fun WarehouseReconnectRecoveryEffect(
    realtimeSignals: WarehouseRealtimeSignals,
    isBusy: () -> Boolean,
    onRecover: (hadInFlight: Boolean) -> Unit,
) {
    LaunchedEffect(realtimeSignals) {
        realtimeSignals.reconnectTick.collectLatest {
            val hadInFlight = isBusy()
            onRecover(hadInFlight)
        }
    }
}
