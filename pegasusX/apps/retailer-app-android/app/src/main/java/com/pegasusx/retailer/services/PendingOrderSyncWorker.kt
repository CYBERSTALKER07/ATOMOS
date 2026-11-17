package com.pegasusx.retailer.services

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.pegasusx.retailer.BuildConfig
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.PendingOrderDao
import com.pegasusx.retailer.data.local.PendingOrderEntity
import com.pegasusx.retailer.data.model.ProcurementOrderRequest
import com.pegasusx.retailer.data.model.UnifiedCheckoutRequest
import com.pegasusx.retailer.util.OrderLineContractHelper
import com.pegasusx.retailer.util.PendingOrderConflictKind
import com.pegasusx.retailer.util.PendingOrderErrorAssessment
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json

data class PendingOrderStatusNotice(
    val orderId: Long,
    val endpoint: String,
    val status: String,
    val message: String,
    val keptInQueue: Boolean,
)

data class PendingOrderSyncSummary(
    val attempted: Int,
    val succeeded: Int,
    val parked: Int,
    val retried: Int,
    val purged: Int,
    val notices: List<PendingOrderStatusNotice>,
) {
    val shouldRetry: Boolean get() = parked > 0 || retried > 0
}

object PendingOrderStatusBus {
    private val _latestNotice = MutableStateFlow<PendingOrderStatusNotice?>(null)
    val latestNotice: StateFlow<PendingOrderStatusNotice?> = _latestNotice.asStateFlow()

    fun emit(notice: PendingOrderStatusNotice) {
        _latestNotice.value = notice
    }

    fun clear() {
        _latestNotice.value = null
    }
}

/**
 * Pure Kotlin processor shared between [PendingOrderSyncWorker] and ViewModel-triggered flushes
 * so replay logic (including 409 variants, 429, and procurement endpoints) can be tested without
 * Android framework globals.
 */
class PendingOrderSyncProcessor(
    private val api: PegasusApi,
    private val pendingOrderDao: PendingOrderDao,
    private val json: Json = OrderLineContractHelper.lenientJson,
    private val onStatusNotice: (PendingOrderStatusNotice) -> Unit = { PendingOrderStatusBus.emit(it) },
) {
    suspend fun processAll(): PendingOrderSyncSummary {
        val pending = pendingOrderDao.getAll()
        if (pending.isEmpty()) {
            return PendingOrderSyncSummary(0, 0, 0, 0, 0, emptyList())
        }

        var succeeded = 0
        var parked = 0
        var retried = 0
        var purged = 0
        val notices = mutableListOf<PendingOrderStatusNotice>()

        for (order in pending) {
            try {
                dispatch(order)
                pendingOrderDao.deleteById(order.id)
                succeeded++
                if (order.status in PARKED_STATUSES) {
                    val notice = PendingOrderStatusNotice(
                        orderId = order.id,
                        endpoint = order.endpoint,
                        status = "SYNCED",
                        message = "Queued order synced after ${order.status.lowercase().replace('_', ' ')}.",
                        keptInQueue = false,
                    )
                    notices.add(notice)
                    onStatusNotice(notice)
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                val assessment = assessFailure(e)
                if (assessment.shouldDelete) {
                    pendingOrderDao.deleteById(order.id)
                    purged++
                } else {
                    pendingOrderDao.updateStatusAndRetry(order.id, assessment.status, assessment.message)
                    if (assessment.kind in PARKED_CONFLICT_KINDS) {
                        parked++
                    } else {
                        retried++
                    }
                }
                if (assessment.shouldNotify) {
                    val notice = PendingOrderStatusNotice(
                        orderId = order.id,
                        endpoint = order.endpoint,
                        status = assessment.status,
                        message = assessment.message,
                        keptInQueue = !assessment.shouldDelete,
                    )
                    notices.add(notice)
                    onStatusNotice(notice)
                }
            }
        }

        return PendingOrderSyncSummary(
            attempted = pending.size,
            succeeded = succeeded,
            parked = parked,
            retried = retried,
            purged = purged,
            notices = notices,
        )
    }

    private suspend fun dispatch(order: PendingOrderEntity) {
        when {
            order.endpoint == "/v1/checkout/unified" -> {
                val request = json.decodeFromString(UnifiedCheckoutRequest.serializer(), order.payloadJson)
                api.unifiedCheckout(request, order.idempotencyKey)
            }
            order.endpoint == "/v1/order/create" || order.endpoint == "/v1/ai/preorder" -> {
                val decoded = runCatching {
                    json.decodeFromString(ProcurementOrderRequest.serializer(), order.payloadJson)
                }.getOrNull()
                if (decoded != null) {
                    if (order.endpoint == "/v1/order/create") {
                        api.createOrder(decoded, order.idempotencyKey)
                    } else {
                        api.createProcurementOrder(order.endpoint, decoded, order.idempotencyKey)
                    }
                } else {
                    val rawElement = json.parseToJsonElement(order.payloadJson)
                    api.postRawQueuedOrder(order.endpoint, rawElement, order.idempotencyKey)
                }
            }
            order.endpoint == "/v1/orders/confirm-preorder" ||
                order.endpoint == "/v1/orders/edit-preorder" ||
                order.endpoint == "/v1/retailer/orders/confirm-ai" ||
                order.endpoint.startsWith("/v1/") -> {
                val rawElement = json.parseToJsonElement(order.payloadJson)
                api.postRawQueuedOrder(order.endpoint, rawElement, order.idempotencyKey)
            }
            else -> throw IllegalArgumentException("Unsupported queued endpoint: ${order.endpoint}")
        }
    }

    private fun assessFailure(error: Exception): PendingOrderErrorAssessment {
        if (error is SerializationException || error is IllegalArgumentException) {
            return PendingOrderErrorAssessment(
                kind = PendingOrderConflictKind.PERMANENT_CLIENT_ERROR,
                status = "DISCARDED",
                message = error.message ?: "Invalid queued order payload removed from queue.",
                shouldDelete = true,
                shouldNotify = true,
                retryable = false,
            )
        }
        return OrderLineContractHelper.assessPendingOrderError(error)
    }

    private companion object {
        val PARKED_STATUSES = setOf(
            PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL,
            PendingOrderEntity.STATUS_IN_PROGRESS,
            PendingOrderEntity.STATUS_RATE_LIMITED,
        )
        val PARKED_CONFLICT_KINDS = setOf(
            PendingOrderConflictKind.OWNER_APPROVAL_REQUIRED,
            PendingOrderConflictKind.REQUEST_IN_PROGRESS,
            PendingOrderConflictKind.RATE_LIMITED,
            PendingOrderConflictKind.UNAUTHORIZED,
        )
    }
}

/**
 * Drains pending_orders when network reconnects (checkout + procurement).
 * Idempotency keys prevent double-apply if the original request succeeded during the outage.
 * Preserves queued orders on 409 owner_approval_required, 409 request_in_progress, and 429 rate-limit.
 */
@HiltWorker
class PendingOrderSyncWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val api: PegasusApi,
    private val pendingOrderDao: PendingOrderDao,
    private val json: Json,
) : CoroutineWorker(appContext, params) {

    companion object {
        const val TAG = "PendingOrderSyncWorker"
        const val WORK_NAME = "retailer_pending_order_sync"
        private const val CHANNEL_ID = "retailer_pending_orders"
    }

    override suspend fun doWork(): Result {
        val summary = PendingOrderSyncProcessor(
            api = api,
            pendingOrderDao = pendingOrderDao,
            json = json,
            onStatusNotice = { notice ->
                PendingOrderStatusBus.emit(notice)
                showStatusNotification(applicationContext, notice)
            },
        ).processAll()

        if (summary.attempted == 0) return Result.success()
        if (BuildConfig.DEBUG) {
            Log.d(
                TAG,
                "Processed ${summary.attempted} queued order(s): " +
                    "succeeded=${summary.succeeded}, parked=${summary.parked}, " +
                    "retried=${summary.retried}, purged=${summary.purged}",
            )
        }
        return if (summary.shouldRetry) Result.retry() else Result.success()
    }

    private fun showStatusNotification(context: Context, notice: PendingOrderStatusNotice) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) !=
            PackageManager.PERMISSION_GRANTED
        ) {
            if (BuildConfig.DEBUG) {
                Log.d(TAG, "Skipping pending-order status notification: POST_NOTIFICATIONS not granted")
            }
            return
        }
        try {
            val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as? NotificationManager
                ?: return
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                val channel = NotificationChannel(
                    CHANNEL_ID,
                    "Pending order sync",
                    NotificationManager.IMPORTANCE_DEFAULT,
                )
                manager.createNotificationChannel(channel)
            }
            val title = when (notice.status) {
                PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL -> "Order waiting for owner approval"
                PendingOrderEntity.STATUS_IN_PROGRESS -> "Order sync in progress"
                PendingOrderEntity.STATUS_RATE_LIMITED -> "Order sync rate-limited"
                "SYNCED", "SYNCED_IDEMPOTENT" -> "Queued order synced"
                else -> "Order queue notice"
            }
            val notification = NotificationCompat.Builder(context, CHANNEL_ID)
                .setSmallIcon(R.drawable.ic_launcher_foreground)
                .setContentTitle(title)
                .setContentText(notice.message)
                .setStyle(NotificationCompat.BigTextStyle().bigText(notice.message))
                .setAutoCancel(true)
                .build()
            manager.notify((notice.orderId % Int.MAX_VALUE).toInt().coerceAtLeast(1), notification)
        } catch (e: SecurityException) {
            Log.w(TAG, "Unable to show pending-order status notification: ${e.message}")
        } catch (e: Exception) {
            Log.w(TAG, "Notification dispatch failed: ${e.message}")
        }
    }
}
