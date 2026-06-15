package com.pegasusx.warehouse.data.remote

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Handler
import android.os.Looper
import com.pegasusx.warehouse.BuildConfig
import com.pegasusx.warehouse.data.model.WarehouseLiveEvent
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancelChildren
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.util.concurrent.TimeUnit

enum class WarehouseRealtimeStatus {
    IDLE,
    CONNECTING,
    LIVE,
    RECONNECTING,
    OFFLINE,
}

class WarehouseRealtimeClient(
    context: Context,
    private val json: Json = Json {
        ignoreUnknownKeys = true
        coerceInputValues = true
        encodeDefaults = true
    },
    private val client: OkHttpClient = OkHttpClient.Builder()
        .readTimeout(0, TimeUnit.MILLISECONDS)
        .build(),
) {
    private val appContext = context.applicationContext
    private val connectivityManager = appContext.getSystemService(ConnectivityManager::class.java)
    private val mainHandler = Handler(Looper.getMainLooper())
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val networkCallback = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) {
            networkAvailable = true
            if (!manualDisconnect && webSocket == null) {
                connectInternal(isReconnect = reconnectAttempt > 0)
            }
        }

        override fun onLost(network: Network) {
            networkAvailable = hasNetworkConnectivity()
            if (!networkAvailable && !manualDisconnect) {
                reconnectJob?.cancel()
                webSocket?.cancel()
                webSocket = null
                notifyState(WarehouseRealtimeStatus.OFFLINE)
            }
        }

        override fun onUnavailable() {
            networkAvailable = false
            if (!manualDisconnect) {
                notifyState(WarehouseRealtimeStatus.OFFLINE)
            }
        }
    }

    private var webSocket: WebSocket? = null
    private var reconnectAttempt = 0
    private var pendingRetryAfterMs: Long? = null
    private var reconnectJob: Job? = null
    private var manualDisconnect = true
    private var networkAvailable = hasNetworkConnectivity()
    private var stateHandler: ((WarehouseRealtimeStatus) -> Unit)? = null
    private var eventHandler: ((WarehouseLiveEvent) -> Unit)? = null
    private var onReconnectHandler: (() -> Unit)? = null
    private var hasConnectedOnce = false

    init {
        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()
        connectivityManager?.registerNetworkCallback(request, networkCallback)
    }

    fun connect(
        onStateChange: (WarehouseRealtimeStatus) -> Unit,
        onEvent: (WarehouseLiveEvent) -> Unit,
        onReconnect: () -> Unit = {},
    ) {
        stateHandler = onStateChange
        eventHandler = onEvent
        onReconnectHandler = onReconnect
        manualDisconnect = false
        reconnectAttempt = 0
        connectInternal(isReconnect = false)
    }

    fun disconnect() {
        manualDisconnect = true
        reconnectJob?.cancel()
        reconnectJob = null
        webSocket?.close(1000, "closing")
        webSocket = null
        notifyState(WarehouseRealtimeStatus.IDLE)
    }

    fun dispose() {
        disconnect()
        runCatching { connectivityManager?.unregisterNetworkCallback(networkCallback) }
        scope.coroutineContext.cancelChildren()
    }

    private fun connectInternal(isReconnect: Boolean) {
        if (manualDisconnect) return
        reconnectJob?.cancel()
        reconnectJob = null

        val token = TokenHolder.token
        if (token.isNullOrBlank() || !networkAvailable) {
            notifyState(WarehouseRealtimeStatus.OFFLINE)
            return
        }

        val baseUrl = BuildConfig.API_BASE_URL.trimEnd('/').toHttpUrlOrNull() ?: run {
            notifyState(WarehouseRealtimeStatus.OFFLINE)
            return
        }
        val wsUrl = baseUrl.newBuilder()
            .scheme(if (baseUrl.isHttps) "wss" else "ws")
            .encodedPath("/v1/ws")
            .addQueryParameter("token", token)
            .build()
        val request = Request.Builder().url(wsUrl).build()

        notifyState(if (isReconnect) WarehouseRealtimeStatus.RECONNECTING else WarehouseRealtimeStatus.CONNECTING)
        webSocket?.cancel()
        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                val wasReconnect = hasConnectedOnce
                hasConnectedOnce = true
                reconnectAttempt = 0
                notifyState(WarehouseRealtimeStatus.LIVE)
                if (wasReconnect) {
                    mainHandler.post { onReconnectHandler?.invoke() }
                }
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                runCatching { json.decodeFromString<WarehouseLiveEvent>(text) }
                    .getOrNull()
                    ?.let { event ->
                        notifyState(WarehouseRealtimeStatus.LIVE)
                        mainHandler.post { eventHandler?.invoke(event) }
                    }
            }

            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                webSocket.close(code, reason)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                handleSocketDrop()
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                pendingRetryAfterMs = ReconnectBackoff.retryAfterMs(response)
                handleSocketDrop()
            }
        })
    }

    private fun handleSocketDrop() {
        webSocket = null
        scheduleReconnect()
    }

    private fun scheduleReconnect() {
        if (manualDisconnect) return
        if (!networkAvailable) {
            notifyState(WarehouseRealtimeStatus.OFFLINE)
            return
        }

        reconnectAttempt += 1
        notifyState(WarehouseRealtimeStatus.RECONNECTING)
        reconnectJob?.cancel()
        reconnectJob = scope.launch {
            val delayMs = ReconnectBackoff.delayMs(
                reconnectAttempt,
                retryAfterMs = pendingRetryAfterMs,
            )
            pendingRetryAfterMs = null
            delay(delayMs)
            if (!manualDisconnect) {
                connectInternal(isReconnect = true)
            }
        }
    }

    private fun notifyState(status: WarehouseRealtimeStatus) {
        mainHandler.post { stateHandler?.invoke(status) }
    }

    private fun hasNetworkConnectivity(): Boolean {
        val activeNetwork = connectivityManager?.activeNetwork ?: return false
        val capabilities = connectivityManager.getNetworkCapabilities(activeNetwork) ?: return false
        return capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
    }
}