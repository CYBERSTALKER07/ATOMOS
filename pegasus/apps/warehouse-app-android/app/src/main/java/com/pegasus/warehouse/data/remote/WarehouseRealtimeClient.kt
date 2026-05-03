package com.pegasus.warehouse.data.remote

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Handler
import android.os.Looper
import com.pegasus.warehouse.BuildConfig
import com.pegasus.warehouse.data.model.WarehouseLiveEvent
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
import kotlin.math.min

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
    private var reconnectJob: Job? = null
    private var manualDisconnect = true
    private var networkAvailable = hasNetworkConnectivity()
    private var stateHandler: ((WarehouseRealtimeStatus) -> Unit)? = null
    private var eventHandler: ((WarehouseLiveEvent) -> Unit)? = null

    init {
        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()
        connectivityManager?.registerNetworkCallback(request, networkCallback)
    }

    fun connect(
        onStateChange: (WarehouseRealtimeStatus) -> Unit,
        onEvent: (WarehouseLiveEvent) -> Unit,
    ) {
        stateHandler = onStateChange
        eventHandler = onEvent
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
            .encodedPath("/ws/warehouse")
            .addQueryParameter("token", token)
            .build()
        val request = Request.Builder().url(wsUrl).build()

        notifyState(if (isReconnect) WarehouseRealtimeStatus.RECONNECTING else WarehouseRealtimeStatus.CONNECTING)
        webSocket?.cancel()
        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                reconnectAttempt = 0
                notifyState(WarehouseRealtimeStatus.LIVE)
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
            val delayMs = min(30_000L, 1_000L shl (reconnectAttempt - 1).coerceAtMost(4))
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