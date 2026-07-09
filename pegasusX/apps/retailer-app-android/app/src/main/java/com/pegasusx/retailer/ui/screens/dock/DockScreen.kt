package com.pegasusx.retailer.ui.screens.dock

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ExpandLess
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.QrCode2
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.retailer.data.model.TrackingOrder
import com.pegasusx.retailer.ui.components.PegasusEmptyState
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.retailer.ui.components.TrackingQROverlay
import com.pegasusx.retailer.ui.theme.PillShape
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.ui.theme.StatusGreen
import com.pegasusx.retailer.ui.theme.StatusOrange
import com.pegasusx.retailer.ui.theme.StatusRed
import java.text.NumberFormat
import java.util.Locale

private val stateLabels = mapOf(
    "DISPATCHED" to "Dispatched",
    "IN_TRANSIT" to "In Transit",
    "ARRIVING" to "Arriving",
    "ARRIVED" to "Arrived",
    "AWAITING_PAYMENT" to "Awaiting Payment",
)

@Composable
fun DockScreen(
    viewModel: DockViewModel,
    modifier: Modifier = Modifier,
) {
    val uiState by viewModel.state.collectAsState()

    TrackingQROverlay(
        visible = uiState.activeQrOrder != null,
        order = uiState.activeQrOrder,
        onDismiss = viewModel::dismissQr,
    )

    Column(modifier = modifier.fillMaxSize()) {
        uiState.syncMessage?.let { message ->
            PegasusRuntimeBanner(
                tone = if (uiState.isRefreshing) PegasusRuntimeTone.Refreshing else PegasusRuntimeTone.Warning,
                message = message,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                onRetry = if (!uiState.isRefreshing) viewModel::refresh else null,
            )
        }

        DockSummaryRow(
            queueCount = uiState.activeOrders.size,
            arrivedCount = uiState.arrivedCount,
            approachingCount = uiState.approachingCount,
            isRefreshing = uiState.isRefreshing,
            onRefresh = viewModel::refresh,
        )

        when {
            uiState.isLoading && uiState.orders.isEmpty() -> {
                Column(
                    modifier = Modifier.fillMaxSize(),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center,
                ) {
                    CircularProgressIndicator()
                    Spacer(modifier = Modifier.height(12.dp))
                    Text("Loading dock queue...", color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
                }
            }
            uiState.supplierGroups.isEmpty() -> {
                PegasusEmptyState(
                    icon = Icons.Default.LocalShipping,
                    title = "Dock Queue Empty",
                    message = "Inbound deliveries grouped by supplier will appear here.",
                )
            }
            else -> {
                LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                    contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
                    items(uiState.supplierGroups, key = { it.supplierId }) { group ->
                        SupplierDockSection(
                            group = group,
                            expanded = group.supplierId in uiState.expandedSupplierIds,
                            revealedTokenOrderIds = uiState.revealedTokenOrderIds,
                            onToggleExpand = { viewModel.toggleSupplier(group.supplierId) },
                            onToggleToken = viewModel::toggleTokenReveal,
                            onShowQr = viewModel::showQr,
                        )
                    }
                    item(span = { GridItemSpan(maxLineSpan) }) { Spacer(modifier = Modifier.height(24.dp)) }
                }
            }
        }
    }
}

@Composable
private fun DockSummaryRow(
    queueCount: Int,
    arrivedCount: Int,
    approachingCount: Int,
    isRefreshing: Boolean,
    onRefresh: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        DockMetricCard(label = "Queue", value = queueCount.toString(), modifier = Modifier.weight(1f))
        DockMetricCard(label = "Arrived", value = arrivedCount.toString(), modifier = Modifier.weight(1f))
        DockMetricCard(label = "Approaching", value = approachingCount.toString(), modifier = Modifier.weight(1f))
        IconButton(onClick = onRefresh, enabled = !isRefreshing) {
            if (isRefreshing) {
                CircularProgressIndicator(modifier = Modifier.size(20.dp), strokeWidth = 2.dp)
            } else {
                Icon(Icons.Default.Refresh, contentDescription = "Refresh")
            }
        }
    }
}

@Composable
private fun DockMetricCard(label: String, value: String, modifier: Modifier = Modifier) {
    Surface(
        modifier = modifier,
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
        tonalElevation = 1.dp,
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(
                label.uppercase(),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
            )
            Text(value, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
        }
    }
}

@Composable
private fun SupplierDockSection(
    group: SupplierDockGroup,
    expanded: Boolean,
    revealedTokenOrderIds: Set<String>,
    onToggleExpand: () -> Unit,
    onToggleToken: (String) -> Unit,
    onShowQr: (String) -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
        tonalElevation = 1.dp,
    ) {
        Column {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable(onClick = onToggleExpand)
                    .padding(16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(group.supplierName, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
                    Text(
                        "${group.orders.size} orders · ${formatDockAmount(group.totalAmount)} UZS",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                    )
                }
                if (group.hasApproaching) {
                    StatusPill("Approaching", StatusOrange)
                    Spacer(modifier = Modifier.width(6.dp))
                }
                if (group.hasArrived) {
                    StatusPill("Arrived", StatusGreen)
                    Spacer(modifier = Modifier.width(6.dp))
                }
                Icon(
                    if (expanded) Icons.Default.ExpandLess else Icons.Default.ExpandMore,
                    contentDescription = if (expanded) "Collapse" else "Expand",
                )
            }

            AnimatedVisibility(
                visible = expanded,
                enter = expandVertically(),
                exit = shrinkVertically(),
            ) {
                Column(modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)) {
                    group.orders.forEach { order ->
                        DockOrderRow(
                            order = order,
                            tokenRevealed = order.orderId in revealedTokenOrderIds,
                            onToggleToken = { onToggleToken(order.orderId) },
                            onShowQr = { onShowQr(order.orderId) },
                        )
                        Spacer(modifier = Modifier.height(8.dp))
                    }
                }
            }
        }
    }
}

@Composable
private fun DockOrderRow(
    order: TrackingOrder,
    tokenRevealed: Boolean,
    onToggleToken: () -> Unit,
    onShowQr: () -> Unit,
) {
    val stateColor = when (order.state) {
        "ARRIVED", "AWAITING_PAYMENT" -> StatusGreen
        "ARRIVING" -> StatusOrange
        "IN_TRANSIT", "DISPATCHED" -> StatusOrange
        else -> MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
    }

    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f),
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Row(verticalAlignment = Alignment.Top) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        "Order #${order.orderId.takeLast(8)}",
                        style = MaterialTheme.typography.labelLarge,
                        fontWeight = FontWeight.SemiBold,
                    )
                    Text(
                        "${order.items.size} items · ${formatDockAmount(order.totalAmount)} UZS",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                    )
                }
                StatusPill(stateLabels[order.state] ?: order.state, stateColor)
            }

            if (order.isApproaching || order.state == "ARRIVING") {
                Spacer(modifier = Modifier.height(8.dp))
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(8.dp))
                        .background(StatusOrange.copy(alpha = 0.12f))
                        .padding(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(Icons.Default.Warning, contentDescription = null, tint = StatusOrange, modifier = Modifier.size(16.dp))
                    Spacer(modifier = Modifier.width(6.dp))
                    Text("Driver approaching", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.SemiBold)
                }
            }

            val canRevealQr = order.deliveryToken.isNotBlank() &&
                (order.state == "ARRIVED" || order.state == "AWAITING_PAYMENT" || order.isApproaching)

            if (canRevealQr) {
                Spacer(modifier = Modifier.height(10.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        if (tokenRevealed) "Hide QR" else "Reveal QR",
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier
                            .clip(PillShape)
                            .clickable(onClick = onToggleToken)
                            .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.1f))
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    )
                    if (tokenRevealed) {
                        Text(
                            "Show Fullscreen",
                            style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold),
                            color = Color.White,
                            modifier = Modifier
                                .clip(PillShape)
                                .clickable(onClick = onShowQr)
                                .background(MaterialTheme.colorScheme.primary)
                                .padding(horizontal = 12.dp, vertical = 6.dp),
                        )
                    }
                }
                if (tokenRevealed) {
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Default.QrCode2, contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(modifier = Modifier.width(6.dp))
                        Text(
                            "Token ready for driver scan",
                            style = MaterialTheme.typography.labelSmall,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun StatusPill(text: String, color: Color) {
    Text(
        text,
        style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp, fontWeight = FontWeight.Bold),
        color = color,
        modifier = Modifier
            .clip(PillShape)
            .background(color.copy(alpha = 0.12f))
            .padding(horizontal = 8.dp, vertical = 4.dp),
    )
}

private fun formatDockAmount(amount: Long): String {
    return NumberFormat.getNumberInstance(Locale.US).format(amount).replace(',', ' ')
}
