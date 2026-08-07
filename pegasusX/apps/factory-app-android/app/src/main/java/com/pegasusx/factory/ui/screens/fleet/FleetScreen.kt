package com.pegasusx.factory.ui.screens.fleet

import androidx.compose.ui.res.stringResource

import androidx.compose.ui.unit.dp

import androidx.compose.foundation.lazy.grid.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.DirectionsCar
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.FactoryFleetLiveRoute
import com.pegasusx.factory.data.model.Vehicle
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasusx.factory.ui.components.FactoryOpsListCard
import com.pegasusx.factory.ui.components.FactorySectionTitle
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FleetScreen(
    api: FactoryApi,
    onBack: () -> Unit,
) {
    var vehicles by remember { mutableStateOf<List<Vehicle>>(emptyList()) }
    var liveRoutes by remember { mutableStateOf<List<FactoryFleetLiveRoute>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        if (!silent) {
            loading = true
        }
        error = null
        scope.launch {
            try {
                val resp = api.getFleet()
                if (resp.isSuccessful && resp.body() != null) {
                    vehicles = resp.body()!!.vehicles
                } else {
                    error = "Failed (${resp.code()})"
                }
                val liveResp = api.getFleetLiveMap()
                if (liveResp.isSuccessful) {
                    liveRoutes = liveResp.body()?.routes.orEmpty()
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                if (!silent) {
                    loading = false
                }
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.TransferUpdate,
            FactoryRealtimeEventType.ManifestUpdate,
        ),
    ) {
        load(silent = true)
    }

    val available = vehicles.count { it.status.equals("AVAILABLE", ignoreCase = true) }
    val assigned = vehicles.size - available

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text("Fleet")
                        Text(
                            text = stringResource(R.string.mobile_factory_ui_vehicle_roster_and_assignment_status),
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading && vehicles.isEmpty() -> PegasusLoadingState(
                title = stringResource(R.string.mobile_factory_ui_loading_fleet),
                body = "Fetching the current factory vehicle roster and assignments.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load fleet",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            vehicles.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No vehicles registered",
                body = "There are no vehicles registered in the factory fleet yet.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md)
    ) {
                item {
                    FleetSummaryCard(
                        total = vehicles.size,
                        available = available,
                        assigned = assigned,
                    )
                }
                if (liveRoutes.isNotEmpty()) {
                    item {
                        FactorySectionTitle(title = stringResource(R.string.mobile_factory_ui_live_drivers))
                    }
                    items(liveRoutes, key = { it.manifestId }) { route ->
                        val loc = route.driverLocation
                        val lat = loc?.lat?.takeIf { it != 0.0 } ?: loc?.latitude
                        val lng = loc?.lng?.takeIf { it != 0.0 } ?: loc?.longitude
                        FactoryOpsListCard(
                            headline = route.driverName.ifBlank { route.driverId.ifBlank { route.manifestId } },
                            supporting = buildString {
                                append(route.manifestState)
                                if (lat != null && lng != null) {
                                    append(" · %.5f, %.5f".format(lat, lng))
                                } else {
                                    append(" · waiting for GPS")
                                }
                                if (route.locationStale) append(" · stale")
                            },
                            status = if (route.liveLocationAvailable) "LIVE" else "OFFLINE",
                        )
                    }
                }
                item {
                    FactorySectionTitle(title = stringResource(R.string.supplier_portal_org_fleet_components_vehicle_table_text_vehicle_roster))
                }
                items(vehicles, key = { it.id }) { vehicle ->
                    FactoryOpsListCard(
                        headline = vehicle.plateNumber,
                        supporting = buildString {
                            append(vehicle.driverName.ifBlank { "Unassigned" })
                            append(" · ${vehicle.capacityKg.toInt()}kg · ${vehicle.capacityL.toInt()}L")
                        },
                        status = vehicle.status,
                    )
                }
            }
        }
    }
}

@Composable
private fun FleetSummaryCard(
    total: Int,
    available: Int,
    assigned: Int,
) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                text = stringResource(R.string.mobile_factory_ui_fleet_capacity),
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = stringResource(R.string.mobile_factory_ui_track_available_vehicles_and_current_assignments_for_outbound_di),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FactoryMetricTile("Total", total.toString(), Modifier.weight(1f))
                FactoryMetricTile("Available", available.toString(), Modifier.weight(1f))
                FactoryMetricTile("Assigned", assigned.toString(), Modifier.weight(1f))
            }
        }
    }
}
