package com.pegasusx.supplier.ui.screens.orders

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
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
    val showVetActions = state.filter == OrderFilterTab.REVIEW || state.filter == OrderFilterTab.ACTIVE

    Scaffold(
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
                        val vetting = state.vettingId == order.orderId
                        SupplierOpsListCard(
                            headline = order.orderId.take(12),
                            supporting = buildString {
                                append(amount)
                                append(" · Retailer ")
                                append(order.retailerId.take(8))
                                order.updatedAt.takeIf { it.isNotBlank() }?.let { append(" · $it") }
                            },
                            status = order.status.ifBlank { order.decision },
                        )
                        if (showVetActions && order.status.equals("AWAITING_REVIEW", ignoreCase = true)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                            ) {
                                OutlinedButton(
                                    onClick = { viewModel.vetOrder(order, "REJECTED") },
                                    enabled = !vetting,
                                    modifier = Modifier.weight(1f),
                                ) { Text("Reject") }
                                Button(
                                    onClick = { viewModel.vetOrder(order, "APPROVED") },
                                    enabled = !vetting,
                                    modifier = Modifier.weight(1f),
                                ) { Text("Approve") }
                            }
                        }
                    }
                }
            }
        }
    }
}
