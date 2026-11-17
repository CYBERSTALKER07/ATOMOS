package com.pegasusx.retailer.ui.screens.dashboard.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.AccountBalanceWallet
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Bolt
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.Edit
import androidx.compose.material.icons.rounded.LocalShipping
import androidx.compose.material.icons.rounded.Remove
import androidx.compose.material.icons.rounded.Schedule
import androidx.compose.material.icons.rounded.ShoppingBag
import androidx.compose.material.icons.rounded.Shield
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.local.RetailerWorkspaceMode
import com.pegasusx.retailer.data.model.FastHomePendingProposal
import com.pegasusx.retailer.data.model.RetailerFastHomeResponse
import com.pegasusx.retailer.ui.theme.PillShape
import com.pegasusx.retailer.ui.theme.SquircleShape
import com.pegasusx.retailer.ui.theme.StatusGreen
import com.pegasusx.retailer.ui.theme.StatusOrange
import com.pegasusx.retailer.ui.theme.StatusRed
import com.pegasusx.retailer.util.CutoffCountdownState
import com.pegasusx.retailer.util.OrderLineContractHelper

@Composable
fun WorkspaceModeToggleStrip(
    mode: RetailerWorkspaceMode,
    pendingApprovalsCount: Int,
    onSelectMode: (RetailerWorkspaceMode) -> Unit,
    onOpenApprovals: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        SingleChoiceSegmentedButtonRow(modifier = Modifier.fillMaxWidth()) {
            val options = listOf(
                RetailerWorkspaceMode.FAST to stringResource(R.string.workspace_mode_fast),
                RetailerWorkspaceMode.FULL to stringResource(R.string.workspace_mode_full),
            )
            options.forEachIndexed { index, (optionMode, label) ->
                SegmentedButton(
                    selected = mode == optionMode,
                    onClick = { onSelectMode(optionMode) },
                    shape = SegmentedButtonDefaults.itemShape(index = index, count = options.size),
                    icon = {
                        SegmentedButtonDefaults.Icon(active = mode == optionMode) {
                            Icon(
                                imageVector = if (optionMode == RetailerWorkspaceMode.FAST) {
                                    Icons.Rounded.Bolt
                                } else {
                                    Icons.Rounded.Check
                                },
                                contentDescription = null,
                                modifier = Modifier.size(SegmentedButtonDefaults.IconSize),
                            )
                        }
                    },
                ) {
                    Text(label, fontWeight = FontWeight.Medium)
                }
            }
        }

        if (pendingApprovalsCount > 0) {
            Surface(
                onClick = onOpenApprovals,
                shape = SquircleShape,
                color = StatusOrange.copy(alpha = 0.14f),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Row(
                    modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Icon(Icons.Rounded.Shield, contentDescription = null, tint = StatusOrange, modifier = Modifier.size(18.dp))
                    Text(
                        text = stringResource(R.string.fast_home_approvals_badge_fmt, pendingApprovalsCount),
                        style = MaterialTheme.typography.labelLarge,
                        color = StatusOrange,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
            }
        }
    }
}

@Composable
fun FastHomeCardsSection(
    fastHome: RetailerFastHomeResponse?,
    countdown: CutoffCountdownState,
    actionPending: Boolean,
    actionFeedback: String?,
    onOpenDeliveries: () -> Unit,
    onOpenCredit: () -> Unit,
    onConfirmProposal: (FastHomePendingProposal) -> Unit,
    onHoldProposal: (FastHomePendingProposal) -> Unit,
    onEditProposal: (FastHomePendingProposal, Map<String, Int>) -> Unit,
) {
    var showEditDialog by remember { mutableStateOf(false) }

    Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
        // Header banner
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Surface(
                shape = PillShape,
                color = MaterialTheme.colorScheme.primaryContainer,
            ) {
                Row(
                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    Icon(
                        Icons.Rounded.Bolt,
                        contentDescription = null,
                        modifier = Modifier.size(14.dp),
                        tint = MaterialTheme.colorScheme.onPrimaryContainer,
                    )
                    Text(
                        text = stringResource(R.string.fast_home_badge),
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onPrimaryContainer,
                    )
                }
            }
            Text(
                text = stringResource(R.string.fast_home_title),
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = stringResource(R.string.fast_home_subtitle),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }

        if (!actionFeedback.isNullOrBlank()) {
            Surface(
                shape = SquircleShape,
                color = MaterialTheme.colorScheme.secondaryContainer,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    text = actionFeedback,
                    modifier = Modifier.padding(12.dp),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSecondaryContainer,
                )
            }
        }

        // Card 1: Tomorrow's Delivery
        val delivery = fastHome?.tomorrowDelivery
        ElevatedCard(
            shape = SquircleShape,
            colors = CardDefaults.elevatedCardColors(containerColor = MaterialTheme.colorScheme.surface),
            modifier = Modifier.fillMaxWidth(),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Icon(Icons.Rounded.LocalShipping, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                        Text(
                            text = stringResource(R.string.fast_home_tomorrow_delivery_title),
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.SemiBold,
                        )
                    }
                    if (!delivery?.state.isNullOrBlank()) {
                        Surface(shape = PillShape, color = StatusGreen.copy(alpha = 0.14f)) {
                            Text(
                                text = delivery!!.state.replace('_', ' '),
                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                                style = MaterialTheme.typography.labelSmall,
                                color = StatusGreen,
                                fontWeight = FontWeight.SemiBold,
                            )
                        }
                    }
                }
                if (delivery == null) {
                    Text(
                        text = stringResource(R.string.fast_home_no_delivery),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                } else {
                    Text(
                        text = OrderLineContractHelper.formatWholeSom(delivery.amountMinor, delivery.currency),
                        style = MaterialTheme.typography.headlineSmall,
                        fontWeight = FontWeight.Bold,
                    )
                    val details = listOfNotNull(
                        delivery.supplierName.takeIf { it.isNotBlank() },
                        delivery.deliveryDate?.take(10),
                        delivery.windowLabel?.takeIf { it.isNotBlank() },
                        if (delivery.itemCount > 0) "${delivery.itemCount}x" else null,
                    ).joinToString(" · ")
                    if (details.isNotBlank()) {
                        Text(
                            text = details,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
                OutlinedButton(
                    onClick = onOpenDeliveries,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(stringResource(R.string.fast_home_view_deliveries))
                }
            }
        }

        // Card 2: What We Owe
        val balanceMinor = fastHome?.balanceOwedMinor ?: 0L
        val overdueMinor = fastHome?.overdueMinor ?: 0L
        val headroomMinor = fastHome?.creditHeadroomMinor ?: 0L
        val currency = fastHome?.currency ?: "UZS"

        ElevatedCard(
            shape = SquircleShape,
            colors = CardDefaults.elevatedCardColors(containerColor = MaterialTheme.colorScheme.surface),
            modifier = Modifier.fillMaxWidth(),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Icon(
                            Icons.Rounded.AccountBalanceWallet,
                            contentDescription = null,
                            tint = if (overdueMinor > 0L) StatusRed else MaterialTheme.colorScheme.primary,
                        )
                        Text(
                            text = stringResource(R.string.fast_home_balance_title),
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.SemiBold,
                        )
                    }
                    if (overdueMinor > 0L) {
                        Surface(shape = PillShape, color = StatusRed.copy(alpha = 0.14f)) {
                            Text(
                                text = stringResource(
                                    R.string.fast_home_overdue_fmt,
                                    OrderLineContractHelper.formatWholeSom(overdueMinor, currency),
                                ),
                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                                style = MaterialTheme.typography.labelSmall,
                                color = StatusRed,
                                fontWeight = FontWeight.SemiBold,
                            )
                        }
                    }
                }
                Text(
                    text = OrderLineContractHelper.formatWholeSom(balanceMinor, currency),
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.Bold,
                    color = if (overdueMinor > 0L) StatusRed else MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = stringResource(
                        R.string.fast_home_headroom_fmt,
                        OrderLineContractHelper.formatWholeSom(headroomMinor, currency),
                    ),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                FilledTonalButton(
                    onClick = onOpenCredit,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(stringResource(R.string.fast_home_pay_now))
                }
            }
        }

        // Card 3: Today's Proposed Order + Server-Synced Cutoff Countdown
        val proposal = fastHome?.pendingProposal
        ElevatedCard(
            shape = SquircleShape,
            colors = CardDefaults.elevatedCardColors(containerColor = MaterialTheme.colorScheme.surface),
            modifier = Modifier.fillMaxWidth(),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Icon(Icons.Rounded.ShoppingBag, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                        Text(
                            text = stringResource(R.string.fast_home_proposal_title),
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.SemiBold,
                        )
                    }
                    Surface(
                        shape = PillShape,
                        color = if (countdown.expired) StatusRed.copy(alpha = 0.14f) else StatusOrange.copy(alpha = 0.14f),
                    ) {
                        Row(
                            modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(4.dp),
                        ) {
                            Icon(
                                Icons.Rounded.Schedule,
                                contentDescription = null,
                                modifier = Modifier.size(14.dp),
                                tint = if (countdown.expired) StatusRed else StatusOrange,
                            )
                            Text(
                                text = if (countdown.expired) {
                                    stringResource(R.string.fast_home_cutoff_expired)
                                } else {
                                    "${stringResource(R.string.fast_home_cutoff_label)}: ${countdown.formatted}"
                                },
                                style = MaterialTheme.typography.labelSmall,
                                color = if (countdown.expired) StatusRed else StatusOrange,
                                fontWeight = FontWeight.SemiBold,
                            )
                        }
                    }
                }

                if (proposal == null) {
                    Text(
                        text = stringResource(R.string.fast_home_no_proposal),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                } else {
                    Text(
                        text = OrderLineContractHelper.formatWholeSom(proposal.amountMinor, proposal.currency),
                        style = MaterialTheme.typography.headlineSmall,
                        fontWeight = FontWeight.Bold,
                    )
                    val meta = listOfNotNull(
                        proposal.supplierName.takeIf { it.isNotBlank() },
                        proposal.proposedDeliveryDate?.take(10),
                        if (proposal.itemCount > 0) "${proposal.itemCount}x" else null,
                    ).joinToString(" · ")
                    if (meta.isNotBlank()) {
                        Text(
                            text = meta,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    proposal.items.take(4).forEach { line ->
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                        ) {
                            Text(
                                text = "${line.productName.ifBlank { line.sku }} ×${line.quantity}",
                                style = MaterialTheme.typography.bodySmall,
                                modifier = Modifier.weight(1f),
                            )
                            Text(
                                text = OrderLineContractHelper.formatWholeSom(
                                    line.lineTotalMinor.takeIf { it > 0L } ?: (line.unitPriceMinor * line.quantity),
                                    proposal.currency,
                                ),
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = FontWeight.Medium,
                            )
                        }
                    }

                    HorizontalDivider(modifier = Modifier.padding(vertical = 2.dp))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Button(
                            onClick = { onConfirmProposal(proposal) },
                            enabled = !actionPending,
                            modifier = Modifier.weight(1f),
                        ) {
                            Icon(Icons.Rounded.Check, contentDescription = null, modifier = Modifier.size(16.dp))
                            Spacer(modifier = Modifier.width(4.dp))
                            Text(stringResource(R.string.fast_home_confirm_order))
                        }
                        if (proposal.items.isNotEmpty()) {
                            OutlinedButton(
                                onClick = { showEditDialog = true },
                                enabled = !actionPending,
                            ) {
                                Icon(Icons.Rounded.Edit, contentDescription = null, modifier = Modifier.size(16.dp))
                                Spacer(modifier = Modifier.width(4.dp))
                                Text(stringResource(R.string.fast_home_edit_order))
                            }
                        }
                        TextButton(
                            onClick = { onHoldProposal(proposal) },
                            enabled = !actionPending,
                        ) {
                            Icon(Icons.Rounded.Close, contentDescription = null, modifier = Modifier.size(16.dp))
                            Text(stringResource(R.string.fast_home_hold_order))
                        }
                    }
                }
            }
        }
    }

    if (showEditDialog && fastHome?.pendingProposal != null) {
        val proposal = fastHome.pendingProposal
        val quantities = remember(proposal.orderId) {
            mutableStateMapOf<String, Int>().apply {
                proposal.items.forEach { item ->
                    val key = item.sku.ifBlank { item.productId }
                    put(key, item.quantity)
                }
            }
        }
        AlertDialog(
            onDismissRequest = { showEditDialog = false },
            title = { Text(stringResource(R.string.fast_home_edit_dialog_title)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    proposal.items.forEach { item ->
                        val key = item.sku.ifBlank { item.productId }
                        val qty = quantities[key] ?: item.quantity
                        val step = item.packSize.coerceAtLeast(1)
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.SpaceBetween,
                        ) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(item.productName.ifBlank { key }, style = MaterialTheme.typography.bodyMedium)
                                Text(
                                    OrderLineContractHelper.formatWholeSom(item.unitPriceMinor, proposal.currency),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                IconButton(
                                    onClick = { quantities[key] = (qty - step).coerceAtLeast(0) },
                                ) {
                                    Icon(Icons.Rounded.Remove, contentDescription = null)
                                }
                                Text("$qty", fontWeight = FontWeight.SemiBold)
                                IconButton(
                                    onClick = { quantities[key] = qty + step },
                                ) {
                                    Icon(Icons.Rounded.Add, contentDescription = null)
                                }
                            }
                        }
                    }
                }
            },
            confirmButton = {
                Button(
                    onClick = {
                        showEditDialog = false
                        onEditProposal(proposal, quantities.toMap())
                    },
                ) {
                    Text(stringResource(R.string.fast_home_save_changes))
                }
            },
            dismissButton = {
                TextButton(onClick = { showEditDialog = false }) {
                    Text(stringResource(R.string.fast_home_cancel))
                }
            },
        )
    }
}
