package com.pegasusx.supplier.ui.screens.dispatch

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.SupplierDispatchPreview
import com.pegasusx.supplier.data.model.SupplierTopologyWarehouse
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.data.remote.TokenHolder
import com.pegasusx.supplier.util.SUPPLIER_RECONNECT_RECOVERY_HINT
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import com.pegasusx.supplier.ui.realtime.SupplierReconnectRecoveryEffect
import com.pegasusx.supplier.ui.components.DispatchPreviewMapLibre
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DispatchPreviewScreen(
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
    onBack: () -> Unit,
) {
    var preview by remember { mutableStateOf<SupplierDispatchPreview?>(null) }
    var warehouses by remember { mutableStateOf<List<SupplierTopologyWarehouse>>(emptyList()) }
    var selectedWarehouseId by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var executing by remember { mutableStateOf(false) }
    var executeMessage by remember { mutableStateOf<String?>(null) }
    var showExecuteConfirm by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val topologyResp = ops.getTopology()
                if (topologyResp.isSuccessful) {
                    warehouses = topologyResp.body()?.warehouses.orEmpty()
                }
                val resp = ops.getDispatchPreview(selectedWarehouseId)
                preview = if (resp.isSuccessful) resp.body() else null
                if (!resp.isSuccessful) error = "Preview failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(selectedWarehouseId) { load() }

    SupplierReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { executing },
    ) { hadInFlight ->
        if (hadInFlight) {
            executing = false
            executeMessage = SUPPLIER_RECONNECT_RECOVERY_HINT
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Dispatch preview") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading && preview == null -> SupplierLoadingState("Loading dispatch preview…", "Auto-dispatch snapshot")
            error != null && preview == null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Preview unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            else -> Column(
                modifier = Modifier
                    .padding(padding)
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                if (warehouses.isNotEmpty()) {
                    Text("Warehouse scope", style = MaterialTheme.typography.titleSmall)
                    warehouses.forEach { wh ->
                        FilterChip(
                            selected = selectedWarehouseId == wh.warehouseId,
                            onClick = {
                                selectedWarehouseId = if (selectedWarehouseId == wh.warehouseId) null else wh.warehouseId
                            },
                            label = { Text(wh.name.ifBlank { wh.warehouseId }) },
                        )
                    }
                }
                preview?.let { p ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text("Pending orders", style = MaterialTheme.typography.titleMedium)
                            Text("${p.pendingCount}", style = MaterialTheme.typography.headlineMedium)
                            Text("Available drivers: ${p.availableDriverCount}", style = MaterialTheme.typography.bodyMedium)
                            Text(
                                "Undispatched bucket: ${p.undispatchedOrders.size}",
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                    if (p.proposedRoutes.isNotEmpty()) {
                        Text("Smart suggest route map", style = MaterialTheme.typography.titleSmall)
                        p.optimizerSource?.takeIf { it.isNotBlank() }?.let { source ->
                            Text(
                                "Source: $source",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        DispatchPreviewMapLibre(
                            routes = p.proposedRoutes,
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(320.dp),
                        )
                        p.proposedRoutes.forEachIndexed { index, route ->
                            val label = route.driverName?.takeIf { it.isNotBlank() }
                                ?: route.driverId?.takeIf { it.isNotBlank() }
                                ?: "Route ${index + 1}"
                            val stops = route.stopCount ?: route.orderIds.size
                            Text(
                                "$label · $stops stops",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
                executeMessage?.let { msg ->
                    Text(msg, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
                }
                Button(
                    onClick = { showExecuteConfirm = true },
                    enabled = !loading && !executing && preview != null,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(if (executing) "Executing…" else "Execute auto-dispatch") }
                OutlinedButton(onClick = { load() }, enabled = !loading) { Text("Refresh preview") }
            }
        }
    }

    if (showExecuteConfirm) {
        AlertDialog(
            onDismissRequest = { if (!executing) showExecuteConfirm = false },
            title = { Text("Execute dispatch?") },
            text = { Text("This assigns pending orders to available drivers. Confirm to proceed.") },
            confirmButton = {
                TextButton(
                    onClick = {
                        scope.launch {
                            executing = true
                            executeMessage = null
                            try {
                                val p = preview
                                val routeFingerprint = p?.let {
                                    """{"pending":${it.pendingCount},"drivers":${it.availableDriverCount},"undispatched":${it.undispatchedOrders.size}}"""
                                } ?: "[]"
                                val supplierId = TokenHolder.supplierId.orEmpty().ifBlank { "supplier" }
                                val warehouseId = selectedWarehouseId.orEmpty().ifBlank { "default" }
                                val idempotencyKey = SupplierIdempotencyKeys.dispatch(
                                    supplierId,
                                    warehouseId,
                                    "AUTO",
                                    routeFingerprint,
                                )
                                val resp = ops.executeDispatch(selectedWarehouseId, idempotencyKey)
                                executeMessage = if (resp.isSuccessful) {
                                    "Dispatch executed"
                                } else {
                                    "Execute failed (${resp.code()})"
                                }
                                if (resp.isSuccessful) load()
                            } catch (e: Exception) {
                                executeMessage = e.message
                            } finally {
                                executing = false
                                showExecuteConfirm = false
                            }
                        }
                    },
                    enabled = !executing,
                ) { Text("Confirm") }
            },
            dismissButton = {
                TextButton(onClick = { showExecuteConfirm = false }, enabled = !executing) {
                    Text("Cancel")
                }
            },
        )
    }
}
