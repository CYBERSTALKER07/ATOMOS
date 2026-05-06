package com.pegasus.factory.ui.realtime

import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.platform.LocalContext
import com.pegasus.factory.data.remote.FactoryRealtimeClient
import com.pegasus.factory.data.remote.FactoryRealtimeEventType

@Composable
fun FactoryRealtimeReloadEffect(
    eventTypes: Set<FactoryRealtimeEventType>,
    onEvent: () -> Unit,
) {
    val context = LocalContext.current
    val latestOnEvent = rememberUpdatedState(onEvent)
    val latestEventTypes = rememberUpdatedState(eventTypes)
    val realtimeClient = remember { FactoryRealtimeClient(context) }

    DisposableEffect(realtimeClient) {
        realtimeClient.connect(
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
