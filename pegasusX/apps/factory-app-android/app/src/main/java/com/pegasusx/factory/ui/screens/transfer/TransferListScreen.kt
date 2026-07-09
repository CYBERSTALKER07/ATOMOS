package com.pegasusx.factory.ui.screens.transfer

import androidx.compose.ui.unit.dp

import androidx.compose.foundation.lazy.grid.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.Transfer
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.components.FactoryStatusChip
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

private val STATE_FILTERS = listOf("ALL", "DRAFT", "APPROVED", "LOADING", "DISPATCHED", "IN_TRANSIT", "ARRIVED", "RECEIVED", "CANCELLED")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransferListScreen(
    api: FactoryApi,
    onTransferClick: (String) -> Unit,
    onCreateTransfer: () -> Unit,
    onBack: () -> Unit,
) {
    var transfers by remember { mutableStateOf<List<Transfer>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var selectedFilter by remember { mutableStateOf("ALL") }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        if (!silent) {
            loading = true
        }
        error = null
        scope.launch {
            try {
                val state = if (selectedFilter == "ALL") null else selectedFilter
                val resp = api.getTransfers(state = state)
                if (resp.isSuccessful && resp.body() != null) {
                    transfers = resp.body()!!.transfers
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                if (!silent) {
                    loading = false
                }
            }
        }
    }

    LaunchedEffect(selectedFilter) { load() }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.TransferUpdate,
            FactoryRealtimeEventType.ManifestUpdate,
        ),
    ) {
        load(silent = true)
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text("Transfers")
                        Text(
                            text = "Factory-to-warehouse movement pipeline",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = onCreateTransfer) { Icon(Icons.Default.Add, "Create transfer") }
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            Row(
                modifier = Modifier
                    .horizontalScroll(rememberScrollState())
                    .padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                STATE_FILTERS.forEach { filter ->
                    FilterChip(
                        selected = selectedFilter == filter,
                        onClick = { selectedFilter = filter },
                        label = { Text(filter, style = MaterialTheme.typography.labelSmall) },
                    )
                }
            }

            when {
                loading && transfers.isEmpty() -> PegasusLoadingState(
                    title = "Loading transfers",
                    body = "Fetching the current transfer pipeline for this factory.",
                    modifier = Modifier.fillMaxSize(),
                )
                error != null -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Unable to load transfers",
                    body = error!!,
                    actionLabel = "Retry",
                    onAction = { load() },
                    modifier = Modifier.fillMaxSize(),
                )
                transfers.isEmpty() -> PegasusStatePane(
                    kind = if (selectedFilter == "ALL") PegasusStateKind.Empty else PegasusStateKind.NoResults,
                    headline = if (selectedFilter == "ALL") "No transfers available" else "No ${selectedFilter.replace('_', ' ')} transfers",
                    body = if (selectedFilter == "ALL") {
                        "Transfers will appear here as soon as warehouse demand enters the factory pipeline."
                    } else {
                        "Adjust the active state filter or wait for the next transfer update."
                    },
                    actionLabel = if (selectedFilter == "ALL") null else "Clear Filter",
                    onAction = if (selectedFilter == "ALL") null else ({ selectedFilter = "ALL" }),
                    modifier = Modifier.fillMaxSize(),
                )
                else -> LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md)
    ) {
                    item {
                        TransferListSummary(
                            count = transfers.size,
                            selectedFilter = selectedFilter,
                        )
                    }
                    items(transfers, key = { it.id }) { transfer ->
                        TransferRow(transfer, onClick = { onTransferClick(transfer.id) })
                    }
                }
            }
        }
    }
}

@Composable
private fun TransferRow(transfer: Transfer, onClick: () -> Unit) {
    ElevatedCard(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Row(
                verticalAlignment = Alignment.Top,
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                    Text(
                        text = transfer.warehouseName.ifBlank { transfer.warehouseId.take(8) },
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text(
                        text = "Transfer ${transfer.id.take(8)}",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Column(
                    horizontalAlignment = Alignment.End,
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                ) {
                    FactoryStatusChip(status = transfer.state)
                    FactoryStatusChip(status = transfer.priority.ifBlank { "STANDARD" })
                }
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FactoryMetricTile(
                    label = "Items",
                    value = transfer.totalItems.toString(),
                    modifier = Modifier.weight(1f),
                )
                FactoryMetricTile(
                    label = "Volume",
                    value = "${String.format("%.0f", transfer.totalVolumeL)}L",
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}

@Composable
private fun TransferListSummary(
    count: Int,
    selectedFilter: String,
) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
        ) {
            Text(
                text = "$count transfers in view",
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = if (selectedFilter == "ALL") {
                    "Showing every transfer state across the factory queue."
                } else {
                    "Filtered to $selectedFilter transfers."
                },
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}
