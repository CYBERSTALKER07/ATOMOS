package com.pegasusx.warehouse.ui.screens.drivers

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.AssignVehicleRequest
import com.pegasusx.warehouse.data.model.CreateDriverRequest
import com.pegasusx.warehouse.data.model.Driver
import com.pegasusx.warehouse.data.model.Vehicle
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.realtime.WAREHOUSE_RECONNECT_RECOVERY_HINT
import com.pegasusx.warehouse.ui.realtime.WarehouseReconnectRecoveryEffect
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.launch



@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DriversScreen(
    api: WarehouseApi,
    realtimeSignals: WarehouseRealtimeSignals,
    onBack: (() -> Unit)? = null,
) {
    var drivers by remember { mutableStateOf<List<Driver>>(emptyList()) }
    var vehicles by remember { mutableStateOf<List<Vehicle>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var showCreate by remember { mutableStateOf(false) }
    var createdPin by remember { mutableStateOf<String?>(null) }
    var assignDriver by remember { mutableStateOf<Driver?>(null) }
    var assigningDriverId by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    WarehouseReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { assigningDriverId != null },
    ) { hadInFlight ->
        if (hadInFlight) {
            assigningDriverId = null
            scope.launch { snackbarHostState.showSnackbar(WAREHOUSE_RECONNECT_RECOVERY_HINT) }
        }
    }

    fun load(silent: Boolean = false) {
        if (!silent) loading = true
        error = null
        scope.launch {
            try {
                val driverResp = api.getDrivers()
                val vehicleResp = api.getVehicles()
                if (driverResp.isSuccessful && driverResp.body() != null) {
                    drivers = driverResp.body()!!.drivers
                } else if (!silent) {
                    error = "Failed (${driverResp.code()})"
                }
                if (vehicleResp.isSuccessful && vehicleResp.body() != null) {
                    vehicles = vehicleResp.body()!!.vehicles
                } else if (!silent && error == null) {
                    error = "Failed (${vehicleResp.code()})"
                }
            } catch (e: Exception) {
                if (!silent) error = e.message ?: "Network error"
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    LaunchedEffect(Unit) {
        realtimeSignals.refreshTick.collect { load(silent = true) }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Drivers") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } } },
                actions = {
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                    IconButton(onClick = { showCreate = true }) { Icon(Icons.Default.Add, "Add") }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        when {
            loading && drivers.isEmpty() -> PegasusLoadingState(
                title = "Loading drivers…",
                body = "Fleet driver roster",
                modifier = Modifier.padding(innerPadding),
            )
            error != null && drivers.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Drivers unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.padding(innerPadding),
            )
            drivers.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No drivers",
                body = "Fleet drivers will appear here.",
                modifier = Modifier.padding(innerPadding),
            )
            else -> DriversList(
                drivers = drivers,
                vehicles = vehicles,
                assigningDriverId = assigningDriverId,
                onAssignClick = { assignDriver = it },
                modifier = Modifier.padding(innerPadding)
            )
        }
    }

    // Create driver dialog
    if (showCreate) {
        CreateDriverDialog(
            api = api,
            realtimeSignals = realtimeSignals,
            onDismiss = { showCreate = false },
            onCreated = { pin ->
                createdPin = pin
                showCreate = false
                load()
            },
        )
    }

    // PIN display dialog
    if (createdPin != null) {
        AlertDialog(
            onDismissRequest = { createdPin = null },
            title = { Text("Driver Created") },
            text = {
                Column {
                    Text("One-time PIN — save it now:")
                    Spacer(Modifier.height(PegasusSpacing.md))
                    Text(createdPin!!, style = MaterialTheme.typography.headlineMedium, color = MaterialTheme.colorScheme.primary)
                }
            },
            confirmButton = { TextButton(onClick = { createdPin = null }) { Text("Done") } },
        )
    }

    if (assignDriver != null) {
        AssignVehicleDialog(
            driver = assignDriver!!,
            vehicles = vehicles.filter { it.isActive || it.vehicleId == assignDriver!!.vehicleId },
            onDismiss = { assignDriver = null },
            onAssign = { vehicleId ->
                assigningDriverId = assignDriver!!.driverId
                scope.launch {
                    try {
                        val resp = api.assignDriverVehicle(
                            assignDriver!!.driverId,
                            AssignVehicleRequest(vehicleId = vehicleId),
                            WarehouseIdempotencyKeys.assignDriverVehicle(assignDriver!!.driverId, vehicleId),
                        )
                        if (resp.isSuccessful) {
                            assignDriver = null
                            load()
                            snackbarHostState.showSnackbar("Driver assignment updated")
                        } else {
                            error = "Failed (${resp.code()})"
                        }
                    } catch (e: Exception) {
                        error = e.message ?: "Network error"
                    } finally {
                        assigningDriverId = null
                    }
                }
            },
        )
    }
}

@Composable
private fun AssignVehicleDialog(
    driver: Driver,
    vehicles: List<Vehicle>,
    onDismiss: () -> Unit,
    onAssign: (String?) -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Assign Vehicle") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                Text(driver.name, style = MaterialTheme.typography.titleSmall)
                TextButton(onClick = { onAssign(null) }) {
                    Text("Unassign")
                }
                vehicles.forEach { vehicle ->
                    TextButton(onClick = { onAssign(vehicle.vehicleId) }) {
                        Text(vehicleLabel(vehicle))
                    }
                }
            }
        },
        confirmButton = { TextButton(onClick = onDismiss) { Text("Close") } },
    )
}



@Composable
private fun CreateDriverDialog(
    api: WarehouseApi,
    realtimeSignals: WarehouseRealtimeSignals,
    onDismiss: () -> Unit,
    onCreated: (String) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var submitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    WarehouseReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { submitting },
    ) { hadInFlight ->
        if (hadInFlight) {
            submitting = false
            error = WAREHOUSE_RECONNECT_RECOVERY_HINT
        }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add Driver") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
                OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(value = phone, onValueChange = { phone = it }, label = { Text("Phone") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                if (error != null) Text(error!!, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    submitting = true; error = null
                    scope.launch {
                        try {
                            val resp = api.createDriver(
                                CreateDriverRequest(name = name, phone = phone),
                                WarehouseIdempotencyKeys.createDriver(phone),
                            )
                            if (resp.isSuccessful && resp.body() != null) onCreated(resp.body()!!.pin)
                            else error = "Failed (${resp.code()})"
                        } catch (e: Exception) { error = e.message ?: "Error" }
                        finally { submitting = false }
                    }
                },
                enabled = !submitting && name.isNotBlank() && phone.isNotBlank(),
            ) {
                if (submitting) CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                else Text("Create")
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
