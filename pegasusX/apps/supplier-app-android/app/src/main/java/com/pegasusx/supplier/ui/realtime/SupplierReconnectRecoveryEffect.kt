package com.pegasusx.supplier.ui.realtime

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import kotlinx.coroutines.flow.collectLatest

@Composable
fun SupplierReconnectRecoveryEffect(
    realtimeSignals: SupplierRealtimeSignals,
    isBusy: () -> Boolean,
    onRecover: (hadInFlight: Boolean) -> Unit,
) {
    LaunchedEffect(realtimeSignals) {
        realtimeSignals.reconnectTick.collectLatest {
            onRecover(isBusy())
        }
    }
}
