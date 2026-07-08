package com.pegasusx.supplier.ui.screens.orders

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.components.formatMinorAmount
import com.pegasus.design.showFullScreenLoading
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.OrderFilterTab
import com.pegasusx.supplier.ui.viewmodel.OrdersViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrdersScreen(
    viewModel: OrdersViewModel = hiltViewModel(),
) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(state.reassignMessage) {
        state.reassignMessage?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.dismissReassignMessage()
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
        topBar = {
            TopAppBar(
                title = { Text("Orders") },
                actions = {
                    IconButton(onClick = { viewModel.load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
    ) { padding ->
        Column(Modifier.padding(padding).fillMaxSize()) {
            ScrollableTabRow(selectedTabIndex = state.filter.ordinal) {
                OrderFilterTab.entries.forEach { tab ->
                    Tab(
                        selected = state.filter == tab,
                        onClick = { viewModel.setFilter(tab) },
                        text = { Text(tab.name.lowercase().replaceFirstChar { it.uppercase() }) },
                    )
                }
            }
            when {
                showFullScreenLoading(state.loading, state.orders.isNotEmpty()) -> SupplierLoadingState(
                    title = "Loading orders…",
                    body = "Supplier order queue",
                )
                state.error != null -> SupplierStatePane(
                    kind = SupplierStateKind.Error,
                    headline = "Orders unavailable",
                    body = state.error!!,
                    actionLabel = "Retry",
                    onAction = { viewModel.load() },
                )
                state.orders.isEmpty() -> SupplierStatePane(
                    kind = SupplierStateKind.Empty,
                    headline = "No orders",
                    body = "Orders for this filter will appear here.",
                )
                else -> LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    items(state.orders, key = { it.orderId }) { order ->
                        val amount = formatMinorAmount(order.totalMinor, order.currency)
                        SupplierOpsListCard(
                            headline = order.orderId.take(12),
                            supporting = buildString {
                                append(amount)
                                append(" · Retailer ")
                                append(order.retailerId.take(8))
                                order.updatedAt.takeIf { it.isNotBlank() }?.let { append(" · $it") }
                            },
                            status = order.status.ifBlank { order.decision },
                            onReassign = if (viewModel.canWarehouseOps(order)) {
                                { viewModel.openReassignDialog(order.orderId) }
                            } else null,
                        )
                    }
                }
            }
        }
    }

    state.reassignTarget?.let { orderId ->
        val recs = state.reassignRecommendations
        AlertDialog(
            onDismissRequest = { viewModel.closeReassignDialog() },
            title = { Text("Reassign Order ${orderId.take(8)}") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (recs == null) {
                        CircularProgressIndicator(modifier = Modifier.align(Alignment.CenterHorizontally))
                    } else if (recs.recommendations.isEmpty()) {
                        Text("No suitable trucks available.")
                    } else {
                        Text(
                            "${recs.retailerName} • %.1f VU".format(recs.orderVolumeVu),
                            style = MaterialTheme.typography.bodyMedium,
                        )
                        LazyColumn(
                            verticalArrangement = Arrangement.spacedBy(6.dp),
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(280.dp),
                        ) {
                            items(recs.recommendations, key = { it.driverId }) { rec ->
                                ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                    Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                                        Row(verticalAlignment = Alignment.CenterVertically) {
                                            Text(
                                                rec.driverName.ifBlank { rec.driverId.take(8) },
                                                style = MaterialTheme.typography.titleSmall,
                                                modifier = Modifier.weight(1f)
                                            )
                                            Text("score %.2f".format(rec.score), style = MaterialTheme.typography.labelMedium)
                                        }
                                        Text(
                                            listOfNotNull(
                                                rec.licensePlate.takeIf { it.isNotBlank() },
                                                rec.vehicleClass.takeIf { it.isNotBlank() }
                                            ).joinToString(" • "),
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        )
                                        Row(
                                            modifier = Modifier.fillMaxWidth(),
                                            horizontalArrangement = Arrangement.spacedBy(8.dp, Alignment.End)
                                        ) {
                                            OutlinedButton(
                                                onClick = {
                                                    viewModel.applyReassign(orderId, rec.driverId, true)
                                                },
                                                enabled = !state.isReassigning,
                                                contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp),
                                            ) { Text("Partial") }
                                            Button(
                                                onClick = {
                                                    viewModel.applyReassign(orderId, rec.driverId, false)
                                                },
                                                enabled = !state.isReassigning,
                                                contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp),
                                            ) { Text("Complete") }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            },
            confirmButton = {},
            dismissButton = {
                TextButton(onClick = { viewModel.closeReassignDialog() }, enabled = !state.isReassigning) {
                    Text("Close")
                }
            }
        )
    }
}
