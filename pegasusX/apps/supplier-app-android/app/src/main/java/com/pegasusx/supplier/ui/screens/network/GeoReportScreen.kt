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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GeoReportScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
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
                title = { Text("Geo report") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading geo report…", "H3 coverage")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Geo report unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            lanes.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No coverage",
                body = "No active lanes to report on.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg)) {
                            Text(
                                "Estimated H3 cells in service",
                                style = MaterialTheme.typography.labelMedium,
                                color = MaterialTheme.colorScheme.outline,
                            )
                            Text(
                                lanes.sumOf { it.h3Cells }.toString(),
                                style = MaterialTheme.typography.headlineMedium,
                            )
                        }
                    }
                }
                items(lanes, key = { it.laneId }) { lane ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg)) {
                            Text(lane.name.ifEmpty { lane.warehouseId }, style = MaterialTheme.typography.titleMedium)
                            Text(
                                "%d cells · %.0f%% utilization today".format(lane.h3Cells, lane.utilizationPct),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.outline,
                            )
                        }
                    }
                }
            }
        }
    }
}
