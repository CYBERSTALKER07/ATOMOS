package com.pegasusx.supplier.data.remote

import android.util.Log
import com.pegasusx.supplier.BuildConfig
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
data class SupplierWSMessage(
    val type: String,
    @SerialName("trace_id") val traceId: String? = null,
    @SerialName("minimum_version") val minimumVersion: String? = null,
)

@Singleton
class SupplierWebSocket @Inject constructor(
    private val client: OkHttpClient,
    private val json: Json,
) {
    companion object {
        private const val TAG = "SupplierWebSocket"
        private const val BASE_RECONNECT_DELAY_MS = 2_000L
        private const val MAX_RECONNECT_DELAY_MS = 60_000L
        private const val MAX_RECONNECT_ATTEMPTS = 10
    }

    private var socket: WebSocket? = null
    private val reconnectExecutor = Executors.newSingleThreadScheduledExecutor()
    private var reconnectTask: ScheduledFuture<*>? = null
    private val intentionalClose = AtomicBoolean(false)
    private val reconnectAttempt = AtomicInteger(0)
    private val _messages = MutableSharedFlow<SupplierWSMessage>(extraBufferCapacity = 32)
    val messages: SharedFlow<SupplierWSMessage> = _messages.asSharedFlow()

    private var currentBaseUrl: String? = null

    fun connect(baseUrl: String, token: String) {
        if (socket != null) return
        reconnectTask?.cancel(false)
        intentionalClose.set(false)
        currentBaseUrl = baseUrl.trimEnd('/')
        val wsUrl = buildWsUrl(currentBaseUrl!!, token)
        val request = Request.Builder().url(wsUrl)
            .header("X-App-Version", BuildConfig.VERSION_NAME)
            .build()
        socket = client.newWebSocket(request, listener)
    }

    fun disconnect() {
        intentionalClose.set(true)
        reconnectTask?.cancel(false)
        socket?.close(1000, "client_close")
        socket = null
    }

    private fun buildWsUrl(base: String, token: String): String {
        val http = base.replace("https://", "wss://").replace("http://", "ws://")
        return "$http/v1/ws?token=${java.net.URLEncoder.encode(token, "UTF-8")}" +
            "&platform=android&version=${BuildConfig.VERSION_NAME}"
    }

    private val listener = object : WebSocketListener() {
        override fun onOpen(webSocket: WebSocket, response: Response) {
            reconnectAttempt.set(0)
            Log.i(TAG, "supplier ws connected")
        }

        override fun onMessage(webSocket: WebSocket, text: String) {
            runCatching {
                json.decodeFromString<SupplierWSMessage>(text)
            }.onSuccess { msg ->
                if (msg.type == "SYSTEM_APP_OUTDATED") {
                    Log.w(TAG, "app outdated minimum=${msg.minimumVersion}")
                }
                _messages.tryEmit(msg)
            }
        }

        override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
            Log.w(TAG, "supplier ws failure", t)
            scheduleReconnect()
        }

        override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
            if (!intentionalClose.get()) scheduleReconnect()
        }
    }

    private fun scheduleReconnect() {
        if (intentionalClose.get()) return
        val attempt = reconnectAttempt.incrementAndGet()
        if (attempt > MAX_RECONNECT_ATTEMPTS) return
        val delay = minOf(BASE_RECONNECT_DELAY_MS * attempt, MAX_RECONNECT_DELAY_MS)
        reconnectTask?.cancel(false)
        reconnectTask = reconnectExecutor.schedule({
            socket = null
            val base = currentBaseUrl ?: return@schedule
            val token = TokenHolder.token ?: return@schedule
            connect(base, token)
        }, delay, TimeUnit.MILLISECONDS)
    }
}
