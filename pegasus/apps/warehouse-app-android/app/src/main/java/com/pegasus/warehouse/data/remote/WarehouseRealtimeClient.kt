package com.pegasus.warehouse.data.remote

import com.pegasus.warehouse.BuildConfig
import com.pegasus.warehouse.data.model.WarehouseLiveEvent
import kotlinx.serialization.json.Json
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.util.concurrent.TimeUnit

class WarehouseRealtimeClient(
    private val json: Json = Json {
        ignoreUnknownKeys = true
        coerceInputValues = true
        encodeDefaults = true
    },
    private val client: OkHttpClient = OkHttpClient.Builder()
        .readTimeout(0, TimeUnit.MILLISECONDS)
        .build(),
) {
    private var webSocket: WebSocket? = null

    fun connect(onEvent: (WarehouseLiveEvent) -> Unit) {
        val token = TokenHolder.token ?: return
        disconnect()
        val baseUrl = BuildConfig.API_BASE_URL.trimEnd('/').toHttpUrlOrNull() ?: return
        val wsUrl = baseUrl.newBuilder()
            .scheme(if (baseUrl.isHttps) "wss" else "ws")
            .encodedPath("/ws/warehouse")
            .addQueryParameter("token", token)
            .build()
        val request = Request.Builder().url(wsUrl).build()

        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onMessage(webSocket: WebSocket, text: String) {
                runCatching { json.decodeFromString<WarehouseLiveEvent>(text) }
                    .getOrNull()
                    ?.let(onEvent)
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                disconnect()
            }
        })
    }

    fun disconnect() {
        webSocket?.close(1000, "closing")
        webSocket = null
    }
}