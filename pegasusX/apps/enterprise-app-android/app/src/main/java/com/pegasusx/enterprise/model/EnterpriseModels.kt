package com.pegasusx.enterprise.model

import androidx.compose.runtime.Immutable
import kotlinx.serialization.Serializable

@Immutable
@Serializable
data class EnterpriseRequest(
    val id: String,
    val name: String,
    val email: String,
    val company: String? = null,
    val phone: String? = null,
    val role: String? = "Supplier Operations",
    val fleet_size: String? = null,
    val inquiry_type: String = "demo",
    val subject: String? = null,
    val message: String? = null,
    val status: String = "new",
    val priority: String = "normal",
    val admin_notes: String? = null,
    val source_page: String? = "website",
    val created_at: String,
    val updated_at: String? = null
) {
    val initials: String
        get() {
            val parts = name.trim().split("\\s+".toRegex())
            return when {
                parts.isEmpty() || parts[0].isEmpty() -> "P"
                parts.size == 1 -> parts[0].take(2).uppercase()
                else -> "${parts[0].first()}${parts[1].first()}".uppercase()
            }
        }

    val isUrgent: Boolean
        get() = priority.equals("urgent", ignoreCase = true) || 
                status.equals("new", ignoreCase = true) ||
                (message?.contains("urgent", ignoreCase = true) == true)
}

enum class SortOption(val label: String) {
    NEWEST("Newest Submissions"),
    OLDEST("Oldest Pending"),
    PRIORITY("Highest Priority"),
    COMPANY_NAME("Company (A-Z)")
}

@Immutable
data class TelemetryKpi(
    val totalCount: Int = 0,
    val newCount: Int = 0,
    val inReviewCount: Int = 0,
    val qualifiedCount: Int = 0,
    val demoCount: Int = 0,
    val pilotCount: Int = 0,
    val avgResponseLatencyHours: Double = 1.6,
    val dispatchHealthPercent: Int = 99
)

@Immutable
data class EnterpriseDashboardUiState(
    val isLoading: Boolean = false,
    val isRefreshing: Boolean = false,
    val requests: List<EnterpriseRequest> = emptyList(),
    val filteredRequests: List<EnterpriseRequest> = emptyList(),
    val selectedCategory: String = "all",
    val selectedStatus: String = "all",
    val searchQuery: String = "",
    val sortOption: SortOption = SortOption.NEWEST,
    val activeInquiry: EnterpriseRequest? = null,
    val isSendingReply: Boolean = false,
    val userNotice: String? = null,
    val errorMessage: String? = null,
    val kpi: TelemetryKpi = TelemetryKpi()
)

@Serializable
data class EnterpriseApiResponse<T>(
    val success: Boolean,
    val data: T? = null,
    val count: Int? = null,
    val message: String? = null,
    val error: String? = null
)

@Serializable
data class ClientReplyPayload(
    val request_id: String,
    val recipient: String,
    val subject: String,
    val message: String
)

@Serializable
data class StatusUpdatePayload(
    val id: String,
    val status: String,
    val admin_notes: String? = null
)
