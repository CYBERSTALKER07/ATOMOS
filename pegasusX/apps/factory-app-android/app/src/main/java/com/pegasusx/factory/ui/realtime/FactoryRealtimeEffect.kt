package com.pegasusx.factory.ui.realtime

import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.platform.LocalContext
import com.pegasusx.factory.data.remote.FactoryRealtimeClient
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasusx.factory.data.remote.FactoryRealtimeStatus

@Composable
fun FactoryRealtimeReloadEffect(
    eventTypes: Set<FactoryRealtimeEventType>,
    onStatusChange: (FactoryRealtimeStatus) -> Unit = {},
    onEvent: () -> Unit,
) {
    val context = LocalContext.current
    val latestOnEvent = rememberUpdatedState(onEvent)
    val latestOnStatusChange = rememberUpdatedState(onStatusChange)
    val latestEventTypes = rememberUpdatedState(eventTypes)
    val realtimeClient = remember { FactoryRealtimeClient(context) }

    DisposableEffect(realtimeClient) {
        realtimeClient.connect(
            onStateChange = { status ->
                latestOnStatusChange.value(status)
            },
            onEvent = { event ->
                val eventType = event.eventType ?: return@connect
                if (!latestEventTypes.value.contains(eventType)) {
                    return@connect
                }
                latestOnEvent.value()
            },
        )

        onDispose {
            realtimeClient.dispose()
        }
    }
}
