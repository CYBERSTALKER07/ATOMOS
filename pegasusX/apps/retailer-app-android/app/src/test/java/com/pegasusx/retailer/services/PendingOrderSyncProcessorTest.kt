package com.pegasusx.retailer.services

import com.pegasusx.retailer.data.local.PendingOrderEntity
import com.pegasusx.retailer.data.model.CheckoutLineItem
import com.pegasusx.retailer.data.model.ProblemDetail
import com.pegasusx.retailer.data.model.ProblemDetailException
import com.pegasusx.retailer.data.model.ProcurementOrderItem
import com.pegasusx.retailer.data.model.ProcurementOrderRequest
import com.pegasusx.retailer.data.model.UnifiedCheckoutRequest
import com.pegasusx.retailer.testsupport.FakePendingOrderDao
import com.pegasusx.retailer.testsupport.RecordingPegasusApiConfig
import com.pegasusx.retailer.testsupport.httpJsonException
import com.pegasusx.retailer.util.OrderLineContractHelper
import java.io.IOException
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.encodeToString
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PendingOrderSyncProcessorTest {

    private val json = OrderLineContractHelper.lenientJson

    private fun checkoutEntity(id: Long = 1L, key: String = "idem-1"): PendingOrderEntity {
        val req = UnifiedCheckoutRequest(
            retailerId = "ret-1",
            paymentGateway = "CASH",
            items = listOf(
                CheckoutLineItem(
                    skuId = "sku-1",
                    quantity = 6,
                    unitPrice = 10_000L,
                    mustHave = true,
                    allowSubstitutes = false,
                ),
            ),
            autoConfirmOptIn = true,
        )
        return PendingOrderEntity(
            id = id,
            endpoint = "/v1/checkout/unified",
            method = "POST",
            payloadJson = json.encodeToString(req),
            idempotencyKey = key,
        )
    }

    private fun procurementEntity(
        id: Long = 2L,
        endpoint: String = "/v1/order/create",
        key: String = "idem-proc-1",
    ): PendingOrderEntity {
        val req = ProcurementOrderRequest(
            retailerId = "ret-1",
            items = listOf(
                ProcurementOrderItem(
                    productId = "prod-1",
                    quantity = 12,
                    mustHave = false,
                    allowSubstitutes = true,
                ),
            ),
            autoConfirmOptIn = true,
        )
        return PendingOrderEntity(
            id = id,
            endpoint = endpoint,
            method = "POST",
            payloadJson = json.encodeToString(req),
            idempotencyKey = key,
        )
    }

    @Test
    fun doesNotPurge_on409OwnerApprovalRequired_andEmitsNotice() = runTest {
        val dao = FakePendingOrderDao().apply {
            rows.add(checkoutEntity(id = 10L))
        }
        val apiCfg = RecordingPegasusApiConfig().apply {
            onUnifiedCheckout = { _, _ ->
                throw httpJsonException(409, """{"code":"owner_approval_required","message":"Pending owner sign-off"}""")
            }
        }
        val notices = mutableListOf<PendingOrderStatusNotice>()
        val processor = PendingOrderSyncProcessor(apiCfg.build(), dao, json) { notices.add(it) }

        val summary = processor.processAll()

        assertEquals(1, dao.rows.size)
        assertTrue(dao.deletedIds.isEmpty())
        assertEquals(PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL, dao.rows.single().status)
        assertEquals(1, summary.parked)
        assertTrue(summary.shouldRetry)
        assertEquals(1, notices.size)
        assertEquals(PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL, notices.single().status)
        assertTrue(notices.single().keptInQueue)
    }

    @Test
    fun doesNotPurge_on409RequestInProgress_onProcurementOrder() = runTest {
        val dao = FakePendingOrderDao().apply {
            rows.add(procurementEntity(id = 20L, endpoint = "/v1/order/create"))
        }
        val apiCfg = RecordingPegasusApiConfig().apply {
            onCreateOrder = { _, _ ->
                throw ProblemDetailException(
                    ProblemDetail(
                        status = 409,
                        code = "request_in_progress",
                        detail = "Order submission still processing",
                    ),
                )
            }
        }
        val notices = mutableListOf<PendingOrderStatusNotice>()
        val processor = PendingOrderSyncProcessor(apiCfg.build(), dao, json) { notices.add(it) }

        val summary = processor.processAll()

        assertEquals(1, dao.rows.size)
        assertTrue(dao.deletedIds.isEmpty())
        assertEquals(PendingOrderEntity.STATUS_IN_PROGRESS, dao.rows.single().status)
        assertEquals(1, summary.parked)
        assertTrue(summary.shouldRetry)
        assertEquals(PendingOrderEntity.STATUS_IN_PROGRESS, notices.single().status)
    }

    @Test
    fun doesNotPurge_on429RateLimit_bothHttpAndProblemDetail() = runTest {
        val dao = FakePendingOrderDao().apply {
            rows.add(checkoutEntity(id = 30L, key = "k-30"))
            rows.add(procurementEntity(id = 31L, endpoint = "/v1/ai/preorder", key = "k-31"))
        }
        val apiCfg = RecordingPegasusApiConfig().apply {
            onUnifiedCheckout = { _, _ ->
                throw ProblemDetailException(
                    ProblemDetail(status = 429, code = "rate_limit_exceeded", detail = "Too many requests"),
                )
            }
            onCreateProcurementOrder = { _, _, _ ->
                throw httpJsonException(429, """{"code":"rate_limit_exceeded","message":"Slow down"}""")
            }
        }
        val notices = mutableListOf<PendingOrderStatusNotice>()
        val processor = PendingOrderSyncProcessor(apiCfg.build(), dao, json) { notices.add(it) }

        val summary = processor.processAll()

        assertEquals(2, dao.rows.size)
        assertTrue(dao.deletedIds.isEmpty())
        assertTrue(dao.rows.all { it.status == PendingOrderEntity.STATUS_RATE_LIMITED })
        assertEquals(2, summary.parked)
        assertTrue(summary.shouldRetry)
        assertEquals(2, notices.size)
        assertTrue(notices.all { it.keptInQueue && it.status == PendingOrderEntity.STATUS_RATE_LIMITED })
    }

    @Test
    fun purgesTrueIdempotent409_andReplaysSecondaryProcurementEndpoints() = runTest {
        val dao = FakePendingOrderDao().apply {
            rows.add(checkoutEntity(id = 40L, key = "dup-checkout"))
            rows.add(procurementEntity(id = 41L, endpoint = "/v1/ai/preorder", key = "proc-key-41"))
            rows.add(
                PendingOrderEntity(
                    id = 42L,
                    endpoint = "/v1/orders/edit-preorder",
                    method = "POST",
                    payloadJson = """{"order_id":"ord-edit-42","items":[{"sku_id":"s1","quantity":6}]}""",
                    idempotencyKey = "proc-raw-42",
                ),
            )
        }
        val apiCfg = RecordingPegasusApiConfig().apply {
            onUnifiedCheckout = { _, _ ->
                throw httpJsonException(409, """{"code":"duplicate_order","message":"Already created"}""")
            }
        }
        val processor = PendingOrderSyncProcessor(apiCfg.build(), dao, json)

        val summary = processor.processAll()

        assertTrue(dao.rows.isEmpty())
        assertEquals(listOf(40L, 41L, 42L), dao.deletedIds)
        assertEquals(2, summary.succeeded)
        assertEquals(1, summary.purged)
        assertFalse(summary.shouldRetry)
        assertEquals("/v1/ai/preorder", apiCfg.procurementCalls.single().first)
        assertEquals("proc-key-41", apiCfg.procurementCalls.single().third)
        assertEquals("/v1/orders/edit-preorder", apiCfg.rawQueuedCalls.single().first)
        assertEquals("proc-raw-42", apiCfg.rawQueuedCalls.single().third)
    }

    @Test
    fun retainsOrderOnIoFailure_thenSucceedsAndClearsQueue() = runTest {
        val dao = FakePendingOrderDao().apply {
            rows.add(checkoutEntity(id = 50L))
        }
        var shouldFail = true
        val apiCfg = RecordingPegasusApiConfig().apply {
            onUnifiedCheckout = { _, _ ->
                if (shouldFail) throw IOException("No route to host")
                com.pegasusx.retailer.data.model.UnifiedCheckoutResponse(status = "CONFIRMED", invoiceId = "inv-50")
            }
        }
        val processor = PendingOrderSyncProcessor(apiCfg.build(), dao, json)

        val firstAttempt = processor.processAll()
        assertEquals(1, firstAttempt.retried)
        assertTrue(firstAttempt.shouldRetry)
        assertEquals(1, dao.rows.size)

        shouldFail = false
        val secondAttempt = processor.processAll()
        assertEquals(1, secondAttempt.succeeded)
        assertFalse(secondAttempt.shouldRetry)
        assertTrue(dao.rows.isEmpty())
    }
}
