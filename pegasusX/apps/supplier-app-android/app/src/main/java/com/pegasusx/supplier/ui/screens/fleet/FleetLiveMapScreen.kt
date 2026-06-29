package com.pegasusx.supplier.ui.screens.fleet

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.SupplierFleetLiveRoute
import com.pegasusx.supplier.data.model.ExceptionMapCell
import com.pegasusx.supplier.ui.components.FleetLiveMapLibre
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FleetLiveMapScreen(
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
    onBack: () -> Unit,
) {
    var routes by remember { mutableStateOf<List<SupplierFleetLiveRoute>>(emptyList()) }
    var exceptionCells by remember { mutableStateOf<List<ExceptionMapCell>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    suspend fun load(silent: Boolean = false) {
        if (!silent) loading = true
        error = null
        try {
            val resp = ops.getFleetLiveMap()
            routes = if (resp.isSuccessful) resp.body()?.routes.orEmpty() else emptyList()
            val exc = ops.getExceptionMap()
            exceptionCells = if (exc.isSuccessful) exc.body()?.cells.orEmpty() else emptyList()
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
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    TextButton(onClick = { scope.launch { load() } }) { Text("Refresh") }
                },
            )
        },
    ) { padding ->
        when {
            loading && routes.isEmpty() -> SupplierLoadingState(
                "Loading live fleet…",
                "Sealed manifest routes",
                modifier = Modifier.padding(padding),
            )
            error != null && routes.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Live fleet unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { scope.launch { load() } },
            )
            routes.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No active routes",
                body = "Sealed manifests with route geometry appear here during dispatch.",
                modifier = Modifier.padding(padding),
            )
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
                            .height(320.dp),
                    )
                    Spacer(Modifier.height(PegasusSpacing.sm))
                }
                if (exceptionCells.isNotEmpty()) {
                    item {
                        Text("Exception weather (24h)", style = MaterialTheme.typography.titleMedium)
                        Spacer(Modifier.height(PegasusSpacing.xs))
                    }
                    items(exceptionCells, key = { it.h3Cell }) { cell ->
                        ElevatedCard(Modifier.fillMaxWidth()) {
                            Column(Modifier.padding(PegasusSpacing.md)) {
                                Text("${cell.severity.uppercase()} · total ${cell.counts["total"] ?: 0}")
                                Text(
                                    "shop-closed ${cell.counts["shop_closed"] ?: 0} · delayed ${cell.counts["delayed"] ?: 0}",
                                    style = MaterialTheme.typography.bodySmall,
                                )
                            }
                        }
                    }
                    item { Spacer(Modifier.height(PegasusSpacing.sm)) }
                }
                items(routes, key = { it.manifestId }) { route ->
                    FleetLiveRouteCard(route)
                }
            }
        }
    }
}

@Composable
private fun FleetLiveRouteCard(route: SupplierFleetLiveRoute) {
    val pointCount = route.routeGeometry?.coordinates?.size ?: 0
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
            Text(route.driverName ?: route.driverId, style = MaterialTheme.typography.titleMedium)
            Text("${route.manifestState} · ${route.stopCountLabel(pointCount)}", style = MaterialTheme.typography.bodyMedium)
            Text("Manifest ${route.manifestId.take(8)}…", style = MaterialTheme.typography.bodySmall)
            if (route.liveLocationAvailable && route.driverLocation != null) {
                val stale = route.locationStale == true
                val location = route.driverLocation
                Text(
                    if (stale) "GPS stale · ${location.latitude}, ${location.longitude}"
                    else "GPS live · ${location.latitude}, ${location.longitude}",
                    style = MaterialTheme.typography.labelMedium,
                    color = if (stale) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary,
                )
            } else {
                Text("No live GPS", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

private fun SupplierFleetLiveRoute.stopCountLabel(pointCount: Int): String {
    val stops = routeGeometry?.stopCount
    return when {
        stops != null && stops > 0 -> "$stops stops · $pointCount geometry points"
        pointCount > 0 -> "$pointCount geometry points"
        else -> "geometry pending"
    }
}
