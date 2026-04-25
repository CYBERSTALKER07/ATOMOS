package com.thelab.driver.data.remote

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
        private const val RECONNECT_DELAY_MS = 3000L
        private const val MAX_RECONNECT_ATTEMPTS = 10
    }

    private var socket: WebSocket? = null
    private val _messages = MutableSharedFlow<DriverWSMessage>(extraBufferCapacity = 16)
    val messages: SharedFlow<DriverWSMessage> = _messages.asSharedFlow()

    private var currentBaseUrl: String? = null
    private var currentToken: String? = null
    private var reconnectAttempts = 0
    private var shouldReconnect = true

    fun connect(baseUrl: String, driverId: String, token: String) {
        disconnect()
        shouldReconnect = true
        reconnectAttempts = 0
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
                reconnectAttempts = 0
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
                scheduleReconnect()
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                Log.d(TAG, "Closed: $reason")
                if (code != 1000) scheduleReconnect()
            }
        })
    }

    private fun scheduleReconnect() {
        if (!shouldReconnect) return
        if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
            Log.e(TAG, "Max reconnect attempts reached ($MAX_RECONNECT_ATTEMPTS)")
            return
        }
        reconnectAttempts++
        val delay = RECONNECT_DELAY_MS * reconnectAttempts
        Log.d(TAG, "Reconnecting in ${delay}ms (attempt $reconnectAttempts)")
        Thread {
            try {
                Thread.sleep(delay)
                val url = currentBaseUrl ?: return@Thread
                val tok = currentToken ?: return@Thread
                if (shouldReconnect) connectInternal(url, tok)
            } catch (_: InterruptedException) {}
        }.start()
    }

    fun disconnect() {
        shouldReconnect = false
        socket?.close(1000, "Driver disconnected")
        socket = null
    }
}
