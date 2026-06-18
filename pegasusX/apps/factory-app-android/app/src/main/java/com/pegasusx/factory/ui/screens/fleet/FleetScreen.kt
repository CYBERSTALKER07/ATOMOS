package com.pegasusx.factory.ui.screens.fleet

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.DirectionsCar
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.Vehicle
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasusx.factory.ui.components.FactoryLoadingState
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasusx.factory.ui.components.FactoryOpsListCard
import com.pegasusx.factory.ui.components.FactorySectionTitle
import com.pegasusx.factory.ui.components.FactoryStateKind
import com.pegasusx.factory.ui.components.FactoryStatePane
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
                            text = "Vehicle roster and assignment status",
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
            loading && vehicles.isEmpty() -> FactoryLoadingState(
                title = "Loading fleet",
                body = "Fetching the current factory vehicle roster and assignments.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> FactoryStatePane(
                kind = FactoryStateKind.Error,
                headline = "Unable to load fleet",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            vehicles.isEmpty() -> FactoryStatePane(
                kind = FactoryStateKind.Empty,
                headline = "No vehicles registered",
                body = "There are no vehicles registered in the factory fleet yet.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                item {
                    FleetSummaryCard(
                        total = vehicles.size,
                        available = available,
                        assigned = assigned,
                    )
                }
                item {
                    FactorySectionTitle(title = "Vehicle roster")
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
                text = "Fleet capacity",
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = "Track available vehicles and current assignments for outbound dispatch.",
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
