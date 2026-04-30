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
import com.pegasus.factory.ui.theme.LabSpacing
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
                    Spacer(Modifier.height(LabSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(LabSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(LabSpacing.sm),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                item { BayHeader("Ready for Loading", approved.size) }
                items(approved, key = { it.id }) { transfer ->
                    TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
                }
                item { Spacer(Modifier.height(LabSpacing.lg)) }
                item { BayHeader("Now Loading", loadingState.size) }
                items(loadingState, key = { it.id }) { transfer ->
                    TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
                }
                item { Spacer(Modifier.height(LabSpacing.lg)) }
                item { BayHeader("Dispatched", dispatched.size) }
                items(dispatched, key = { it.id }) { transfer ->
                    TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
                }
            }
        }
    }
}

@Composable
private fun BayHeader(title: String, count: Int) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.padding(vertical = LabSpacing.sm),
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.width(LabSpacing.sm))
        Badge { Text("$count") }
    }
}

@Composable
private fun TransferCard(transfer: Transfer, onClick: () -> Unit) {
    ElevatedCard(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            modifier = Modifier.padding(LabSpacing.lg),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = transfer.warehouseName.ifBlank { transfer.warehouseId.take(8) },
                    style = MaterialTheme.typography.titleSmall,
                )
                Spacer(Modifier.height(2.dp))
                Text(
                    text = "${transfer.totalItems} items · ${String.format("%.0f", transfer.totalVolumeL)}L",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            AssistChip(
                onClick = {},
                label = { Text(transfer.priority, style = MaterialTheme.typography.labelSmall) },
            )
        }
    }
}
