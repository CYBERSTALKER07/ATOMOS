package com.pegasus.warehouse.ui.screens.dispatch

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
import com.pegasus.warehouse.data.model.DispatchPreview
import com.pegasus.warehouse.data.model.CreateWarehouseDispatchLockRequest
import com.pegasus.warehouse.data.model.CreateWarehouseSupplyRequestRequest
import com.pegasus.warehouse.data.model.WarehouseDispatchLock
import com.pegasus.warehouse.data.model.WarehouseSupplyRequest
import com.pegasus.warehouse.data.model.WarehouseSupplyRequestTransitionRequest
import com.pegasus.warehouse.data.remote.WarehouseApi
import com.pegasus.warehouse.data.remote.WarehouseRealtimeClient
import com.pegasus.warehouse.data.remote.WarehouseRealtimeStatus
import com.pegasus.warehouse.ui.theme.LabSpacing
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DispatchScreen(
    api: WarehouseApi,
    onBack: () -> Unit,
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
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }
    val realtimeClient = remember(context) { WarehouseRealtimeClient(context) }

    val hasActiveManualDispatchLock = dispatchLocks.any { lock -> lock.lockType == "MANUAL_DISPATCH" }

    fun load() {
        loading = true; error = null
        scope.launch {
            try {
                val previewResp = api.getDispatchPreview()
                val supplyResp = api.getSupplyRequests()
                val lockResp = api.getDispatchLocks()
                if (previewResp.isSuccessful && previewResp.body() != null) preview = previewResp.body()!!
                else error = "Failed (${previewResp.code()})"
                if (supplyResp.isSuccessful && supplyResp.body() != null) supplyRequests = supplyResp.body()!!
                else if (error == null) error = "Failed (${supplyResp.code()})"
                if (lockResp.isSuccessful && lockResp.body() != null) dispatchLocks = lockResp.body()!!
                else if (error == null) error = "Failed (${lockResp.code()})"
            } catch (e: Exception) { error = e.message ?: "Network error" }
            finally { loading = false }
        }
    }

    val reloadSupplyRequests: () -> Unit = {
        scope.launch {
            runCatching { api.getSupplyRequests() }
                .onSuccess { response ->
                    if (response.isSuccessful && response.body() != null) {
                        supplyRequests = response.body()!!
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
                        dispatchLocks = response.body()!!
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
                api.createSupplyRequest(
                    CreateWarehouseSupplyRequestRequest(
                        factoryId = factoryId,
                        priority = priority,
                        notes = notes,
                    )
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
                    actionMessage = DispatchActionMessage("Supply Request Failed", "Failed (${response.code()})")
                }
            }.onFailure { throwable ->
                actionMessage = DispatchActionMessage("Supply Request Failed", throwable.message ?: "Network error")
            }
        }
    }

    fun cancelSupplyRequest(request: WarehouseSupplyRequest) {
        scope.launch {
            runCatching {
                api.transitionSupplyRequest(
                    request.requestId,
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
                    actionMessage = DispatchActionMessage("Cancellation Failed", "Failed (${response.code()})")
                }
            }.onFailure { throwable ->
                actionMessage = DispatchActionMessage("Cancellation Failed", throwable.message ?: "Network error")
            }
        }
    }

    fun acquireDispatchLock() {
        scope.launch {
            runCatching { api.createDispatchLock(CreateWarehouseDispatchLockRequest(lockType = "MANUAL_DISPATCH")) }
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
                        actionMessage = DispatchActionMessage("Lock Failed", "Failed (${response.code()})")
                    }
                }
                .onFailure { throwable ->
                    actionMessage = DispatchActionMessage("Lock Failed", throwable.message ?: "Network error")
                }
        }
    }

    fun releaseDispatchLock(lock: WarehouseDispatchLock) {
        scope.launch {
            runCatching { api.releaseDispatchLock(lock.lockId) }
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
                        actionMessage = DispatchActionMessage("Release Failed", "Failed (${response.code()})")
                    }
                }
                .onFailure { throwable ->
                    actionMessage = DispatchActionMessage("Release Failed", throwable.message ?: "Network error")
                }
        }
    }

    LaunchedEffect(Unit) { load() }

    DisposableEffect(lifecycleOwner, realtimeClient) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_START -> realtimeClient.connect(
                    onStateChange = { realtimeStatus = it },
                    onEvent = { liveEvent ->
                        when (liveEvent.type) {
                            "SUPPLY_REQUEST_UPDATE" -> reloadSupplyRequests()
                            "DISPATCH_LOCK_CHANGE" -> {
                                reloadDispatchLocks()
                                load()
                            }
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
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } },
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
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(LabSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            preview != null -> Column(modifier = Modifier.fillMaxSize().padding(innerPadding)) {
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
                            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                                Text("All orders dispatched", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        } else {
                            LazyColumn(contentPadding = PaddingValues(LabSpacing.lg), verticalArrangement = Arrangement.spacedBy(LabSpacing.md)) {
                                items(preview!!.undispatchedOrders, key = { it.orderId }) { o ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f)) {
                                                Text(o.retailerName.ifBlank { o.orderId.take(8) }, style = MaterialTheme.typography.titleSmall)
                                                Text(
                                                    fmt.format(o.totalUzs) + " UZS · ${o.itemCount} items",
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                )
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                    1 -> {
                        if (preview!!.availableDrivers.isEmpty() && preview!!.unavailableDrivers.isEmpty()) {
                            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                                Text("No available drivers", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        } else {
                            LazyColumn(contentPadding = PaddingValues(LabSpacing.lg), verticalArrangement = Arrangement.spacedBy(LabSpacing.md)) {
                                if (preview!!.availableDrivers.isNotEmpty()) {
                                    item {
                                        Text(
                                            "Available",
                                            style = MaterialTheme.typography.labelLarge,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        )
                                    }
                                }
                                items(preview!!.availableDrivers, key = { it.driverId }) { d ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f)) {
                                                Text(d.name, style = MaterialTheme.typography.titleSmall)
                                                Text(
                                                    d.vehicleLabel.ifBlank { d.phone.ifBlank { d.truckStatus.ifBlank { "No vehicle" } } },
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                )
                                            }
                                            AssistChip(onClick = {}, label = { Text(d.truckStatus.ifBlank { "IDLE" }) })
                                        }
                                    }
                                }
                                if (preview!!.unavailableDrivers.isNotEmpty()) {
                                    item {
                                        Text(
                                            "Vehicle Unavailable",
                                            style = MaterialTheme.typography.labelLarge,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        )
                                    }
                                }
                                items(preview!!.unavailableDrivers, key = { "unavailable-${it.driverId}" }) { d ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(LabSpacing.xs)) {
                                                Text(d.name, style = MaterialTheme.typography.titleSmall)
                                                Text(
                                                    d.vehicleLabel.ifBlank { d.phone.ifBlank { "Assigned vehicle unavailable" } },
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                )
                                                if (!d.unavailableReason.isNullOrBlank()) {
                                                    Text(
                                                        vehicleUnavailableReasonLabel(d.unavailableReason),
                                                        style = MaterialTheme.typography.labelSmall,
                                                        color = MaterialTheme.colorScheme.tertiary,
                                                    )
                                                }
                                            }
                                            AssistChip(onClick = {}, label = { Text(d.truckStatus.ifBlank { "IDLE" }) })
                                        }
                                    }
                                }
                            }
                        }
                    }
                    2 -> {
                        if (supplyRequests.isEmpty()) {
                            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                                Text("No active supply requests", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        } else {
                            LazyColumn(contentPadding = PaddingValues(LabSpacing.lg), verticalArrangement = Arrangement.spacedBy(LabSpacing.md)) {
                                items(supplyRequests, key = { it.requestId }) { request ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(LabSpacing.xs)) {
                                                Text(request.requestId.take(8), style = MaterialTheme.typography.titleSmall)
                                                Text(
                                                    "${request.state} · ${request.priority} · ${request.totalVolumeVu.toInt()} VU",
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                    maxLines = 1,
                                                    overflow = TextOverflow.Ellipsis,
                                                )
                                            }
                                            Column(horizontalAlignment = Alignment.End, verticalArrangement = Arrangement.spacedBy(LabSpacing.sm)) {
                                                SuggestionChip(onClick = {}, label = { Text(request.state) })
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
                            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                                Text("Dispatch is currently unlocked", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        } else {
                            LazyColumn(contentPadding = PaddingValues(LabSpacing.lg), verticalArrangement = Arrangement.spacedBy(LabSpacing.md)) {
                                items(dispatchLocks, key = { it.lockId }) { lock ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(LabSpacing.xs)) {
                                                Text(lock.lockType, style = MaterialTheme.typography.titleSmall)
                                                Text(
                                                    lock.lockedBy.ifBlank { lock.lockId.take(8) },
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                    maxLines = 1,
                                                    overflow = TextOverflow.Ellipsis,
                                                )
                                            }
                                            Column(horizontalAlignment = Alignment.End, verticalArrangement = Arrangement.spacedBy(LabSpacing.sm)) {
                                                SuggestionChip(
                                                    onClick = {},
                                                    label = { Text(lock.warehouseId.ifBlank { "Global" }.take(8)) },
                                                )
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
            modifier = Modifier.fillMaxWidth().padding(horizontal = LabSpacing.lg, vertical = LabSpacing.sm),
            shape = MaterialTheme.shapes.medium,
        ) {
            Text(
                text = config.first,
                style = MaterialTheme.typography.bodySmall,
                modifier = Modifier.padding(horizontal = LabSpacing.lg, vertical = LabSpacing.md),
            )
        }
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
            Column(verticalArrangement = Arrangement.spacedBy(LabSpacing.md)) {
                OutlinedTextField(
                    value = factoryId,
                    onValueChange = { factoryId = it },
                    label = { Text("Factory ID") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                Text("Priority", style = MaterialTheme.typography.labelMedium)
                Row(horizontalArrangement = Arrangement.spacedBy(LabSpacing.sm)) {
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
