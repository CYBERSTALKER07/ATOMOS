package com.pegasus.factory.data.remote

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Handler
import android.os.Looper
import com.pegasus.factory.BuildConfig
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancelChildren
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.util.concurrent.TimeUnit
import kotlin.math.min

enum class FactoryRealtimeStatus {
    IDLE,
    CONNECTING,
    LIVE,
    RECONNECTING,
    OFFLINE,
}

enum class FactoryRealtimeEventType(val wireName: String) {
    SupplyRequestUpdate("FACTORY_SUPPLY_REQUEST_UPDATE"),
    TransferUpdate("FACTORY_TRANSFER_UPDATE"),
    ManifestUpdate("FACTORY_MANIFEST_UPDATE");

    companion object {
        fun fromWireName(value: String): FactoryRealtimeEventType? {
            return entries.firstOrNull { it.wireName == value }
        }
    }
}

@Serializable
data class FactoryLiveEvent(
    @SerialName("type") val type: String,
    @SerialName("timestamp") val timestamp: String? = null,
) {
    val eventType: FactoryRealtimeEventType?
        get() = FactoryRealtimeEventType.fromWireName(type)
}

class FactoryRealtimeClient(
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
                notifyState(FactoryRealtimeStatus.OFFLINE)
            }
        }

        override fun onUnavailable() {
            networkAvailable = false
            if (!manualDisconnect) {
                notifyState(FactoryRealtimeStatus.OFFLINE)
            }
        }
    }

    private var webSocket: WebSocket? = null
    private var reconnectAttempt = 0
    private var reconnectJob: Job? = null
    private var manualDisconnect = true
    private var networkAvailable = hasNetworkConnectivity()
    private var stateHandler: ((FactoryRealtimeStatus) -> Unit)? = null
    private var eventHandler: ((FactoryLiveEvent) -> Unit)? = null

    init {
        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()
        connectivityManager?.registerNetworkCallback(request, networkCallback)
    }

    fun connect(
        onStateChange: (FactoryRealtimeStatus) -> Unit = {},
        onEvent: (FactoryLiveEvent) -> Unit,
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
        notifyState(FactoryRealtimeStatus.IDLE)
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
            notifyState(FactoryRealtimeStatus.OFFLINE)
            return
        }

        val baseUrl = BuildConfig.API_BASE_URL.trimEnd('/').toHttpUrlOrNull() ?: run {
            notifyState(FactoryRealtimeStatus.OFFLINE)
            return
        }

        val wsUrl = baseUrl.newBuilder()
            .scheme(if (baseUrl.isHttps) "wss" else "ws")
            .encodedPath("/v1/ws/factory")
            .addQueryParameter("token", token)
            .build()

        val request = Request.Builder().url(wsUrl).build()

        notifyState(if (isReconnect) FactoryRealtimeStatus.RECONNECTING else FactoryRealtimeStatus.CONNECTING)
        webSocket?.cancel()
        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                reconnectAttempt = 0
                notifyState(FactoryRealtimeStatus.LIVE)
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                runCatching { json.decodeFromString<FactoryLiveEvent>(text) }
                    .getOrNull()
                    ?.let { event ->
                        notifyState(FactoryRealtimeStatus.LIVE)
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
            notifyState(FactoryRealtimeStatus.OFFLINE)
            return
        }

        reconnectAttempt += 1
        notifyState(FactoryRealtimeStatus.RECONNECTING)
        reconnectJob?.cancel()
        reconnectJob = scope.launch {
            val delayMs = min(30_000L, 1_000L shl (reconnectAttempt - 1).coerceAtMost(4))
            delay(delayMs)
            if (!manualDisconnect) {
                connectInternal(isReconnect = true)
            }
        }
    }

    private fun notifyState(status: FactoryRealtimeStatus) {
        mainHandler.post { stateHandler?.invoke(status) }
    }

    private fun hasNetworkConnectivity(): Boolean {
        val activeNetwork = connectivityManager?.activeNetwork ?: return false
        val capabilities = connectivityManager.getNetworkCapabilities(activeNetwork) ?: return false
        return capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
    }
}
