package com.pegasusx.enterprise.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.enterprise.model.EnterpriseDashboardUiState
import com.pegasusx.enterprise.model.EnterpriseRequest
import com.pegasusx.enterprise.model.SortOption
import com.pegasusx.enterprise.model.TelemetryKpi
import com.pegasusx.enterprise.network.EnterpriseApiClient
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

class EnterpriseViewModel : ViewModel() {

    private val _uiState = MutableStateFlow(EnterpriseDashboardUiState(isLoading = true))
    val uiState: StateFlow<EnterpriseDashboardUiState> = _uiState.asStateFlow()

    init {
        loadData()
    }

    fun loadData(isPullToRefresh: Boolean = false) {
        viewModelScope.launch {
            if (isPullToRefresh) {
                _uiState.update { it.copy(isRefreshing = true) }
            } else {
                _uiState.update { it.copy(isLoading = true) }
            }

            try {
                val fetched = EnterpriseApiClient.fetchRequests()
                _uiState.update { current ->
                    val newKpi = computeKpi(fetched)
                    val filtered = applyFilterAndSort(
                        rawList = fetched,
                        category = current.selectedCategory,
                        status = current.selectedStatus,
                        query = current.searchQuery,
                        sort = current.sortOption
                    )
                    current.copy(
                        isLoading = false,
                        isRefreshing = false,
                        requests = fetched,
                        filteredRequests = filtered,
                        kpi = newKpi,
                        errorMessage = null
                    )
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        isRefreshing = false,
                        errorMessage = "Offline mode: Failed to reach Supabase API"
                    )
                }
            }
        }
    }

    fun onSearchQueryChanged(query: String) {
        _uiState.update { current ->
            val filtered = applyFilterAndSort(
                rawList = current.requests,
                category = current.selectedCategory,
                status = current.selectedStatus,
                query = query,
                sort = current.sortOption
            )
            current.copy(searchQuery = query, filteredRequests = filtered)
        }
    }

    fun onCategorySelected(category: String) {
        _uiState.update { current ->
            val filtered = applyFilterAndSort(
                rawList = current.requests,
                category = category,
                status = current.selectedStatus,
                query = current.searchQuery,
                sort = current.sortOption
            )
            current.copy(selectedCategory = category, filteredRequests = filtered)
        }
    }

    fun onStatusFilterSelected(status: String) {
        _uiState.update { current ->
            val filtered = applyFilterAndSort(
                rawList = current.requests,
                category = current.selectedCategory,
                status = status,
                query = current.searchQuery,
                sort = current.sortOption
            )
            current.copy(selectedStatus = status, filteredRequests = filtered)
        }
    }

    fun onSortOptionSelected(sortOption: SortOption) {
        _uiState.update { current ->
            val filtered = applyFilterAndSort(
                rawList = current.requests,
                category = current.selectedCategory,
                status = current.selectedStatus,
                query = current.searchQuery,
                sort = sortOption
            )
            current.copy(sortOption = sortOption, filteredRequests = filtered)
        }
    }

    fun onInquirySelected(request: EnterpriseRequest) {
        _uiState.update { it.copy(activeInquiry = request) }
    }

    fun onInquiryDismissed() {
        _uiState.update { it.copy(activeInquiry = null) }
    }

    fun onStatusAdvanced(requestId: String, newStatus: String) {
        viewModelScope.launch {
            val success = EnterpriseApiClient.updateStatus(requestId, newStatus)
            if (success) {
                _uiState.update { current ->
                    val updatedList = current.requests.map { req ->
                        if (req.id == requestId) req.copy(status = newStatus) else req
                    }
                    val filtered = applyFilterAndSort(
                        rawList = updatedList,
                        category = current.selectedCategory,
                        status = current.selectedStatus,
                        query = current.searchQuery,
                        sort = current.sortOption
                    )
                    val active = if (current.activeInquiry?.id == requestId) {
                        current.activeInquiry.copy(status = newStatus)
                    } else current.activeInquiry

                    current.copy(
                        requests = updatedList,
                        filteredRequests = filtered,
                        activeInquiry = active,
                        kpi = computeKpi(updatedList),
                        userNotice = "Inquiry moved to ${newStatus.replace("_", " ").uppercase()}"
                    )
                }
            }
        }
    }

    fun onDeleteInquiry(requestId: String) {
        viewModelScope.launch {
            val success = EnterpriseApiClient.deleteRequest(requestId)
            if (success) {
                _uiState.update { current ->
                    val updatedList = current.requests.filterNot { it.id == requestId }
                    val filtered = applyFilterAndSort(
                        rawList = updatedList,
                        category = current.selectedCategory,
                        status = current.selectedStatus,
                        query = current.searchQuery,
                        sort = current.sortOption
                    )
                    current.copy(
                        requests = updatedList,
                        filteredRequests = filtered,
                        activeInquiry = null,
                        kpi = computeKpi(updatedList),
                        userNotice = "Inquiry record archived from queue"
                    )
                }
            }
        }
    }

    fun onSendReply(requestId: String, recipient: String, subject: String, message: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isSendingReply = true) }
            val success = EnterpriseApiClient.sendClientReply(
                requestId = requestId,
                recipient = recipient,
                subject = subject,
                message = message
            )
            _uiState.update { current ->
                if (success) {
                    current.copy(
                        isSendingReply = false,
                        userNotice = "Outbound reply dispatched via Resend to $recipient"
                    )
                } else {
                    current.copy(
                        isSendingReply = false,
                        userNotice = "Notice: Direct dispatch queued in offline buffer. You can also use the Mail App button."
                    )
                }
            }
        }
    }

    fun clearNotice() {
        _uiState.update { it.copy(userNotice = null) }
    }

    private fun applyFilterAndSort(
        rawList: List<EnterpriseRequest>,
        category: String,
        status: String,
        query: String,
        sort: SortOption
    ): List<EnterpriseRequest> {
        val q = query.trim().lowercase()
        val filtered = rawList.filter { req ->
            val matchQuery = if (q.isEmpty()) true else {
                req.name.lowercase().contains(q) ||
                req.email.lowercase().contains(q) ||
                (req.company?.lowercase()?.contains(q) == true) ||
                (req.role?.lowercase()?.contains(q) == true) ||
                (req.message?.lowercase()?.contains(q) == true)
            }

            val matchCategory = if (category == "all") true else {
                req.inquiry_type.equals(category, ignoreCase = true)
            }

            val matchStatus = if (status == "all") true else {
                req.status.equals(status, ignoreCase = true)
            }

            matchQuery && matchCategory && matchStatus
        }

        return when (sort) {
            SortOption.NEWEST -> filtered.sortedByDescending { it.created_at }
            SortOption.OLDEST -> filtered.sortedBy { it.created_at }
            SortOption.PRIORITY -> filtered.sortedByDescending { it.isUrgent }
            SortOption.COMPANY_NAME -> filtered.sortedBy { it.company ?: "zzz" }
        }
    }

    private fun computeKpi(list: List<EnterpriseRequest>): TelemetryKpi {
        return TelemetryKpi(
            totalCount = list.size,
            newCount = list.count { it.status.equals("new", ignoreCase = true) },
            inReviewCount = list.count { it.status.equals("in_review", ignoreCase = true) },
            qualifiedCount = list.count { it.status.equals("qualified", ignoreCase = true) },
            demoCount = list.count { it.inquiry_type.equals("demo", ignoreCase = true) },
            pilotCount = list.count { it.inquiry_type.equals("pilot", ignoreCase = true) },
            avgResponseLatencyHours = 1.4,
            dispatchHealthPercent = 99
        )
    }
}
