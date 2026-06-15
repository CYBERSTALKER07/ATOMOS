package com.pegasusx.factory.ui.realtime

import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.platform.LocalContext
import com.pegasusx.factory.data.remote.FactoryRealtimeClient
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasusx.factory.data.remote.FactoryRealtimeStatus
import com.pegasusx.factory.data.remote.reconcileFactorySession
import com.pegasusx.factory.data.remote.FactoryApi
import kotlinx.coroutines.launch

@Composable
fun FactoryRealtimeReloadEffect(
    api: FactoryApi? = null,
    eventTypes: Set<FactoryRealtimeEventType>,
    onStatusChange: (FactoryRealtimeStatus) -> Unit = {},
    onEvent: () -> Unit,
    onReconnect: () -> Unit = {},
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val latestOnEvent = rememberUpdatedState(onEvent)
    val latestOnStatusChange = rememberUpdatedState(onStatusChange)
    val latestOnReconnect = rememberUpdatedState(onReconnect)
    val latestEventTypes = rememberUpdatedState(eventTypes)
    val latestApi = rememberUpdatedState(api)
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
            onReconnect = {
                latestApi.value?.let { factoryApi ->
                    scope.launch { reconcileFactorySession(factoryApi) }
                }
                latestOnReconnect.value()
            },
        )

        onDispose {
            realtimeClient.dispose()
        }
    }
}
