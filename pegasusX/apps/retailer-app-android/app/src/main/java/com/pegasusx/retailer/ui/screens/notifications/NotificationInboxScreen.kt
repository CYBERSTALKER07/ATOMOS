package com.pegasusx.retailer.ui.screens.notifications

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.AccountBalanceWallet
import androidx.compose.material.icons.outlined.CheckCircle
import androidx.compose.material.icons.outlined.DoneAll
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.Inventory2
import androidx.compose.material.icons.outlined.LocalShipping
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Payments
import androidx.compose.material.icons.outlined.Place
import androidx.compose.material.icons.outlined.SwapHoriz
import androidx.compose.material.icons.outlined.SyncAlt
import androidx.compose.material.icons.outlined.Update
import androidx.compose.material.icons.outlined.Verified
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
fun NotificationInboxScreen(
    onBack: () -> Unit,
    viewModel: NotificationInboxViewModel = hiltViewModel(),
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
                IconButton(onClick = { viewModel.refresh() }) {
                    if (state.isRefreshing) {
                        CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                    } else {
                        Icon(Icons.Outlined.SyncAlt, contentDescription = "Refresh notifications")
                    }
                }
                if (state.unreadCount > 0) {
                    TextButton(onClick = { viewModel.markAllRead() }) {
                        Icon(
                            Icons.Outlined.DoneAll,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp),
                        )
                        Text(
                            "Read all",
                            modifier = Modifier.padding(start = 4.dp),
                            style = MaterialTheme.typography.labelMedium,
                        )
                    }
                }
            },
            colors = TopAppBarDefaults.topAppBarColors(
                containerColor = MaterialTheme.colorScheme.surface,
            ),
        )

        if (state.loadIssue != null || state.isRefreshing) {
            val loadIssue = state.loadIssue
            val issueMessage = when {
                loadIssue != null -> state.error ?: state.syncMessage.orEmpty()
                else -> "Syncing notifications..."
            }
            val containerColor = when (loadIssue) {
                NotificationLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
                NotificationLoadIssue.OFFLINE -> MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.5f)
                NotificationLoadIssue.ERROR -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.35f)
                null -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
            }
            val contentColor = when (loadIssue) {
                NotificationLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.onErrorContainer
                NotificationLoadIssue.OFFLINE -> MaterialTheme.colorScheme.onTertiaryContainer
                NotificationLoadIssue.ERROR -> MaterialTheme.colorScheme.onErrorContainer
                null -> MaterialTheme.colorScheme.onPrimaryContainer
            }

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(containerColor)
                    .padding(horizontal = 12.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = issueMessage,
                    modifier = Modifier.weight(1f),
                    style = MaterialTheme.typography.labelMedium,
                    color = contentColor,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                if (loadIssue != null) {
                    TextButton(onClick = { viewModel.refresh() }) {
                        Text("Retry", color = contentColor)
                    }
                }
            }
        }

        when {
            state.loading && state.items.isEmpty() -> {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            }
            state.items.isEmpty() -> {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Icon(
                            Icons.Outlined.Notifications,
                            contentDescription = null,
                            modifier = Modifier.size(48.dp),
                            tint = MaterialTheme.colorScheme.outlineVariant,
                        )
                        Text(
                            when (state.loadIssue) {
                                NotificationLoadIssue.RESTRICTED -> "Notifications access restricted"
                                NotificationLoadIssue.OFFLINE -> "Notifications are offline"
                                NotificationLoadIssue.ERROR -> "Notifications unavailable"
                                null -> "No notifications yet"
                            },
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 12.dp),
                        )
                    }
                }
            }
            else -> {
                LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        modifier = Modifier.fillMaxSize()
    ) {
                    items(state.items, key = { it.id }) { notif ->
                        NotificationRow(
                            notification = notif,
                            onClick = { if (notif.readAt == null) viewModel.markRead(notif.id) },
                        )
                        HorizontalDivider()
                    }
                    if (state.hasMore) {
                        item(key = "inbox-load-more-sentinel") {
                            androidx.compose.runtime.LaunchedEffect(state.items.size, state.isLoadingMore) {
                                if (!state.isLoadingMore) {
                                    viewModel.loadMore()
                                }
                            }
                            if (state.isLoadingMore) {
                                Box(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(16.dp),
                                    contentAlignment = Alignment.Center,
                                ) {
                                    CircularProgressIndicator(modifier = Modifier.size(24.dp), strokeWidth = 2.dp)
                                }
                            } else {
                                Spacer(modifier = Modifier.height(1.dp))
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun NotificationRow(
    notification: NotificationItem,
    onClick: () -> Unit,
) {
    val isUnread = notification.readAt == null
    val bg = if (isUnread) {
        MaterialTheme.colorScheme.surfaceContainerHigh
    } else {
        MaterialTheme.colorScheme.surface
    }
    val icon = typeIcon(notification.type)
    val iconTint = if (isUnread) {
        MaterialTheme.colorScheme.primary
    } else {
        MaterialTheme.colorScheme.outline
    }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(bg)
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Icon(
            icon,
            contentDescription = null,
            tint = iconTint,
            modifier = Modifier
                .size(24.dp)
                .padding(top = 2.dp),
        )
        Column(modifier = Modifier.weight(1f)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text(
                    notification.title,
                    style = MaterialTheme.typography.labelLarge.copy(
                        fontWeight = if (isUnread) FontWeight.SemiBold else FontWeight.Normal,
                        color = if (isUnread) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.outline,
                    ),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                Text(
                    timeAgo(notification.createdAt),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Text(
                notification.body,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.padding(top = 2.dp),
            )
        }
        if (isUnread) {
            Box(
                modifier = Modifier
                    .size(8.dp)
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primary)
                    .align(Alignment.CenterVertically),
            )
        }
    }
}

private fun typeIcon(type: String): ImageVector = when (type) {
    "ORDER_DISPATCHED" -> Icons.Outlined.LocalShipping
    "DRIVER_ARRIVED" -> Icons.Outlined.Place
    "ORDER_STATUS_CHANGED" -> Icons.Outlined.SyncAlt
    "ORDER_REASSIGNED" -> Icons.Outlined.SwapHoriz
    "PAYLOAD_READY_TO_SEAL" -> Icons.Outlined.Inventory2
    "PAYLOAD_SEALED" -> Icons.Outlined.Verified
    "SETTLEMENT_REQUIRED" -> Icons.Outlined.AccountBalanceWallet
    "DELIVERY_SESSION_UPDATED" -> Icons.Outlined.Update
    "PAYMENT_SETTLED", "GLOBAL_PAYNT_SETTLED" -> Icons.Outlined.Payments
    "PAYMENT_FAILED", "PAYMENT_EXPIRED", "GLOBAL_PAYNT_FAILED", "GLOBAL_PAYNT_EXPIRED" -> Icons.Outlined.ErrorOutline
    "ORDER_COMPLETED" -> Icons.Outlined.CheckCircle
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
    } catch (_: Exception) {
        ""
    }
}
