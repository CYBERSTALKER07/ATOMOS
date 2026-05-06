package com.pegasus.factory.ui.screens.transfer

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasus.factory.data.model.Transfer
import com.pegasus.factory.data.remote.FactoryApi
import com.pegasus.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasus.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

private val STATE_FILTERS = listOf("ALL", "DRAFT", "APPROVED", "LOADING", "DISPATCHED", "IN_TRANSIT", "ARRIVED", "RECEIVED", "CANCELLED")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransferListScreen(
    api: FactoryApi,
    onTransferClick: (String) -> Unit,
    onBack: () -> Unit,
) {
    var transfers by remember { mutableStateOf<List<Transfer>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var selectedFilter by remember { mutableStateOf("ALL") }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
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
                loading = false
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
        load()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Transfers") },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
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
            // Filter chips
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
                loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
                error != null -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(error!!, color = MaterialTheme.colorScheme.error)
                        Spacer(Modifier.height(PegasusSpacing.lg))
                        Button(onClick = { load() }) { Text("Retry") }
                    }
                }
                transfers.isEmpty() -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    EmptyTransferListState(selectedFilter = selectedFilter)
                }
                else -> LazyColumn(
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
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
                    TransferTag(
                        text = transfer.state,
                        containerColor = MaterialTheme.colorScheme.secondaryContainer,
                        contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
                    )
                    TransferTag(
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
                TransferMetric("Items", transfer.totalItems.toString(), Modifier.weight(1f))
                TransferMetric("Volume", "${String.format("%.0f", transfer.totalVolumeL)}L", Modifier.weight(1f))
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

@Composable
private fun EmptyTransferListState(selectedFilter: String) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        Text(
            text = "No transfers found",
            style = MaterialTheme.typography.titleMedium,
        )
        Text(
            text = if (selectedFilter == "ALL") {
                "There are no transfers available right now."
            } else {
                "There are no $selectedFilter transfers in the current queue."
            },
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun TransferMetric(
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
            Text(value, style = MaterialTheme.typography.titleMedium)
            Text(
                text = label,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun TransferTag(
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
