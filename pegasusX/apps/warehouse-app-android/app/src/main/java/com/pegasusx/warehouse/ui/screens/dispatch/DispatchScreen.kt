package com.pegasusx.warehouse.ui.screens.dispatch

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.DispatchPreview
import com.pegasusx.warehouse.data.model.PulseEvent
import com.pegasusx.warehouse.data.model.CreateWarehouseDispatchLockRequest
import com.pegasusx.warehouse.data.model.UpdateVehicleRequest
import com.pegasusx.warehouse.data.model.Vehicle
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import com.pegasusx.warehouse.util.filterHandoffPulseEvents
import com.pegasusx.warehouse.data.model.CreateWarehouseSupplyRequestRequest
import com.pegasusx.warehouse.data.model.WarehouseDispatchLock
import com.pegasusx.warehouse.data.model.WarehouseSupplyRequest
import com.pegasusx.warehouse.data.model.WarehouseSupplyRequestTransitionRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeClient
import com.pegasusx.warehouse.data.remote.reconcileWarehouseSession
import com.pegasusx.warehouse.ui.screens.supply.CreateSupplyRequestDialog
import com.pegasusx.warehouse.ui.screens.supply.SupplyRequestFormResult
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.components.DispatchPreviewMapLibre
import com.pegasusx.warehouse.ui.components.DispatchDriverList
import com.pegasusx.warehouse.ui.components.FleetLiveMapSection
import com.pegasusx.warehouse.ui.components.HandoffTimelineSection
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.warehouse.ui.components.OrderDetailOpenMode
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasusx.warehouse.ui.components.WarehouseOpsListCard
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeStatus
import com.pegasusx.warehouse.ui.realtime.WAREHOUSE_RECONNECT_RECOVERY_HINT
import com.pegasusx.warehouse.ui.realtime.WarehouseReconnectRecoveryEffect
import com.pegasusx.warehouse.ui.screens.vehicles.FleetTruckDispatchCard
import com.pegasusx.warehouse.ui.screens.vehicles.vehicleUnavailableReasonLabel
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.google.gson.JsonArray
import com.google.gson.JsonObject
import kotlinx.coroutines.launch
import com.pegasusx.warehouse.ui.components.OrderOpsCard
import java.text.NumberFormat
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import java.util.Locale

private const val DISPATCH_TETRIS_BUFFER = 0.95

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DispatchScreen(
    api: WarehouseApi,
    opsRepository: WarehouseOperationsRepository,
    realtimeSignals: WarehouseRealtimeSignals,
    onVehicleClick: (String) -> Unit = {},
    onOrderClick: (String) -> Unit = {},
    onBack: (() -> Unit)? = null,
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    var preview by remember { mutableStateOf<DispatchPreview?>(null) }
    var fleetVehicles by remember { mutableStateOf<List<Vehicle>>(emptyList()) }
    var vehicleReasons by remember { mutableStateOf<Map<String, String>>(emptyMap()) }
    var vehicleNotes by remember { mutableStateOf<Map<String, String>>(emptyMap()) }
    var mutatingFleetVehicleId by remember { mutableStateOf<String?>(null) }
    var fleetAlert by remember { mutableStateOf<String?>(null) }
    var supplyRequests by remember { mutableStateOf<List<WarehouseSupplyRequest>>(emptyList()) }
    var dispatchLocks by remember { mutableStateOf<List<WarehouseDispatchLock>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var tab by remember { mutableIntStateOf(0) }
    var realtimeStatus by remember { mutableStateOf(WarehouseRealtimeStatus.IDLE) }
    var showCreateSupplyRequest by remember { mutableStateOf(false) }
    var showAcquireDispatchLock by remember { mutableStateOf(false) }
    var requestPendingCancellation by remember { mutableStateOf<WarehouseSupplyRequest?>(null) }
    var lockPendingRelease by remember { mutableStateOf<WarehouseDispatchLock?>(null) }
    var actionMessage by remember { mutableStateOf<DispatchActionMessage?>(null) }
    var executing by remember { mutableStateOf(false) }
    var selectedDriverId by remember { mutableStateOf("") }
    var selectedOrderIds by remember { mutableStateOf(setOf<String>()) }
    var driverMenuExpanded by remember { mutableStateOf(false) }
    var capacityWarnings by remember { mutableStateOf<List<com.pegasusx.warehouse.data.model.DispatchCapacityWarning>>(emptyList()) }
    var showCapacityDialog by remember { mutableStateOf(false) }
    var capacityDialogAutoMode by remember { mutableStateOf(false) }
    var showSmartConfirm by remember { mutableStateOf(false) }
    var dispatchMode by remember { mutableStateOf("smart") }
    var proposeTarget by remember { mutableStateOf<String?>(null) }
    var rejectTarget by remember { mutableStateOf<String?>(null) }
    var opsReasonInput by remember { mutableStateOf("") }
    var opsActingId by remember { mutableStateOf<String?>(null) }
    var handoffEvents by remember { mutableStateOf<List<PulseEvent>>(emptyList()) }
    var handoffLoading by remember { mutableStateOf(true) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }
    val realtimeClient = remember(context) { WarehouseRealtimeClient(context) }

    val hasActiveManualDispatchLock = dispatchLocks.any { lock -> lock.lockType == "MANUAL_DISPATCH" }
    fun codeMessage(code: Int, fallback: String = "Request failed"): String {
        return if (code == 403) {
            "Permission denied for this warehouse scope."
        } else {
            "$fallback ($code)"
        }
    }

    fun loadVehicles() {
        scope.launch {
            runCatching { api.getVehicles() }
                .onSuccess { response ->
                    if (response.isSuccessful && response.body() != null) {
                        fleetVehicles = response.body()!!.vehicles
                    }
                }
        }
    }

    fun load(silent: Boolean = false) {
        if (!silent) loading = true
        error = null
        scope.launch {
            try {
                val previewResp = api.getDispatchPreview()
                val supplyResp = api.getSupplyRequests()
                val lockResp = api.getDispatchLocks()
                val vehiclesResp = api.getVehicles()
                if (previewResp.isSuccessful && previewResp.body() != null) preview = previewResp.body()!!
                else if (!silent) error = codeMessage(previewResp.code(), "Failed to load dispatch preview")
                if (supplyResp.isSuccessful && supplyResp.body() != null) {
                    supplyRequests = supplyResp.body()!!.resolved()
                }
                else if (!silent && error == null) error = codeMessage(supplyResp.code(), "Failed to load supply requests")
                if (lockResp.isSuccessful && lockResp.body() != null) {
                    dispatchLocks = lockResp.body()!!.locks
                }
                else if (!silent && error == null) error = codeMessage(lockResp.code(), "Failed to load dispatch locks")
                if (vehiclesResp.isSuccessful && vehiclesResp.body() != null) {
                    fleetVehicles = vehiclesResp.body()!!.vehicles
                }
                handoffLoading = true
                val pulseResp = api.getPulse()
                if (pulseResp.isSuccessful && pulseResp.body() != null) {
                    handoffEvents = filterHandoffPulseEvents(pulseResp.body()!!.events)
                } else {
                    handoffEvents = emptyList()
                }
                handoffLoading = false
            } catch (e: Exception) {
                if (!silent) error = e.message ?: "Network error"
            } finally {
                if (!silent) loading = false
            }
        }
    }

    fun updateFleetVehicle(vehicle: Vehicle, isActive: Boolean, reason: String? = null, note: String? = null) {
        mutatingFleetVehicleId = vehicle.vehicleId
        fleetAlert = null
        scope.launch {
            try {
                val resp = api.updateVehicle(
                    vehicle.vehicleId,
                    UpdateVehicleRequest(
                        isActive = isActive,
                        unavailableReason = if (isActive) null else reason,
                        unavailableNote = if (isActive) null else note,
                    ),
                    WarehouseIdempotencyKeys.updateVehicle(vehicle.vehicleId, isActive, reason),
                )
                if (resp.isSuccessful) {
                    fleetAlert = if (isActive) {
                        "${vehicle.label.ifBlank { vehicle.licensePlate }} restored to dispatch"
                    } else {
                        "${vehicle.label.ifBlank { vehicle.licensePlate }} excluded from dispatch"
                    }
                    loadVehicles()
                    load()
                } else {
                    error = codeMessage(resp.code(), "Failed to update truck")
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                mutatingFleetVehicleId = null
            }
        }
    }

    val reloadSupplyRequests: () -> Unit = {
        scope.launch {
            runCatching { api.getSupplyRequests() }
                .onSuccess { response ->
                    if (response.isSuccessful && response.body() != null) {
                        supplyRequests = response.body()!!.resolved()
                    }
                }
                .onFailure { throwable ->
                    error = throwable.message ?: "Network error"
                }
        }
    }

    val reloadDispatchLocks: () -> Unit = {
        scope.launch {
            runCatching { api.getDispatchLocks() }
                .onSuccess { response ->
                    if (response.isSuccessful && response.body() != null) {
                        dispatchLocks = response.body()!!.locks
                    }
                }
                .onFailure { throwable ->
                    error = throwable.message ?: "Network error"
                }
        }
    }

    fun createSupplyRequest(form: SupplyRequestFormResult) {
        scope.launch {
            runCatching {
                val mode = if (form.useDemandForecast) "FORECAST" else "MANUAL"
                val key = WarehouseIdempotencyKeys.createSupplyRequest(form.factoryId, mode, form.notes)
                api.createSupplyRequest(
                    key,
                    CreateWarehouseSupplyRequestRequest(
                        factoryId = form.factoryId,
                        priority = form.priority,
                        notes = form.notes,
                        items = form.items,
                        useDemandForecast = form.useDemandForecast,
                        requestedDeliveryDate = form.requestedDeliveryDate,
                    ),
                )
            }.onSuccess { response ->
                if (response.isSuccessful && response.body() != null) {
                    val body = response.body()!!
                    actionMessage = DispatchActionMessage(
                        title = "Supply Request Submitted",
                        message = "Request ${body.requestId.take(8)} is now ${body.state}.",
                    )
                    showCreateSupplyRequest = false
                    reloadSupplyRequests()
                } else {
                    actionMessage = DispatchActionMessage("Supply Request Failed", codeMessage(response.code()))
                }
            }.onFailure { throwable ->
                actionMessage = DispatchActionMessage("Supply Request Failed", throwable.message ?: "Network error")
            }
        }
    }

    fun cancelSupplyRequest(request: WarehouseSupplyRequest) {
        scope.launch {
            runCatching {
                val key = WarehouseIdempotencyKeys.supplyRequestTransition(request.requestId, "CANCEL")
                api.transitionSupplyRequest(
                    request.requestId,
                    key,
                    WarehouseSupplyRequestTransitionRequest(action = "CANCEL"),
                )
            }.onSuccess { response ->
                if (response.isSuccessful && response.body() != null) {
                    val body = response.body()!!
                    actionMessage = DispatchActionMessage(
                        title = "Supply Request Cancelled",
                        message = "Request ${body.requestId.take(8)} moved to ${body.state}.",
                    )
                    requestPendingCancellation = null
                    reloadSupplyRequests()
                } else {
                    actionMessage = DispatchActionMessage("Cancellation Failed", codeMessage(response.code()))
                }
            }.onFailure { throwable ->
                actionMessage = DispatchActionMessage("Cancellation Failed", throwable.message ?: "Network error")
            }
        }
    }

    fun acquireDispatchLock() {
        scope.launch {
            runCatching {
                api.createDispatchLock(
                    CreateWarehouseDispatchLockRequest(lockType = "MANUAL_DISPATCH"),
                    WarehouseIdempotencyKeys.dispatchLockAcquire(),
                )
            }
                .onSuccess { response ->
                    if (response.isSuccessful && response.body() != null) {
                        val body = response.body()!!
                        actionMessage = DispatchActionMessage(
                            title = "Dispatch Locked",
                            message = "${body.lockType} is now active for this warehouse scope.",
                        )
                        showAcquireDispatchLock = false
                        reloadDispatchLocks()
                        load()
                    } else {
                        actionMessage = DispatchActionMessage("Lock Failed", codeMessage(response.code()))
                    }
                }
                .onFailure { throwable ->
                    actionMessage = DispatchActionMessage("Lock Failed", throwable.message ?: "Network error")
                }
        }
    }

    fun runManualDispatch(forceCapacity: Boolean = false) {
        if (executing || selectedDriverId.isBlank() || selectedOrderIds.isEmpty()) return
        executing = true
        scope.launch {
            runCatching {
                val orderIds = JsonArray()
                selectedOrderIds.forEach { orderIds.add(it) }
                val route = JsonObject().apply {
                    addProperty("driver_id", selectedDriverId)
                    add("order_ids", orderIds)
                }
                val routes = JsonArray().apply { add(route) }
                val sortedOrderIds = selectedOrderIds.sorted()
                val orderIdsBody = JsonArray()
                sortedOrderIds.forEach { orderIdsBody.add(it) }
                val body = JsonObject().apply {
                    addProperty("mode", "MANUAL")
                    addProperty("force_capacity", forceCapacity)
                    add("order_ids", orderIdsBody)
                    add("routes", routes)
                }
                val routeFingerprint = buildString {
                    append("""{"mode":"MANUAL","force_capacity":$forceCapacity,"routes":[{"driver_id":"$selectedDriverId","order_ids":""")
                    append(sortedOrderIds.joinToString(prefix = "[", postfix = "]") { "\"$it\"" })
                    append("}]}")
                }
                val idempotencyKey = WarehouseIdempotencyKeys.dispatch(selectedDriverId, routeFingerprint)
                api.executeDispatch(idempotencyKey, body)
            }.onSuccess { response ->
                if (response.isSuccessful && response.body() != null) {
                    val result = response.body()!!
                    when (result.status) {
                        "plan_stale" -> {
                            actionMessage = DispatchActionMessage(
                                title = "Plan stale",
                                message = "Refresh preview and try smart dispatch again.",
                            )
                            load()
                        }
                        "capacity_exceeded" -> {
                            capacityWarnings = result.capacityWarnings
                            capacityDialogAutoMode = false
                            showCapacityDialog = true
                        }
                        "dispatched" -> {
                            actionMessage = DispatchActionMessage(
                                title = "Dispatch Committed",
                                message = "Assigned ${result.ordersAssigned} order(s). Payloader loading gate is active.",
                            )
                            selectedOrderIds = emptySet()
                            load()
                        }
                        else -> {
                            val warning = result.warnings.firstOrNull() ?: "Dispatch did not commit."
                            actionMessage = DispatchActionMessage("Dispatch Incomplete", warning)
                        }
                    }
                } else {
                    actionMessage = DispatchActionMessage("Dispatch Failed", codeMessage(response.code()))
                }
            }.onFailure { throwable ->
                actionMessage = DispatchActionMessage("Dispatch Failed", throwable.message ?: "Network error")
            }
            executing = false
        }
    }

    fun runSmartDispatch(forceCapacity: Boolean = false, acceptPartial: Boolean = false) {
        val orders = preview?.undispatchedOrders.orEmpty()
        val orderIds = if (selectedOrderIds.isNotEmpty()) selectedOrderIds.sorted() else orders.map { it.orderId }
        if (executing || orderIds.isEmpty()) return
        executing = true
        showSmartConfirm = false
        scope.launch {
            runCatching {
                val orderIdsJson = JsonArray()
                orderIds.forEach { orderIdsJson.add(it) }
                val body = JsonObject().apply {
                    addProperty("mode", "AUTO")
                    add("order_ids", orderIdsJson)
                    addProperty("force_capacity", forceCapacity)
                    addProperty("accept_partial", acceptPartial)
                    preview?.planFingerprint?.let { addProperty("plan_fingerprint", it) }
                }
                val routeFingerprint = """{"mode":"AUTO","order_ids":${orderIdsJson},"force_capacity":$forceCapacity,"accept_partial":$acceptPartial}"""
                val idempotencyKey = WarehouseIdempotencyKeys.dispatch("smart-dispatch", routeFingerprint)
                api.executeDispatch(idempotencyKey, body)
            }.onSuccess { response ->
                if (response.isSuccessful && response.body() != null) {
                    val result = response.body()!!
                    when (result.status) {
                        "plan_stale" -> {
                            actionMessage = DispatchActionMessage(
                                title = "Plan stale",
                                message = "Refresh preview and try smart dispatch again.",
                            )
                            load()
                        }
                        "capacity_exceeded" -> {
                            capacityWarnings = result.capacityWarnings
                            capacityDialogAutoMode = true
                            showCapacityDialog = true
                        }
                        "dispatched" -> {
                            val orphanNote = if (result.orphanOrderIds.isNotEmpty()) {
                                " ${result.orphanOrderIds.size} order(s) could not be assigned."
                            } else ""
                            actionMessage = DispatchActionMessage(
                                title = "Smart Dispatch Committed",
                                message = "Assigned ${result.ordersAssigned} order(s).$orphanNote",
                            )
                            selectedOrderIds = emptySet()
                            load()
                        }
                        else -> {
                            val warning = result.warnings.firstOrNull() ?: "Smart dispatch did not commit."
                            actionMessage = DispatchActionMessage("Smart Dispatch Incomplete", warning)
                        }
                    }
                } else {
                    actionMessage = DispatchActionMessage("Smart Dispatch Failed", codeMessage(response.code()))
                }
            }.onFailure { throwable ->
                actionMessage = DispatchActionMessage("Smart Dispatch Failed", throwable.message ?: "Network error")
            }
            executing = false
        }
    }

    fun releaseDispatchLock(lock: WarehouseDispatchLock) {
        scope.launch {
            runCatching {
                api.releaseDispatchLock(
                    lock.lockId,
                    WarehouseIdempotencyKeys.dispatchLockRelease(lock.lockId),
                )
            }
                .onSuccess { response ->
                    if (response.isSuccessful && response.body() != null) {
                        val body = response.body()!!
                        actionMessage = DispatchActionMessage(
                            title = "Dispatch Lock Released",
                            message = "Lock ${body.lockId.take(8)} is now ${body.status}.",
                        )
                        lockPendingRelease = null
                        reloadDispatchLocks()
                        load()
                    } else {
                        actionMessage = DispatchActionMessage("Release Failed", codeMessage(response.code()))
                    }
                }
                .onFailure { throwable ->
                    actionMessage = DispatchActionMessage("Release Failed", throwable.message ?: "Network error")
                }
        }
    }

    LaunchedEffect(Unit) { load() }

    LaunchedEffect(Unit) {
        realtimeSignals.refreshTick.collect { load(silent = true) }
    }

    WarehouseReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { executing },
    ) { hadInFlight ->
        if (hadInFlight) {
            executing = false
            actionMessage = DispatchActionMessage(
                title = "Connection restored",
                message = WAREHOUSE_RECONNECT_RECOVERY_HINT,
            )
        }
    }

    DisposableEffect(lifecycleOwner, realtimeClient) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_START -> realtimeClient.connect(
                    onStateChange = { realtimeStatus = it },
                    onEvent = { liveEvent ->
                        when (liveEvent.type) {
                            "SUPPLY_REQUEST_UPDATE" -> reloadSupplyRequests()
                            "DRIVER_AVAILABILITY_CHANGED", "VEHICLE_AVAILABILITY_CHANGED" -> {
                                fleetAlert = "Fleet availability updated"
                                loadVehicles()
                                load()
                            }
                            "DISPATCH_LOCK_CHANGE", "DISPATCH_COMMITTED", "DISPATCH_PLAN_UPDATED" -> {
                                reloadDispatchLocks()
                                load()
                            }
                        }
                    },
                    onReconnect = {
                        scope.launch {
                            reconcileWarehouseSession(api)
                            realtimeSignals.bump()
                            load()
                        }
                    },
                )
                Lifecycle.Event.ON_STOP -> realtimeClient.disconnect()
                else -> Unit
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
            realtimeClient.dispose()
        }
    }

    proposeTarget?.let { orderId ->
        val datePickerState = rememberDatePickerState(initialSelectedDateMillis = System.currentTimeMillis())
        var showReasonDialog by remember(orderId) { mutableStateOf(false) }

        if (showReasonDialog) {
            AlertDialog(
                onDismissRequest = { showReasonDialog = false },
                title = { Text("Reason for new delivery date") },
                text = {
                    OutlinedTextField(
                        value = opsReasonInput,
                        onValueChange = { opsReasonInput = it },
                        label = { Text("Reason (required)") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                },
                confirmButton = {
                    TextButton(
                        onClick = {
                            val selectedMillis = datePickerState.selectedDateMillis ?: return@TextButton
                            val iso = Instant.ofEpochMilli(selectedMillis)
                                .atOffset(ZoneOffset.ofHours(5))
                                .withHour(12).withMinute(0).withSecond(0)
                                .format(DateTimeFormatter.ISO_OFFSET_DATE_TIME)
                            opsActingId = orderId
                            scope.launch {
                                try {
                                    val resp = opsRepository.proposeOrderDelivery(orderId, iso, opsReasonInput.trim())
                                    actionMessage = DispatchActionMessage(
                                        title = if (resp.isSuccessful) "Date proposed" else "Propose failed",
                                        message = if (resp.isSuccessful) {
                                            "Retailer notified — they can accept or reject."
                                        } else {
                                            "HTTP ${resp.code()}"
                                        },
                                    )
                                    proposeTarget = null
                                    opsReasonInput = ""
                                    load()
                                } catch (e: Exception) {
                                    actionMessage = DispatchActionMessage("Propose failed", e.message ?: "Error")
                                } finally {
                                    opsActingId = null
                                }
                            }
                        },
                        enabled = opsReasonInput.isNotBlank(),
                    ) { Text("Send proposal") }
                },
                dismissButton = { TextButton(onClick = { showReasonDialog = false }) { Text("Back") } },
            )
        }

        DatePickerDialog(
            onDismissRequest = { proposeTarget = null },
            confirmButton = { TextButton(onClick = { showReasonDialog = true }) { Text("Next") } },
            dismissButton = { TextButton(onClick = { proposeTarget = null }) { Text("Cancel") } },
        ) {
            DatePicker(state = datePickerState)
        }
    }

    rejectTarget?.let { orderId ->
        AlertDialog(
            onDismissRequest = { rejectTarget = null; opsReasonInput = "" },
            title = { Text("Cancel order") },
            text = {
                OutlinedTextField(
                    value = opsReasonInput,
                    onValueChange = { opsReasonInput = it },
                    label = { Text("Reason (required)") },
                    modifier = Modifier.fillMaxWidth(),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        if (opsReasonInput.isBlank()) return@TextButton
                        opsActingId = orderId
                        scope.launch {
                            try {
                                val resp = opsRepository.rejectOrder(orderId, opsReasonInput.trim())
                                actionMessage = DispatchActionMessage(
                                    title = if (resp.isSuccessful) "Order cancelled" else "Cancel failed",
                                    message = if (resp.isSuccessful) {
                                        "Retailer notified."
                                    } else {
                                        "HTTP ${resp.code()}"
                                    },
                                )
                                rejectTarget = null
                                opsReasonInput = ""
                                load()
                            } catch (e: Exception) {
                                actionMessage = DispatchActionMessage("Cancel failed", e.message ?: "Error")
                            } finally {
                                opsActingId = null
                            }
                        }
                    },
                    enabled = opsReasonInput.isNotBlank(),
                ) { Text("Cancel order") }
            },
            dismissButton = { TextButton(onClick = { rejectTarget = null; opsReasonInput = "" }) { Text("Back") } },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Dispatch") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } } },
                actions = {
                    if (tab == 2) {
                        IconButton(onClick = { showCreateSupplyRequest = true }) { Icon(Icons.Default.Add, "New request") }
                    }
                    if (tab == 3 && !hasActiveManualDispatchLock) {
                        IconButton(onClick = { showAcquireDispatchLock = true }) { Icon(Icons.Default.Lock, "Lock dispatch") }
                    }
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading && preview == null -> PegasusLoadingState(
                title = "Loading dispatch…",
                body = "Orders, drivers, supply, and locks",
                modifier = Modifier.padding(innerPadding),
            )
            error != null && preview == null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Dispatch unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.padding(innerPadding),
            )
            preview != null -> Column(modifier = Modifier.fillMaxSize().padding(innerPadding)) {
                FleetLiveMapSection(
                    ops = opsRepository,
                    realtimeSignals = realtimeSignals,
                    modifier = Modifier.padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
                    mapHeight = 280.dp,
                )
                HandoffTimelineSection(
                    events = handoffEvents,
                    loading = handoffLoading,
                )
                TabRow(selectedTabIndex = tab) {
                    Tab(selected = tab == 0, onClick = { tab = 0 }, text = { Text("Orders (${preview!!.undispatchedOrders.size})") })
                    Tab(selected = tab == 1, onClick = { tab = 1 }, text = { Text("Drivers (${preview!!.availableDrivers.size + preview!!.unavailableDrivers.size})") })
                    Tab(selected = tab == 2, onClick = { tab = 2 }, text = { Text("Supply (${supplyRequests.size})") })
                    Tab(selected = tab == 3, onClick = { tab = 3 }, text = { Text("Locks (${dispatchLocks.size})") })
                }

                RealtimeStatusBanner(status = realtimeStatus)

                fleetAlert?.let { alert ->
                    Text(
                        alert,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.tertiary,
                        modifier = Modifier.padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.xs),
                    )
                }

                when (tab) {
                    0 -> {
                        DispatchOrderList(
                            preview = preview!!,
                            fleetVehicles = fleetVehicles,
                            vehicleReasons = vehicleReasons,
                            vehicleNotes = vehicleNotes,
                            mutatingFleetVehicleId = mutatingFleetVehicleId,
                            dispatchMode = dispatchMode,
                            selectedDriverId = selectedDriverId,
                            selectedOrderIds = selectedOrderIds,
                            executing = executing,
                            opsActingId = opsActingId,
                            driverMenuExpanded = driverMenuExpanded,
                            fmt = fmt,
                            onDispatchModeChange = { dispatchMode = it },
                            onDriverSelect = { selectedDriverId = it },
                            onDriverMenuExpandChange = { driverMenuExpanded = it },
                            onToggleOrder = { orderId, checked ->
                                selectedOrderIds = if (checked) {
                                    selectedOrderIds + orderId
                                } else {
                                    selectedOrderIds - orderId
                                }
                            },
                            onManualDispatch = { runManualDispatch(false) },
                            onSmartDispatch = { showSmartConfirm = true },
                            onProposeDate = { orderId -> proposeTarget = orderId; opsReasonInput = "" },
                            onReject = { orderId -> rejectTarget = orderId; opsReasonInput = "" },
                            onOrderClick = onOrderClick,
                            onVehicleClick = onVehicleClick,
                            onVehicleReasonChange = { vehicleId, reason ->
                                vehicleReasons = vehicleReasons + (vehicleId to reason)
                            },
                            onVehicleNoteChange = { vehicleId, note ->
                                vehicleNotes = vehicleNotes + (vehicleId to note)
                            },
                            onMarkVehicleUnavailable = { vehicle, reason, note ->
                                val finalNote = if (reason == "OTHER") note.trim().takeIf { it.isNotEmpty() } else null
                                updateFleetVehicle(vehicle, false, reason, finalNote)
                            },
                            onRestoreVehicle = { vehicle -> updateFleetVehicle(vehicle, true) },
                        )
                    }
                    1 -> {
                        DispatchDriverList(
                            availableDrivers = preview!!.availableDrivers,
                            unavailableDrivers = preview!!.unavailableDrivers,
                        )
                    }
                    2 -> {
                        if (supplyRequests.isEmpty()) {
                            PegasusStatePane(
                                kind = PegasusStateKind.Empty,
                                headline = "No supply requests",
                                body = "Active factory supply requests will appear here.",
                            )
                        } else {
                            LazyVerticalGrid(
                                columns = GridCells.Adaptive(minSize = 340.dp),
                                contentPadding = PaddingValues(PegasusSpacing.lg),
                                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                            ) {
                                items(supplyRequests, key = { it.requestId }) { request ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(PegasusSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                                                Text(request.requestId.take(8), style = MaterialTheme.typography.titleSmall)
                                                Text(
                                                    "${request.priority} · ${request.totalVolumeVu.toInt()} VU",
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                    maxLines = 1,
                                                    overflow = TextOverflow.Ellipsis,
                                                )
                                            }
                                            Column(horizontalAlignment = Alignment.End, verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                                                WarehouseStatusChip(status = request.state)
                                                if (request.state in setOf("DRAFT", "SUBMITTED", "ACKNOWLEDGED")) {
                                                    TextButton(onClick = { requestPendingCancellation = request }) {
                                                        Text("Cancel")
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                    3 -> {
                        if (dispatchLocks.isEmpty()) {
                            PegasusStatePane(
                                kind = PegasusStateKind.Empty,
                                headline = "Dispatch unlocked",
                                body = "No active dispatch locks for this warehouse scope.",
                            )
                        } else {
                            LazyVerticalGrid(
                                columns = GridCells.Adaptive(minSize = 340.dp),
                                contentPadding = PaddingValues(PegasusSpacing.lg),
                                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                            ) {
                                items(dispatchLocks, key = { it.lockId }) { lock ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(PegasusSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                                                Text(lock.lockType, style = MaterialTheme.typography.titleSmall)
                                                Text(
                                                    lock.lockedBy.ifBlank { lock.lockId.take(8) },
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                    maxLines = 1,
                                                    overflow = TextOverflow.Ellipsis,
                                                )
                                            }
                                            Column(horizontalAlignment = Alignment.End, verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                                                WarehouseStatusChip(status = "ACTIVE")
                                                TextButton(onClick = { lockPendingRelease = lock }) {
                                                    Text("Release")
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    if (showCreateSupplyRequest) {
        CreateSupplyRequestDialog(
            api = api,
            onDismiss = { showCreateSupplyRequest = false },
            onCreate = { form -> createSupplyRequest(form) },
        )
    }

    if (showAcquireDispatchLock) {
        AlertDialog(
            onDismissRequest = { showAcquireDispatchLock = false },
            title = { Text("Lock Dispatch") },
            text = { Text("Acquire a MANUAL_DISPATCH lock to freeze auto-dispatch changes until you release it.") },
            confirmButton = {
                Button(onClick = { acquireDispatchLock() }) {
                    Text("Lock Dispatch")
                }
            },
            dismissButton = { TextButton(onClick = { showAcquireDispatchLock = false }) { Text("Cancel") } },
        )
    }

    if (requestPendingCancellation != null) {
        AlertDialog(
            onDismissRequest = { requestPendingCancellation = null },
            title = { Text("Cancel Supply Request") },
            text = { Text("Cancel request ${requestPendingCancellation!!.requestId.take(8)}?") },
            confirmButton = {
                Button(onClick = { cancelSupplyRequest(requestPendingCancellation!!) }) {
                    Text("Cancel Request")
                }
            },
            dismissButton = { TextButton(onClick = { requestPendingCancellation = null }) { Text("Keep") } },
        )
    }

    if (lockPendingRelease != null) {
        AlertDialog(
            onDismissRequest = { lockPendingRelease = null },
            title = { Text("Release Dispatch Lock") },
            text = { Text("Release ${lockPendingRelease!!.lockType} for this warehouse scope?") },
            confirmButton = {
                Button(onClick = { releaseDispatchLock(lockPendingRelease!!) }) {
                    Text("Release")
                }
            },
            dismissButton = { TextButton(onClick = { lockPendingRelease = null }) { Text("Keep") } },
        )
    }

    if (showSmartConfirm) {
        AlertDialog(
            onDismissRequest = { showSmartConfirm = false },
            title = { Text("Run smart dispatch?") },
            text = {
                val count = if (selectedOrderIds.isNotEmpty()) selectedOrderIds.size else preview?.undispatchedOrders?.size ?: 0
                Text("Assign $count order(s) using the optimizer across available trucks.")
            },
            confirmButton = {
                Button(onClick = { runSmartDispatch() }) { Text("Smart Dispatch") }
            },
            dismissButton = { TextButton(onClick = { showSmartConfirm = false }) { Text("Cancel") } },
        )
    }

    if (showCapacityDialog) {
        AlertDialog(
            onDismissRequest = { showCapacityDialog = false },
            title = { Text("Capacity exceeded") },
            text = {
                Column {
                    Text(
                        if (capacityDialogAutoMode) {
                            "Smart dispatch cannot fit all orders. Accept partial to dispatch feasible routes, or force to override."
                        } else {
                            "Selected orders exceed the truck effective capacity (95% buffer)."
                        },
                    )
                    capacityWarnings.forEach { warning ->
                        Column(modifier = Modifier.padding(top = PegasusSpacing.sm)) {
                            Text(
                                "${"%.1f".format(warning.loadedVu)} VU loaded / ${"%.1f".format(warning.effectiveMaxVu)} VU max",
                                style = MaterialTheme.typography.bodySmall,
                            )
                            if (warning.suggestedUnselectOrderIds.isNotEmpty()) {
                                Text(
                                    "Suggested unselect: ${warning.suggestedUnselectOrderIds.joinToString { it.take(8) }}",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            if (warning.suggestedDeferOrderIds.isNotEmpty()) {
                                Text(
                                    "Suggested defer: ${warning.suggestedDeferOrderIds.joinToString { it.take(8) }}",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }
                    }
                    if (!capacityDialogAutoMode) {
                        val suggestedIds = capacityWarnings.flatMap { it.suggestedUnselectOrderIds }.toSet()
                        if (suggestedIds.isNotEmpty()) {
                            TextButton(onClick = {
                                selectedOrderIds = selectedOrderIds - suggestedIds
                                showCapacityDialog = false
                            }) { Text("Apply suggestion") }
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    showCapacityDialog = false
                    if (capacityDialogAutoMode) {
                        runSmartDispatch(forceCapacity = true)
                    } else {
                        runManualDispatch(forceCapacity = true)
                    }
                }) { Text("Force dispatch") }
            },
            dismissButton = {
                Row {
                    if (capacityDialogAutoMode) {
                        TextButton(onClick = {
                            showCapacityDialog = false
                            runSmartDispatch(acceptPartial = true)
                        }) { Text("Accept partial") }
                    }
                    TextButton(onClick = { showCapacityDialog = false }) { Text("Cancel") }
                }
            },
        )
    }

    if (actionMessage != null) {
        AlertDialog(
            onDismissRequest = { actionMessage = null },
            title = { Text(actionMessage!!.title) },
            text = { Text(actionMessage!!.message) },
            confirmButton = { TextButton(onClick = { actionMessage = null }) { Text("OK") } },
        )
    }
}

@Composable
private fun RealtimeStatusBanner(status: WarehouseRealtimeStatus) {
    val config = when (status) {
        WarehouseRealtimeStatus.IDLE, WarehouseRealtimeStatus.LIVE -> null
        WarehouseRealtimeStatus.CONNECTING -> Triple("Connecting live warehouse updates…", MaterialTheme.colorScheme.secondaryContainer, MaterialTheme.colorScheme.onSecondaryContainer)
        WarehouseRealtimeStatus.RECONNECTING -> Triple("Live updates reconnecting. Current data may be stale.", MaterialTheme.colorScheme.tertiaryContainer, MaterialTheme.colorScheme.onTertiaryContainer)
        WarehouseRealtimeStatus.OFFLINE -> Triple("Offline. Live updates are paused until the network returns.", MaterialTheme.colorScheme.errorContainer, MaterialTheme.colorScheme.onErrorContainer)
    }

    if (config != null) {
        Surface(
            color = config.second,
            contentColor = config.third,
            modifier = Modifier.fillMaxWidth().padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
            shape = MaterialTheme.shapes.medium,
        ) {
            Text(
                text = config.first,
                style = MaterialTheme.typography.bodySmall,
                modifier = Modifier.padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.md),
            )
        }
    }
}

private data class DispatchActionMessage(
    val title: String,
    val message: String,
)
