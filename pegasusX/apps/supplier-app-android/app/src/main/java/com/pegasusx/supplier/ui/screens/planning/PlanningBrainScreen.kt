package com.pegasusx.supplier.ui.screens.planning

import androidx.compose.ui.res.stringResource

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
import com.pegasusx.supplier.data.model.ForecastConfidence
import com.pegasusx.supplier.data.model.PlanningScenarioInput
import com.pegasusx.supplier.data.model.PlanningScenarioResult
import com.pegasusx.supplier.data.model.PlanningSAndOPSnapshot
import com.pegasusx.supplier.data.model.SparsityGateResult
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.util.brainForecastLine
import com.pegasusx.supplier.util.factoryPlanningDisabledCode
import com.pegasusx.supplier.util.forecastConfidenceFromDemand
import com.pegasusx.supplier.util.planBrainTabFromQuery
import com.pegasusx.supplier.ui.components.SupplierKpiTile
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch
import kotlin.math.roundToInt
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlanningBrainScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var tab by remember { mutableStateOf(planBrainTabFromQuery(null)) }
    var sandop by remember { mutableStateOf<PlanningSAndOPSnapshot?>(null) }
    var scenario by remember { mutableStateOf<PlanningScenarioResult?>(null) }
    var downtimeHours by remember { mutableFloatStateOf(8.0f) }
    var demandDeltaPct by remember { mutableFloatStateOf(10.0f) }
    var loading by remember { mutableStateOf(true) }
    var running by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var demandConfidence by remember { mutableStateOf<ForecastConfidence?>(null) }
    var retailerId by remember { mutableStateOf("") }
    var sparsity by remember { mutableStateOf<SparsityGateResult?>(null) }
    var pushStatus by remember { mutableStateOf<String?>(null) }
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
        ops.getDemandToday().body()?.let { demandConfidence = forecastConfidenceFromDemand(it) }
        loading = false
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Plan & Brain") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading…", body = "", modifier = Modifier.padding(padding))
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    TabRow(selectedTabIndex = if (tab == "brain") 1 else 0) {
                        Tab(selected = tab == "planning", onClick = { tab = "planning" }, text = { Text("Planning") })
                        Tab(selected = tab == "brain", onClick = { tab = "brain" }, text = { Text("Digital Brain") })
                    }
                }
                if (tab == "brain") {
                    item {
                        Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            demandConfidence?.let { ForecastConfidenceView(it) }
                            if (brainForecastLine(demandConfidence, emptyList()) == null) {
                                Text("No forecast line", style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }
                    item {
                        Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            OutlinedTextField(value = retailerId, onValueChange = { retailerId = it }, label = { Text("Retailer id") })
                            Button(onClick = {
                                scope.launch {
                                    sparsity = ops.getPlanningSparsity(retailerId.trim()).body()
                                }
                            }) { Text("Check sparsity") }
                            sparsity?.let {
                                Text(
                                    if (it.allowed) "allowed · ${it.label}" else "blocked · ${it.blockedReason ?: it.label}",
                                    style = MaterialTheme.typography.bodySmall,
                                )
                            }
                        }
                    }
                    item {
                        Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Button(onClick = {
                                scope.launch {
                                    val resp = ops.postPlanningPredictivePush(
                                        SupplierIdempotencyKeys.planningPredictivePush(SupplierIdempotencyKeys.supplierScopeId()),
                                    )
                                    pushStatus = factoryPlanningDisabledCode(resp.code(), resp.errorBody()?.string().orEmpty())
                                        ?: if (resp.isSuccessful) "preview ${resp.body()?.transfers ?: 0} transfers" else "push_failed"
                                }
                            }) { Text("Preview push") }
                            pushStatus?.let { Text(it, style = MaterialTheme.typography.bodySmall) }
                        }
                    }
                } else {
                if (error != null && sandop == null) {
                    item {
                        PegasusStatePane(
                            kind = PegasusStateKind.Error,
                            headline = "Planning unavailable",
                            body = error!!,
                        )
                    }
                }
                item {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        SupplierSectionTitle("S&OP snapshot")
                        Text(
                            "Read-only what-if and lightweight S&OP",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
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
                            Text(stringResource(R.string.mobile_supplier_ui_factory_downtime_roundtointh, downtimeHours.roundToInt()), style = MaterialTheme.typography.bodySmall)
                            Slider(value = downtimeHours, onValueChange = { downtimeHours = it }, valueRange = 0f..168f, steps = 167)
                            Text(stringResource(R.string.mobile_supplier_ui_demand_delta_roundtoint, demandDeltaPct.roundToInt()), style = MaterialTheme.typography.bodySmall)
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
                                    stringResource(R.string.mobile_supplier_ui_sla_risk_roundtoint_fleet_fleetvolumeorders_stockouts_size, result.slaRiskPct.roundToInt(), result.fleetVolumeOrders, result.stockoutSkus.size),
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
}
