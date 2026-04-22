package com.thelab.driver.ui.screens.notifications

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.DoneAll
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.LocalShipping
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Payments
import androidx.compose.material.icons.outlined.Place
import androidx.compose.material.icons.outlined.SyncAlt
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import java.time.Duration
import java.time.Instant

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DriverNotificationInboxScreen(
    onBack: () -> Unit,
    viewModel: DriverNotificationInboxViewModel = hiltViewModel(),
) {
    val state by viewModel.uiState.collectAsState()

    Column(modifier = Modifier.fillMaxSize()) {
        TopAppBar(
            title = {
                Text(
                    "Notifications",
                    style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                )
            },
            navigationIcon = {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Outlined.ArrowBack, "Back")
                }
            },
            actions = {
                if (state.unreadCount > 0) {
                    TextButton(onClick = { viewModel.markAllRead() }) {
                        Icon(Icons.Outlined.DoneAll, null, modifier = Modifier.size(18.dp))
                        Text("Read all", modifier = Modifier.padding(start = 4.dp), style = MaterialTheme.typography.labelMedium)
                    }
                }
            },
            colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.surface),
        )

        when {
            state.loading -> {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            }
            state.items.isEmpty() -> {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Icon(Icons.Outlined.Notifications, null, Modifier.size(48.dp), tint = MaterialTheme.colorScheme.outlineVariant)
                        Text("No notifications yet", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 12.dp))
                    }
                }
            }
            else -> {
                LazyColumn(modifier = Modifier.fillMaxSize()) {
                    items(state.items, key = { it.id }) { notif ->
                        DriverNotificationRow(notif) { if (notif.readAt == null) viewModel.markRead(notif.id) }
                        HorizontalDivider()
                    }
                }
            }
        }
    }
}

@Composable
private fun DriverNotificationRow(notification: DriverNotificationItem, onClick: () -> Unit) {
    val isUnread = notification.readAt == null
    val bg = if (isUnread) MaterialTheme.colorScheme.surfaceContainerHigh else MaterialTheme.colorScheme.surface
    val icon = typeIcon(notification.type)
    val iconTint = if (isUnread) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline

    Row(
        modifier = Modifier.fillMaxWidth().background(bg).clickable(onClick = onClick).padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Icon(icon, null, tint = iconTint, modifier = Modifier.size(24.dp).padding(top = 2.dp))
        Column(modifier = Modifier.weight(1f)) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(notification.title, style = MaterialTheme.typography.labelLarge.copy(fontWeight = if (isUnread) FontWeight.SemiBold else FontWeight.Normal, color = if (isUnread) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.outline), maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f, fill = false))
                Text(timeAgo(notification.createdAt), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Text(notification.body, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 2, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(top = 2.dp))
        }
        if (isUnread) {
            Box(Modifier.size(8.dp).clip(CircleShape).background(MaterialTheme.colorScheme.primary).align(Alignment.CenterVertically))
        }
    }
}

private fun typeIcon(type: String): ImageVector = when (type) {
    "ORDER_DISPATCHED" -> Icons.Outlined.LocalShipping
    "DRIVER_ARRIVED" -> Icons.Outlined.Place
    "ORDER_STATUS_CHANGED" -> Icons.Outlined.SyncAlt
    "PAYMENT_SETTLED" -> Icons.Outlined.Payments
    "PAYMENT_FAILED" -> Icons.Outlined.ErrorOutline
    else -> Icons.Outlined.Notifications
}

private fun timeAgo(iso: String): String {
    return try {
        val then = Instant.parse(iso)
        val diff = Duration.between(then, Instant.now())
        val mins = diff.toMinutes()
        when {
            mins < 1 -> "now"
            mins < 60 -> "${mins}m"
            mins < 1440 -> "${mins / 60}h"
            else -> "${mins / 1440}d"
        }
    } catch (_: Exception) { "" }
}
