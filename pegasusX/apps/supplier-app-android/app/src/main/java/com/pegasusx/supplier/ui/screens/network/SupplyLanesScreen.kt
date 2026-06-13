package com.pegasusx.supplier.ui.screens.network

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierSupplyLaneRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import kotlin.math.min

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SupplyLanesScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var lanes by remember { mutableStateOf<List<SupplierSupplyLaneRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getSupplyLanes()
                if (resp.isSuccessful) lanes = resp.body()?.lanes ?: emptyList()
                else error = "Failed (${resp.code()})"
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
                title = { Text("Supply lanes") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading supply lanes…", "Warehouse lanes")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Supply lanes unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            lanes.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No lanes",
                body = "No active warehouse lanes. Configure nodes on topology.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(lanes, key = { it.laneId }) { lane ->
                    LaneCard(lane)
                }
            }
        }
    }
}

@Composable
private fun LaneCard(lane: SupplierSupplyLaneRow) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(
            Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text(lane.name.ifEmpty { lane.warehouseId }, style = MaterialTheme.typography.titleMedium)
                Text("${lane.h3Cells} cells", style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.primary)
            }
            LaneMetric("Active drivers", lane.drivers.toString())
            LaneMetric("Orders today", lane.ordersToday.toString())
            LaneMetric("Capacity limit", lane.capacity.toString())
            val tint = when {
                lane.utilizationPct > 85 -> MaterialTheme.colorScheme.error
                lane.utilizationPct > 75 -> MaterialTheme.colorScheme.tertiary
                else -> MaterialTheme.colorScheme.primary
            }
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text("Lane utilization", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
                Text("%.0f%%".format(lane.utilizationPct), style = MaterialTheme.typography.bodySmall, color = tint)
            }
            LinearProgressIndicator(
                progress = { (min(100.0, maxOf(0.0, lane.utilizationPct)) / 100.0).toFloat() },
                modifier = Modifier.fillMaxWidth(),
                color = tint,
            )
        }
    }
}

@Composable
private fun LaneMetric(label: String, value: String) {
    Row(
        Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(label, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.outline)
        Text(value, style = MaterialTheme.typography.bodyMedium)
    }
}
