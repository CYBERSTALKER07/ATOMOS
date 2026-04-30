package com.thelab.payload.services

import com.thelab.payload.BuildConfig
import com.thelab.payload.data.model.WsMessage
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import javax.inject.Inject
import javax.inject.Singleton

/**
 * PayloadWebSocket — process-singleton wrapper around an OkHttp [WebSocket]
 * pointed at `${BuildConfig.WS_BASE_URL}/v1/ws/payloader?token=...`.
 *
 * Mirrors the Expo `payload-terminal/App.tsx` behaviour:
 *   - reconnects 3s after every close/error while a token is configured
 *   - exposes `online` as a [StateFlow] (UI chip + offline-queue trigger)
 *   - exposes inbound notification frames as a [SharedFlow] for the ViewModel
 *
 * Backend wire shape (notification_dispatcher.go::HandleEvent):
 *   `{ "type": "<EVENT_NAME>", "title": "...", "body": "...", "channel": "PUSH" }`
 * Anything carrying a `title`+`body` is rendered as an in-app notification.
 */
@Singleton
class PayloadWebSocket @Inject constructor(
    private val okHttp: OkHttpClient,
    private val json: Json,
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val _online = MutableStateFlow(false)
    val online: StateFlow<Boolean> = _online.asStateFlow()

    private val _frames = MutableSharedFlow<WsMessage>(extraBufferCapacity = 32)
    val frames: SharedFlow<WsMessage> = _frames.asSharedFlow()

    private val _onReconnect = MutableSharedFlow<Unit>(extraBufferCapacity = 4)
    /** Emits each time the socket transitions closed → open (drains offline queue). */
    val onReconnect: SharedFlow<Unit> = _onReconnect.asSharedFlow()

    private var socket: WebSocket? = null
    private var reconnectJob: Job? = null
    private var token: String? = null

    fun connect(authToken: String) {
        if (token == authToken && socket != null) return
        token = authToken
        openSocket()
    }

    fun disconnect() {
        token = null
        reconnectJob?.cancel(); reconnectJob = null
        socket?.close(NORMAL_CLOSURE, "logout")
        socket = null
        _online.value = false
    }

    private fun openSocket() {
        val authToken = token ?: return
        socket?.close(NORMAL_CLOSURE, "reopen")
        val url = BuildConfig.WS_BASE_URL.trimEnd('/') +
            "/v1/ws/payloader?token=" + java.net.URLEncoder.encode(authToken, "UTF-8")
        val req = Request.Builder().url(url).build()
        socket = okHttp.newWebSocket(req, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                _online.value = true
                scope.launch { _onReconnect.emit(Unit) }
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                val frame = runCatching { json.decodeFromString(WsMessage.serializer(), text) }
                    .getOrNull() ?: return
                if (frame.title.isNullOrEmpty() && frame.body.isNullOrEmpty()) return
                scope.launch { _frames.emit(frame) }
            }

            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                webSocket.close(NORMAL_CLOSURE, null)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                _online.value = false
                scheduleReconnect()
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                _online.value = false
                scheduleReconnect()
            }
        })
    }

    private fun scheduleReconnect() {
        if (token == null) return
        if (reconnectJob?.isActive == true) return
        reconnectJob = scope.launch {
            delay(RECONNECT_DELAY_MS)
            openSocket()
        }
    }

    private companion object {
        const val NORMAL_CLOSURE = 1000
        const val RECONNECT_DELAY_MS = 3_000L
    }
}
