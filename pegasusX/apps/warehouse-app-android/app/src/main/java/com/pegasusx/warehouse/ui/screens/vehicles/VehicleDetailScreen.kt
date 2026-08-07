package com.pegasusx.warehouse.ui.screens.vehicles

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
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
import com.pegasusx.warehouse.data.model.UpdateVehicleRequest
import com.pegasusx.warehouse.data.model.Vehicle
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.ui.components.WarehouseStatusChip
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun VehicleDetailScreen(
    api: WarehouseApi,
    vehicleId: String,
    realtimeSignals: WarehouseRealtimeSignals,
    onBack: (() -> Unit)? = null,
) {
    var vehicle by remember { mutableStateOf<Vehicle?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var mutating by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load(silent: Boolean = false) {
        if (!silent) loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getVehicle(vehicleId)
                if (resp.isSuccessful && resp.body()?.vehicle != null) {
                    vehicle = resp.body()!!.vehicle
                } else if (!silent) {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                if (!silent) error = e.message ?: "Network error"
            } finally {
                if (!silent) loading = false
            }
        }
    }

    fun updateAvailability(isActive: Boolean, reason: String? = null, note: String? = null) {
        val current = vehicle ?: return
        mutating = true
        scope.launch {
            try {
                val resp = api.updateVehicle(
                    current.vehicleId,
                    UpdateVehicleRequest(
                        isActive = isActive,
                        unavailableReason = if (isActive) null else reason,
                        unavailableNote = if (isActive) null else note,
                    ),
                    WarehouseIdempotencyKeys.updateVehicle(current.vehicleId, isActive, reason),
                )
                if (resp.isSuccessful) {
                    load()
                    snackbarHostState.showSnackbar(if (isActive) "Truck restored" else "Truck marked unavailable")
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                mutating = false
            }
        }
    }

    LaunchedEffect(vehicleId) { load() }

    LaunchedEffect(Unit) {
        realtimeSignals.refreshTick.collect { load(silent = true) }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(vehicle?.label?.ifBlank { vehicle?.licensePlate } ?: "Truck") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                        }
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = stringResource(R.string.portal_page_orders_action_refresh))
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        when {
            loading && vehicle == null -> PegasusLoadingState(
                title = stringResource(R.string.mobile_warehouse_ui_loading_truck),
                body = "Fleet vehicle details",
                modifier = Modifier.padding(innerPadding),
            )
            error != null && vehicle == null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Truck unavailable",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.padding(innerPadding),
            )
            vehicle != null -> {
                val v = vehicle!!
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(innerPadding)
                        .verticalScroll(rememberScrollState())
                        .padding(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text(v.label.ifBlank { v.licensePlate }, style = MaterialTheme.typography.headlineSmall)
                        Text(
                            stringResource(R.string.mobile_warehouse_ui_licenseplate_vehicleclass_capacityvu_vu, v.licensePlate, v.vehicleClass, v.capacityVu),
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        WarehouseStatusChip(
                            status = if (v.isActive) v.status.ifBlank { "ACTIVE" } else "UNAVAILABLE",
                        )
                    }

                    WarehouseSectionTitle("Assignment")
                    Text(
                        v.assignedDriverName.ifBlank { "Unassigned" },
                        style = MaterialTheme.typography.bodyMedium,
                    )

                    WarehouseSectionTitle("Dispatch impact")
                    Text(
                        if (v.isActive) {
                            "This truck is eligible for manual and smart dispatch."
                        } else {
                            "Excluded from dispatch: ${formatUnavailableReason(v.unavailableReason, v.unavailableNote)}"
                        },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.fillMaxWidth(),
                    )

                    VehicleAvailabilityPanel(
                        vehicle = v,
                        mutating = mutating,
                        onMarkUnavailable = { reason, note -> updateAvailability(false, reason, note) },
                        onRestore = { updateAvailability(true) },
                    )
                }
            }
        }
    }
}
