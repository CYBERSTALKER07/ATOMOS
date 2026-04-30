package com.thelab.warehouse.ui.screens.inventory

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.thelab.warehouse.data.model.InventoryAdjustRequest
import com.thelab.warehouse.data.model.InventoryItem
import com.thelab.warehouse.data.remote.WarehouseApi
import com.thelab.warehouse.ui.theme.LabSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InventoryScreen(
    api: WarehouseApi,
    onBack: () -> Unit,
) {
    var items by remember { mutableStateOf<List<InventoryItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var lowOnly by remember { mutableStateOf(false) }
    var adjustItem by remember { mutableStateOf<InventoryItem?>(null) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load() {
        loading = true; error = null
        scope.launch {
            try {
                val resp = api.getInventory(lowStock = if (lowOnly) true else null)
                if (resp.isSuccessful && resp.body() != null) items = resp.body()!!.items
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) { error = e.message ?: "Network error" }
            finally { loading = false }
        }
    }

    LaunchedEffect(lowOnly) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Inventory") },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } },
                actions = {
                    FilterChip(
                        selected = lowOnly,
                        onClick = { lowOnly = !lowOnly },
                        label = { Text("Low") },
                        leadingIcon = if (lowOnly) {{ Icon(Icons.Default.Warning, null, modifier = Modifier.size(16.dp)) }} else null,
                        modifier = Modifier.padding(end = LabSpacing.sm),
                    )
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(LabSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            items.isEmpty() -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Text("No inventory items", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(LabSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(LabSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(items, key = { it.productId }) { item ->
                    val isLow = item.quantity <= item.reorderThreshold
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(item.productName, style = MaterialTheme.typography.titleSmall)
                                Text(
                                    "Qty: ${item.quantity} · Reorder at: ${item.reorderThreshold}",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            if (isLow) {
                                AssistChip(
                                    onClick = {},
                                    label = { Text("LOW", style = MaterialTheme.typography.labelSmall) },
                                    colors = AssistChipDefaults.assistChipColors(containerColor = MaterialTheme.colorScheme.errorContainer),
                                )
                            }
                            Spacer(Modifier.width(LabSpacing.sm))
                            TextButton(onClick = { adjustItem = item }) { Text("Adjust") }
                        }
                    }
                }
            }
        }
    }

    if (adjustItem != null) {
        AdjustDialog(
            item = adjustItem!!,
            api = api,
            onDismiss = { adjustItem = null },
            onAdjusted = { adjustItem = null; load(); scope.launch { snackbarHostState.showSnackbar("Inventory adjusted") } },
        )
    }
}

@Composable
private fun AdjustDialog(
    item: InventoryItem,
    api: WarehouseApi,
    onDismiss: () -> Unit,
    onAdjusted: () -> Unit,
) {
    var qty by remember { mutableStateOf(item.quantity.toString()) }
    var submitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Adjust ${item.productName}") },
        text = {
            Column {
                OutlinedTextField(value = qty, onValueChange = { qty = it }, label = { Text("New Quantity") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                if (error != null) { Spacer(Modifier.height(LabSpacing.sm)); Text(error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    val q = qty.toIntOrNull() ?: return@Button
                    submitting = true; error = null
                    scope.launch {
                        try {
                            val resp = api.adjustInventory(InventoryAdjustRequest(productId = item.productId, quantity = q))
                            if (resp.isSuccessful) onAdjusted()
                            else error = "Failed (${resp.code()})"
                        } catch (e: Exception) { error = e.message ?: "Error" }
                        finally { submitting = false }
                    }
                },
                enabled = !submitting && qty.toIntOrNull() != null,
            ) { Text("Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
