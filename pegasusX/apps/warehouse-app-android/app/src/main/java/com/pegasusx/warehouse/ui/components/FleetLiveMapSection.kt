package com.pegasusx.warehouse.ui.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.WarehouseFleetLiveRoute
import com.pegasusx.warehouse.data.remote.TokenHolder
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

@Composable
fun FleetLiveMapSection(
    ops: WarehouseOperationsRepository,
    realtimeSignals: WarehouseRealtimeSignals,
    modifier: Modifier = Modifier,
    mapHeight: androidx.compose.ui.unit.Dp = 320.dp,
    onOpenFullMap: (() -> Unit)? = null,
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

    ElevatedCard(modifier = modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column {
                    Text("Live fleet", style = MaterialTheme.typography.titleMedium)
                    Text(stringResource(R.string.mobile_warehouse_ui_size_active_routeif_else_s, routes.size, if (routes.size == 1) "" else "s"),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                    TextButton(onClick = { scope.launch { load() } }) { Text("Refresh") }
                    if (onOpenFullMap != null) {
                        TextButton(onClick = onOpenFullMap) { Text("Expand") }
                    }
                }
            }

            when {
                loading && routes.isEmpty() -> Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(mapHeight),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator()
                }
                error != null && routes.isEmpty() -> Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(mapHeight),
                    contentAlignment = Alignment.Center,
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(error!!, color = MaterialTheme.colorScheme.error)
                        TextButton(onClick = { scope.launch { load() } }) { Text("Retry") }
                    }
                }
                routes.isEmpty() -> Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(mapHeight),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        "No active routes",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                else -> FleetLiveMapLibre(
                    routes = routes,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(mapHeight),
                )
            }
        }
    }
}
