package com.pegasus.warehouse.ui.screens.dispatch

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
import androidx.compose.ui.text.style.TextOverflow
import com.pegasus.warehouse.data.model.DispatchPreview
import com.pegasus.warehouse.data.model.WarehouseDispatchLock
import com.pegasus.warehouse.data.model.WarehouseSupplyRequest
import com.pegasus.warehouse.data.remote.WarehouseApi
import com.pegasus.warehouse.data.remote.WarehouseRealtimeClient
import com.pegasus.warehouse.ui.theme.LabSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DispatchScreen(
    api: WarehouseApi,
    onBack: () -> Unit,
) {
    var preview by remember { mutableStateOf<DispatchPreview?>(null) }
    var supplyRequests by remember { mutableStateOf<List<WarehouseSupplyRequest>>(emptyList()) }
    var dispatchLocks by remember { mutableStateOf<List<WarehouseDispatchLock>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var tab by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }
    val realtimeClient = remember { WarehouseRealtimeClient() }

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

    fun reloadSupplyRequests() {
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

    fun reloadDispatchLocks() {
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

    LaunchedEffect(Unit) { load() }

    DisposableEffect(Unit) {
        realtimeClient.connect { event ->
            when (event.type) {
                "SUPPLY_REQUEST_UPDATE" -> reloadSupplyRequests()
                "DISPATCH_LOCK_CHANGE" -> reloadDispatchLocks()
            }
        }
        onDispose { realtimeClient.disconnect() }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Dispatch") },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } },
                actions = { IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") } },
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
                    Tab(selected = tab == 1, onClick = { tab = 1 }, text = { Text("Drivers (${preview!!.availableDrivers.size})") })
                    Tab(selected = tab == 2, onClick = { tab = 2 }, text = { Text("Supply (${supplyRequests.size})") })
                    Tab(selected = tab == 3, onClick = { tab = 3 }, text = { Text("Locks (${dispatchLocks.size})") })
                }
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
                        if (preview!!.availableDrivers.isEmpty()) {
                            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                                Text("No available drivers", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        } else {
                            LazyColumn(contentPadding = PaddingValues(LabSpacing.lg), verticalArrangement = Arrangement.spacedBy(LabSpacing.md)) {
                                items(preview!!.availableDrivers, key = { it.driverId }) { d ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f)) {
                                                Text(d.name, style = MaterialTheme.typography.titleSmall)
                                                Text(
                                                    d.vehicleLabel.ifBlank { d.truckStatus.ifBlank { "No vehicle" } },
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
                                            SuggestionChip(onClick = {}, label = { Text(request.state) })
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
                                            SuggestionChip(
                                                onClick = {},
                                                label = { Text(lock.warehouseId.ifBlank { "Global" }.take(8)) },
                                            )
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
