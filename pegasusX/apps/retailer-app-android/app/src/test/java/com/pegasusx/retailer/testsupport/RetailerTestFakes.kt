package com.pegasusx.retailer.testsupport

import com.pegasusx.retailer.data.api.CartRealtimeEventSource
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.RetailerWSMessage
import com.pegasusx.retailer.data.local.PendingOrderDao
import com.pegasusx.retailer.data.local.PendingOrderEntity
import com.pegasusx.retailer.data.local.RetailerUserIdProvider
import com.pegasusx.retailer.data.model.ApprovalDecisionRequest
import com.pegasusx.retailer.data.model.ApprovalDecisionResponse
import com.pegasusx.retailer.data.model.CheckoutPreviewResponse
import com.pegasusx.retailer.data.model.CheckoutQuoteRequest
import com.pegasusx.retailer.data.model.CheckoutQuoteResponse
import com.pegasusx.retailer.data.model.ProcurementOrderRequest
import com.pegasusx.retailer.data.model.ProcurementOrderResponse
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.RetailerApprovalsListResponse
import com.pegasusx.retailer.data.model.RetailerPaymentCatalogResponse
import com.pegasusx.retailer.data.model.SellerMandateResponse
import com.pegasusx.retailer.data.model.SellerMandateUpdateRequest
import com.pegasusx.retailer.data.model.UnifiedCheckoutRequest
import com.pegasusx.retailer.data.model.UnifiedCheckoutResponse
import com.pegasusx.retailer.services.PendingOrderSyncEnqueuer
import java.lang.reflect.Proxy
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import retrofit2.HttpException
import retrofit2.Response

class FakePendingOrderDao : PendingOrderDao {
    private var nextId = 1L
    val rows = mutableListOf<PendingOrderEntity>()
    val deletedIds = mutableListOf<Long>()

    override suspend fun insert(order: PendingOrderEntity) {
        val assignedId = if (order.id == 0L) nextId++ else order.id
        rows.add(order.copy(id = assignedId))
    }

    override suspend fun getAll(): List<PendingOrderEntity> =
        rows.sortedBy { it.createdAt }

    override suspend fun deleteById(id: Long): Int {
        deletedIds.add(id)
        val before = rows.size
        rows.removeAll { it.id == id }
        return if (rows.size < before) 1 else 0
    }

    override suspend fun incrementRetry(id: Long, lastError: String): Int {
        val index = rows.indexOfFirst { it.id == id }
        if (index < 0) return 0
        val current = rows[index]
        rows[index] = current.copy(retryCount = current.retryCount + 1, lastError = lastError)
        return 1
    }

    override suspend fun updateStatusAndRetry(id: Long, status: String, lastError: String): Int {
        val index = rows.indexOfFirst { it.id == id }
        if (index < 0) return 0
        val current = rows[index]
        rows[index] = current.copy(
            retryCount = current.retryCount + 1,
            status = status,
            lastError = lastError,
        )
        return 1
    }

    override suspend fun updateStatus(id: Long, status: String, lastError: String?): Int {
        val index = rows.indexOfFirst { it.id == id }
        if (index < 0) return 0
        val current = rows[index]
        rows[index] = current.copy(status = status, lastError = lastError)
        return 1
    }
}

class FakeSyncEnqueuer : PendingOrderSyncEnqueuer {
    var enqueueCount = 0
        private set

    override fun enqueue() {
        enqueueCount += 1
    }
}

class FakeUserIdProvider(private val userId: String? = "retailer-1") : RetailerUserIdProvider {
    override fun getUserId(): String? = userId
}

class FakeRealtimeEventSource : CartRealtimeEventSource {
    private val _events = MutableSharedFlow<RetailerWSMessage>(extraBufferCapacity = 8)
    override val events: SharedFlow<RetailerWSMessage> = _events.asSharedFlow()
    var connectCalls = 0
        private set

    override fun connect() {
        connectCalls += 1
    }
}

class RecordingPegasusApiConfig {
    var onUnifiedCheckout: ((UnifiedCheckoutRequest, String?) -> UnifiedCheckoutResponse)? = null
    var onCheckoutPreview: ((UnifiedCheckoutRequest) -> CheckoutPreviewResponse)? = null
    var onCreateOrder: ((ProcurementOrderRequest, String?) -> ProcurementOrderResponse)? = null
    var onCreateProcurementOrder: ((String, ProcurementOrderRequest, String?) -> ProcurementOrderResponse)? = null
    var onPostRawQueuedOrder: ((String, JsonElement, String?) -> JsonElement)? = null
    var onGetRetailerMe: (() -> JsonElement)? = null
    var onGetRetailerApprovals: ((String?, Int, String?) -> RetailerApprovalsListResponse)? = null
    var onApproveRetailerRequest: ((String, ApprovalDecisionRequest) -> ApprovalDecisionResponse)? = null
    var onRejectRetailerRequest: ((String, ApprovalDecisionRequest) -> ApprovalDecisionResponse)? = null
    var onGetSellerMandate: ((String) -> SellerMandateResponse)? = null
    var onUpdateSellerMandate: ((String, SellerMandateUpdateRequest) -> SellerMandateResponse)? = null
    var catalogProducts: List<Product> = emptyList()
    var paymentCatalog: RetailerPaymentCatalogResponse = RetailerPaymentCatalogResponse()

    val unifiedCalls = mutableListOf<Pair<UnifiedCheckoutRequest, String?>>()
    val procurementCalls = mutableListOf<Triple<String, ProcurementOrderRequest, String?>>()
    val rawQueuedCalls = mutableListOf<Triple<String, JsonElement, String?>>()
    val mandateUpdates = mutableListOf<Pair<String, SellerMandateUpdateRequest>>()

    fun build(): PegasusApi {
        val fallbackProxy = Proxy.newProxyInstance(
            PegasusApi::class.java.classLoader,
            arrayOf(PegasusApi::class.java),
        ) { _, method, _ ->
            throw UnsupportedOperationException("Unexpected API call in test: ${method.name}")
        } as PegasusApi
        return object : PegasusApi by fallbackProxy {
            override suspend fun unifiedCheckout(
                body: UnifiedCheckoutRequest,
                idempotencyKey: String,
            ): UnifiedCheckoutResponse {
                unifiedCalls.add(body to idempotencyKey)
                return onUnifiedCheckout?.invoke(body, idempotencyKey)
                    ?: UnifiedCheckoutResponse(status = "CONFIRMED", invoiceId = "inv-1", total = 10_000L)
            }

            override suspend fun checkoutPreview(body: UnifiedCheckoutRequest): CheckoutPreviewResponse {
                return onCheckoutPreview?.invoke(body) ?: CheckoutPreviewResponse(ok = true)
            }

            override suspend fun createOrder(
                body: ProcurementOrderRequest,
                idempotencyKey: String?,
            ): ProcurementOrderResponse {
                procurementCalls.add(Triple("/v1/order/create", body, idempotencyKey))
                return onCreateOrder?.invoke(body, idempotencyKey)
                    ?: ProcurementOrderResponse(status = "CREATED", orderId = "ord-proc-1")
            }

            override suspend fun createProcurementOrder(
                path: String,
                body: ProcurementOrderRequest,
                idempotencyKey: String?,
            ): ProcurementOrderResponse {
                procurementCalls.add(Triple(path, body, idempotencyKey))
                return onCreateProcurementOrder?.invoke(path, body, idempotencyKey)
                    ?: ProcurementOrderResponse(status = "CREATED", orderId = "ord-proc-2")
            }

            override suspend fun postRawQueuedOrder(
                path: String,
                body: JsonElement,
                idempotencyKey: String?,
            ): JsonElement {
                rawQueuedCalls.add(Triple(path, body, idempotencyKey))
                return onPostRawQueuedOrder?.invoke(path, body, idempotencyKey) ?: JsonObject(emptyMap())
            }

            override suspend fun getPaymentCatalog(): RetailerPaymentCatalogResponse = paymentCatalog

            override suspend fun getCards(): JsonElement = JsonArray(emptyList())

            override suspend fun getCartSync(): JsonElement = JsonObject(mapOf("items" to JsonArray(emptyList())))

            override suspend fun postCartSync(
                body: Map<String, @JvmSuppressWildcards Any>,
                idempotencyKey: String?,
            ): JsonElement = JsonObject(mapOf("ok" to kotlinx.serialization.json.JsonPrimitive(true)))

            override suspend fun checkoutQuote(body: CheckoutQuoteRequest): CheckoutQuoteResponse {
                return CheckoutQuoteResponse(
                    supplierId = body.supplierId,
                    subtotalMinor = 10_000L,
                    discountMinor = 0L,
                    totalMinor = 10_000L,
                )
            }

            override suspend fun getCatalogProducts(
                categoryId: String?,
                supplierId: String?,
                retailerId: String?,
                limit: Int?,
                offset: Int?,
            ): List<Product> = catalogProducts

            override suspend fun getRetailerMe(): JsonElement {
                return onGetRetailerMe?.invoke()
                    ?: JsonObject(mapOf("retailer_role" to kotlinx.serialization.json.JsonPrimitive("OWNER"), "is_owner" to kotlinx.serialization.json.JsonPrimitive(true)))
            }

            override suspend fun getRetailerApprovals(
                status: String?,
                limit: Int,
                cursor: String?,
            ): RetailerApprovalsListResponse {
                return onGetRetailerApprovals?.invoke(status, limit, cursor) ?: RetailerApprovalsListResponse()
            }

            override suspend fun approveRetailerRequest(
                requestId: String,
                body: ApprovalDecisionRequest,
            ): ApprovalDecisionResponse {
                return onApproveRetailerRequest?.invoke(requestId, body)
                    ?: ApprovalDecisionResponse(requestId = requestId, status = "APPROVED")
            }

            override suspend fun rejectRetailerRequest(
                requestId: String,
                body: ApprovalDecisionRequest,
            ): ApprovalDecisionResponse {
                return onRejectRetailerRequest?.invoke(requestId, body)
                    ?: ApprovalDecisionResponse(requestId = requestId, status = "REJECTED")
            }

            override suspend fun getSellerMandate(userId: String): SellerMandateResponse {
                return onGetSellerMandate?.invoke(userId) ?: SellerMandateResponse(userId = userId)
            }

            override suspend fun updateSellerMandate(
                userId: String,
                body: SellerMandateUpdateRequest,
            ): SellerMandateResponse {
                mandateUpdates.add(userId to body)
                return onUpdateSellerMandate?.invoke(userId, body) ?: SellerMandateResponse(userId = userId)
            }
        }
    }
}

fun httpJsonException(status: Int, jsonBody: String): HttpException {
    val body = jsonBody.toResponseBody("application/json".toMediaType())
    return HttpException(Response.error<Any>(status, body))
}
