package com.pegasus.factory.ui.screens.loadingbay

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasus.factory.data.model.DispatchRequest
import com.pegasus.factory.data.model.Transfer
import com.pegasus.factory.data.remote.FactoryApi
import com.pegasus.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasus.factory.ui.theme.PegasusSpacing
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
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getLoadingBayTransfers()
                if (resp.isSuccessful && resp.body() != null) {
                    transfers = resp.body()!!.transfers
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
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
            load()
        }
    }

    val approved = transfers.filter { it.state == "APPROVED" }
    val loadingState = transfers.filter { it.state == "LOADING" }
    val dispatched = transfers.filter { it.state == "DISPATCHED" }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Loading Bay") },
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
                                val resp = api.dispatch(DispatchRequest(transferIds = ids))
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
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                item {
                    BayOverviewCard(
                        readyCount = approved.size,
                        loadingCount = loadingState.size,
                        dispatchedCount = dispatched.size,
                    )
                }
                item { BayHeader("Ready for Loading", approved.size) }
                if (approved.isEmpty()) {
                    item { EmptyBayState("No approved transfers are waiting at the bay.") }
                } else {
                    items(approved, key = { it.id }) { transfer ->
                        TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
                    }
                }
                item { BayHeader("Now Loading", loadingState.size) }
                if (loadingState.isEmpty()) {
                    item { EmptyBayState("Nothing is actively loading right now.") }
                } else {
                    items(loadingState, key = { it.id }) { transfer ->
                        TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
                    }
                }
                item { BayHeader("Dispatched", dispatched.size) }
                if (dispatched.isEmpty()) {
                    item { EmptyBayState("No transfers have been dispatched in the current view.") }
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
private fun BayHeader(title: String, count: Int) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.padding(top = PegasusSpacing.sm),
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.width(PegasusSpacing.sm))
        Badge { Text("$count") }
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
                    MetaPill(
                        text = transfer.state,
                        containerColor = MaterialTheme.colorScheme.secondaryContainer,
                        contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
                    )
                    MetaPill(
                        text = transfer.priority.ifBlank { "STANDARD" },
                        containerColor = MaterialTheme.colorScheme.surfaceContainerHighest,
                        contentColor = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                BayMetric(
                    label = "Items",
                    value = transfer.totalItems.toString(),
                    modifier = Modifier.weight(1f),
                )
                BayMetric(
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
                BayMetric("Ready", readyCount.toString(), Modifier.weight(1f))
                BayMetric("Loading", loadingCount.toString(), Modifier.weight(1f))
                BayMetric("Out", dispatchedCount.toString(), Modifier.weight(1f))
            }
        }
    }
}

@Composable
private fun BayMetric(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier,
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.surfaceContainer,
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
        ) {
            Text(value, style = MaterialTheme.typography.titleLarge)
            Text(
                text = label,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun EmptyBayState(message: String) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.surfaceContainerLowest,
    ) {
        Text(
            text = message,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(PegasusSpacing.lg),
        )
    }
}

@Composable
private fun MetaPill(
    text: String,
    containerColor: androidx.compose.ui.graphics.Color,
    contentColor: androidx.compose.ui.graphics.Color,
) {
    Surface(
        shape = MaterialTheme.shapes.small,
        color = containerColor,
        contentColor = contentColor,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelMedium,
            modifier = Modifier.padding(horizontal = PegasusSpacing.sm, vertical = PegasusSpacing.xs),
        )
    }
}
