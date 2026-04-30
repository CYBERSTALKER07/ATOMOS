package com.pegasus.retailer.ui.screens.orders

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import com.pegasus.retailer.ui.theme.PillShape
import com.pegasus.retailer.ui.theme.SoftSquircleShape
import androidx.compose.material.icons.outlined.QrCode2
import androidx.compose.material.icons.rounded.AutoAwesome
import androidx.compose.material.icons.rounded.Inventory2
import androidx.compose.material.icons.rounded.Receipt
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.TabRowDefaults
import androidx.compose.material3.TabRowDefaults.tabIndicatorOffset
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasus.retailer.data.model.DemandForecast
import com.pegasus.retailer.data.model.Order
import com.pegasus.retailer.data.model.OrderStatus
import com.pegasus.retailer.ui.components.CountdownTimer
import com.pegasus.retailer.ui.components.LabEmptyState
import com.pegasus.retailer.ui.components.ShimmerOrderList
import com.pegasus.retailer.ui.components.OrderDetailSheet
import com.pegasus.retailer.ui.components.OrderStatusBadge
import com.pegasus.retailer.ui.components.QROverlay
import com.pegasus.retailer.ui.components.statusColor
import com.pegasus.retailer.ui.theme.StatusGreen
import com.pegasus.retailer.ui.theme.StatusOrange
import com.pegasus.retailer.ui.theme.StatusRed
import com.pegasus.retailer.ui.theme.StatusTeal
import kotlinx.coroutines.launch

private enum class OrderTab(
    val title: String,
    val icon: androidx.compose.ui.graphics.vector.ImageVector,
) {
    ACTIVE("Active", Icons.Rounded.Inventory2),
    ORDERED("Ordered", Icons.Rounded.Receipt),
    AI_PLANNED("AI Planned", Icons.Rounded.AutoAwesome),
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrdersScreen(
    viewModel: OrdersViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val pagerState = rememberPagerState(pageCount = { 3 })
    val scope = rememberCoroutineScope()

    var selectedOrder by remember { mutableStateOf<Order?>(null) }
    var qrOrder by remember { mutableStateOf<Order?>(null) }
    var correctionForecast by remember { mutableStateOf<DemandForecast?>(null) }
    var correctionAmount by remember { mutableStateOf("") }

    selectedOrder?.let { order ->
        OrderDetailSheet(
            order = order,
            onDismiss = { selectedOrder = null },
            onShowQR = {
                qrOrder = order
                selectedOrder = null
            },
            onCancel = {
                viewModel.cancelOrder(order.id)
                selectedOrder = null
            },
        )
    }

    QROverlay(
        visible = qrOrder != null,
        order = qrOrder,
        onDismiss = { qrOrder = null },
    )

    // RLHF Correction Dialog
    correctionForecast?.let { forecast ->
        AlertDialog(
            onDismissRequest = { correctionForecast = null; correctionAmount = "" },
            title = { Text("Correct Prediction") },
            text = {
                Column {
                    Text("${forecast.productName} — AI predicted ${forecast.predictedQuantity} units")
                    Spacer(modifier = Modifier.height(12.dp))
                    androidx.compose.material3.OutlinedTextField(
                        value = correctionAmount,
                        onValueChange = { correctionAmount = it.filter { c -> c.isDigit() } },
                        label = { Text("Correct amount") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    correctionAmount.toLongOrNull()?.let { amt ->
                        viewModel.correctPrediction(forecast.id, amt)
                    }
                    correctionForecast = null; correctionAmount = ""
                }) { Text("Submit") }
            },
            dismissButton = {
                Row {
                    TextButton(onClick = {
                        viewModel.rejectPrediction(forecast.id)
                        correctionForecast = null; correctionAmount = ""
                    }) { Text("Reject", color = StatusRed) }
                    TextButton(onClick = { correctionForecast = null; correctionAmount = "" }) { Text("Cancel") }
                }
            },
        )
    }

    PullToRefreshBox(
        isRefreshing = uiState.isLoading,
        onRefresh = viewModel::refresh,
        modifier = Modifier.fillMaxSize(),
    ) {
        Column(modifier = Modifier.fillMaxSize()) {
            // ── M3 Icon Tabs ──
            TabRow(
                selectedTabIndex = pagerState.currentPage,
                containerColor = MaterialTheme.colorScheme.surface,
                contentColor = MaterialTheme.colorScheme.onSurface,
                indicator = { tabPositions ->
                    if (pagerState.currentPage < tabPositions.size) {
                        TabRowDefaults.SecondaryIndicator(
                            modifier = Modifier.tabIndicatorOffset(tabPositions[pagerState.currentPage]),
                            color = MaterialTheme.colorScheme.primary,
                        )
                    }
                },
                divider = { HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.3f)) },
            ) {
                OrderTab.entries.forEachIndexed { index, tab ->
                    val selected = pagerState.currentPage == index
                    Tab(
                        selected = selected,
                        onClick = { scope.launch { pagerState.animateScrollToPage(index) } },
                        icon = {
                            Icon(
                                imageVector = tab.icon,
                                contentDescription = tab.title,
                                modifier = Modifier.size(20.dp),
                            )
                        },
                        text = {
                            Text(
                                tab.title,
                                style = MaterialTheme.typography.labelMedium.copy(fontWeight = if (selected) FontWeight.Bold else FontWeight.Medium),
                            )
                        },
                        selectedContentColor = MaterialTheme.colorScheme.primary,
                        unselectedContentColor = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                }
            }

            // ── Pager Content ──
            HorizontalPager(state = pagerState, modifier = Modifier.fillMaxSize()) { page ->
                when (page) {
                    0 -> ActiveOrdersList(
                        orders = uiState.activeOrders,
                        isLoading = uiState.isLoading,
                        onDetailsCash = { selectedOrder = it },
                        onQRCash = { qrOrder = it },
                    )
                    1 -> OrderedList(
                        orders = uiState.pendingOrders,
                        isLoading = uiState.isLoading,
                        onDetailsCash = { selectedOrder = it },
                        onCancel = viewModel::cancelOrder,
                    )
                    2 -> AiPlannedList(
                        predictions = uiState.predictions,
                        isLoading = uiState.isLoading,
                        onPreorder = viewModel::requestPreorder,
                        onCorrect = { correctionForecast = it },
                        onReject = { viewModel.rejectPrediction(it.id) },
                    )
                }
            }
        }
    }
}

// ────────────────────────────────────────────────────────────────
// Section A: Active Orders (LOADED/IN_TRANSIT/ARRIVED)
// ────────────────────────────────────────────────────────────────

@Composable
private fun ActiveOrdersList(
    orders: List<Order>,
    isLoading: Boolean = false,
    onDetailsCash: (Order) -> Unit,
    onQRCash: (Order) -> Unit,
) {
    if (isLoading && orders.isEmpty()) {
        ShimmerOrderList()
        return
    }
    if (orders.isEmpty()) {
        LabEmptyState(icon = Icons.Rounded.Inventory2, title = "No Active Orders", message = "Orders being prepared or en route will appear here")
        return
    }
    LazyColumn(contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        itemsIndexed(orders, key = { _, o -> o.id }) { _, order ->
            ActiveOrderCard(
                order = order,
                onDetailsCash = { onDetailsCash(order) },
                onQRCash = { onQRCash(order) },
            )
        }
        item { Spacer(modifier = Modifier.height(32.dp)) }
    }
}

// ────────────────────────────────────────────────────────────────
// Section B: Ordered (PENDING — cancelable, AI badge if AI)
// ────────────────────────────────────────────────────────────────

@Composable
private fun OrderedList(
    orders: List<Order>,
    isLoading: Boolean = false,
    onDetailsCash: (Order) -> Unit,
    onCancel: (String) -> Unit,
) {
    if (isLoading && orders.isEmpty()) {
        ShimmerOrderList()
        return
    }
    if (orders.isEmpty()) {
        LabEmptyState(icon = Icons.Rounded.Receipt, title = "No Pending Orders", message = "Orders awaiting dispatch will appear here")
        return
    }
    LazyColumn(contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        itemsIndexed(orders, key = { _, o -> o.id }) { _, order ->
            OrderedCard(
                order = order,
                onDetailsCash = { onDetailsCash(order) },
                onCancel = { onCancel(order.id) },
            )
        }
        item { Spacer(modifier = Modifier.height(32.dp)) }
    }
}

// ────────────────────────────────────────────────────────────────
// Section C: AI Planned Orders (Future forecasts)
// ────────────────────────────────────────────────────────────────

@Composable
private fun AiPlannedList(
    predictions: List<DemandForecast>,
    isLoading: Boolean = false,
    onPreorder: (DemandForecast) -> Unit,
    onCorrect: (DemandForecast) -> Unit,
    onReject: (DemandForecast) -> Unit,
) {
    if (isLoading && predictions.isEmpty()) {
        ShimmerOrderList(count = 3)
        return
    }
    if (predictions.isEmpty()) {
        LabEmptyState(icon = Icons.Rounded.AutoAwesome, title = "No AI Predictions", message = "AI-predicted orders based on your history will appear here")
        return
    }
    LazyColumn(contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        itemsIndexed(predictions, key = { _, f -> f.id }) { _, forecast ->
            AiPlannedCard(
                forecast = forecast,
                onPreorder = { onPreorder(forecast) },
                onCorrect = { onCorrect(forecast) },
                onReject = { onReject(forecast) },
            )
        }
        item { Spacer(modifier = Modifier.height(32.dp)) }
    }
}

// ── Active Order Card (3-Step Circular Progress) ──
@Composable
private fun ActiveOrderCard(
    order: Order,
    onDetailsCash: () -> Unit,
    onQRCash: () -> Unit,
) {
    val progress = order.status.progressFraction
    val ringColor = order.status.statusColor()

    Surface(
        modifier = Modifier.fillMaxWidth().shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.Top) {
                // 3-Step determinate progress ring
                Box(modifier = Modifier.size(44.dp), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(
                        progress = { progress },
                        modifier = Modifier.size(44.dp),
                        color = ringColor,
                        trackColor = ringColor.copy(alpha = 0.15f),
                        strokeWidth = 6.dp,
                        strokeCap = StrokeCap.Round,
                    )
                    Text(
                        order.status.ringLabel,
                        style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.Bold),
                        color = ringColor,
                    )
                }
                Spacer(modifier = Modifier.width(12.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("Order #${order.id.takeLast(3)}", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
                        if (order.isAiGenerated) {
                            Spacer(modifier = Modifier.width(6.dp))
                            Text(
                                "AI",
                                style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.ExtraBold),
                                color = Color.White,
                                modifier = Modifier
                                    .background(MaterialTheme.colorScheme.primary, RoundedCornerShape(4.dp))
                                    .padding(horizontal = 5.dp, vertical = 1.dp),
                            )
                        }
                    }
                    Text("${order.itemCount} items · ${order.displayTotal}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                }
                OrderStatusBadge(order.status)
            }

            // ── Order Status Timeline ──
            Spacer(modifier = Modifier.height(14.dp))
            OrderStatusTimeline(currentStep = order.status.timelineStepIndex)

            // Countdown
            if (order.estimatedDelivery != null) {
                Spacer(modifier = Modifier.height(10.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    CountdownTimer(
                        targetIso = order.estimatedDelivery,
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold, fontSize = 12.sp),
                        color = StatusGreen,
                    )
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("until arrival", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f))
                }
            }

            // Tag pills
            if (order.items.isNotEmpty()) {
                Spacer(modifier = Modifier.height(10.dp))
                Row(
                    modifier = Modifier.horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    order.items.take(3).forEach { item ->
                        Text(
                            item.productName.split(" ").take(2).joinToString(" "),
                            style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp),
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                            modifier = Modifier
                                .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f), PillShape)
                                .padding(horizontal = 8.dp, vertical = 4.dp),
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.2f))
            Spacer(modifier = Modifier.height(12.dp))

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                // Details
                Text(
                    "Details",
                    style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onDetailsCash() }
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f), PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
                // QR
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onQRCash() }
                        .background(MaterialTheme.colorScheme.primary, PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                ) {
                    Icon(Icons.Outlined.QrCode2, contentDescription = null, modifier = Modifier.size(12.dp), tint = MaterialTheme.colorScheme.onPrimary)
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Show QR", style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold), color = MaterialTheme.colorScheme.onPrimary)
                }
            }
        }
    }
}

// ── Ordered Card (PENDING — cancelable, AI badge if AI-generated) ──
@Composable
private fun OrderedCard(
    order: Order,
    onDetailsCash: () -> Unit,
    onCancel: () -> Unit,
) {
    val ringColor = order.status.statusColor()

    Surface(
        modifier = Modifier.fillMaxWidth().shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                // Progress ring (1/3 for PENDING)
                Box(modifier = Modifier.size(44.dp), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(
                        progress = { order.status.progressFraction },
                        modifier = Modifier.size(44.dp),
                        color = ringColor,
                        trackColor = ringColor.copy(alpha = 0.15f),
                        strokeWidth = 6.dp,
                        strokeCap = StrokeCap.Round,
                    )
                    Text(
                        order.status.ringLabel,
                        style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.Bold),
                        color = ringColor,
                    )
                }
                Spacer(modifier = Modifier.width(12.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("Order #${order.id.takeLast(3)}", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
                        if (order.isAiGenerated) {
                            Spacer(modifier = Modifier.width(6.dp))
                            Text(
                                "AI",
                                style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.ExtraBold),
                                color = Color.White,
                                modifier = Modifier
                                    .background(MaterialTheme.colorScheme.primary, RoundedCornerShape(4.dp))
                                    .padding(horizontal = 5.dp, vertical = 1.dp),
                            )
                        }
                    }
                    Text("${order.itemCount} items · ${order.displayTotal}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                }
                OrderStatusBadge(order.status)
            }

            // Tag pills
            if (order.items.isNotEmpty()) {
                Spacer(modifier = Modifier.height(10.dp))
                Row(
                    modifier = Modifier.horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    order.items.take(3).forEach { item ->
                        Text(
                            item.productName.split(" ").take(2).joinToString(" "),
                            style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp),
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                            modifier = Modifier
                                .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f), PillShape)
                                .padding(horizontal = 8.dp, vertical = 4.dp),
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.2f))
            Spacer(modifier = Modifier.height(12.dp))

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                // Cancel
                Text(
                    "Cancel",
                    style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                    color = StatusRed,
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onCancel() }
                        .background(StatusRed.copy(alpha = 0.1f), PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
                // Details
                Text(
                    "Details",
                    style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onDetailsCash() }
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f), PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
            }
        }
    }
}

// ── AI Planned Card (Future forecasts with execution date) ──
@Composable
private fun AiPlannedCard(
    forecast: DemandForecast,
    onPreorder: () -> Unit,
    onCorrect: () -> Unit,
    onReject: () -> Unit,
) {
    val color = when {
        forecast.confidence >= 0.8 -> StatusGreen
        forecast.confidence >= 0.6 -> StatusOrange
        else -> StatusRed
    }
    val trackColor = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.3f)

    Surface(
        modifier = Modifier.fillMaxWidth().shadow(3.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                // Confidence ring
                Box(
                    modifier = Modifier.size(40.dp).drawBehind {
                        val sw = 3.dp.toPx()
                        val arcSize = Size(size.width - sw, size.height - sw)
                        val tl = Offset(sw / 2, sw / 2)
                        drawArc(trackColor, 0f, 360f, false, topLeft = tl, size = arcSize, style = Stroke(sw))
                        drawArc(color, -90f, (forecast.confidence * 360).toFloat(), false, topLeft = tl, size = arcSize, style = Stroke(sw, cap = StrokeCap.Round))
                    },
                    contentAlignment = Alignment.Center,
                ) {
                    Text(forecast.confidencePercent, style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.Bold), color = color)
                }
                Spacer(modifier = Modifier.width(12.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Text(forecast.productName, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    Text("${forecast.predictedQuantity} units", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                }
                Spacer(modifier = Modifier.width(8.dp))
                // Execution date + pre-order
                Column(horizontalAlignment = Alignment.End) {
                    Text(
                        forecast.suggestedOrderDate,
                        style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Medium),
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        "Pre-Order",
                        style = MaterialTheme.typography.labelSmall.copy(fontSize = 11.sp, fontWeight = FontWeight.Bold),
                        color = Color.White,
                        modifier = Modifier
                            .clip(PillShape)
                            .clickable { onPreorder() }
                            .background(MaterialTheme.colorScheme.primary, PillShape)
                            .padding(horizontal = 10.dp, vertical = 5.dp),
                    )
                }
            }
            // Execution date banner
            Spacer(modifier = Modifier.height(8.dp))
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.2f), SoftSquircleShape)
                    .padding(horizontal = 10.dp, vertical = 6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Rounded.AutoAwesome, contentDescription = null, modifier = Modifier.size(12.dp), tint = MaterialTheme.colorScheme.primary)
                Spacer(modifier = Modifier.width(6.dp))
                Text(
                    "AI will place on ${forecast.suggestedOrderDate}",
                    style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                )
            }
            // RLHF action row
            Spacer(modifier = Modifier.height(8.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    "Correct",
                    style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onCorrect() }
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f), PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
                Text(
                    "Reject",
                    style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold),
                    color = StatusRed,
                    modifier = Modifier
                        .clip(PillShape)
                        .clickable { onReject() }
                        .background(StatusRed.copy(alpha = 0.1f), PillShape)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
            }
        }
    }
}

// ── Order Status Timeline (horizontal stepper) ──
@Composable
private fun OrderStatusTimeline(currentStep: Int) {
    val steps = OrderStatus.timelineSteps
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.15f),
                RoundedCornerShape(10.dp),
            )
            .padding(horizontal = 10.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        steps.forEachIndexed { index, (label, _) ->
            val isCompleted = index < currentStep
            val isActive = index == currentStep
            val dotColor = when {
                isCompleted -> StatusGreen
                isActive -> StatusTeal
                else -> MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.4f)
            }
            val labelColor = when {
                isCompleted -> MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                isActive -> StatusTeal
                else -> MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f)
            }
            val labelWeight = if (isActive) FontWeight.Bold else FontWeight.Medium

            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier.weight(1f),
            ) {
                Box(
                    modifier = Modifier
                        .size(if (isActive) 10.dp else 8.dp)
                        .background(dotColor, CircleShape),
                )
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = label,
                    style = MaterialTheme.typography.labelSmall.copy(
                        fontSize = 8.sp,
                        fontWeight = labelWeight,
                        lineHeight = 10.sp,
                    ),
                    color = labelColor,
                    maxLines = 1,
                )
            }
        }
    }
}
