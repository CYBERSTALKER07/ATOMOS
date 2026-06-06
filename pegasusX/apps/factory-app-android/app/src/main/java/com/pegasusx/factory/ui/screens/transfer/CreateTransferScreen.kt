package com.pegasusx.factory.ui.screens.transfer

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExposedDropdownMenu
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import com.pegasusx.factory.data.model.CreateTransferRequest
import com.pegasusx.factory.data.model.FleetDriverRow
import com.pegasusx.factory.data.model.FleetVehicleRow
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.ui.components.FactoryLoadingState
import com.pegasusx.factory.ui.components.FactoryStateKind
import com.pegasusx.factory.ui.components.FactoryStatePane
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreateTransferScreen(
    api: FactoryApi,
    onBack: () -> Unit,
    onCreated: (String) -> Unit,
) {
    var loadingFleet by remember { mutableStateOf(true) }
    var submitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var drivers by remember { mutableStateOf<List<FleetDriverRow>>(emptyList()) }
    var vehicles by remember { mutableStateOf<List<FleetVehicleRow>>(emptyList()) }
    var orderId by remember { mutableStateOf("") }
    var totalVu by remember { mutableStateOf("25") }
    var driverId by remember { mutableStateOf<String?>(null) }
    var vehicleId by remember { mutableStateOf<String?>(null) }
    var driverMenuExpanded by remember { mutableStateOf(false) }
    var vehicleMenuExpanded by remember { mutableStateOf(false) }
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    fun loadFleet() {
        loadingFleet = true
        error = null
        scope.launch {
            try {
                val driversResp = api.getFleetDrivers()
                val vehiclesResp = api.getFleetVehicles()
                if (!driversResp.isSuccessful || !vehiclesResp.isSuccessful) {
                    error = "Failed to load fleet (${driversResp.code()}/${vehiclesResp.code()})"
                } else {
                    drivers = driversResp.body()?.drivers.orEmpty()
                    vehicles = vehiclesResp.body()?.vehicles.orEmpty()
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loadingFleet = false
            }
        }
    }

    LaunchedEffect(Unit) { loadFleet() }

    fun submit() {
        val parsedVu = totalVu.toLongOrNull()
        if (parsedVu == null || parsedVu <= 0) {
            scope.launch { snackbarHostState.showSnackbar("Total VU must be a positive number") }
            return
        }
        submitting = true
        scope.launch {
            try {
                val resp = api.createTransfer(
                    CreateTransferRequest(
                        orderId = orderId.trim().ifBlank { null },
                        totalVu = parsedVu,
                        driverId = driverId,
                        vehicleId = vehicleId,
                    ),
                )
                if (resp.isSuccessful && resp.body() != null) {
                    onCreated(resp.body()!!.transferId)
                } else {
                    snackbarHostState.showSnackbar("Create failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Create failed")
            } finally {
                submitting = false
            }
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
        topBar = {
            TopAppBar(
                title = { Text("Create Transfer") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                    }
                },
            )
        },
    ) { innerPadding ->
        when {
            loadingFleet -> FactoryLoadingState(
                title = "Preparing form",
                body = "Loading fleet assignment options.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> FactoryStatePane(
                kind = FactoryStateKind.Error,
                headline = "Unable to load fleet",
                body = error!!,
                actionLabel = "Retry",
                onAction = { loadFleet() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding)
                    .verticalScroll(rememberScrollState())
                    .padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Text(
                    text = "Stage a factory-to-warehouse movement. Volume is captured in VU.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                OutlinedTextField(
                    value = orderId,
                    onValueChange = { orderId = it },
                    label = { Text("Order ID (optional)") },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
                OutlinedTextField(
                    value = totalVu,
                    onValueChange = { totalVu = it },
                    label = { Text("Total VU") },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
                ExposedDropdownMenuBox(
                    expanded = driverMenuExpanded,
                    onExpandedChange = { driverMenuExpanded = it },
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    OutlinedTextField(
                        value = drivers.firstOrNull { it.driverId == driverId }?.name ?: "Unassigned",
                        onValueChange = {},
                        readOnly = true,
                        label = { Text("Driver") },
                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = driverMenuExpanded) },
                        modifier = Modifier
                            .menuAnchor()
                            .fillMaxWidth(),
                    )
                    ExposedDropdownMenu(
                        expanded = driverMenuExpanded,
                        onDismissRequest = { driverMenuExpanded = false },
                    ) {
                        DropdownMenuItem(
                            text = { Text("Unassigned") },
                            onClick = {
                                driverId = null
                                driverMenuExpanded = false
                            },
                        )
                        drivers.forEach { driver ->
                            DropdownMenuItem(
                                text = { Text("${driver.name}${if (driver.onShift) " (on shift)" else ""}") },
                                onClick = {
                                    driverId = driver.driverId
                                    driverMenuExpanded = false
                                },
                            )
                        }
                    }
                }
                ExposedDropdownMenuBox(
                    expanded = vehicleMenuExpanded,
                    onExpandedChange = { vehicleMenuExpanded = it },
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    OutlinedTextField(
                        value = vehicles.firstOrNull { it.vehicleId == vehicleId }?.let { "${it.plateNo} · ${it.state}" }
                            ?: "Unassigned",
                        onValueChange = {},
                        readOnly = true,
                        label = { Text("Vehicle") },
                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = vehicleMenuExpanded) },
                        modifier = Modifier
                            .menuAnchor()
                            .fillMaxWidth(),
                    )
                    ExposedDropdownMenu(
                        expanded = vehicleMenuExpanded,
                        onDismissRequest = { vehicleMenuExpanded = false },
                    ) {
                        DropdownMenuItem(
                            text = { Text("Unassigned") },
                            onClick = {
                                vehicleId = null
                                vehicleMenuExpanded = false
                            },
                        )
                        vehicles.forEach { vehicle ->
                            DropdownMenuItem(
                                text = { Text("${vehicle.plateNo} · ${vehicle.state}") },
                                onClick = {
                                    vehicleId = vehicle.vehicleId
                                    vehicleMenuExpanded = false
                                },
                            )
                        }
                    }
                }
                Button(
                    onClick = { submit() },
                    enabled = !submitting,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(if (submitting) "Creating…" else "Create transfer")
                }
            }
        }
    }
}
