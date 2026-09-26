package com.pegasusx.supplier.ui.screens.analytics

import androidx.compose.ui.res.stringResource

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.ShowChart
import androidx.compose.material.icons.filled.Storage
import androidx.compose.material.icons.filled.Warning
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.PlanningScenarioInput
import com.pegasusx.supplier.data.model.PlanningScenarioResult
import com.pegasusx.supplier.data.model.PlanningSAndOPSnapshot
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierKpiTile
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch
import kotlin.math.roundToInt
import com.pegasusx.supplier.R

@Composable
fun PlanningBrainSection(ops: SupplierOperationsRepository) {
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

    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        SupplierSectionTitle("Planning sandbox")
        Text(
            "Read-only what-if and lightweight S&OP",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        if (error != null) {
            Text(error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
        }
        if (loading) {
            Text("Loading S&OP…", style = MaterialTheme.typography.bodySmall)
        } else {
            sandop?.let { snap ->
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm), modifier = Modifier.fillMaxWidth()) {
                    SupplierKpiTile("Factory cap", snap.factoryCapacityUnits.toString(), Icons.Default.Storage, Modifier.weight(1f))
                    SupplierKpiTile("WH inbound", snap.warehouseInboundCapUnits.toString(), Icons.Default.LocalShipping, Modifier.weight(1f))
                }
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm), modifier = Modifier.fillMaxWidth()) {
                    SupplierKpiTile("Utilization", "${snap.utilizationPct.roundToInt()}%", Icons.Default.ShowChart, Modifier.weight(1f))
                    SupplierKpiTile("Alert", if (snap.capacityAlert) "Breach" else "OK", Icons.Default.Warning, Modifier.weight(1f))
                }
            }
        }
        ElevatedCard(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(PegasusSpacing.md), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                Text("Scenario run", style = MaterialTheme.typography.titleSmall)
                Text(stringResource(R.string.mobile_supplier_ui_factory_downtime_roundtointh, downtimeHours.roundToInt()), style = MaterialTheme.typography.bodySmall)
                Slider(
                    value = downtimeHours,
                    onValueChange = { downtimeHours = it },
                    valueRange = 0f..168f,
                    steps = 167,
                )
                Text(stringResource(R.string.mobile_supplier_ui_demand_delta_roundtoint, demandDeltaPct.roundToInt()), style = MaterialTheme.typography.bodySmall)
                Slider(
                    value = demandDeltaPct,
                    onValueChange = { demandDeltaPct = it },
                    valueRange = -50f..200f,
                    steps = 50,
                )
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
            }
        }
    }
}
