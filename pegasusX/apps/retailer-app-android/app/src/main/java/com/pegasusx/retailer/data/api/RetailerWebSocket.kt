package com.pegasusx.retailer.data.api

import com.pegasusx.retailer.BuildConfig
import com.pegasusx.retailer.data.local.TokenManager
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
import java.util.concurrent.TimeUnit
import java.util.concurrent.Executors
import java.util.concurrent.ScheduledFuture
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger
import javax.inject.Inject
import javax.inject.Singleton

// ── Inbound WebSocket message from backend ──

@Serializable
data class RetailerWSMessage(
    @SerialName("type") val type: String,
    @SerialName("order_id") val orderId: String = "",
    @SerialName("invoice_id") val invoiceId: String = "",
    @SerialName("session_id") val sessionId: String = "",
    @SerialName("amount") val amount: Long = 0,
    @SerialName("original_amount") val originalAmount: Long = 0,
    @SerialName("available_card_gateways") val availableCardGateways: List<String> = emptyList(),
    @SerialName("message") val message: String = "",
    @SerialName("delivery_token") val deliveryToken: String = "",
    @SerialName("payment_method") val paymentMethod: String = "",
    @SerialName("gateway") val gateway: String = "",
    @SerialName("adjusted_amount") val adjustedAmount: Long = 0,
    @SerialName("driver_latitude") val driverLatitude: Double? = null,
    @SerialName("driver_longitude") val driverLongitude: Double? = null,
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("supplier_name") val supplierName: String = "",
    @SerialName("state") val state: String = "",
    @SerialName("timestamp") val timestamp: String = "",
    @SerialName("driver_id") val driverId: String = "",
    @SerialName("driver_name") val driverName: String = "",
    @SerialName("attempt_id") val attemptId: String = "",
    @SerialName("options") val options: List<String> = emptyList(),
)

val defaultShopClosedOptions = listOf(
    "OPEN_NOW",
    "5_MIN",
    "CALL_ME",
    "CLOSED_TODAY",
    "RESCHEDULE",
    "CREDIT_LEAVE",
    "CANCEL",
    "AUTHORIZE_BYPASS",
)

fun RetailerWSMessage.toShopClosedAlert(): ShopClosedAlert? {
    if (type != "SHOP_CLOSED" && type != "SHOP_CLOSED_ALERT") return null
    if (orderId.isBlank()) return null
    val resolvedDriver = driverName.ifBlank { driverId.ifBlank { "Driver" } }
    val resolvedOptions = options.ifEmpty { defaultShopClosedOptions }
    return ShopClosedAlert(
        orderId = orderId,
        driverName = resolvedDriver,
        attemptId = attemptId,
        options = resolvedOptions,
    )
}

data class ShopClosedAlert(
    val orderId: String,
    val driverName: String,
    val attemptId: String,
    val options: List<String>,
)

@Singleton
class RetailerWebSocket @Inject constructor(
    private val tokenManager: TokenManager,
    private val json: Json,
) {
    private var socket: WebSocket? = null
    private var client: OkHttpClient? = null
    private val reconnectExecutor = Executors.newSingleThreadScheduledExecutor()
    private var reconnectTask: ScheduledFuture<*>? = null
    private val intentionalClose = AtomicBoolean(false)
    private val reconnectAttempt = AtomicInteger(0)

    private val _events = MutableSharedFlow<RetailerWSMessage>(extraBufferCapacity = 16)
    val events: SharedFlow<RetailerWSMessage> = _events.asSharedFlow()

    private val _reconnects = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val reconnects: SharedFlow<Unit> = _reconnects.asSharedFlow()

    companion object {
        private const val BASE_DELAY_MS = 2_000L
        private const val MAX_DELAY_MS = 60_000L
        private const val MAX_RECONNECT_ATTEMPTS = 10
    }

    fun connect() {
        if (socket != null) return
        reconnectTask?.cancel(false)
        reconnectTask = null
        intentionalClose.set(false)
        val token = tokenManager.getToken() ?: return

        val wsBase = BuildConfig.WS_URL
        val url = "$wsBase/v1/ws"

        val okClient = OkHttpClient.Builder()
            .pingInterval(30, TimeUnit.SECONDS)
            .build()
        client = okClient

        val request = Request.Builder()
            .url(url)
            .addHeader("Authorization", "Bearer $token")
            .build()

        socket = okClient.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                val wasReconnect = reconnectAttempt.getAndSet(0) > 0
                if (wasReconnect) {
                    _reconnects.tryEmit(Unit)
                }
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                try {
                    val msg = json.decodeFromString<RetailerWSMessage>(text)
                    _events.tryEmit(msg)
                } catch (_: Exception) {
                    // ignore malformed messages
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                socket = null
                if (intentionalClose.get()) return
                val attempt = reconnectAttempt.getAndIncrement()
                if (attempt >= MAX_RECONNECT_ATTEMPTS) return
                val delay = ReconnectBackoff.delayMs(attempt, BASE_DELAY_MS, MAX_DELAY_MS, ReconnectBackoff.retryAfterMs(response))
                reconnectTask?.cancel(false)
                reconnectTask = reconnectExecutor.schedule({ connect() }, delay, TimeUnit.MILLISECONDS)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                socket = null
                if (intentionalClose.get()) return
                val attempt = reconnectAttempt.getAndIncrement()
                if (attempt >= MAX_RECONNECT_ATTEMPTS) return
                val delay = ReconnectBackoff.delayMs(attempt, BASE_DELAY_MS, MAX_DELAY_MS, -1)
                reconnectTask?.cancel(false)
                reconnectTask = reconnectExecutor.schedule({ connect() }, delay, TimeUnit.MILLISECONDS)
            }
        })
    }

    fun disconnect() {
        intentionalClose.set(true)
        reconnectTask?.cancel(false)
        reconnectTask = null
        socket?.close(1000, "App closing")
        socket = null
        client?.dispatcher?.executorService?.shutdown()
        client = null
    }
}
