package com.pegasusx.driver.data.remote

import android.util.Log
import com.pegasusx.driver.data.model.TelemetryPayload
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.util.concurrent.atomic.AtomicBoolean
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class TelemetrySocket @Inject constructor(
    private val client: OkHttpClient,
    private val json: Json
) {
    private var socket: WebSocket? = null
    private val _connectionState = Channel<ConnectionState>(Channel.CONFLATED)
    val connectionState: Flow<ConnectionState> = _connectionState.receiveAsFlow()

    private var lastBaseUrl: String? = null
    private var lastToken: String? = null
    private var reconnectAttempt = 0
    private var reconnectJob: Job? = null
    private var intentionalDisconnect = false
    private var pendingRetryAfterMs: Long? = null
    private val connected = AtomicBoolean(false)
    private val scope = CoroutineScope(Dispatchers.IO)

    companion object {
        private const val TAG = "TelemetrySocket"
        private const val BASE_DELAY_MS = 5_000L
        private const val MAX_DELAY_MS = 60_000L
    }

    fun connect(baseUrl: String, token: String) {
        intentionalDisconnect = false
        lastBaseUrl = baseUrl
        lastToken = token
        reconnectAttempt = 0
        reconnectJob?.cancel()
        establishConnection(baseUrl, token)
    }

    /**
     * Network regained: zero attempt counter and reconnect immediately if we still
     * have credentials and were not intentionally stopped (§8.8).
     */
    fun resetAndReconnect() {
        if (intentionalDisconnect) return
        val url = lastBaseUrl ?: return
        val token = lastToken ?: return
        reconnectAttempt = 0
        reconnectJob?.cancel()
        reconnectJob = null
        _connectionState.trySend(ConnectionState.RECONNECTING)
        Log.d(TAG, "resetAndReconnect — network regained")
        establishConnection(url, token)
    }

    private fun establishConnection(baseUrl: String, token: String) {
        connected.set(false)
        socket?.close(1000, null)
        socket = null

        val wsUrl = baseUrl
            .replace("http://", "ws://")
            .replace("https://", "wss://")
            .plus("/v1/ws?sv=2")

        val request = Request.Builder()
            .url(wsUrl)
            .addHeader("Authorization", "Bearer $token")
            .build()

        socket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                reconnectAttempt = 0
                connected.set(true)
                _connectionState.trySend(ConnectionState.CONNECTED)
                Log.d(TAG, "Connected")
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                Log.e(TAG, "Connection failed", t)
                connected.set(false)
                pendingRetryAfterMs = ReconnectBackoff.retryAfterMs(response)
                scheduleReconnect()
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                connected.set(false)
                if (!intentionalDisconnect) {
                    scheduleReconnect()
                } else {
                    _connectionState.trySend(ConnectionState.DISCONNECTED)
                }
            }
        })
    }

    private fun scheduleReconnect() {
        if (intentionalDisconnect) return
        val url = lastBaseUrl ?: return
        val token = lastToken ?: return

        reconnectAttempt++
        // Hold at max delay forever — never give up mid-shift (§8.8).
        _connectionState.trySend(ConnectionState.RECONNECTING)
        val delayMs = ReconnectBackoff.delayMs(
            reconnectAttempt - 1,
            BASE_DELAY_MS,
            MAX_DELAY_MS,
            pendingRetryAfterMs,
        )
        pendingRetryAfterMs = null
        Log.d(TAG, "Reconnecting in ${delayMs}ms (attempt $reconnectAttempt)")

        reconnectJob?.cancel()
        reconnectJob = scope.launch {
            delay(delayMs)
            establishConnection(url, token)
        }
    }

    fun send(payload: TelemetryPayload): Boolean {
        if (!connected.get()) return false
        val data = json.encodeToString(payload)
        return socket?.send(data) ?: false
    }

    fun disconnect() {
        intentionalDisconnect = true
        reconnectJob?.cancel()
        reconnectJob = null
        connected.set(false)
        socket?.close(1000, "Driver stopped transit")
        socket = null
        _connectionState.trySend(ConnectionState.DISCONNECTED)
    }

    fun isConnected(): Boolean = connected.get()
}
