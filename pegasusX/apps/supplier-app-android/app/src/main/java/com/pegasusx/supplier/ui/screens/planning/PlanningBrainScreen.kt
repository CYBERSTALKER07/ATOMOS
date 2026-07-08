package com.pegasusx.supplier.ui.screens.planning

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.ShowChart
import androidx.compose.material.icons.filled.Storage
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.PlanningScenarioInput
import com.pegasusx.supplier.data.model.PlanningScenarioResult
import com.pegasusx.supplier.data.model.PlanningSAndOPSnapshot
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierKpiTile
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch
import kotlin.math.roundToInt

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlanningBrainScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var sandop by remember { mutableStateOf<PlanningSAndOPSnapshot?>(null) }
    var scenario by remember { mutableStateOf<PlanningScenarioResult?>(null) }
    var downtimeHours by remember { mutableFloatStateOf(8.0f) }
    var demandDeltaPct by remember { mutableFloatStateOf(10.0f) }
    var loading by remember { mutableStateOf(true) }
    var running by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        loading = true
        error = null
        val resp = ops.getPlanningSAndOP()
        if (resp.isSuccessful) {
            sandop = resp.body()
        } else {
            error = "S&OP unavailable (${resp.code()})"
        }
        loading = false
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Planning sandbox") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading…", body = "", modifier = Modifier.padding(padding))
            error != null && sandop == null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Planning unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    SupplierSectionTitle("S&OP snapshot")
                    Text(
                        "Read-only what-if and lightweight S&OP",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                sandop?.let { snap ->
                    item {
                        Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm), modifier = Modifier.fillMaxWidth()) {
                            SupplierKpiTile("Factory cap", snap.factoryCapacityUnits.toString(), Icons.Default.Storage, Modifier.weight(1f))
                            SupplierKpiTile("WH inbound", snap.warehouseInboundCapUnits.toString(), Icons.Default.LocalShipping, Modifier.weight(1f))
                        }
                    }
                    item {
                        Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm), modifier = Modifier.fillMaxWidth()) {
                            SupplierKpiTile("Utilization", "${snap.utilizationPct.roundToInt()}%", Icons.Default.ShowChart, Modifier.weight(1f))
                            SupplierKpiTile("Alert", if (snap.capacityAlert) "Breach" else "OK", Icons.Default.Warning, Modifier.weight(1f))
                        }
                    }
                }
                item {
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.md), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text("Scenario run", style = MaterialTheme.typography.titleSmall)
                            Text("Factory downtime: ${downtimeHours.roundToInt()}h", style = MaterialTheme.typography.bodySmall)
                            Slider(value = downtimeHours, onValueChange = { downtimeHours = it }, valueRange = 0f..168f, steps = 167)
                            Text("Demand delta: ${demandDeltaPct.roundToInt()}%", style = MaterialTheme.typography.bodySmall)
                            Slider(value = demandDeltaPct, onValueChange = { demandDeltaPct = it }, valueRange = -50f..200f, steps = 50)
                            Button(
                                enabled = !running,
                                onClick = {
                                    scope.launch {
                                        running = true
                                        error = null
                                        val scopeId = SupplierIdempotencyKeys.supplierScopeId()
                                        val key = SupplierIdempotencyKeys.planningScenario(
                                            scopeId,
                                            downtimeHours.roundToInt(),
                                            demandDeltaPct.toDouble(),
                                        )
                                        val resp = ops.runPlanningScenario(
                                            PlanningScenarioInput(
                                                factoryDowntimeHours = downtimeHours.roundToInt(),
                                                demandDeltaPct = demandDeltaPct.toDouble(),
                                                horizonDays = 7,
                                            ),
                                            key,
                                        )
                                        if (resp.isSuccessful) {
                                            scenario = resp.body()
                                        } else {
                                            error = "Scenario failed (${resp.code()})"
                                        }
                                        running = false
                                    }
                                },
                            ) {
                                Text(if (running) "Running…" else "Run scenario")
                            }
                            scenario?.let { result ->
                                Text(
                                    "SLA risk ${result.slaRiskPct.roundToInt()}% · fleet ${result.fleetVolumeOrders} · stockouts ${result.stockoutSkus.size}",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            error?.let {
                                Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }
                }
            }
        }
    }
}
