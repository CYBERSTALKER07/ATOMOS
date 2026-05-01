package com.pegasus.retailer.ui.screens.notifications

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import javax.inject.Inject

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
)

data class NotificationInboxState(
    val items: List<NotificationItem> = emptyList(),
    val unreadCount: Int = 0,
    val loading: Boolean = true,
)

@HiltViewModel
class NotificationInboxViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(NotificationInboxState())
    val uiState: StateFlow<NotificationInboxState> = _uiState.asStateFlow()

    init {
        loadNotifications()
    }

    private fun loadNotifications() {
        viewModelScope.launch {
            try {
                val resp = api.getNotifications(limit = 50)
                _uiState.update {
                    it.copy(
                        items = resp.notifications,
                        unreadCount = resp.unreadCount,
                        loading = false,
                    )
                }
            } catch (_: Exception) {
                _uiState.update { it.copy(loading = false) }
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
                    )
                }
            } catch (_: Exception) { /* best effort */ }
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
                    )
                }
            } catch (_: Exception) { /* best effort */ }
        }
    }
}
