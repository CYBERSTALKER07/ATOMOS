package com.pegasusx.supplier.ui.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.supplier.data.model.JoinRequestItem
import com.pegasusx.supplier.data.remote.MigrationRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class JoinRequestsUiState(
    val statusFilter: String = "PENDING",
    val requests: List<JoinRequestItem> = emptyList(),
    val rejectionReasonById: Map<String, String> = emptyMap(),
    val loading: Boolean = true,
    val busyId: String? = null,
    val error: String? = null,
    val notice: String? = null,
)

@HiltViewModel
class JoinRequestsViewModel @Inject constructor(
    private val repo: MigrationRepository,
    private val realtimeSignals: SupplierRealtimeSignals,
) : ViewModel() {
    private val _state = MutableStateFlow(JoinRequestsUiState())
    val state: StateFlow<JoinRequestsUiState> = _state.asStateFlow()

    init {
        load()
        viewModelScope.launch {
            realtimeSignals.refreshTick.collect { load(silent = true) }
        }
        viewModelScope.launch {
            realtimeSignals.reconnectTick.collect { load(silent = true) }
        }
    }

    fun setStatusFilter(status: String) {
        _state.update { it.copy(statusFilter = status, notice = null) }
        load()
    }

    fun setRejectionReason(requestId: String, reason: String) {
        _state.update {
            it.copy(rejectionReasonById = it.rejectionReasonById + (requestId to reason))
        }
    }

    fun load(silent: Boolean = false) {
        val status = _state.value.statusFilter
        viewModelScope.launch {
            if (!silent && _state.value.requests.isEmpty()) {
                _state.update { it.copy(loading = true, error = null) }
            } else if (!silent) {
                _state.update { it.copy(error = null) }
            }
            try {
                val resp = repo.listJoinRequests(status = status)
                if (resp.isSuccessful) {
                    _state.update {
                        it.copy(
                            requests = resp.body()?.requests.orEmpty(),
                            loading = false,
                        )
                    }
                } else if (!silent || _state.value.requests.isEmpty()) {
                    _state.update {
                        it.copy(loading = false, error = "Failed to load join requests (${resp.code()})")
                    }
                } else {
                    _state.update { it.copy(loading = false) }
                }
            } catch (e: Exception) {
                if (!silent || _state.value.requests.isEmpty()) {
                    _state.update { it.copy(loading = false, error = e.message ?: "Network error") }
                } else {
                    _state.update { it.copy(loading = false) }
                }
            }
        }
    }

    fun decide(requestId: String, approve: Boolean) {
        if (_state.value.busyId != null) return
        val reason = _state.value.rejectionReasonById[requestId].orEmpty().trim()
        val key = SupplierIdempotencyKeys.joinRequestDecide(
            requestId = requestId,
            decision = if (approve) "approve" else "reject",
        )
        viewModelScope.launch {
            _state.update { it.copy(busyId = requestId, error = null, notice = null) }
            try {
                val resp = if (approve) {
                    repo.approveJoinRequest(requestId, reason, key)
                } else {
                    repo.rejectJoinRequest(requestId, reason, key)
                }
                if (resp.isSuccessful) {
                    _state.update { it.copy(busyId = null, notice = "Decision recorded") }
                    load(silent = true)
                } else {
                    _state.update { it.copy(busyId = null, error = "Decision failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(busyId = null, error = e.message ?: "Decision failed") }
            }
        }
    }
}
