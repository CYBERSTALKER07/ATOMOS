package com.pegasus.factory.ui.screens.transfer

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasus.factory.data.model.Transfer
import com.pegasus.factory.data.model.TransitionRequest
import com.pegasus.factory.data.remote.FactoryApi
import com.pegasus.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransferDetailScreen(
    api: FactoryApi,
    transferId: String,
    onBack: () -> Unit,
) {
    var transfer by remember { mutableStateOf<Transfer?>(null) }
    var loading by remember { mutableStateOf(true) }
    var transitioning by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getTransfer(transferId)
                if (resp.isSuccessful && resp.body() != null) {
                    transfer = resp.body()!!
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

    fun transition(target: String) {
        transitioning = true
        scope.launch {
            try {
                val resp = api.transitionTransfer(transferId, TransitionRequest(targetState = target))
                if (resp.isSuccessful && resp.body() != null) {
                    transfer = resp.body()!!
                    snackbarHostState.showSnackbar("Transitioned to $target")
                } else {
                    snackbarHostState.showSnackbar("Failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Error")
            } finally {
                transitioning = false
            }
        }
    }

    LaunchedEffect(transferId) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Transfer Detail") },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
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
            transfer != null -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                item {
                    TransferOverviewCard(transfer = transfer!!)
                }
                item {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        SummaryCard("Items", "${transfer!!.totalItems}", Modifier.weight(1f))
                        SummaryCard("Volume", "${String.format("%.0f", transfer!!.totalVolumeL)}L", Modifier.weight(1f))
                    }
                }

                item {
                    val state = transfer!!.state
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                    ) {
                        if (state == "APPROVED") {
                            Button(
                                onClick = { transition("LOADING") },
                                enabled = !transitioning,
                                modifier = Modifier
                                    .weight(1f)
                                    .height(PegasusSpacing.xxxl),
                            ) { Text("Start loading") }
                        }
                        if (state == "LOADING") {
                            FilledTonalButton(
                                onClick = { transition("DISPATCHED") },
                                enabled = !transitioning,
                                modifier = Modifier
                                    .weight(1f)
                                    .height(PegasusSpacing.xxxl),
                            ) { Text("Mark dispatched") }
                        }
                        if (state != "APPROVED" && state != "LOADING") {
                            Surface(
                                modifier = Modifier.fillMaxWidth(),
                                shape = MaterialTheme.shapes.medium,
                                color = MaterialTheme.colorScheme.surfaceContainerLowest,
                            ) {
                                Text(
                                    text = "No manual transition is available for the current state.",
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    modifier = Modifier.padding(PegasusSpacing.lg),
                                )
                            }
                        }
                    }
                }

                item {
                    HorizontalDivider()
                    Spacer(Modifier.height(PegasusSpacing.sm))
                    Text("Items", style = MaterialTheme.typography.titleMedium)
                }

                items(transfer!!.items) { item ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            modifier = Modifier.padding(PegasusSpacing.lg),
                            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                        ) {
                            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                                Text(
                                    text = item.productName.ifBlank { item.productId.take(8) },
                                    style = MaterialTheme.typography.titleMedium,
                                )
                                Text(
                                    text = item.productId.take(8),
                                    style = MaterialTheme.typography.labelMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                            ) {
                                SummaryCard("Qty", "${item.quantity}", Modifier.weight(1f))
                                SummaryCard("Available", "${item.quantityAvailable}", Modifier.weight(1f))
                                SummaryCard("Volume", "${String.format("%.1f", item.unitVolumeL)}L", Modifier.weight(1f))
                            }
                        }
                    }
                }

                if (transfer!!.notes.isNotBlank()) {
                    item {
                        Spacer(Modifier.height(PegasusSpacing.sm))
                        HorizontalDivider()
                        Spacer(Modifier.height(PegasusSpacing.sm))
                        Text("Notes", style = MaterialTheme.typography.titleMedium)
                        Spacer(Modifier.height(PegasusSpacing.xs))
                        Surface(
                            modifier = Modifier.fillMaxWidth(),
                            shape = MaterialTheme.shapes.medium,
                            color = MaterialTheme.colorScheme.surfaceContainerLowest,
                        ) {
                            Text(
                                text = transfer!!.notes,
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(PegasusSpacing.lg),
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun TransferOverviewCard(transfer: Transfer) {
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
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                Text(
                    text = transfer.warehouseName.ifBlank { transfer.warehouseId.take(8) },
                    style = MaterialTheme.typography.titleLarge,
                )
                Text(
                    text = "Transfer ${transfer.id.take(8)}",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Row(
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                DetailTag(
                    text = transfer.state,
                    containerColor = MaterialTheme.colorScheme.secondaryContainer,
                    contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
                )
                DetailTag(
                    text = transfer.priority.ifBlank { "STANDARD" },
                    containerColor = MaterialTheme.colorScheme.surfaceContainerHighest,
                    contentColor = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun DetailTag(
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

@Composable
private fun SummaryCard(label: String, value: String, modifier: Modifier = Modifier) {
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
                label,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}
