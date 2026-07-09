package com.pegasusx.factory.ui.screens.loadingbay

import androidx.compose.ui.unit.dp

import androidx.compose.foundation.lazy.grid.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.DispatchRequest
import com.pegasusx.factory.data.model.PulseEvent
import com.pegasusx.factory.data.model.Transfer
import com.pegasusx.factory.util.filterHandoffPulseEvents
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasusx.factory.ui.components.FactorySectionHeader
import com.pegasusx.factory.ui.components.HandoffTimelineSection
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.components.FactoryStatusChip
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.PegasusSpacing
import com.pegasusx.factory.util.FactoryIdempotencyKeys
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LoadingBayScreen(
    api: FactoryApi,
    onTransferClick: (String) -> Unit,
    onBack: () -> Unit,
) {
    var transfers by remember { mutableStateOf<List<Transfer>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var dispatching by remember { mutableStateOf(false) }
    var handoffEvents by remember { mutableStateOf<List<PulseEvent>>(emptyList()) }
    var handoffLoading by remember { mutableStateOf(true) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load(silent: Boolean = false) {
        if (!silent) {
            loading = true
        }
        error = null
        scope.launch {
            try {
                val resp = api.getLoadingBayTransfers()
                if (resp.isSuccessful && resp.body() != null) {
                    transfers = resp.body()!!.transfers
                } else {
                    error = "Failed (${resp.code()})"
                }
                handoffLoading = true
                val pulseResp = api.getPulse()
                handoffEvents = if (pulseResp.isSuccessful && pulseResp.body() != null) {
                    filterHandoffPulseEvents(pulseResp.body()!!.events)
                } else {
                    emptyList()
                }
                handoffLoading = false
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                if (!silent) {
                    loading = false
                }
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.TransferUpdate,
            FactoryRealtimeEventType.ManifestUpdate,
        ),
    ) {
        if (!dispatching) {
            load(silent = transfers.isNotEmpty())
        }
    }

    val approved = transfers.filter { it.state == "APPROVED" }
    val loadingState = transfers.filter { it.state == "LOADING" }
    val dispatched = transfers.filter { it.state == "DISPATCHED" }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text("Loading Bay")
                        Text(
                            text = "Approved, loading, and dispatched queues",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
        floatingActionButton = {
            if (loadingState.isNotEmpty()) {
                ExtendedFloatingActionButton(
                    text = { Text(if (dispatching) "Dispatching…" else "Batch Dispatch") },
                    icon = { Icon(Icons.Default.LocalShipping, null) },
                    onClick = {
                        if (dispatching) return@ExtendedFloatingActionButton
                        dispatching = true
                        scope.launch {
                            try {
                                val ids = loadingState.map { it.id }
                                val resp = api.dispatch(
                                    DispatchRequest(transferIds = ids),
                                    FactoryIdempotencyKeys.batchDispatch(ids),
                                )
                                if (resp.isSuccessful) {
                                    snackbarHostState.showSnackbar("Dispatched ${ids.size} transfers")
                                    load()
                                } else {
                                    snackbarHostState.showSnackbar("Dispatch failed (${resp.code()})")
                                }
                            } catch (e: Exception) {
                                snackbarHostState.showSnackbar(e.message ?: "Error")
                            } finally {
                                dispatching = false
                            }
                        }
                    },
                )
            }
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        when {
            loading && transfers.isEmpty() -> PegasusLoadingState(
                title = "Loading bay status",
                body = "Fetching approved, loading, and dispatched transfer groups for the bay.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load loading bay",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md)
    ) {
                item {
                    BayOverviewCard(
                        readyCount = approved.size,
                        loadingCount = loadingState.size,
                        dispatchedCount = dispatched.size,
                    )
                }
                item {
                    HandoffTimelineSection(
                        events = handoffEvents,
                        loading = handoffLoading,
                    )
                }
                item { FactorySectionHeader(title = "Ready for Loading", count = approved.size) }
                if (approved.isEmpty()) {
                    item(span = { GridItemSpan(maxLineSpan) }) { 
                        PegasusStatePane(
                            kind = PegasusStateKind.Empty,
                            headline = "Empty Queue",
                            body = "No approved transfers are waiting at the bay."
                        )
                    }
                } else {
                    items(approved, key = { it.id }) { transfer ->
                        TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
                    }
                }
                item { FactorySectionHeader(title = "Now Loading", count = loadingState.size) }
                if (loadingState.isEmpty()) {
                    item(span = { GridItemSpan(maxLineSpan) }) { 
                        PegasusStatePane(
                            kind = PegasusStateKind.Empty,
                            headline = "Empty Queue",
                            body = "Nothing is actively loading right now."
                        )
                    }
                } else {
                    items(loadingState, key = { it.id }) { transfer ->
                        TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
                    }
                }
                item { FactorySectionHeader(title = "Dispatched", count = dispatched.size) }
                if (dispatched.isEmpty()) {
                    item(span = { GridItemSpan(maxLineSpan) }) { 
                        PegasusStatePane(
                            kind = PegasusStateKind.Empty,
                            headline = "Empty Queue",
                            body = "No transfers have been dispatched in the current view."
                        )
                    }
                } else {
                    items(dispatched, key = { it.id }) { transfer ->
                        TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
                    }
                }
            }
        }
    }
}

@Composable
private fun TransferCard(transfer: Transfer, onClick: () -> Unit) {
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
private fun BayOverviewCard(
    readyCount: Int,
    loadingCount: Int,
    dispatchedCount: Int,
) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                text = "Loading bay flow",
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = "Track approved transfers, active loading work, and dispatched volume from one queue.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FactoryMetricTile("Ready", readyCount.toString(), Modifier.weight(1f))
                FactoryMetricTile("Loading", loadingCount.toString(), Modifier.weight(1f))
                FactoryMetricTile("Out", dispatchedCount.toString(), Modifier.weight(1f))
            }
        }
    }
}
