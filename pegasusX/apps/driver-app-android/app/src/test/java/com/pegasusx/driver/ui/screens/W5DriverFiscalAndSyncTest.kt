package com.pegasusx.driver.ui.screens

import com.pegasusx.driver.data.local.PendingMutationDao
import com.pegasusx.driver.data.model.OpenFiscalResponse
import com.pegasusx.driver.data.model.PendingMutationEntity
import com.pegasusx.driver.data.model.SyncEscalationRequest
import com.pegasusx.driver.data.model.SyncEscalationResponse
import com.pegasusx.driver.ui.screens.manifest.ManifestUiState
import com.pegasusx.driver.ui.screens.manifest.applyOpenFiscalSnapshot
import com.pegasusx.driver.ui.screens.offload.FISCAL_RETRY_QUEUED_GUIDANCE
import com.pegasusx.driver.ui.screens.offload.PaymentWaitingUiState
import com.pegasusx.driver.ui.screens.offload.continuePaymentWaitingNextStop
import com.pegasusx.driver.ui.screens.offload.reducePaymentWaitingFiscalStatus
import com.pegasusx.driver.ui.screens.sync.SyncQueueCoordinator
import com.pegasusx.driver.util.DriverIdempotencyKeys
import com.pegasusx.driver.util.DriverLocaleManager
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class W5DriverFiscalAndSyncTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun localeManager_defaultsToUzbekAndNormalizesSupportedCodes() {
        assertEquals("uz", DriverLocaleManager.DEFAULT_LOCALE)
        assertEquals("uz", DriverLocaleManager.normalizeLocaleTag(null))
        assertEquals("uz", DriverLocaleManager.normalizeLocaleTag("fr"))
        assertEquals("ru", DriverLocaleManager.normalizeLocaleTag(" RU "))
        assertEquals("en", DriverLocaleManager.normalizeLocaleTag("EN"))
        assertEquals(listOf("uz", "ru", "en"), DriverLocaleManager.supportedLanguages.map { it.code })
        assertEquals("O'zbekcha", DriverLocaleManager.displayName("uz"))
        assertEquals("Русский", DriverLocaleManager.displayName("ru"))
        assertEquals("English", DriverLocaleManager.displayName("en"))
    }

    @Test
    fun openFiscal_retryQueuedDoesNotFreezeCashBag() {
        val raw = """
            {
              "open_fiscal_count": 2,
              "retry_queued_count": 2,
              "fiscal_issue_count": 0,
              "order_ids": ["ord-1", "ord-2"],
              "retry_order_ids": ["ord-1", "ord-2"],
              "cash_bag_frozen": false
            }
        """.trimIndent()
        val snap = json.decodeFromString<OpenFiscalResponse>(raw)
        val state = applyOpenFiscalSnapshot(ManifestUiState(), snap)

        assertEquals(2L, state.openFiscalCount)
        assertEquals(2L, state.retryQueuedCount)
        assertEquals(0L, state.fiscalIssueCount)
        assertFalse("Retry-queued fiscal must not block cash-bag close", state.cashBagFrozen)

        val lockedSnap = json.decodeFromString<OpenFiscalResponse>(
            """{"open_fiscal_count":1,"retry_queued_count":0,"fiscal_issue_count":1,"cash_bag_frozen":true}"""
        )
        val lockedState = applyOpenFiscalSnapshot(ManifestUiState(), lockedSnap)
        assertTrue(lockedState.cashBagFrozen)
        assertEquals(1L, lockedState.fiscalIssueCount)
    }

    @Test
    fun paymentWaiting_nonBlockingFiscalRetryQueuedAllowsContinuingToNextStop() {
        val initial = PaymentWaitingUiState(orderId = "ORD-42", amount = 150_000_00)
        val queued = reducePaymentWaitingFiscalStatus(initial, "FISCAL_RETRY_QUEUED")

        assertTrue(queued.paymentSettled)
        assertFalse(queued.fiscalizing)
        assertTrue(queued.fiscalFailed)
        assertNull(queued.error)
        assertNotNull(queued.statusNotice)
        assertFalse(queued.statusNotice!!.contains("call supervisor", ignoreCase = true))
        assertTrue(queued.statusNotice!!.contains("Soliq", ignoreCase = true))

        val issue = reducePaymentWaitingFiscalStatus(initial, "PENDING_FISCAL_ISSUE")
        assertTrue(issue.fiscalFailed)
        assertNull(issue.error)

        val nextStop = continuePaymentWaitingNextStop(queued)
        assertTrue("Driver must be able to continue to next stop without supervisor call", nextStop.completed)
        assertFalse(nextStop.fiscalizing)

        assertFalse(FISCAL_RETRY_QUEUED_GUIDANCE.contains("call supervisor", ignoreCase = true))
    }

    @Test
    fun syncQueue_escalatePreservesDeadRowsWhenServerFailsOrRejects() = runTest {
        val row = deadMutation("mut-1", "v1/order/deliver", "ORD-1")
        val dao = FakePendingMutationDao(listOf(row))
        var flushCount = 0

        val failingCoordinator = SyncQueueCoordinator(
            dao = dao,
            escalateFn = { _, _ -> throw RuntimeException("offline") },
            scheduleFlush = { flushCount++ },
        )
        val errorMsg = failingCoordinator.escalateDeadToSupport()
        assertNotNull(errorMsg)
        assertTrue(errorMsg!!.contains("kept safely", ignoreCase = true))
        assertEquals(1, dao.getDead().size)
        assertFalse(dao.clearDeadCalled)
        assertEquals(0, flushCount)

        val rejectedCoordinator = SyncQueueCoordinator(
            dao = dao,
            escalateFn = { _, _ -> SyncEscalationResponse(ok = false, escalated = 0, escalationId = "") },
            scheduleFlush = { flushCount++ },
        )
        val rejectedMsg = rejectedCoordinator.escalateDeadToSupport()
        assertTrue(rejectedMsg!!.contains("preserved", ignoreCase = true))
        assertEquals(1, dao.getDead().size)
    }

    @Test
    fun syncQueue_escalateDeletesOnlyAfterConfirmationAndRetryRequeuesItem() = runTest {
        val row1 = deadMutation("mut-a", "v1/order/collect-cash", "ORD-101")
        val row2 = deadMutation("mut-b", "v1/order/deliver", "ORD-102")
        val dao = FakePendingMutationDao(listOf(row1, row2))
        var capturedRequest: SyncEscalationRequest? = null
        var capturedKey: String? = null
        var flushCount = 0

        val coordinator = SyncQueueCoordinator(
            dao = dao,
            escalateFn = { req, key ->
                // Dead rows must still be intact before confirmation returns.
                assertEquals(1, dao.getDead().size)
                capturedRequest = req
                capturedKey = key
                SyncEscalationResponse(ok = true, escalated = 1, escalationId = "esc-99")
            },
            scheduleFlush = { flushCount++ },
        )

        val retryMsg = coordinator.retryDeadItem(row1)
        assertTrue(retryMsg.contains("v1/order/collect-cash"))
        assertEquals(1, flushCount)
        assertEquals(1, dao.getPending().size)
        assertEquals(1, dao.getDead().size)

        val escalateMsg = coordinator.escalateDeadToSupport()
        assertTrue(escalateMsg!!.contains("esc-99"))
        assertEquals(listOf("mut-b"), capturedRequest?.items?.map { it.id })
        assertEquals(DriverIdempotencyKeys.syncEscalation(listOf("mut-b")), capturedKey)
        assertEquals(0, dao.getDead().size)
        assertEquals(1, dao.getPending().size)
        assertFalse("Legacy destructive clearDead must never be used", dao.clearDeadCalled)
    }

    @Test
    fun syncEscalationWireContract_serializesSnakeCaseFields() {
        val item = com.pegasusx.driver.data.model.SyncEscalationItem(
            id = "m-1",
            endpoint = "v1/order/deliver",
            orderId = "ORD-77",
            payloadJson = """{"cash":100}""",
            lastError = "HTTP 500",
            attemptCount = 5,
            clientTimestampIso = "2026-11-13T03:00:00Z",
        )
        val encoded = json.encodeToString(SyncEscalationRequest(listOf(item)))
        assertTrue(encoded.contains("\"order_id\":\"ORD-77\""))
        assertTrue(encoded.contains("\"payload_json\""))
        assertTrue(encoded.contains("\"last_error\":\"HTTP 500\""))
        assertTrue(encoded.contains("\"attempt_count\":5"))
        assertTrue(encoded.contains("\"client_timestamp_iso\":\"2026-11-13T03:00:00Z\""))
    }

    private fun deadMutation(id: String, endpoint: String, orderId: String) = PendingMutationEntity(
        id = id,
        endpoint = endpoint,
        payloadJson = """{"order_id":"$orderId"}""",
        idempotencyKey = "idem-$id",
        attemptCount = 5,
        lastError = "timeout",
        status = "DEAD",
        orderId = orderId,
        clientTimestampIso = "2026-11-13T01:00:00Z",
    )

    private class FakePendingMutationDao(initial: List<PendingMutationEntity>) : PendingMutationDao {
        private val rows = MutableStateFlow(initial)
        var clearDeadCalled = false

        override suspend fun getPending(): List<PendingMutationEntity> =
            rows.value.filter { it.status == "PENDING" }

        override suspend fun getDead(): List<PendingMutationEntity> =
            rows.value.filter { it.status == "DEAD" }

        override suspend fun getAll(): List<PendingMutationEntity> = rows.value

        override fun observePendingCount(): Flow<Int> = rows.map { list -> list.count { it.status == "PENDING" } }

        override fun observeCount(): Flow<Int> = rows.map { it.size }

        override fun observePending(): Flow<List<PendingMutationEntity>> =
            rows.map { list -> list.filter { it.status == "PENDING" } }

        override fun observeDead(): Flow<List<PendingMutationEntity>> =
            rows.map { list -> list.filter { it.status == "DEAD" } }

        override suspend fun insert(mutation: PendingMutationEntity) {
            rows.value = rows.value.filterNot { it.id == mutation.id } + mutation
        }

        override suspend fun deleteById(id: String) {
            rows.value = rows.value.filterNot { it.id == id }
        }

        override suspend fun deleteByIdempotencyKey(key: String) {
            rows.value = rows.value.filterNot { it.idempotencyKey == key }
        }

        override suspend fun recordAttempt(id: String, error: String) {
            rows.value = rows.value.map {
                if (it.id == id) it.copy(attemptCount = it.attemptCount + 1, lastError = error) else it
            }
        }

        override suspend fun markDead(id: String, error: String) {
            rows.value = rows.value.map {
                if (it.id == id) it.copy(status = "DEAD", lastError = error) else it
            }
        }

        override suspend fun requeueDeadById(id: String) {
            rows.value = rows.value.map {
                if (it.id == id) it.copy(status = "PENDING", attemptCount = 0, lastError = "") else it
            }
        }

        override suspend fun deleteByIds(ids: List<String>) {
            val idSet = ids.toSet()
            rows.value = rows.value.filterNot { it.id in idSet }
        }

        @Deprecated("test")
        override suspend fun clearDead() {
            clearDeadCalled = true
            rows.value = rows.value.filterNot { it.status == "DEAD" }
        }
    }
}
