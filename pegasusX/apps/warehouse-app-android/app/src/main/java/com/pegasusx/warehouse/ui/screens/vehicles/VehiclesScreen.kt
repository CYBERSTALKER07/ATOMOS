package com.pegasusx.warehouse.ui.screens.vehicles

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.CreateVehicleRequest
import com.pegasusx.warehouse.data.model.Vehicle
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.realtime.WAREHOUSE_RECONNECT_RECOVERY_HINT
import com.pegasusx.warehouse.ui.realtime.WarehouseReconnectRecoveryEffect
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasus.design.showFullScreenLoading
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.launch
import com.pegasusx.warehouse.R

private val VEHICLE_CLASSES = listOf("CLASS_A" to "50 VU", "CLASS_B" to "150 VU", "CLASS_C" to "400 VU")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun VehiclesScreen(
    api: WarehouseApi,
    realtimeSignals: WarehouseRealtimeSignals,
    onVehicleClick: (String) -> Unit = {},
    onBack: (() -> Unit)? = null,
) {
    var vehicles by remember { mutableStateOf<List<Vehicle>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var showCreate by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load(silent: Boolean = false) {
        if (!silent) loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getVehicles()
                if (resp.isSuccessful && resp.body() != null) vehicles = resp.body()!!.vehicles
                else if (!silent) error = "Failed (${resp.code()})"
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
                title = { Text("Trucks") },
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
            loading && vehicles.isEmpty() -> PegasusLoadingState(
                title = stringResource(R.string.mobile_warehouse_ui_loading_trucks),
                body = "Fleet vehicle roster",
                modifier = Modifier.padding(innerPadding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Trucks unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.padding(innerPadding),
            )
            vehicles.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No trucks",
                body = "Fleet trucks will appear here.",
                modifier = Modifier.padding(innerPadding),
            )
            else -> VehiclesList(
                vehicles = vehicles,
                onVehicleClick = onVehicleClick,
                modifier = Modifier.padding(innerPadding)
            )
        }
    }

    if (showCreate) {
        CreateVehicleDialog(
            api = api,
            realtimeSignals = realtimeSignals,
            onDismiss = { showCreate = false },
            onCreated = {
                showCreate = false
                load()
                scope.launch { snackbarHostState.showSnackbar("Truck created") }
            },
        )
    }
}

@Composable
private fun CreateVehicleDialog(
    api: WarehouseApi,
    realtimeSignals: WarehouseRealtimeSignals,
    onDismiss: () -> Unit,
    onCreated: () -> Unit,
) {
    var label by remember { mutableStateOf("") }
    var plate by remember { mutableStateOf("") }
    var selectedClass by remember { mutableStateOf(VEHICLE_CLASSES[0].first) }
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
        title = { Text("Add Truck") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
                OutlinedTextField(value = label, onValueChange = { label = it }, label = { Text("Label") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(value = plate, onValueChange = { plate = it }, label = { Text("License Plate") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                Text("Vehicle Class", style = MaterialTheme.typography.labelMedium)
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    VEHICLE_CLASSES.forEach { (cls, cap) ->
                        FilterChip(
                            selected = selectedClass == cls,
                            onClick = { selectedClass = cls },
                            label = { Text(stringResource(R.string.mobile_warehouse_ui_cls_cap, cls, cap)) },
                        )
                    }
                }
                if (error != null) Text(error!!, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    submitting = true; error = null
                    scope.launch {
                        try {
                            val resp = api.createVehicle(
                                CreateVehicleRequest(label = label, licensePlate = plate, vehicleClass = selectedClass),
                                WarehouseIdempotencyKeys.createVehicle(plate),
                            )
                            if (resp.isSuccessful) onCreated()
                            else error = "Failed (${resp.code()})"
                        } catch (e: Exception) { error = e.message ?: "Error" }
                        finally { submitting = false }
                    }
                },
                enabled = !submitting && label.isNotBlank() && plate.isNotBlank(),
            ) {
                if (submitting) CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                else Text("Create")
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
