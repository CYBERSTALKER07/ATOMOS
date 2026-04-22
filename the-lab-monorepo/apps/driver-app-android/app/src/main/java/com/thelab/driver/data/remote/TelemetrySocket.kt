package com.thelab.driver.data.remote

import android.util.Log
import com.thelab.driver.data.model.TelemetryPayload
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
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.math.min
import kotlin.math.pow

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
    private val scope = CoroutineScope(Dispatchers.IO)

    companion object {
        private const val TAG = "TelemetrySocket"
        private const val BASE_DELAY_MS = 5_000L
        private const val MAX_DELAY_MS = 60_000L
        private const val MAX_RECONNECT_ATTEMPTS = 10
    }

    enum class ConnectionState { CONNECTED, DISCONNECTED, RECONNECTING }

    fun connect(baseUrl: String, token: String) {
        intentionalDisconnect = false
        lastBaseUrl = baseUrl
        lastToken = token
        reconnectAttempt = 0
        reconnectJob?.cancel()
        establishConnection(baseUrl, token)
    }

    private fun establishConnection(baseUrl: String, token: String) {
        socket?.close(1000, null)
        socket = null

        val wsUrl = baseUrl
            .replace("http://", "ws://")
            .replace("https://", "wss://")
            .plus("/v1/ws/telemetry")

        val request = Request.Builder()
            .url(wsUrl)
            .addHeader("Authorization", "Bearer $token")
            .build()

        socket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                reconnectAttempt = 0
                _connectionState.trySend(ConnectionState.CONNECTED)
                Log.d(TAG, "Connected")
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                Log.e(TAG, "Connection failed", t)
                scheduleReconnect()
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
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
        if (reconnectAttempt > MAX_RECONNECT_ATTEMPTS) {
            Log.e(TAG, "Max reconnect attempts reached, giving up")
            _connectionState.trySend(ConnectionState.DISCONNECTED)
            return
        }

        _connectionState.trySend(ConnectionState.RECONNECTING)
        val delayMs = min(BASE_DELAY_MS * 2.0.pow(reconnectAttempt - 1).toLong(), MAX_DELAY_MS)
        Log.d(TAG, "Reconnecting in ${delayMs}ms (attempt $reconnectAttempt)")

        reconnectJob?.cancel()
        reconnectJob = scope.launch {
            delay(delayMs)
            establishConnection(url, token)
        }
    }

    fun send(payload: TelemetryPayload): Boolean {
        val data = json.encodeToString(payload)
        return socket?.send(data) ?: false
    }

    fun disconnect() {
        intentionalDisconnect = true
        reconnectJob?.cancel()
        reconnectJob = null
        socket?.close(1000, "Driver stopped transit")
        socket = null
        _connectionState.trySend(ConnectionState.DISCONNECTED)
    }
}
