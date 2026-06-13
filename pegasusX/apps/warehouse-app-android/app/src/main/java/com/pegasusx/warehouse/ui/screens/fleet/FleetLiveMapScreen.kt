package com.pegasusx.warehouse.ui.screens.fleet

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back") } } },
                actions = {
                    TextButton(onClick = { scope.launch { load() } }) { Text("Refresh") }
                },
            )
        },
    ) { padding ->
        when {
            loading && routes.isEmpty() -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = androidx.compose.ui.Alignment.Center,
            ) {
                CircularProgressIndicator()
            }
            error != null && routes.isEmpty() -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = androidx.compose.ui.Alignment.Center,
            ) {
                Column(horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Button(onClick = { scope.launch { load() } }) { Text("Retry") }
                }
            }
            routes.isEmpty() -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = androidx.compose.ui.Alignment.Center,
            ) {
                Text(
                    "No active routes",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                item {
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
            Text("${route.manifestState} · ${route.stopCountLabel(pointCount)}", style = MaterialTheme.typography.bodyMedium)
            Text("Manifest ${route.manifestId.take(8)}…", style = MaterialTheme.typography.bodySmall)
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
