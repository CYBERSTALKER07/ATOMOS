package com.pegasusx.supplier.ui.screens.orders

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.design.network.showFullScreenLoading
import com.pegasusx.supplier.data.model.SupplierOrder
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.supplier.ui.components.formatMinorAmount
import com.pegasusx.supplier.ui.screens.dispatch.DispatchPreviewScreen
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.OrderFilterTab
import com.pegasusx.supplier.ui.viewmodel.OrdersViewModel
import com.pegasusx.supplier.R

private enum class OrdersHubSurface { Queue, Dispatch }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrdersHubScreen(
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
    onOrderClick: (SupplierOrder) -> Unit,
    initialStatus: String? = null,
    viewModel: OrdersViewModel = hiltViewModel(),
) {
    var surface by remember { mutableStateOf(OrdersHubSurface.Queue) }
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(initialStatus) {
        if (!initialStatus.isNullOrBlank()) {
            viewModel.setCommandStatus(initialStatus)
        }
    }

    Scaffold(
        topBar = {
            Column {
                TopAppBar(
                    title = { Text(if (surface == OrdersHubSurface.Queue) "Orders" else "Dispatch") },
                    actions = {
                        if (surface == OrdersHubSurface.Queue) {
                            IconButton(onClick = { viewModel.load() }) {
                                Icon(Icons.Default.Refresh, contentDescription = stringResource(R.string.portal_page_orders_action_refresh))
                            }
                        }
                    },
                )
                TabRow(selectedTabIndex = surface.ordinal) {
                    Tab(
                        selected = surface == OrdersHubSurface.Queue,
                        onClick = { surface = OrdersHubSurface.Queue },
                        text = { Text("Orders") },
                    )
                    Tab(
                        selected = surface == OrdersHubSurface.Dispatch,
                        onClick = { surface = OrdersHubSurface.Dispatch },
                        text = { Text("Dispatch") },
                    )
                }
                if (surface == OrdersHubSurface.Queue && state.commandStatus != null) {
                    TextButton(onClick = { viewModel.clearCommandStatus() }) {
                        Text("Filtered by ${state.commandStatus?.replace('_', ' ')} · Clear")
                    }
                }
                if (surface == OrdersHubSurface.Queue) {
                    ScrollableTabRow(selectedTabIndex = state.filter.ordinal) {
                        OrderFilterTab.entries.forEach { tab ->
                            Tab(
                                selected = state.filter == tab,
                                onClick = { viewModel.setFilter(tab) },
                                text = {
                                    Text(
                                        when (tab) {
                                            OrderFilterTab.SCHEDULED -> "Scheduled"
                                            else -> tab.name.lowercase().replaceFirstChar { it.uppercase() }
                                        },
                                    )
                                },
                            )
                        }
                    }
                }
            }
        },
    ) { padding ->
        when (surface) {
            OrdersHubSurface.Queue -> OrdersQueueContent(
                modifier = Modifier.padding(padding),
                state = state,
                onRetry = { viewModel.load() },
                onOrderClick = onOrderClick,
                canWarehouseOps = viewModel::canWarehouseOps,
                onDelay = { order, proposedDate, reason -> viewModel.proposeWarehouseOrder(order, proposedDate, reason) },
                onReject = { order, reason -> viewModel.rejectWarehouseOrder(order, reason) },
            )
            OrdersHubSurface.Dispatch -> DispatchPreviewScreen(
                ops = ops,
                realtimeSignals = realtimeSignals,
                embedded = true,
                modifier = Modifier.padding(padding),
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun OrdersQueueContent(
    modifier: Modifier = Modifier,
    state: com.pegasusx.supplier.ui.viewmodel.OrdersUiState,
    onRetry: () -> Unit,
    onOrderClick: (SupplierOrder) -> Unit,
    canWarehouseOps: (SupplierOrder) -> Boolean,
    onDelay: (SupplierOrder, String, String) -> Unit,
    onReject: (SupplierOrder, String) -> Unit,
) {
    when {
        showFullScreenLoading(state.loading, state.orders.isNotEmpty()) -> PegasusLoadingState(
            title = stringResource(R.string.mobile_supplier_ui_loading_orders),
            body = "Supplier order queue",
            modifier = modifier,
        )
        state.error != null -> PegasusStatePane(
            kind = PegasusStateKind.Error,
            headline = "Orders unavailable",
            body = state.error!!,
            actionLabel = "Retry",
            onAction = onRetry,
            modifier = modifier,
        )
        state.orders.isEmpty() -> PegasusStatePane(
            kind = PegasusStateKind.Empty,
            headline = "No orders",
            body = "Orders for this filter will appear here.",
            modifier = modifier,
        )
        else -> LazyVerticalGrid(
            columns = GridCells.Adaptive(minSize = 340.dp),
            modifier = modifier.fillMaxSize(),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            items(state.orders, key = { it.orderId }) { order ->
                val amount = formatMinorAmount(order.totalMinor, order.currency)
                var menuExpanded by remember(order.orderId) { mutableStateOf(false) }
                var rejectDialog by remember(order.orderId) { mutableStateOf(false) }
                var delayDialog by remember(order.orderId) { mutableStateOf(false) }
                var reason by remember(order.orderId) { mutableStateOf("") }
                val warehouseOps = canWarehouseOps(order)

                if (rejectDialog) {
                    AlertDialog(
                        onDismissRequest = { rejectDialog = false; reason = "" },
                        title = { Text("Reject order") },
                        text = {
                            OutlinedTextField(
                                value = reason,
                                onValueChange = { reason = it },
                                label = { Text("Reason") },
                                modifier = Modifier.fillMaxWidth(),
                            )
                        },
                        confirmButton = {
                            TextButton(
                                onClick = {
                                    onReject(order, reason.trim())
                                    rejectDialog = false
                                    reason = ""
                                },
                                enabled = reason.isNotBlank(),
                            ) { Text("Reject") }
                        },
                        dismissButton = { TextButton(onClick = { rejectDialog = false; reason = "" }) { Text("Cancel") } },
                    )
                }
                if (delayDialog) {
                    val datePickerState = rememberDatePickerState(initialSelectedDateMillis = System.currentTimeMillis())
                    var showReasonStep by remember(order.orderId) { mutableStateOf(false) }
                    if (showReasonStep) {
                        AlertDialog(
                            onDismissRequest = { showReasonStep = false },
                            title = { Text("Reason for new delivery date") },
                            text = {
                                OutlinedTextField(
                                    value = reason,
                                    onValueChange = { reason = it },
                                    label = { Text("Reason (required)") },
                                    modifier = Modifier.fillMaxWidth(),
                                )
                            },
                            confirmButton = {
                                TextButton(
                                    onClick = {
                                        val millis = datePickerState.selectedDateMillis ?: return@TextButton
                                        val iso = java.time.Instant.ofEpochMilli(millis)
                                            .atOffset(java.time.ZoneOffset.ofHours(5))
                                            .withHour(12).withMinute(0).withSecond(0)
                                            .format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME)
                                        onDelay(order, iso, reason.trim())
                                        delayDialog = false
                                        showReasonStep = false
                                        reason = ""
                                    },
                                    enabled = reason.isNotBlank(),
                                ) { Text("Notify retailer") }
                            },
                            dismissButton = { TextButton(onClick = { showReasonStep = false }) { Text("Back") } },
                        )
                    } else {
                        DatePickerDialog(
                            onDismissRequest = { delayDialog = false },
                            confirmButton = { TextButton(onClick = { showReasonStep = true }) { Text("Next") } },
                            dismissButton = { TextButton(onClick = { delayDialog = false }) { Text("Cancel") } },
                        ) {
                            DatePicker(state = datePickerState)
                        }
                    }
                }

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    SupplierOpsListCard(
                        modifier = Modifier
                            .weight(1f)
                            .clickable { onOrderClick(order) },
                        headline = order.retailerId.ifBlank { order.orderId.take(12) },
                        supporting = buildString {
                            append(amount)
                            append(" · ")
                            append(order.orderId.take(12))
                            order.updatedAt.takeIf { it.isNotBlank() }?.let { append(" · $it") }
                        },
                        status = order.status.ifBlank { order.decision },
                    )
                    Box {
                        IconButton(onClick = { menuExpanded = true }, modifier = Modifier.size(44.dp)) {
                            Icon(Icons.Default.MoreVert, contentDescription = stringResource(R.string.supplier_portal_orders_order_kebab_menu_text_order_actions))
                        }
                        DropdownMenu(expanded = menuExpanded, onDismissRequest = { menuExpanded = false }) {
                            DropdownMenuItem(
                                text = { Text("View details") },
                                onClick = { menuExpanded = false; onOrderClick(order) },
                            )
                            if (warehouseOps) {
                                DropdownMenuItem(
                                    text = { Text("Delay delivery") },
                                    onClick = { menuExpanded = false; delayDialog = true },
                                )
                                DropdownMenuItem(
                                    text = { Text("Reject") },
                                    onClick = { menuExpanded = false; rejectDialog = true },
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
