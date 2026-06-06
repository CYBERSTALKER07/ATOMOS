package com.pegasusx.supplier.ui.screens.fleet

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.FleetDriver
import com.pegasusx.supplier.data.model.FleetVehicle
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FleetScreen(api: SupplierApi, ops: SupplierOperationsRepository) {
    var tab by remember { mutableIntStateOf(0) }
    var drivers by remember { mutableStateOf<List<FleetDriver>>(emptyList()) }
    var vehicles by remember { mutableStateOf<List<FleetVehicle>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val d = api.getFleetDrivers()
                val v = api.getFleetVehicles()
                drivers = if (d.isSuccessful) d.body()?.items.orEmpty() else emptyList()
                vehicles = if (v.isSuccessful) v.body()?.items.orEmpty() else emptyList()
                runCatching { ops.getFleetOrders() }
                if (!d.isSuccessful || !v.isSuccessful) {
                    error = "Failed to load fleet"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Fleet") },
                actions = {
                    TextButton(onClick = { load() }) { Text("Refresh") }
                },
            )
        },
    ) { padding ->
        Column(Modifier.padding(padding).fillMaxSize()) {
            TabRow(selectedTabIndex = tab) {
                Tab(selected = tab == 0, onClick = { tab = 0 }, text = { Text("Drivers") })
                Tab(selected = tab == 1, onClick = { tab = 1 }, text = { Text("Vehicles") })
            }
            when {
                loading -> SupplierLoadingState("Loading fleet…", "Drivers and vehicles")
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
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        items(drivers, key = { it.driverId }) { d ->
            ElevatedCard(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(PegasusSpacing.lg)) {
                    Text(d.name, style = MaterialTheme.typography.titleMedium)
                    Text(d.phone, style = MaterialTheme.typography.bodySmall)
                    Text("${d.homeNodeType} ${d.homeNodeId}", style = MaterialTheme.typography.labelSmall)
                }
            }
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
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        items(vehicles, key = { it.vehicleId }) { v ->
            ElevatedCard(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(PegasusSpacing.lg)) {
                    Text(v.licensePlate, style = MaterialTheme.typography.titleMedium)
                    Text(v.label ?: v.vehicleId, style = MaterialTheme.typography.bodySmall)
                }
            }
        }
    }
}
