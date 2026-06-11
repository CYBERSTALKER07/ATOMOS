package com.pegasusx.retailer.ui.screens.notifications

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import retrofit2.HttpException
import javax.inject.Inject

private const val inboxPageSize = 50

@Serializable
data class NotificationItem(
    @SerialName("id") val id: String,
    @SerialName("type") val type: String = "",
    @SerialName("title") val title: String = "",
    @SerialName("body") val body: String = "",
    @SerialName("payload") val payload: String = "",
    @SerialName("channel") val channel: String = "",
    @SerialName("read_at") val readAt: String? = null,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class NotificationsResponse(
    @SerialName("notifications") val notifications: List<NotificationItem> = emptyList(),
    @SerialName("unread_count") val unreadCount: Int = 0,
    @SerialName("has_more") val hasMore: Boolean = false,
)

enum class NotificationLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class NotificationInboxState(
    val items: List<NotificationItem> = emptyList(),
    val unreadCount: Int = 0,
    val loading: Boolean = true,
    val isRefreshing: Boolean = false,
    val isLoadingMore: Boolean = false,
    val hasMore: Boolean = false,
    val error: String? = null,
    val loadIssue: NotificationLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            NotificationLoadIssue.RESTRICTED -> "Notifications access is restricted for this account"
            NotificationLoadIssue.OFFLINE -> "Offline mode active. Showing latest cached notifications"
            NotificationLoadIssue.ERROR -> "Notification sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class NotificationInboxViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(NotificationInboxState())
    val uiState: StateFlow<NotificationInboxState> = _uiState.asStateFlow()
    private var nextOffset = 0

    init {
        loadNotifications(initial = true)
    }

    fun refresh() {
        loadNotifications(initial = false, reset = true)
    }

    fun loadMore() {
        val state = _uiState.value
        if (state.loading || state.isRefreshing || state.isLoadingMore || !state.hasMore) {
            return
        }
        viewModelScope.launch {
            _uiState.update { it.copy(isLoadingMore = true, error = null) }
            try {
                val page = api.getNotifications(limit = inboxPageSize, offset = nextOffset)
                nextOffset += page.notifications.size
                _uiState.update {
                    it.copy(
                        items = it.items + page.notifications,
                        unreadCount = page.unreadCount,
                        hasMore = page.hasMore,
                        isLoadingMore = false,
                        loadIssue = null,
                    )
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isLoadingMore = false,
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun loadNotifications(initial: Boolean, reset: Boolean = true) {
        viewModelScope.launch {
            if (reset) {
                nextOffset = 0
            }
            _uiState.update {
                it.copy(
                    loading = if (initial) true else it.loading,
                    isRefreshing = if (initial) false else true,
                    error = null,
                )
            }
            try {
                val page = api.getNotifications(limit = inboxPageSize, offset = 0)
                nextOffset = page.notifications.size
                _uiState.update {
                    it.copy(
                        items = page.notifications,
                        unreadCount = page.unreadCount,
                        hasMore = page.hasMore,
                        loading = false,
                        isRefreshing = false,
                        error = null,
                        loadIssue = null,
                    )
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        loading = false,
                        isRefreshing = false,
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun markRead(notificationId: String) {
        viewModelScope.launch {
            try {
                api.markNotificationsRead(mapOf("notification_ids" to listOf(notificationId)))
                _uiState.update { state ->
                    state.copy(
                        items = state.items.map { n ->
                            if (n.id == notificationId) n.copy(readAt = "now") else n
                        },
                        unreadCount = (state.unreadCount - 1).coerceAtLeast(0),
                        error = null,
                        loadIssue = null,
                    )
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun markAllRead() {
        viewModelScope.launch {
            try {
                api.markNotificationsRead(mapOf("mark_all" to true))
                _uiState.update { state ->
                    state.copy(
                        items = state.items.map { it.copy(readAt = it.readAt ?: "now") },
                        unreadCount = 0,
                        error = null,
                        loadIssue = null,
                    )
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): NotificationLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> NotificationLoadIssue.RESTRICTED
            error is IOException -> NotificationLoadIssue.OFFLINE
            else -> NotificationLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: NotificationLoadIssue): String {
        return when (issue) {
            NotificationLoadIssue.RESTRICTED -> "Notifications access is restricted for this account"
            NotificationLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            NotificationLoadIssue.ERROR -> error.message ?: "Notification request failed"
        }
    }
}
