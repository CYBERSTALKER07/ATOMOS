package com.thelab.factory.ui.screens.transfer

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
import com.thelab.factory.data.model.Transfer
import com.thelab.factory.data.remote.FactoryApi
import com.thelab.factory.ui.theme.LabSpacing
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
                    .padding(horizontal = LabSpacing.lg, vertical = LabSpacing.sm),
                horizontalArrangement = Arrangement.spacedBy(LabSpacing.sm),
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
                        Spacer(Modifier.height(LabSpacing.lg))
                        Button(onClick = { load() }) { Text("Retry") }
                    }
                }
                transfers.isEmpty() -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Text("No transfers", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                else -> LazyColumn(
                    contentPadding = PaddingValues(LabSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(LabSpacing.sm),
                ) {
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
        Row(
            modifier = Modifier.padding(LabSpacing.lg),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = transfer.warehouseName.ifBlank { transfer.warehouseId.take(8) },
                    style = MaterialTheme.typography.titleSmall,
                )
                Text(
                    text = "${transfer.totalItems} items · ${transfer.priority}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            AssistChip(
                onClick = {},
                label = { Text(transfer.state, style = MaterialTheme.typography.labelSmall) },
            )
        }
    }
}
