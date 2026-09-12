package com.pegasusx.driver.data.remote

import android.util.Log
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
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
    @SerialName("command_id") val commandId: String? = null,
    @SerialName("trace_id") val traceId: String? = null,
    @SerialName("required_schema_version") val requiredSchemaVersion: Int? = null,
    @SerialName("client_schema_version") val clientSchemaVersion: Int? = null,
    @SerialName("blocked_event_type") val blockedEventType: String? = null,
    val amount: Long? = null,
    val message: String? = null,
    val response: String? = null,
    @SerialName("bypass_token") val bypassToken: String? = null,
    @SerialName("attempt_id") val attemptId: String? = null,
    val status: String? = null,
    val state: String? = null,
    @SerialName("payment_method") val paymentMethod: String? = null
)

data class DriverOutdatedState(
    val message: String,
    val blockedEventType: String?,
    val requiredSchemaVersion: Int?,
    val clientSchemaVersion: Int?
)

@Singleton
class DriverWebSocket @Inject constructor(
    private val client: OkHttpClient,
    private val json: Json
) {
    companion object {
        private const val TAG = "DriverWebSocket"
        private const val WS_SCHEMA_VERSION = 2
        private const val BASE_RECONNECT_DELAY_MS = 2_000L
        private const val MAX_RECONNECT_DELAY_MS = 60_000L
    }

    private var socket: WebSocket? = null
    private val reconnectExecutor = Executors.newSingleThreadScheduledExecutor()
    private var reconnectTask: ScheduledFuture<*>? = null
    private val ackExecutor = Executors.newSingleThreadExecutor()
    private val intentionalClose = AtomicBoolean(false)
    private val reconnectAttempt = AtomicInteger(0)
    private val hasConnectedOnce = AtomicBoolean(false)
    private val _onReconnect = MutableSharedFlow<Unit>(extraBufferCapacity = 4)
    /** Emits when the socket transitions closed → open after the initial connection. */
    val onReconnect: SharedFlow<Unit> = _onReconnect.asSharedFlow()
    private val _messages = MutableSharedFlow<DriverWSMessage>(extraBufferCapacity = 16)
    val messages: SharedFlow<DriverWSMessage> = _messages.asSharedFlow()
    private val _outdatedState = MutableStateFlow<DriverOutdatedState?>(null)
    val outdatedState: StateFlow<DriverOutdatedState?> = _outdatedState.asStateFlow()
    private val _connectionState = MutableStateFlow(ConnectionState.DISCONNECTED)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    private var currentBaseUrl: String? = null
    private var currentToken: String? = null
    private var pendingRetryAfterMs: Long? = null
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)

    fun connect(baseUrl: String, driverId: String, token: String) {
        if (socket != null && _connectionState.value == ConnectionState.CONNECTED) return
        reconnectTask?.cancel(false)
        reconnectTask = null
        intentionalClose.set(false)
        reconnectAttempt.set(0)
        hasConnectedOnce.set(false)
        _outdatedState.value = null
        currentBaseUrl = baseUrl
        currentToken = token

        connectInternal(baseUrl, token)
    }

    /**
     * Network regained: zero attempt counter and reconnect immediately if still
     * authenticated and not intentionally closed (§8.8).
     */
    fun resetAndReconnect() {
        if (intentionalClose.get()) return
        val url = currentBaseUrl ?: return
        val token = currentToken ?: return
        reconnectTask?.cancel(false)
        reconnectTask = null
        reconnectAttempt.set(0)
        _connectionState.value = ConnectionState.RECONNECTING
        Log.d(TAG, "resetAndReconnect — network regained")
        connectInternal(url, token)
    }

    private fun connectInternal(baseUrl: String, token: String) {
        val wsUrl = baseUrl
            .replace("http://", "ws://")
            .replace("https://", "wss://")
            .plus("/v1/ws?sv=$WS_SCHEMA_VERSION")

        val request = Request.Builder()
            .url(wsUrl)
            .addHeader("Authorization", "Bearer $token")
            .build()

        socket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                Log.d(TAG, "Connected")
                reconnectAttempt.set(0)
                _connectionState.value = ConnectionState.CONNECTED
                if (hasConnectedOnce.getAndSet(true)) {
                    scope.launch { _onReconnect.emit(Unit) }
                }
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                try {
                    val msg = json.decodeFromString<DriverWSMessage>(text)
                    if (msg.type == "SYSTEM_APP_OUTDATED") {
                        publishOutdatedState(msg)
                    }
                    _messages.tryEmit(msg)
                    maybeAckCommand(msg)
                } catch (e: Exception) {
                    Log.w(TAG, "Failed to parse WS message: $text", e)
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                Log.e(TAG, "Connection failed", t)
                socket = null
                pendingRetryAfterMs = ReconnectBackoff.retryAfterMs(response)
                if (!intentionalClose.get()) {
                    _connectionState.value = ConnectionState.RECONNECTING
                }
                scheduleReconnect()
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                Log.d(TAG, "Closed: $reason")
                socket = null
                if (intentionalClose.get()) {
                    _connectionState.value = ConnectionState.DISCONNECTED
                } else {
                    _connectionState.value = ConnectionState.RECONNECTING
                    scheduleReconnect()
                }
            }
        })
    }

    private fun scheduleReconnect() {
        if (intentionalClose.get()) return
        val attempt = reconnectAttempt.getAndIncrement()
        // Hold at max delay forever — never give up mid-shift (§8.8).
        _connectionState.value = ConnectionState.RECONNECTING
        val delay = ReconnectBackoff.delayMs(
            attempt,
            BASE_RECONNECT_DELAY_MS,
            MAX_RECONNECT_DELAY_MS,
            pendingRetryAfterMs,
        )
        pendingRetryAfterMs = null
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
        _connectionState.value = ConnectionState.DISCONNECTED
    }

    fun clearOutdatedState() {
        _outdatedState.value = null
    }

    private fun publishOutdatedState(msg: DriverWSMessage) {
        val blocked = msg.blockedEventType ?: "this operation"
        val required = msg.requiredSchemaVersion?.toString() ?: "latest"
        val fallbackMessage = "App update required for $blocked (required schema: $required)."

        _outdatedState.value = DriverOutdatedState(
            message = msg.message ?: fallbackMessage,
            blockedEventType = msg.blockedEventType,
            requiredSchemaVersion = msg.requiredSchemaVersion,
            clientSchemaVersion = msg.clientSchemaVersion
        )

        Log.w(
            TAG,
            "Server blocked incompatible event ${msg.blockedEventType} (required schema ${msg.requiredSchemaVersion}, client schema ${msg.clientSchemaVersion})"
        )
    }

    private fun maybeAckCommand(msg: DriverWSMessage) {
        if (msg.type == "SYSTEM_APP_OUTDATED") {
            return
        }
        val commandId = msg.commandId ?: return
        val baseUrl = currentBaseUrl ?: return
        val token = currentToken ?: return

        val endpoint = baseUrl.trimEnd('/') + "/v1/ws/ack"
        val traceId = msg.traceId ?: java.util.UUID.randomUUID().toString()
        val eventType = msg.type.replace("\"", "")
        val body = """
            {
              "command_id": "${commandId.replace("\"", "")}",
              "trace_id": "${traceId.replace("\"", "")}",
              "event_type": "$eventType"
            }
        """.trimIndent()

        ackExecutor.execute {
            try {
                val request = Request.Builder()
                    .url(endpoint)
                    .post(body.toRequestBody("application/json".toMediaType()))
                    .addHeader("Authorization", "Bearer $token")
                    .addHeader("X-Trace-Id", traceId)
                    .build()

                client.newCall(request).execute().use { response ->
                    if (!response.isSuccessful) {
                        Log.w(TAG, "ACK failed for command $commandId: HTTP ${response.code}")
                    }
                }
            } catch (e: Exception) {
                Log.w(TAG, "ACK request failed for command $commandId", e)
            }
        }
    }
}
