package com.pegasusx.supplier.ui.screens.planning

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.ui.Modifier
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import com.pegasusx.supplier.data.model.SeasonalOverrideInput
import com.pegasusx.supplier.data.model.SeasonalOverrideRow
import com.pegasusx.supplier.data.model.SeasonalTemplatesResponse
import com.pegasusx.supplier.data.model.KillSwitchRequest
import com.pegasusx.supplier.data.model.NetworkModeUpdateRequest
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlanningSettingsScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var data by remember { mutableStateOf<SeasonalTemplatesResponse?>(null) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var formError by remember { mutableStateOf<String?>(null) }
    var templateId by remember { mutableStateOf("") }
    var name by remember { mutableStateOf("") }
    var startDate by remember { mutableStateOf("") }
    var endDate by remember { mutableStateOf("") }
    var multiplier by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val snackbar = remember { SnackbarHostState() }

    fun load() {
        scope.launch {
            loading = true
            error = null
            val resp = ops.getSeasonalOverrides()
            if (resp.isSuccessful) {
                data = resp.body()
            } else {
                error = "Failed (${resp.code()})"
            }
            loading = false
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Planning settings") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbar) },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading settings…", body = "", modifier = Modifier.padding(padding))
            error != null && data == null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Planning settings unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item { FactoryPlanningOpsCard(ops) }
                item {
                    SupplierSectionTitle("Custom season")
                    Text(
                        "Date-range overrides for seasonal forecast baselines.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                item {
                    CreateOverrideForm(
                        data = data,
                        templateId = templateId,
                        name = name,
                        startDate = startDate,
                        endDate = endDate,
                        multiplier = multiplier,
                        formError = formError,
                        saving = saving,
                        onTemplateIdChange = { templateId = it },
                        onNameChange = { name = it },
                        onStartDateChange = { startDate = it },
                        onEndDateChange = { endDate = it },
                        onMultiplierChange = { multiplier = it },
                        onSubmit = {
                            if (startDate.isBlank() || endDate.isBlank()) {
                                formError = "Start and end dates are required"
                                return@CreateOverrideForm
                            }
                            scope.launch {
                                saving = true
                                formError = null
                                val scopeId = SupplierIdempotencyKeys.supplierScopeId()
                                val key = SupplierIdempotencyKeys.seasonalOverrideCreate(scopeId, startDate, endDate)
                                val multVal = multiplier.trim().toDoubleOrNull()
                                val resp = ops.createSeasonalOverride(
                                    SeasonalOverrideInput(
                                        templateId = templateId.ifBlank { null },
                                        startDate = startDate.trim(),
                                        endDate = endDate.trim(),
                                        name = name.ifBlank { null },
                                        multiplier = multVal,
                                    ),
                                    key,
                                )
                                if (resp.isSuccessful) {
                                    val row = resp.body()
                                    if (row != null) {
                                        data = data?.copy(overrides = listOf(row) + (data?.overrides.orEmpty()))
                                            ?: SeasonalTemplatesResponse(overrides = listOf(row))
                                    }
                                    name = ""
                                    startDate = ""
                                    endDate = ""
                                    templateId = ""
                                    multiplier = ""
                                    snackbar.showSnackbar("Override created")
                                } else {
                                    formError = "Create failed (${resp.code()})"
                                }
                                saving = false
                            }
                        },
                    )
                }
                item { SupplierSectionTitle("Active overrides") }
                val overrides = data?.overrides.orEmpty()
                item {
                    SeasonalOverridesList(overrides = overrides)
                }
            }
        }
    }
}

private val NETWORK_MODES = listOf("SPEED", "ECONOMY", "BALANCED", "LOW_CARBON", "MANUAL_ONLY")

@Composable
private fun FactoryPlanningOpsCard(ops: SupplierOperationsRepository) {
    var mode by remember { mutableStateOf("") }
    var planningEnabled by remember { mutableStateOf<Boolean?>(null) }
    var status by remember { mutableStateOf<String?>(null) }
    var reason by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun loadMode() {
        scope.launch {
            val resp = ops.getNetworkMode()
            if (resp.isSuccessful) {
                mode = resp.body()?.mode.orEmpty()
                planningEnabled = resp.body()?.planningEnabled
            } else {
                status = "Network mode failed (${resp.code()})"
            }
        }
    }

    LaunchedEffect(Unit) { loadMode() }

    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm), modifier = Modifier.padding(PegasusSpacing.lg)) {
            SupplierSectionTitle("Factory network ops")
            Text(
                "Mode, pull-matrix, predictive-push, kill-switch. Pull-matrix and predictive-push 409 if FACTORY_PLANNING_ENABLED is off.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (planningEnabled == false) {
                Text("Engines off (env flag).", style = MaterialTheme.typography.bodySmall)
            }
            NETWORK_MODES.forEach { m ->
                OutlinedButton(
                    onClick = {
                        busy = true
                        scope.launch {
                            val key = SupplierIdempotencyKeys.networkModePut(SupplierIdempotencyKeys.supplierScopeId(), m)
                            val resp = ops.putNetworkMode(NetworkModeUpdateRequest(mode = m), key)
                            status = if (resp.isSuccessful) {
                                "Mode ${resp.body()?.oldMode} → ${resp.body()?.newMode}"
                            } else {
                                "Mode failed (${resp.code()})"
                            }
                            loadMode()
                            busy = false
                        }
                    },
                    enabled = !busy,
                ) { Text(if (mode == m) "● $m" else m) }
            }
            Button(
                onClick = {
                    busy = true
                    scope.launch {
                        val resp = ops.postPlanningPullMatrix(
                            SupplierIdempotencyKeys.planningPullMatrix(SupplierIdempotencyKeys.supplierScopeId()),
                        )
                        status = when {
                            resp.code() == 409 -> "factory_planning_disabled — engines off until FACTORY_PLANNING_ENABLED is on"
                            resp.isSuccessful -> "Pull-matrix ${resp.body()?.status}: ${resp.body()?.transfers} transfers"
                            else -> "Pull-matrix failed (${resp.code()})"
                        }
                        busy = false
                    }
                },
                enabled = !busy,
            ) { Text("Run pull-matrix") }
            Button(
                onClick = {
                    busy = true
                    scope.launch {
                        val resp = ops.postPlanningPredictivePush(
                            SupplierIdempotencyKeys.planningPredictivePush(SupplierIdempotencyKeys.supplierScopeId()),
                        )
                        status = when {
                            resp.code() == 409 -> "factory_planning_disabled — engines off until FACTORY_PLANNING_ENABLED is on"
                            resp.isSuccessful -> "Predictive-push ${resp.body()?.source}: ${resp.body()?.transfers} transfers, ${resp.body()?.skus} SKUs (${resp.body()?.grain?.ifBlank { "baseline" }})"
                            else -> "Predictive-push failed (${resp.code()})"
                        }
                        busy = false
                    }
                },
                enabled = !busy,
            ) { Text("Predictive push") }
            OutlinedTextField(
                value = reason,
                onValueChange = { reason = it },
                label = { Text("Kill-switch reason (ADMIN)") },
                modifier = Modifier.fillMaxWidth(),
            )
            Button(
                onClick = {
                    if (reason.isBlank()) {
                        status = "Typed reason required"
                        return@Button
                    }
                    busy = true
                    scope.launch {
                        val key = SupplierIdempotencyKeys.planningKillSwitch(
                            SupplierIdempotencyKeys.supplierScopeId(),
                            reason.trim(),
                        )
                        val resp = ops.postPlanningKillSwitch(KillSwitchRequest(reason = reason.trim()), key)
                        status = if (resp.isSuccessful) {
                            "Kill-switch cancelled ${resp.body()?.cancelledTransfers}"
                        } else {
                            "Kill-switch failed (${resp.code()})"
                        }
                        loadMode()
                        busy = false
                    }
                },
                enabled = !busy,
            ) { Text("Kill-switch") }
            status?.let { Text(it, style = MaterialTheme.typography.bodySmall) }
        }
    }
}
