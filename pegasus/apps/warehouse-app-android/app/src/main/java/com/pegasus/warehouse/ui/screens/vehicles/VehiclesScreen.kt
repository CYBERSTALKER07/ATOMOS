package com.pegasus.warehouse.ui.screens.vehicles

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasus.warehouse.data.model.CreateVehicleRequest
import com.pegasus.warehouse.data.model.Vehicle
import com.pegasus.warehouse.data.remote.WarehouseApi
import com.pegasus.warehouse.ui.theme.LabSpacing
import kotlinx.coroutines.launch

private val VEHICLE_CLASSES = listOf("CLASS_A" to "50 VU", "CLASS_B" to "150 VU", "CLASS_C" to "400 VU")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun VehiclesScreen(
    api: WarehouseApi,
    onBack: () -> Unit,
) {
    var vehicles by remember { mutableStateOf<List<Vehicle>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var showCreate by remember { mutableStateOf(false) }
    var mutatingVehicleId by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load() {
        loading = true; error = null
        scope.launch {
            try {
                val resp = api.getVehicles()
                if (resp.isSuccessful && resp.body() != null) vehicles = resp.body()!!.vehicles
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) { error = e.message ?: "Network error" }
            finally { loading = false }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Vehicles") },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } },
                actions = {
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                    IconButton(onClick = { showCreate = true }) { Icon(Icons.Default.Add, "Add") }
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
            else -> LazyColumn(
                contentPadding = PaddingValues(LabSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(LabSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(vehicles, key = { it.vehicleId }) { v ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(v.label.ifBlank { v.licensePlate }, style = MaterialTheme.typography.titleSmall)
                                Text("${v.vehicleClass} · ${v.capacityVu} VU", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                Text(v.assignedDriverName.ifBlank { "Unassigned" }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            Column(horizontalAlignment = Alignment.End, verticalArrangement = Arrangement.spacedBy(LabSpacing.sm)) {
                                AssistChip(onClick = {}, label = { Text(v.status.ifBlank { "AVAILABLE" }, style = MaterialTheme.typography.labelSmall) })
                                OutlinedButton(
                                    onClick = {
                                        mutatingVehicleId = v.vehicleId
                                        scope.launch {
                                            try {
                                                val resp = api.updateVehicle(v.vehicleId, UpdateVehicleRequest(isActive = !v.isActive))
                                                if (resp.isSuccessful) {
                                                    load()
                                                    snackbarHostState.showSnackbar("Vehicle availability updated")
                                                } else {
                                                    error = "Failed (${resp.code()})"
                                                }
                                            } catch (e: Exception) {
                                                error = e.message ?: "Network error"
                                            } finally {
                                                mutatingVehicleId = null
                                            }
                                        }
                                    },
                                    enabled = mutatingVehicleId != v.vehicleId,
                                ) {
                                    if (mutatingVehicleId == v.vehicleId) {
                                        CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                                    } else {
                                        Text(if (v.isActive) "Unavailable" else "Restore")
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    if (showCreate) {
        CreateVehicleDialog(
            api = api,
            onDismiss = { showCreate = false },
            onCreated = { showCreate = false; load(); scope.launch { snackbarHostState.showSnackbar("Vehicle created") } },
        )
    }
}

@Composable
private fun CreateVehicleDialog(
    api: WarehouseApi,
    onDismiss: () -> Unit,
    onCreated: () -> Unit,
) {
    var label by remember { mutableStateOf("") }
    var plate by remember { mutableStateOf("") }
    var selectedClass by remember { mutableStateOf(VEHICLE_CLASSES[0].first) }
    var submitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add Vehicle") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(LabSpacing.md)) {
                OutlinedTextField(value = label, onValueChange = { label = it }, label = { Text("Label") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(value = plate, onValueChange = { plate = it }, label = { Text("License Plate") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                Text("Vehicle Class", style = MaterialTheme.typography.labelMedium)
                Row(horizontalArrangement = Arrangement.spacedBy(LabSpacing.sm)) {
                    VEHICLE_CLASSES.forEach { (cls, cap) ->
                        FilterChip(
                            selected = selectedClass == cls,
                            onClick = { selectedClass = cls },
                            label = { Text("$cls ($cap)") },
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
                            val resp = api.createVehicle(CreateVehicleRequest(label = label, licensePlate = plate, vehicleClass = selectedClass))
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
