package com.pegasusx.warehouse.ui.screens.fleet

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.WarehouseFleetLiveRoute
import com.pegasusx.warehouse.data.remote.TokenHolder
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.components.FleetLiveMapLibre
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import com.pegasusx.warehouse.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FleetLiveMapScreen(
    ops: WarehouseOperationsRepository,
    realtimeSignals: WarehouseRealtimeSignals,
    onBack: (() -> Unit)? = null,
) {
    var routes by remember { mutableStateOf<List<WarehouseFleetLiveRoute>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    suspend fun load(silent: Boolean = false) {
        if (!silent) loading = true
        error = null
        try {
            val resp = ops.getFleetLiveMap(TokenHolder.warehouseId)
            routes = if (resp.isSuccessful) resp.body()?.routes.orEmpty() else emptyList()
            if (!resp.isSuccessful && !silent) error = "Failed (${resp.code()})"
        } catch (e: Exception) {
            if (!silent) error = e.message
        } finally {
            if (!silent) loading = false
        }
    }

    LaunchedEffect(Unit) {
        load()
        while (isActive) {
            delay(15_000)
            load(silent = true)
        }
    }

    LaunchedEffect(realtimeSignals) {
        realtimeSignals.refreshTick.collect {
            load(silent = true)
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Live fleet") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back)) } } },
                actions = {
                    TextButton(onClick = { scope.launch { load() } }) { Text("Refresh") }
                },
            )
        },
    ) { padding ->
        when {
            loading && routes.isEmpty() -> com.pegasus.design.PegasusLoadingState(
                title = stringResource(R.string.mobile_warehouse_ui_loading_fleet_map),
                body = "Fetching live locations and routes",
                modifier = Modifier.padding(padding),
            )
            error != null && routes.isEmpty() -> com.pegasus.design.PegasusStatePane(
                kind = com.pegasus.design.PegasusStateKind.Error,
                headline = "Failed to load map",
                body = error!!,
                actionLabel = "Retry",
                onAction = { scope.launch { load() } },
                modifier = Modifier.padding(padding),
            )
            routes.isEmpty() -> com.pegasus.design.PegasusStatePane(
                kind = com.pegasus.design.PegasusStateKind.Empty,
                headline = "No active routes",
                body = "There are no fleet routes currently active.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item(span = { GridItemSpan(maxLineSpan) }) {
                    FleetLiveMapLibre(
                        routes = routes,
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(360.dp),
                    )
                    Spacer(Modifier.height(PegasusSpacing.sm))
                }
                items(routes, key = { it.manifestId }) { route ->
                    FleetLiveRouteCard(route)
                }
            }
        }
    }
}

@Composable
private fun FleetLiveRouteCard(route: WarehouseFleetLiveRoute) {
    val pointCount = route.routeGeometry?.coordinates?.size ?: 0
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
            Text(route.driverName ?: route.driverId, style = MaterialTheme.typography.titleMedium)
            Text(stringResource(R.string.mobile_warehouse_ui_manifeststate_stopcountlabel, route.manifestState, route.stopCountLabel(pointCount)), style = MaterialTheme.typography.bodyMedium)
            Text(stringResource(R.string.mobile_warehouse_ui_manifest_take, route.manifestId.take(8)), style = MaterialTheme.typography.bodySmall)
            if (route.liveLocationAvailable && route.driverLocation != null) {
                val stale = route.locationStale == true
                val location = route.driverLocation
                val lat = if (location.latitude != 0.0) location.latitude else location.lat
                val lng = if (location.longitude != 0.0) location.longitude else location.lng
                Text(
                    if (stale) "GPS stale · $lat, $lng" else "GPS live · $lat, $lng",
                    style = MaterialTheme.typography.labelMedium,
                    color = if (stale) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary,
                )
            } else {
                Text("No live GPS", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

private fun WarehouseFleetLiveRoute.stopCountLabel(pointCount: Int): String {
    val stops = routeGeometry?.stopCount
    return when {
        stops != null && stops > 0 -> "$stops stops · $pointCount geometry points"
        pointCount > 0 -> "$pointCount geometry points"
        else -> "geometry pending"
    }
}
