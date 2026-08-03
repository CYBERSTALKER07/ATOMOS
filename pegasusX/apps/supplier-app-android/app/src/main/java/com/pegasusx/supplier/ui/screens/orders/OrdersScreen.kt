package com.pegasusx.supplier.ui.screens.orders

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasus.design.showFullScreenLoading
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
                showFullScreenLoading(state.loading, state.orders.isNotEmpty()) -> PegasusLoadingState(
                    title = "Loading orders…",
                    body = "Supplier order queue",
                )
                state.error != null -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Orders unavailable",
                    body = state.error!!,
                    actionLabel = "Retry",
                    onAction = { viewModel.load() },
                )
                state.orders.isEmpty() -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No orders",
                    body = "Orders for this filter will appear here.",
                )
                else -> OrdersList(
                    orders = state.orders,
                    onReassign = { order ->
                        if (viewModel.canWarehouseOps(order)) {
                            viewModel.openReassignDialog(order.orderId)
                        }
                    }
                )
            }
        }
    }

    state.reassignTarget?.let { orderId ->
        ReassignOrderDialog(
            orderId = orderId,
            recs = state.reassignRecommendations,
            isReassigning = state.isReassigning,
            onDismiss = { viewModel.closeReassignDialog() },
            onApplyReassign = { id, driver, partial -> viewModel.applyReassign(id, driver, partial) }
        )
    }
}
