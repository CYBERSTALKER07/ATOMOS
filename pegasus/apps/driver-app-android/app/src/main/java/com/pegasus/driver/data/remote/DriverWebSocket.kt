package com.pegasus.driver.data.remote

import android.util.Log
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.util.concurrent.Executors
import java.util.concurrent.ScheduledFuture
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger
import javax.inject.Inject
import javax.inject.Singleton

@Serializable
data class DriverWSMessage(
    val type: String,
    @SerialName("order_id") val orderId: String? = null,
    val amount: Long? = null,
    val message: String? = null,
    val response: String? = null,
    @SerialName("bypass_token") val bypassToken: String? = null,
    @SerialName("attempt_id") val attemptId: String? = null
)

@Singleton
class DriverWebSocket @Inject constructor(
    private val client: OkHttpClient,
    private val json: Json
) {
    companion object {
        private const val TAG = "DriverWebSocket"
        private const val BASE_RECONNECT_DELAY_MS = 2_000L
        private const val MAX_RECONNECT_DELAY_MS = 60_000L
        private const val MAX_RECONNECT_ATTEMPTS = 10
    }

    private var socket: WebSocket? = null
    private val reconnectExecutor = Executors.newSingleThreadScheduledExecutor()
    private var reconnectTask: ScheduledFuture<*>? = null
    private val intentionalClose = AtomicBoolean(false)
    private val reconnectAttempt = AtomicInteger(0)
    private val _messages = MutableSharedFlow<DriverWSMessage>(extraBufferCapacity = 16)
    val messages: SharedFlow<DriverWSMessage> = _messages.asSharedFlow()

    private var currentBaseUrl: String? = null
    private var currentToken: String? = null

    fun connect(baseUrl: String, driverId: String, token: String) {
        if (socket != null) return
        reconnectTask?.cancel(false)
        reconnectTask = null
        intentionalClose.set(false)
        reconnectAttempt.set(0)
        currentBaseUrl = baseUrl
        currentToken = token

        connectInternal(baseUrl, token)
    }

    private fun connectInternal(baseUrl: String, token: String) {
        val wsUrl = baseUrl
            .replace("http://", "ws://")
            .replace("https://", "wss://")
            .plus("/v1/ws/driver")

        val request = Request.Builder()
            .url(wsUrl)
            .addHeader("Authorization", "Bearer $token")
            .build()

        socket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                Log.d(TAG, "Connected")
                reconnectAttempt.set(0)
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                try {
                    val msg = json.decodeFromString<DriverWSMessage>(text)
                    _messages.tryEmit(msg)
                } catch (e: Exception) {
                    Log.w(TAG, "Failed to parse WS message: $text", e)
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                Log.e(TAG, "Connection failed", t)
                socket = null
                scheduleReconnect()
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                Log.d(TAG, "Closed: $reason")
                socket = null
                if (code != 1000) scheduleReconnect()
            }
        })
    }

    private fun scheduleReconnect() {
        if (intentionalClose.get()) return
        val attempt = reconnectAttempt.getAndIncrement()
        if (attempt >= MAX_RECONNECT_ATTEMPTS) {
            Log.e(TAG, "Max reconnect attempts reached ($MAX_RECONNECT_ATTEMPTS)")
            return
        }
        val delay = (BASE_RECONNECT_DELAY_MS * (1L shl attempt.coerceAtMost(5))).coerceAtMost(MAX_RECONNECT_DELAY_MS)
        Log.d(TAG, "Reconnecting in ${delay}ms (attempt ${attempt + 1})")
        reconnectTask?.cancel(false)
        reconnectTask = reconnectExecutor.schedule(
            {
                val url = currentBaseUrl ?: return@schedule
                val tok = currentToken ?: return@schedule
                if (!intentionalClose.get()) connectInternal(url, tok)
            },
            delay,
            TimeUnit.MILLISECONDS
        )
    }

    fun disconnect() {
        intentionalClose.set(true)
        reconnectTask?.cancel(false)
        reconnectTask = null
        socket?.close(1000, "Driver disconnected")
        socket = null
    }
}
