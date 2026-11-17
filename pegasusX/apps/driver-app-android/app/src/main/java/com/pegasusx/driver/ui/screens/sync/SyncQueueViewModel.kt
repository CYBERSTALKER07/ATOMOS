package com.pegasusx.driver.ui.screens.sync

import android.content.Context
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.driver.data.local.PendingMutationDao
import com.pegasusx.driver.data.model.PendingMutationEntity
import com.pegasusx.driver.data.model.SyncEscalationItem
import com.pegasusx.driver.data.model.SyncEscalationRequest
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.services.OfflineSyncScheduler
import com.pegasusx.driver.util.DriverIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

@HiltViewModel
class SyncQueueViewModel @Inject constructor(
    dao: PendingMutationDao,
    api: DriverApi,
    @ApplicationContext private val appContext: Context,
) : ViewModel() {

    private val coordinator = SyncQueueCoordinator(
        dao = dao,
        escalateFn = { req, key -> api.escalateSyncDeadLetters(request = req, idempotencyKey = key) },
        scheduleFlush = { OfflineSyncScheduler.enqueue(appContext) },
    )

    val pending: StateFlow<List<PendingMutationEntity>> = dao.observePending()
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    val dead: StateFlow<List<PendingMutationEntity>> = dao.observeDead()
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    private val _flushing = MutableStateFlow(false)
    val flushing: StateFlow<Boolean> = _flushing.asStateFlow()

    private val _escalating = MutableStateFlow(false)
    val escalating: StateFlow<Boolean> = _escalating.asStateFlow()

    private val _statusMessage = MutableStateFlow<String?>(null)
    val statusMessage: StateFlow<String?> = _statusMessage.asStateFlow()

    fun flushNow() {
        viewModelScope.launch {
            _flushing.value = true
            _statusMessage.value = coordinator.flushNow()
            _flushing.value = false
        }
    }

    fun retryDeadItem(row: PendingMutationEntity) {
        viewModelScope.launch {
            _statusMessage.value = coordinator.retryDeadItem(row)
        }
    }

    fun escalateDeadToSupport() {
        viewModelScope.launch {
            if (_escalating.value) return@launch
            _escalating.value = true
            try {
                val msg = coordinator.escalateDeadToSupport()
                if (msg != null) {
                    _statusMessage.value = msg
                }
            } finally {
                _escalating.value = false
            }
        }
    }
}

internal class SyncQueueCoordinator(
    private val dao: PendingMutationDao,
    private val escalateFn: suspend (SyncEscalationRequest, String) -> com.pegasusx.driver.data.model.SyncEscalationResponse,
    private val scheduleFlush: () -> Unit,
) {
    fun flushNow(): String {
        scheduleFlush()
        return "Flush scheduled"
    }

    suspend fun retryDeadItem(row: PendingMutationEntity): String {
        dao.requeueDeadById(row.id)
        scheduleFlush()
        return "Re-queued ${row.endpoint} for immediate retry."
    }

    suspend fun escalateDeadToSupport(): String? {
        val itemsToEscalate = dao.getDead()
        if (itemsToEscalate.isEmpty()) return null
        return try {
            val request = SyncEscalationRequest(
                items = itemsToEscalate.map { row ->
                    SyncEscalationItem(
                        id = row.id,
                        endpoint = row.endpoint,
                        orderId = row.orderId.ifBlank { null },
                        payloadJson = row.payloadJson,
                        lastError = row.lastError.ifBlank { null },
                        attemptCount = row.attemptCount,
                        clientTimestampIso = row.clientTimestampIso.ifBlank { null },
                    )
                },
            )
            val resp = escalateFn(
                request,
                DriverIdempotencyKeys.syncEscalation(itemsToEscalate.map { it.id }),
            )
            if (resp.ok) {
                dao.deleteByIds(itemsToEscalate.map { it.id })
                val ref = resp.escalationId.ifBlank { "recorded" }
                "Escalated ${itemsToEscalate.size} item(s) to Support Review ($ref)."
            } else {
                "Support escalation not confirmed; local items preserved."
            }
        } catch (e: Exception) {
            "Could not escalate to Support Review (${e.message ?: "network error"}). Dead-letter records kept safely."
        }
    }
}
