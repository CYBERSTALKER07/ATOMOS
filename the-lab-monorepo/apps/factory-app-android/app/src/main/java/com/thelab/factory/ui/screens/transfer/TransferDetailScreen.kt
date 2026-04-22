package com.thelab.factory.ui.screens.transfer

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
import com.thelab.factory.data.model.Transfer
import com.thelab.factory.data.model.TransitionRequest
import com.thelab.factory.data.remote.FactoryApi
import com.thelab.factory.ui.theme.LabSpacing
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
                    Spacer(Modifier.height(LabSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            transfer != null -> LazyColumn(
                contentPadding = PaddingValues(LabSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(LabSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                // Summary cards row
                item {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(LabSpacing.md),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        SummaryCard("State", transfer!!.state, Modifier.weight(1f))
                        SummaryCard("Priority", transfer!!.priority, Modifier.weight(1f))
                    }
                }
                item {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(LabSpacing.md),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        SummaryCard("Items", "${transfer!!.totalItems}", Modifier.weight(1f))
                        SummaryCard("Volume", "${String.format("%.0f", transfer!!.totalVolumeL)}L", Modifier.weight(1f))
                    }
                }

                // Warehouse
                item {
                    Text(
                        text = "Warehouse: ${transfer!!.warehouseName.ifBlank { transfer!!.warehouseId.take(8) }}",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }

                // Action buttons
                item {
                    val state = transfer!!.state
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(LabSpacing.md),
                    ) {
                        if (state == "APPROVED") {
                            Button(
                                onClick = { transition("LOADING") },
                                enabled = !transitioning,
                            ) { Text("Start Loading") }
                        }
                        if (state == "LOADING") {
                            Button(
                                onClick = { transition("DISPATCHED") },
                                enabled = !transitioning,
                            ) { Text("Mark Dispatched") }
                        }
                    }
                }

                // Items header
                item {
                    HorizontalDivider()
                    Spacer(Modifier.height(LabSpacing.sm))
                    Text("Items", style = MaterialTheme.typography.titleMedium)
                }

                // Items list
                items(transfer!!.items) { item ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Row(
                            modifier = Modifier.padding(LabSpacing.lg),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = item.productName.ifBlank { item.productId.take(8) },
                                    style = MaterialTheme.typography.titleSmall,
                                )
                                Text(
                                    text = "Qty: ${item.quantity} · Available: ${item.quantityAvailable}",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            Text(
                                text = "${String.format("%.1f", item.unitVolumeL)}L/unit",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }

                // Notes
                if (transfer!!.notes.isNotBlank()) {
                    item {
                        Spacer(Modifier.height(LabSpacing.sm))
                        HorizontalDivider()
                        Spacer(Modifier.height(LabSpacing.sm))
                        Text("Notes", style = MaterialTheme.typography.titleMedium)
                        Spacer(Modifier.height(LabSpacing.xs))
                        Text(
                            text = transfer!!.notes,
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun SummaryCard(label: String, value: String, modifier: Modifier = Modifier) {
    ElevatedCard(modifier = modifier) {
        Column(modifier = Modifier.padding(LabSpacing.md)) {
            Text(value, style = MaterialTheme.typography.titleMedium)
            Spacer(Modifier.height(2.dp))
            Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}
