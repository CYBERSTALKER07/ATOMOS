package com.pegasusx.supplier.ui.screens.fleet

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Map
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasus.design.RealtimeRefreshEffect
import com.pegasus.design.showFullScreenLoading
import com.pegasusx.supplier.data.model.FleetDriver
import com.pegasusx.supplier.data.model.FleetVehicle
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FleetScreen(
    api: SupplierApi,
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
    onOpenLiveMap: () -> Unit = {},
) {
    var tab by remember { mutableIntStateOf(0) }
    var drivers by remember { mutableStateOf<List<FleetDriver>>(emptyList()) }
    var vehicles by remember { mutableStateOf<List<FleetVehicle>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val hasData = drivers.isNotEmpty() || vehicles.isNotEmpty()

    fun load(silent: Boolean = false) {
        scope.launch {
            if (!silent) {
                loading = true
                error = null
            }
            try {
                val d = api.getFleetDrivers()
                val v = api.getFleetVehicles()
                drivers = if (d.isSuccessful) d.body()?.items.orEmpty() else drivers
                vehicles = if (v.isSuccessful) v.body()?.items.orEmpty() else vehicles
                runCatching { ops.getFleetOrders() }
                if (!d.isSuccessful || !v.isSuccessful) {
                    if (!silent) error = "Failed to load fleet"
                }
            } catch (e: Exception) {
                if (!silent) error = e.message
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    RealtimeRefreshEffect(
        refreshTick = realtimeSignals.refreshTick,
        reconnectTick = realtimeSignals.reconnectTick,
        onRefresh = { load(silent = it) },
    )

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Fleet") },
                actions = {
                    IconButton(onClick = onOpenLiveMap) {
                        Icon(Icons.Default.Map, contentDescription = "Live map")
                    }
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
    ) { padding ->
        Column(Modifier.padding(padding).fillMaxSize()) {
            TabRow(selectedTabIndex = tab) {
                Tab(selected = tab == 0, onClick = { tab = 0 }, text = { Text("Drivers (${drivers.size})") })
                Tab(selected = tab == 1, onClick = { tab = 1 }, text = { Text("Vehicles (${vehicles.size})") })
            }
            when {
                showFullScreenLoading(loading, hasData) -> SupplierLoadingState("Loading fleet…", "Drivers and vehicles")
                error != null -> SupplierStatePane(
                    kind = SupplierStateKind.Error,
                    headline = "Fleet unavailable",
                    body = error!!,
                    actionLabel = "Retry",
                    onAction = { load() },
                )
                tab == 0 -> FleetDriversList(drivers)
                else -> FleetVehiclesList(vehicles)
            }
        }
    }
}

@Composable
private fun FleetDriversList(drivers: List<FleetDriver>) {
    if (drivers.isEmpty()) {
        SupplierStatePane(SupplierStateKind.Empty, headline = "No drivers", body = "Fleet drivers will appear here.")
        return
    }
    LazyColumn(
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(drivers, key = { it.driverId }) { driver ->
            SupplierOpsListCard(
                headline = driver.name.ifBlank { driver.driverId.take(8) },
                supporting = buildString {
                    if (driver.phone.isNotBlank()) append(driver.phone)
                    if (driver.homeNodeType.isNotBlank() || driver.homeNodeId.isNotBlank()) {
                        if (isNotEmpty()) append(" · ")
                        append("${driver.homeNodeType} ${driver.homeNodeId}".trim())
                    }
                },
                status = if (driver.isActive) "ACTIVE" else "INACTIVE",
            )
        }
    }
}

@Composable
private fun FleetVehiclesList(vehicles: List<FleetVehicle>) {
    if (vehicles.isEmpty()) {
        SupplierStatePane(SupplierStateKind.Empty, headline = "No vehicles", body = "Fleet vehicles will appear here.")
        return
    }
    LazyColumn(
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(vehicles, key = { it.vehicleId }) { vehicle ->
            SupplierOpsListCard(
                headline = vehicle.licensePlate.ifBlank { vehicle.vehicleId.take(8) },
                supporting = buildString {
                    vehicle.label?.takeIf { it.isNotBlank() }?.let { append(it) }
                    if (vehicle.homeNodeType.isNotBlank() || vehicle.homeNodeId.isNotBlank()) {
                        if (isNotEmpty()) append(" · ")
                        append("${vehicle.homeNodeType} ${vehicle.homeNodeId}".trim())
                    }
                },
                status = if (vehicle.isActive) "ACTIVE" else "INACTIVE",
            )
        }
    }
}
