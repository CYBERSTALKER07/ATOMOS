package com.pegasusx.warehouse.ui.screens.inventory

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.InventoryAdjustRequest
import com.pegasusx.warehouse.data.model.InventoryItem
import com.pegasusx.warehouse.data.model.InventoryPolicyPatchRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.realtime.WAREHOUSE_RECONNECT_RECOVERY_HINT
import com.pegasusx.warehouse.ui.realtime.WarehouseReconnectRecoveryEffect
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.ui.components.InventoryStockList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InventoryScreen(
    api: WarehouseApi,
    realtimeSignals: WarehouseRealtimeSignals,
    onBack: (() -> Unit)? = null,
) {
    var items by remember { mutableStateOf<List<InventoryItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var lowOnly by remember { mutableStateOf(false) }
    var adjustItem by remember { mutableStateOf<InventoryItem?>(null) }
    var policySavingId by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load(silent: Boolean = false) {
        if (!silent) loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getInventory(lowStock = if (lowOnly) true else null)
                if (resp.isSuccessful && resp.body() != null) items = resp.body()!!.items
                else if (!silent) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                if (!silent) error = e.message ?: "Network error"
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(lowOnly) { load() }

    LaunchedEffect(Unit) {
        realtimeSignals.refreshTick.collect { load(silent = true) }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Inventory") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } } },
                actions = {
                    FilterChip(
                        selected = lowOnly,
                        onClick = { lowOnly = !lowOnly },
                        label = { Text("Low") },
                        leadingIcon = if (lowOnly) {{ Icon(Icons.Default.Warning, null, modifier = Modifier.size(16.dp)) }} else null,
                        modifier = Modifier.padding(end = PegasusSpacing.sm),
                    )
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        when {
            loading && items.isEmpty() -> PegasusLoadingState(
                title = "Loading inventory…",
                body = "Fetching latest stock quantities",
                modifier = Modifier.fillMaxSize().padding(innerPadding)
            )
            error != null && items.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Inventory unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(innerPadding)
            )
            else -> InventoryStockList(
                items = items,
                policySavingId = policySavingId,
                modifier = Modifier.fillMaxSize().padding(innerPadding),
                onAdjust = { adjustItem = it },
                onPolicyChange = { item, policy ->
                    policySavingId = item.productId
                    scope.launch {
                        try {
                            val resp = api.patchInventoryPolicy(
                                item.productId,
                                InventoryPolicyPatchRequest(outOfStockPolicy = policy),
                                WarehouseIdempotencyKeys.inventoryPolicy(item.productId, policy),
                            )
                            if (resp.isSuccessful) {
                                load()
                                snackbarHostState.showSnackbar("Policy updated")
                            } else {
                                snackbarHostState.showSnackbar("Policy update failed (${resp.code()})")
                            }
                        } catch (e: Exception) {
                            snackbarHostState.showSnackbar(e.message ?: "Policy update failed")
                        } finally {
                            policySavingId = null
                        }
                    }
                },
            )
        }
    }

    if (adjustItem != null) {
        AdjustDialog(
            item = adjustItem!!,
            api = api,
            realtimeSignals = realtimeSignals,
            onDismiss = { adjustItem = null },
            onAdjusted = { adjustItem = null; load(); scope.launch { snackbarHostState.showSnackbar("Inventory adjusted") } },
        )
    }
}

private val inventoryPolicies = listOf("INHERIT", "REJECT", "ACCEPT_BACKORDER")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InventoryPolicyPicker(
    currentPolicy: String,
    saving: Boolean,
    onSelect: (String) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { if (!saving) expanded = !expanded }) {
        OutlinedTextField(
            value = currentPolicy,
            onValueChange = {},
            readOnly = true,
            label = { Text("Out-of-stock policy") },
            trailingIcon = {
                if (saving) {
                    CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded)
                }
            },
            modifier = Modifier.menuAnchor(MenuAnchorType.PrimaryNotEditable).fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            inventoryPolicies.forEach { policy ->
                DropdownMenuItem(
                    text = { Text(policy) },
                    onClick = {
                        expanded = false
                        if (policy != currentPolicy) onSelect(policy)
                    },
                )
            }
        }
    }
}

@Composable
private fun AdjustDialog(
    item: InventoryItem,
    api: WarehouseApi,
    realtimeSignals: WarehouseRealtimeSignals,
    onDismiss: () -> Unit,
    onAdjusted: () -> Unit,
) {
    var qty by remember { mutableStateOf(item.quantity.toString()) }
    var reason by remember { mutableStateOf("") }
    var showConfirm by remember { mutableStateOf(false) }
    var submitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val skuLabel = item.sku.ifBlank { item.productId }

    WarehouseReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { submitting },
    ) { hadInFlight ->
        if (hadInFlight) {
            submitting = false
            error = WAREHOUSE_RECONNECT_RECOVERY_HINT
        }
    }

    fun submitAdjust() {
        val q = qty.toIntOrNull() ?: return
        submitting = true
        error = null
        scope.launch {
            try {
                val trimmedReason = reason.trim().ifBlank { null }
                val resp = api.adjustInventory(
                    InventoryAdjustRequest(productId = item.productId, quantity = q),
                    WarehouseIdempotencyKeys.adjustInventory(item.productId, q),
                )
                if (resp.isSuccessful) onAdjusted()
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message ?: "Error"
            } finally {
                submitting = false
            }
        }
    }

    if (showConfirm) {
        val newQty = qty.toIntOrNull() ?: item.quantity
        AlertDialog(
            onDismissRequest = { if (!submitting) showConfirm = false },
            title = { Text("Confirm inventory change") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    Text("Change $skuLabel from ${item.quantity} to $newQty? This affects retailer availability immediately.")
                    OutlinedTextField(
                        value = reason,
                        onValueChange = { reason = it },
                        label = { Text("Reason (optional)") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    if (error != null) {
                        Text(error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                    }
                }
            },
            confirmButton = {
                Button(onClick = { submitAdjust() }, enabled = !submitting) {
                    Text(if (submitting) "Saving…" else "Confirm")
                }
            },
            dismissButton = {
                TextButton(onClick = { showConfirm = false }, enabled = !submitting) { Text("Back") }
            },
        )
        return
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Adjust ${item.productName}") },
        text = {
            Column {
                OutlinedTextField(value = qty, onValueChange = { qty = it }, label = { Text("New Quantity") }, singleLine = true, modifier = Modifier.fillMaxWidth())
            }
        },
        confirmButton = {
            Button(
                onClick = { showConfirm = true },
                enabled = qty.toIntOrNull() != null && qty.toIntOrNull() != item.quantity,
            ) { Text("Review") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
