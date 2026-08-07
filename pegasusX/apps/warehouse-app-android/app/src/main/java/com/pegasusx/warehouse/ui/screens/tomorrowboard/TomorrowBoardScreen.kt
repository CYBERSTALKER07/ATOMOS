package com.pegasusx.warehouse.ui.screens.tomorrowboard

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.WarehouseOpsBoardOrder
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import kotlinx.coroutines.launch
import java.time.LocalDate

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TomorrowBoardScreen(
    api: WarehouseApi,
    realtimeSignals: WarehouseRealtimeSignals,
    onBack: (() -> Unit)? = null,
) {
    var date by remember { mutableStateOf(LocalDate.now().plusDays(1).toString()) }
    var preorders by remember { mutableStateOf<List<WarehouseOpsBoardOrder>>(emptyList()) }
    var deliverBefore by remember { mutableStateOf<List<WarehouseOpsBoardOrder>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        scope.launch {
            if (!silent) loading = true
            error = null
            try {
                val resp = api.getOpsBoard(date)
                if (resp.isSuccessful) {
                    val body = resp.body()
                    preorders = body?.preorders.orEmpty()
                    deliverBefore = body?.deliverBefore.orEmpty()
                } else if (!silent) {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                if (!silent) error = e.message
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(date) { load() }
    LaunchedEffect(Unit) {
        realtimeSignals.refreshTick.collect { load(silent = true) }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Tomorrow board") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                        }
                    }
                },
            )
        },
    ) { innerPadding ->
        if (loading) {
            Box(Modifier.padding(innerPadding)) {
                com.pegasus.design.PegasusLoadingState(
                    title = stringResource(R.string.mobile_warehouse_ui_loading_board),
                    body = "Fetching operations for $date",
                )
            }
            return@Scaffold
        }

        val rows = preorders.map { it to "Pre-order" } + deliverBefore.map { it to "Deliver by" }

        LazyVerticalGrid(
            columns = GridCells.Adaptive(minSize = 340.dp),
            modifier = Modifier.fillMaxSize().padding(innerPadding).padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item(span = { GridItemSpan(maxLineSpan) }) {
                OutlinedTextField(
                    value = date,
                    onValueChange = { date = it },
                    label = { Text("Date (YYYY-MM-DD)") },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
            }
            if (error != null) {
                item(span = { GridItemSpan(maxLineSpan) }) { 
                    com.pegasus.design.PegasusStatePane(
                        kind = com.pegasus.design.PegasusStateKind.Error,
                        headline = "Failed to load board",
                        body = error!!,
                        actionLabel = "Retry",
                        onAction = { load() }
                    )
                }
            }
            if (rows.isEmpty() && error == null) {
                item(span = { GridItemSpan(maxLineSpan) }) { 
                    com.pegasus.design.PegasusStatePane(
                        kind = com.pegasus.design.PegasusStateKind.Empty,
                        headline = "No operations",
                        body = "No orders scheduled for this date."
                    )
                }
            } else if (rows.isNotEmpty()) {
                items(rows, key = { (order, lane) -> "$lane-${order.orderId}" }) { (order, lane) ->
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(lane, style = MaterialTheme.typography.titleSmall)
                            Text(order.orderId, style = MaterialTheme.typography.bodyMedium)
                            Text(
                                order.deliveryExpectation?.targetLabel ?: order.status,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                }
            }
        }
    }
}
