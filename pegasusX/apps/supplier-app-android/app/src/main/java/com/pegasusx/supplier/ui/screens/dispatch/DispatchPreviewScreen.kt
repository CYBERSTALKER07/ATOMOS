@file:OptIn(ExperimentalMaterial3Api::class)

package com.pegasusx.supplier.ui.screens.dispatch

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.SupplierDispatchManualRoute
import com.pegasusx.supplier.data.model.SupplierDispatchPreview
import com.pegasusx.supplier.data.model.SupplierTopologyWarehouse
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.data.remote.TokenHolder
import com.pegasusx.supplier.util.SUPPLIER_RECONNECT_RECOVERY_HINT
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import com.pegasusx.supplier.ui.realtime.SupplierReconnectRecoveryEffect
import com.pegasusx.supplier.ui.components.DispatchPreviewMapLibre
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonPrimitive
import com.pegasusx.supplier.R

private const val TETRIS_BUFFER = 0.95

private enum class DispatchMode { Manual, Auto }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DispatchPreviewScreen(
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
    onBack: () -> Unit = {},
    embedded: Boolean = false,
    modifier: Modifier = Modifier,
) {
    var preview by remember { mutableStateOf<SupplierDispatchPreview?>(null) }
    var warehouses by remember { mutableStateOf<List<SupplierTopologyWarehouse>>(emptyList()) }
    var selectedWarehouseId by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var executing by remember { mutableStateOf(false) }
    var executeMessage by remember { mutableStateOf<String?>(null) }
    var showAutoConfirm by remember { mutableStateOf(false) }
    var showCapacityOverride by remember { mutableStateOf(false) }
    var dispatchMode by remember { mutableStateOf(DispatchMode.Manual) }
    var selectedDriverId by remember { mutableStateOf("") }
    var selectedOrderIds by remember { mutableStateOf(setOf<String>()) }
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

    LaunchedEffect(preview?.availableDrivers) {
        val drivers = preview?.availableDrivers.orEmpty()
        if (selectedDriverId.isBlank() && drivers.isNotEmpty()) {
            selectedDriverId = drivers.first().driverId
        }
        val valid = drivers.map { it.driverId }.toSet()
        if (selectedDriverId.isNotBlank() && selectedDriverId !in valid) {
            selectedDriverId = drivers.firstOrNull()?.driverId.orEmpty()
        }
    }

    LaunchedEffect(preview?.undispatchedOrders) {
        val valid = preview?.undispatchedOrders?.map { it.orderId }?.toSet().orEmpty()
        selectedOrderIds = selectedOrderIds.intersect(valid)
    }

    SupplierReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { executing },
    ) { hadInFlight ->
        if (hadInFlight) {
            executing = false
            executeMessage = SUPPLIER_RECONNECT_RECOVERY_HINT
        }
    }

    fun executeAuto() {
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
                val resp = ops.executeDispatch(selectedWarehouseId, idempotencyKey, mode = "AUTO")
                executeMessage = if (resp.isSuccessful) {
                    "Auto dispatch executed"
                } else {
                    "Execute failed (${resp.code()})"
                }
                if (resp.isSuccessful) load()
            } catch (e: Exception) {
                executeMessage = e.message
            } finally {
                executing = false
                showAutoConfirm = false
            }
        }
    }

    fun executeManual(forceCapacity: Boolean) {
        val orderIds = selectedOrderIds.toList().sorted()
        if (selectedDriverId.isBlank() || orderIds.isEmpty()) return
        scope.launch {
            executing = true
            executeMessage = null
            try {
                val routeFingerprint = """{"mode":"MANUAL","force_capacity":$forceCapacity,"routes":[{"driver_id":"$selectedDriverId","order_ids":$orderIds}]}"""
                val supplierId = TokenHolder.supplierId.orEmpty().ifBlank { "supplier" }
                val warehouseId = selectedWarehouseId.orEmpty().ifBlank { "default" }
                val idempotencyKey = SupplierIdempotencyKeys.dispatch(
                    supplierId,
                    warehouseId,
                    "MANUAL",
                    routeFingerprint,
                )
                val resp = ops.executeDispatch(
                    warehouseId = selectedWarehouseId,
                    idempotencyKey = idempotencyKey,
                    mode = "MANUAL",
                    forceCapacity = forceCapacity,
                    routes = listOf(SupplierDispatchManualRoute(selectedDriverId, orderIds)),
                )
                if (resp.isSuccessful) {
                    val body = resp.body()
                    val status = (body as? JsonObject)?.get("status")?.jsonPrimitive?.content
                    if (status == "capacity_exceeded") {
                        executeMessage = "Truck capacity exceeded — confirm override or remove orders."
                        showCapacityOverride = true
                    } else {
                        executeMessage = "Manual dispatch committed"
                        selectedOrderIds = emptySet()
                        showCapacityOverride = false
                        load()
                    }
                } else {
                    executeMessage = "Execute failed (${resp.code()})"
                }
            } catch (e: Exception) {
                executeMessage = e.message
            } finally {
                executing = false
            }
        }
    }

    val body: @Composable (Modifier) -> Unit = { contentModifier ->
        DispatchPreviewBody(
            modifier = contentModifier,
            loading = loading,
            error = error,
            warehouses = warehouses,
            selectedWarehouseId = selectedWarehouseId,
            onWarehouseSelected = { selectedWarehouseId = it },
            preview = preview,
            executeMessage = executeMessage,
            executing = executing,
            dispatchMode = dispatchMode,
            onDispatchModeChange = { dispatchMode = it },
            selectedDriverId = selectedDriverId,
            onDriverSelected = { selectedDriverId = it },
            selectedOrderIds = selectedOrderIds,
            onToggleOrder = { orderId ->
                selectedOrderIds = if (orderId in selectedOrderIds) {
                    selectedOrderIds - orderId
                } else {
                    selectedOrderIds + orderId
                }
            },
            onSelectAllOrders = {
                selectedOrderIds = preview?.undispatchedOrders?.map { it.orderId }?.toSet().orEmpty()
            },
            onRefresh = { load() },
            onAutoExecute = { showAutoConfirm = true },
            onManualExecute = { executeManual(forceCapacity = false) },
        )
    }

    if (embedded) {
        body(modifier.fillMaxSize())
    } else {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Dispatch") },
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                        }
                    },
                )
            },
        ) { padding ->
            body(Modifier.padding(padding))
        }
    }

    if (showAutoConfirm) {
        AlertDialog(
            onDismissRequest = { if (!executing) showAutoConfirm = false },
            title = { Text("Execute auto-dispatch?") },
            text = { Text("This assigns pending orders to available drivers via the optimizer.") },
            confirmButton = {
                TextButton(onClick = { executeAuto() }, enabled = !executing) { Text("Confirm") }
            },
            dismissButton = {
                TextButton(onClick = { showAutoConfirm = false }, enabled = !executing) { Text("Cancel") }
            },
        )
    }

    if (showCapacityOverride) {
        AlertDialog(
            onDismissRequest = { if (!executing) showCapacityOverride = false },
            title = { Text("Capacity exceeded") },
            text = { Text("Selected orders exceed truck capacity. Continue anyway?") },
            confirmButton = {
                TextButton(
                    onClick = { executeManual(forceCapacity = true) },
                    enabled = !executing,
                ) { Text("Continue") }
            },
            dismissButton = {
                TextButton(onClick = { showCapacityOverride = false }, enabled = !executing) {
                    Text("Adjust")
                }
            },
        )
    }
}

@Composable
private fun DispatchPreviewBody(
    modifier: Modifier,
    loading: Boolean,
    error: String?,
    warehouses: List<SupplierTopologyWarehouse>,
    selectedWarehouseId: String?,
    onWarehouseSelected: (String?) -> Unit,
    preview: SupplierDispatchPreview?,
    executeMessage: String?,
    executing: Boolean,
    dispatchMode: DispatchMode,
    onDispatchModeChange: (DispatchMode) -> Unit,
    selectedDriverId: String,
    onDriverSelected: (String) -> Unit,
    selectedOrderIds: Set<String>,
    onToggleOrder: (String) -> Unit,
    onSelectAllOrders: () -> Unit,
    onRefresh: () -> Unit,
    onAutoExecute: () -> Unit,
    onManualExecute: () -> Unit,
) {
    val selectedDriver = preview?.availableDrivers?.orEmpty()?.find { it.driverId == selectedDriverId }
    val selectedVolume = preview?.undispatchedOrders.orEmpty()
        .filter { it.orderId in selectedOrderIds }
        .sumOf { it.volumeVu }
    val truckMax = selectedDriver?.maxVolumeVu ?: 0.0
    val truckEffective = truckMax * TETRIS_BUFFER
    val capacityExceeded = truckEffective > 0 && selectedVolume > truckEffective

    when {
        loading && preview == null -> PegasusLoadingState("Loading dispatch preview…", "Dispatch snapshot", modifier)
        error != null && preview == null -> PegasusStatePane(
            kind = PegasusStateKind.Error,
            headline = "Preview unavailable",
            body = error,
            modifier = modifier,
            actionLabel = "Retry",
            onAction = onRefresh,
        )
        else -> Column(
            modifier = modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                FilterChip(
                    selected = dispatchMode == DispatchMode.Manual,
                    onClick = { onDispatchModeChange(DispatchMode.Manual) },
                    label = { Text("Manual truck") },
                )
                FilterChip(
                    selected = dispatchMode == DispatchMode.Auto,
                    onClick = { onDispatchModeChange(DispatchMode.Auto) },
                    label = { Text("Smart assign") },
                )
            }
            if (warehouses.isNotEmpty()) {
                Text("Warehouse scope", style = MaterialTheme.typography.titleSmall)
                LazyRow(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    items(warehouses, key = { it.warehouseId }) { wh ->
                        FilterChip(
                            selected = selectedWarehouseId == wh.warehouseId,
                            onClick = {
                                onWarehouseSelected(
                                    if (selectedWarehouseId == wh.warehouseId) null else wh.warehouseId,
                                )
                            },
                            label = { Text(wh.name.ifBlank { wh.warehouseId }) },
                        )
                    }
                }
            }
            preview?.takeIf { it.planFingerprintMismatch }?.let {
                Text(
                    "Dispatch plan drift — refresh warehouse dispatch before commit.",
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
            preview?.let { p ->
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    KpiCard("Pending", "${p.pendingCount}", Modifier.weight(1f))
                    KpiCard("Drivers", "${p.availableDriverCount}", Modifier.weight(1f))
                    KpiCard("Undispatched", "${p.undispatchedOrders.size}", Modifier.weight(1f))
                }
                if (dispatchMode == DispatchMode.Manual && p.undispatchedOrders.isNotEmpty()) {
                    Text("Manual assignment", style = MaterialTheme.typography.titleSmall)
                    if (truckMax > 0) {
                        Text(stringResource(R.string.mobile_supplier_ui_selected_format_format_2_vu, "%.1f".format(selectedVolume), "%.1f".format(truckEffective)),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    if (capacityExceeded) {
                        Text(
                            "Insufficient truck space for selected orders.",
                            color = MaterialTheme.colorScheme.error,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                    if (p.availableDrivers.isNotEmpty()) {
                        var expanded by remember { mutableStateOf(false) }
                        ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }) {
                            OutlinedTextField(
                                readOnly = true,
                                value = selectedDriver?.name?.ifBlank { selectedDriverId }.orEmpty(),
                                onValueChange = {},
                                label = { Text("Driver") },
                                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded) },
                                modifier = Modifier.menuAnchor().fillMaxWidth(),
                            )
                            ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                                p.availableDrivers.forEach { driver ->
                                    DropdownMenuItem(
                                        text = {
                                            val vu = driver.maxVolumeVu?.let { " · ${it.toInt()} VU" }.orEmpty()
                                            Text(stringResource(R.string.mobile_supplier_ui_driveridvu, driver.name.ifBlank { driver.driverId }, vu))
                                        },
                                        onClick = {
                                            onDriverSelected(driver.driverId)
                                            expanded = false
                                        },
                                    )
                                }
                            }
                        }
                    }
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Text("Orders", style = MaterialTheme.typography.labelLarge)
                        TextButton(onClick = onSelectAllOrders) { Text("Select all") }
                    }
                    p.undispatchedOrders.forEach { order ->
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                        ) {
                            Row(verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
                                Checkbox(
                                    checked = order.orderId in selectedOrderIds,
                                    onCheckedChange = { onToggleOrder(order.orderId) },
                                )
                                Text(order.orderId.take(12), style = MaterialTheme.typography.bodySmall)
                            }
                            Text(stringResource(R.string.mobile_supplier_ui_format_vu, "%.1f".format(order.volumeVu)),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
                if (dispatchMode == DispatchMode.Auto && p.proposedRoutes.isNotEmpty()) {
                    Text("Route map", style = MaterialTheme.typography.titleSmall)
                    p.optimizerSource?.takeIf { it.isNotBlank() }?.let { source ->
                        Text(
                            stringResource(R.string.mobile_supplier_ui_source_source_2, source),
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
                }
            }
            executeMessage?.let { msg ->
                Text(msg, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
            }
            if (dispatchMode == DispatchMode.Auto) {
                Button(
                    onClick = onAutoExecute,
                    enabled = !loading && !executing && preview != null,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(if (executing) "Executing…" else "Execute auto-dispatch") }
            } else {
                Button(
                    onClick = onManualExecute,
                    enabled = !loading && !executing && selectedDriverId.isNotBlank() && selectedOrderIds.isNotEmpty(),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(
                        if (executing) {
                            "Dispatching…"
                        } else {
                            "Manual dispatch (${selectedOrderIds.size})"
                        },
                    )
                }
            }
            OutlinedButton(onClick = onRefresh, enabled = !loading) { Text("Refresh preview") }
        }
    }
}

@Composable
private fun KpiCard(label: String, value: String, modifier: Modifier = Modifier) {
    ElevatedCard(modifier) {
        Column(Modifier.padding(PegasusSpacing.md)) {
            Text(label, style = MaterialTheme.typography.labelMedium)
            Text(value, style = MaterialTheme.typography.headlineSmall)
        }
    }
}
