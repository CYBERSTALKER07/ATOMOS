package com.pegasusx.warehouse.ui.screens.dispatch

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import com.pegasusx.warehouse.data.model.CreateWarehouseDispatchLockRequest
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import com.pegasusx.warehouse.data.model.CreateWarehouseSupplyRequestRequest
import com.pegasusx.warehouse.data.model.WarehouseDispatchLock
import com.pegasusx.warehouse.data.model.WarehouseSupplyRequest
import com.pegasusx.warehouse.data.model.WarehouseSupplyRequestTransitionRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeClient
import com.pegasusx.warehouse.data.remote.reconcileWarehouseSession
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.components.FleetLiveMapSection
import com.pegasusx.warehouse.ui.components.WarehouseLoadingState
import com.pegasusx.warehouse.ui.components.WarehouseOpsListCard
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasusx.warehouse.ui.components.WarehouseStateKind
import com.pegasusx.warehouse.ui.components.WarehouseStatePane
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeStatus
import com.pegasusx.warehouse.ui.realtime.WAREHOUSE_RECONNECT_RECOVERY_HINT
import com.pegasusx.warehouse.ui.realtime.WarehouseReconnectRecoveryEffect
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.google.gson.JsonArray
import com.google.gson.JsonObject
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

private val DISPATCH_UNAVAILABLE_REASON_LABELS = mapOf(
    "MAINTENANCE" to "Maintenance",
    "TRUCK_DAMAGED" to "Truck Damaged",
    "REGULATORY_HOLD" to "Regulatory Hold",
    "MANUAL_HOLD" to "Manual Hold",
)

private const val DISPATCH_TETRIS_BUFFER = 0.95

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DispatchScreen(
    api: WarehouseApi,
    opsRepository: WarehouseOperationsRepository,
    realtimeSignals: WarehouseRealtimeSignals,
    onBack: (() -> Unit)? = null,
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    var preview by remember { mutableStateOf<DispatchPreview?>(null) }
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

    fun load() {
        loading = true; error = null
        scope.launch {
            try {
                val previewResp = api.getDispatchPreview()
                val supplyResp = api.getSupplyRequests()
                val lockResp = api.getDispatchLocks()
                if (previewResp.isSuccessful && previewResp.body() != null) preview = previewResp.body()!!
                else error = codeMessage(previewResp.code(), "Failed to load dispatch preview")
                if (supplyResp.isSuccessful && supplyResp.body() != null) {
                    supplyRequests = supplyResp.body()!!.resolved()
                }
                else if (error == null) error = codeMessage(supplyResp.code(), "Failed to load supply requests")
                if (lockResp.isSuccessful && lockResp.body() != null) {
                    dispatchLocks = lockResp.body()!!.locks
                }
                else if (error == null) error = codeMessage(lockResp.code(), "Failed to load dispatch locks")
            } catch (e: Exception) { error = e.message ?: "Network error" }
            finally { loading = false }
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

    fun createSupplyRequest(factoryId: String, priority: String, notes: String) {
        scope.launch {
            runCatching {
                val key = WarehouseIdempotencyKeys.createSupplyRequest(factoryId, priority, notes)
                api.createSupplyRequest(
                    key,
                    CreateWarehouseSupplyRequestRequest(
                        factoryId = factoryId,
                        priority = priority,
                        notes = notes,
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
                            "DISPATCH_LOCK_CHANGE", "DISPATCH_COMMITTED" -> {
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
            loading -> WarehouseLoadingState(
                title = "Loading dispatch…",
                body = "Orders, drivers, supply, and locks",
                modifier = Modifier.padding(innerPadding),
            )
            error != null -> WarehouseStatePane(
                kind = WarehouseStateKind.Error,
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
                TabRow(selectedTabIndex = tab) {
                    Tab(selected = tab == 0, onClick = { tab = 0 }, text = { Text("Orders (${preview!!.undispatchedOrders.size})") })
                    Tab(selected = tab == 1, onClick = { tab = 1 }, text = { Text("Drivers (${preview!!.availableDrivers.size + preview!!.unavailableDrivers.size})") })
                    Tab(selected = tab == 2, onClick = { tab = 2 }, text = { Text("Supply (${supplyRequests.size})") })
                    Tab(selected = tab == 3, onClick = { tab = 3 }, text = { Text("Locks (${dispatchLocks.size})") })
                }

                RealtimeStatusBanner(status = realtimeStatus)

                when (tab) {
                    0 -> {
                        if (preview!!.undispatchedOrders.isEmpty()) {
                            WarehouseStatePane(
                                kind = WarehouseStateKind.Empty,
                                headline = "All orders dispatched",
                                body = "No undispatched orders remain in the preview queue.",
                            )
                        } else {
                            val selectedDriver = preview!!.availableDrivers.firstOrNull { it.driverId == selectedDriverId }
                            val selectedVolume = preview!!.undispatchedOrders
                                .filter { selectedOrderIds.contains(it.orderId) }
                                .sumOf { it.volumeVu }
                            val effectiveMax = when {
                                selectedDriver?.freeVolumeVu != null && selectedDriver.freeVolumeVu > 0 ->
                                    selectedDriver.freeVolumeVu * DISPATCH_TETRIS_BUFFER
                                else -> (selectedDriver?.maxVolumeVu ?: 0.0) * DISPATCH_TETRIS_BUFFER
                            }
                            Column(modifier = Modifier.fillMaxSize()) {
                                Column(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.md),
                                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                                ) {
                                    Box {
                                        OutlinedButton(
                                            onClick = { driverMenuExpanded = true },
                                            modifier = Modifier.fillMaxWidth(),
                                        ) {
                                            Text(
                                                selectedDriver?.let { "${it.name} · ${it.maxVolumeVu} VU max" }
                                                    ?: "Select truck / driver",
                                                maxLines = 1,
                                                overflow = TextOverflow.Ellipsis,
                                            )
                                        }
                                        DropdownMenu(
                                            expanded = driverMenuExpanded,
                                            onDismissRequest = { driverMenuExpanded = false },
                                        ) {
                                            preview!!.availableDrivers.forEach { driver ->
                                                DropdownMenuItem(
                                                    text = {
                                                        Text(
                                                            "${driver.name} · ${driver.vehicleLabel.ifBlank { driver.truckStatus }}",
                                                            maxLines = 1,
                                                            overflow = TextOverflow.Ellipsis,
                                                        )
                                                    },
                                                    onClick = {
                                                        selectedDriverId = driver.driverId
                                                        driverMenuExpanded = false
                                                    },
                                                )
                                            }
                                        }
                                    }
                                    if (selectedDriver != null) {
                                        Text(
                                            "Loaded ${"%.1f".format(selectedVolume)} / ${"%.1f".format(effectiveMax)} VU effective",
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        )
                                    }
                                    Button(
                                        onClick = { runManualDispatch(false) },
                                        enabled = !executing && selectedDriverId.isNotBlank() && selectedOrderIds.isNotEmpty(),
                                        modifier = Modifier.fillMaxWidth(),
                                    ) {
                                        Text(if (executing) "Dispatching…" else "Manual (${selectedOrderIds.size})")
                                    }
                                    OutlinedButton(
                                        onClick = { showSmartConfirm = true },
                                        enabled = !executing && preview!!.undispatchedOrders.isNotEmpty() && preview!!.availableDrivers.isNotEmpty(),
                                        modifier = Modifier.fillMaxWidth(),
                                    ) {
                                        Text("Smart Dispatch")
                                    }
                                    if (preview!!.fleetEffectiveCapacityVu > 0) {
                                        Text(
                                            "Fleet ${"%.1f".format(preview!!.fleetEffectiveCapacityVu)} VU effective",
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        )
                                    }
                                }
                                LazyColumn(
                                    contentPadding = PaddingValues(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.md),
                                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                                ) {
                                items(preview!!.undispatchedOrders, key = { it.orderId }) { o ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(PegasusSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Checkbox(
                                                checked = selectedOrderIds.contains(o.orderId),
                                                onCheckedChange = { checked ->
                                                    selectedOrderIds = if (checked) {
                                                        selectedOrderIds + o.orderId
                                                    } else {
                                                        selectedOrderIds - o.orderId
                                                    }
                                                },
                                            )
                                            Column(modifier = Modifier.weight(1f)) {
                                                Text(o.retailerName.ifBlank { o.orderId.take(8) }, style = MaterialTheme.typography.titleSmall)
                                                Text(
                                                    fmt.format(o.totalUzs) + " UZS · ${"%.1f".format(o.volumeVu)} VU",
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                )
                                            }
                                        }
                                    }
                                }
                                if (preview!!.proposedRoutes.isNotEmpty() || preview!!.optimizerWarnings.isNotEmpty()) {
                                    item {
                                        Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                                            WarehouseSectionTitle("Smart suggest preview")
                                            preview!!.optimizerSource?.let { source ->
                                                Text("Source: $source", style = MaterialTheme.typography.bodySmall)
                                            }
                                            preview!!.optimizerWarnings.forEach { warning ->
                                                Text(warning, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.tertiary)
                                            }
                                        }
                                    }
                                    items(preview!!.proposedRoutes.size) { index ->
                                        val route = preview!!.proposedRoutes[index]
                                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                            Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
                                                Text(
                                                    route.driverName ?: route.driverId ?: "Driver",
                                                    style = MaterialTheme.typography.titleSmall,
                                                )
                                                Text(
                                                    "${route.stopCount ?: route.orderIds.size} stops · ${"%.1f".format(route.volumeVu ?: 0.0)} VU",
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                )
                                                Text(
                                                    route.orderIds.joinToString(" → "),
                                                    style = MaterialTheme.typography.labelSmall,
                                                    maxLines = 2,
                                                    overflow = TextOverflow.Ellipsis,
                                                )
                                            }
                                        }
                                    }
                                }
                                }
                            }
                        }
                    }
                    1 -> {
                        if (preview!!.availableDrivers.isEmpty() && preview!!.unavailableDrivers.isEmpty()) {
                            WarehouseStatePane(
                                kind = WarehouseStateKind.Empty,
                                headline = "No drivers",
                                body = "Available and unavailable drivers will appear here.",
                            )
                        } else {
                            LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
                                if (preview!!.availableDrivers.isNotEmpty()) {
                                    item { WarehouseSectionTitle("Available") }
                                }
                                items(preview!!.availableDrivers, key = { it.driverId }) { d ->
                                    WarehouseOpsListCard(
                                        headline = d.name,
                                        supporting = d.vehicleLabel.ifBlank { d.phone.ifBlank { d.truckStatus.ifBlank { "No vehicle" } } },
                                        status = d.truckStatus.ifBlank { "IDLE" },
                                    )
                                }
                                if (preview!!.unavailableDrivers.isNotEmpty()) {
                                    item { WarehouseSectionTitle("Vehicle unavailable") }
                                }
                                items(preview!!.unavailableDrivers, key = { "unavailable-${it.driverId}" }) { d ->
                                    WarehouseOpsListCard(
                                        headline = d.name,
                                        supporting = buildString {
                                            append(d.vehicleLabel.ifBlank { d.phone.ifBlank { "Assigned vehicle unavailable" } })
                                            if (!d.unavailableReason.isNullOrBlank()) {
                                                append(" · ")
                                                append(vehicleUnavailableReasonLabel(d.unavailableReason))
                                            }
                                        },
                                        status = d.truckStatus.ifBlank { "UNAVAILABLE" },
                                    )
                                }
                            }
                        }
                    }
                    2 -> {
                        if (supplyRequests.isEmpty()) {
                            WarehouseStatePane(
                                kind = WarehouseStateKind.Empty,
                                headline = "No supply requests",
                                body = "Active factory supply requests will appear here.",
                            )
                        } else {
                            LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
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
                            WarehouseStatePane(
                                kind = WarehouseStateKind.Empty,
                                headline = "Dispatch unlocked",
                                body = "No active dispatch locks for this warehouse scope.",
                            )
                        } else {
                            LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
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
            onDismiss = { showCreateSupplyRequest = false },
            onCreate = { factoryId, priority, notes -> createSupplyRequest(factoryId, priority, notes) },
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

private fun vehicleUnavailableReasonLabel(reason: String): String {
    return DISPATCH_UNAVAILABLE_REASON_LABELS[reason]
        ?: reason.lowercase().split('_').joinToString(" ") { token ->
            token.replaceFirstChar { ch -> ch.titlecase() }
        }
}

@Composable
private fun CreateSupplyRequestDialog(
    onDismiss: () -> Unit,
    onCreate: (String, String, String) -> Unit,
) {
    var factoryId by remember { mutableStateOf("") }
    var priority by remember { mutableStateOf("NORMAL") }
    var notes by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("New Supply Request") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
                OutlinedTextField(
                    value = factoryId,
                    onValueChange = { factoryId = it },
                    label = { Text("Factory ID") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                Text("Priority", style = MaterialTheme.typography.labelMedium)
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    listOf("NORMAL", "URGENT", "CRITICAL").forEach { option ->
                        FilterChip(
                            selected = priority == option,
                            onClick = { priority = option },
                            label = { Text(option) },
                        )
                    }
                }
                OutlinedTextField(
                    value = notes,
                    onValueChange = { notes = it },
                    label = { Text("Notes") },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 3,
                )
                Text(
                    "This submits a warehouse supply request through the backend demand forecast path.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        confirmButton = {
            Button(
                onClick = { onCreate(factoryId.trim(), priority, notes.trim()) },
                enabled = factoryId.isNotBlank(),
            ) {
                Text("Submit")
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

private data class DispatchActionMessage(
    val title: String,
    val message: String,
)
