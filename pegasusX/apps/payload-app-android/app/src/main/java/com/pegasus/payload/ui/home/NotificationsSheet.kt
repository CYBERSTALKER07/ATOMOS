package com.pegasus.payload.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import com.pegasus.payload.data.model.NotificationItem
import com.pegasus.payload.ui.components.HandoffInboxCard
import com.pegasus.design.PegasusStatePane
import com.pegasus.design.PegasusStateKind

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NotificationsSheet(
    items: List<NotificationItem>,
    unreadCount: Int,
    onDismiss: () -> Unit,
    onMarkRead: (String) -> Unit,
    onMarkAllRead: () -> Unit,
    onHandoffAction: (String) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 8.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text("Notifications", style = MaterialTheme.typography.titleLarge)
                if (unreadCount > 0) {
                    TextButton(onClick = onMarkAllRead) { Text("Mark all read") }
                }
            }
            HorizontalDivider()
            if (items.isEmpty()) {
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No notifications",
                    body = "New events will appear here in real time.",
                    modifier = Modifier.fillMaxWidth().heightIn(min = 200.dp),
                )
            } else {
                LazyVerticalGrid(
                    columns = GridCells.Adaptive(minSize = 340.dp),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    items(items, key = { it.notificationId }) { n ->
                        NotificationRow(
                            item = n,
                            onClick = { if (n.isUnread) onMarkRead(n.notificationId) },
                            onHandoffAction = onHandoffAction,
                        )
                        HorizontalDivider()
                    }
                }
            }
            Spacer(Modifier.height(12.dp))
        }
    }
}

@Composable
fun NotificationRow(
    item: NotificationItem,
    onClick: () -> Unit,
    onHandoffAction: (String) -> Unit = {},
) {
    val bg = if (item.isUnread) MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.35f)
             else MaterialTheme.colorScheme.surface
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(bg)
            .clickable(onClick = onClick)
            .padding(horizontal = 8.dp, vertical = 12.dp),
        verticalAlignment = Alignment.Top,
    ) {
        if (item.isUnread) {
            Box(
                Modifier
                    .size(8.dp)
                    .clip(RoundedCornerShape(50))
                    .background(MaterialTheme.colorScheme.primary)
                    .padding(top = 6.dp),
            )
            Spacer(Modifier.size(10.dp))
        } else {
            Spacer(Modifier.size(18.dp))
        }
        Column(Modifier.fillMaxWidth()) {
            Text(item.title.ifEmpty { item.type }, style = MaterialTheme.typography.titleSmall)
            if (item.body.isNotEmpty()) {
                Text(item.body, style = MaterialTheme.typography.bodySmall)
            }
            item.handoffMetadata?.let { metadata ->
                HandoffInboxCard(
                    metadata = metadata,
                    modifier = Modifier.padding(top = 8.dp),
                    onAction = onHandoffAction,
                )
            }
            if (item.createdAt.isNotEmpty()) {
                Text(item.createdAt, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}
