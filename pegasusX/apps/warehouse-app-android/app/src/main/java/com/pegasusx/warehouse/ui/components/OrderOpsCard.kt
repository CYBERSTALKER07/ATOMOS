package com.pegasusx.warehouse.ui.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.AssistChip
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

enum class OrderDetailOpenMode {
    Single,
    Double,
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
fun OrderOpsCard(
    retailerName: String,
    orderId: String,
    state: String,
    amountLabel: String,
    meta: String? = null,
    badge: String? = null,
    enabled: Boolean = true,
    canDelay: Boolean = false,
    canReject: Boolean = false,
    canReassign: Boolean = false,
    showOpsMenu: Boolean = true,
    showQuickActions: Boolean = true,
    detailOpenMode: OrderDetailOpenMode = OrderDetailOpenMode.Single,
    delayLabel: String = "Propose date",
    rejectLabel: String = "Cancel order",
    onOpenDetail: () -> Unit,
    onDelay: (() -> Unit)? = null,
    onReject: (() -> Unit)? = null,
    onReassign: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
    leadingContent: (@Composable () -> Unit)? = null,
) {
    var menuExpanded by remember { mutableStateOf(false) }

    val openModifier = when (detailOpenMode) {
        OrderDetailOpenMode.Single -> Modifier.combinedClickable(
            enabled = enabled,
            onClick = onOpenDetail,
            onDoubleClick = {},
        )
        OrderDetailOpenMode.Double -> Modifier.combinedClickable(
            enabled = enabled,
            onClick = {},
            onDoubleClick = onOpenDetail,
        )
    }

    ElevatedCard(
        modifier = modifier
            .fillMaxWidth()
            .then(openModifier),
    ) {
        Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                leadingContent?.invoke()
                Column(modifier = Modifier.weight(1f)) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            text = retailerName.ifBlank { orderId.take(8) },
                            style = MaterialTheme.typography.titleSmall,
                        )
                        AssistChip(onClick = {}, label = { Text(state, style = MaterialTheme.typography.labelSmall) })
                        badge?.let {
                            AssistChip(onClick = {}, label = { Text(it, style = MaterialTheme.typography.labelSmall) })
                        }
                    }
                    Text(
                        text = orderId,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    meta?.let {
                        Text(
                            text = it,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
                Text(
                    text = amountLabel,
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(end = PegasusSpacing.xs),
                )
                if (showOpsMenu) {
                    Box {
                        IconButton(
                            onClick = { menuExpanded = true },
                            enabled = enabled,
                            modifier = Modifier.size(44.dp),
                        ) {
                            Icon(Icons.Default.MoreVert, contentDescription = stringResource(R.string.supplier_portal_orders_order_kebab_menu_text_order_actions))
                        }
                        DropdownMenu(expanded = menuExpanded, onDismissRequest = { menuExpanded = false }) {
                            DropdownMenuItem(
                                text = { Text("View details") },
                                onClick = {
                                    menuExpanded = false
                                    onOpenDetail()
                                },
                            )
                            if (onDelay != null) {
                                DropdownMenuItem(
                                    text = { Text(delayLabel) },
                                    enabled = canDelay,
                                    onClick = {
                                        menuExpanded = false
                                        onDelay()
                                    },
                                )
                            }
                            if (onReject != null) {
                                DropdownMenuItem(
                                    text = { Text(rejectLabel) },
                                    enabled = canReject,
                                    onClick = {
                                        menuExpanded = false
                                        onReject()
                                    },
                                )
                            }
                            if (onReassign != null) {
                                DropdownMenuItem(
                                    text = { Text("Reassign order") },
                                    enabled = canReassign,
                                    onClick = {
                                        menuExpanded = false
                                        onReassign()
                                    },
                                )
                            }
                        }
                    }
                }
            }
            if (showQuickActions && (onDelay != null || onReject != null || onReassign != null)) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = PegasusSpacing.md),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    if (onDelay != null) {
                        OutlinedButton(
                            onClick = onDelay,
                            enabled = enabled && canDelay,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(delayLabel)
                        }
                    }
                    if (onReject != null) {
                        OutlinedButton(
                            onClick = onReject,
                            enabled = enabled && canReject,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(rejectLabel)
                        }
                    }
                    if (onReassign != null) {
                        OutlinedButton(
                            onClick = onReassign,
                            enabled = enabled && canReassign,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text("Reassign")
                        }
                    }
                }
            }
        }
    }
}
