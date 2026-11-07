package com.pegasusx.supplier.ui.screens.foundry

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Dangerous
import androidx.compose.material.icons.filled.Layers
import androidx.compose.material.icons.filled.Science
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.FoundrySandboxUiState
import com.pegasusx.supplier.ui.viewmodel.FoundrySandboxViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FoundrySandboxScreen(
    viewModel: FoundrySandboxViewModel,
    onBack: () -> Unit,
) {
    val state by viewModel.uiState.collectAsState()
    var selectedTab by remember { mutableStateOf(0) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Foundry Sandbox & Lot Recall", style = MaterialTheme.typography.titleLarge) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            TabRow(selectedTabIndex = selectedTab) {
                Tab(
                    selected = selectedTab == 0,
                    onClick = { selectedTab = 0 },
                    text = { Text("Scenario Sandbox (CoW)") },
                    icon = { Icon(Icons.Default.Layers, contentDescription = null) },
                )
                Tab(
                    selected = selectedTab == 1,
                    onClick = { selectedTab = 1 },
                    text = { Text("Lot Recall Blast-Radius") },
                    icon = { Icon(Icons.Default.Science, contentDescription = null) },
                )
            }

            // Status & Error Banners
            state.errorMessage?.let { err ->
                Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer),
                ) {
                    Row(
                        modifier = Modifier.padding(PegasusSpacing.md),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(Icons.Default.Dangerous, contentDescription = null, tint = MaterialTheme.colorScheme.error)
                        Spacer(modifier = Modifier.width(PegasusSpacing.sm))
                        Text(err, color = MaterialTheme.colorScheme.onErrorContainer, style = MaterialTheme.typography.bodyMedium)
                    }
                }
            }

            state.statusMessage?.let { msg ->
                Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
                    colors = CardDefaults.cardColors(
                        containerColor = if (state.hasConflict) MaterialTheme.colorScheme.errorContainer else MaterialTheme.colorScheme.secondaryContainer,
                    ),
                ) {
                    Row(
                        modifier = Modifier.padding(PegasusSpacing.md),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            if (state.hasConflict) Icons.Default.Warning else Icons.Default.CheckCircle,
                            contentDescription = null,
                            tint = if (state.hasConflict) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSecondaryContainer,
                        )
                        Spacer(modifier = Modifier.width(PegasusSpacing.sm))
                        Text(msg, style = MaterialTheme.typography.bodyMedium)
                    }
                }
            }

            if (selectedTab == 0) {
                ScenarioSandboxTab(state = state, viewModel = viewModel)
            } else {
                LotRecallTab(state = state, viewModel = viewModel)
            }
        }
    }
}

@Composable
private fun ScenarioSandboxTab(
    state: FoundrySandboxUiState,
    viewModel: FoundrySandboxViewModel,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        item {
            Card(
                modifier = Modifier.fillMaxWidth(),
                elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
            ) {
                Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
                    Text("Scenario Isolation Layer", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Text(
                        "Copy-on-Write (CoW) branching creates an ephemeral sandbox without mutating live production state until committed.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Spacer(modifier = Modifier.height(PegasusSpacing.md))
                    OutlinedTextField(
                        value = state.scenarioId,
                        onValueChange = { viewModel.updateScenarioId(it) },
                        label = { Text("Scenario ID") },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                }
            }
        }

        item {
            Card(
                modifier = Modifier.fillMaxWidth(),
                elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
            ) {
                Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
                    Text("Add CoW Patch", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Spacer(modifier = Modifier.height(PegasusSpacing.sm))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                    ) {
                        OutlinedTextField(
                            value = state.objectType,
                            onValueChange = { viewModel.updateObjectType(it) },
                            label = { Text("Object Type") },
                            modifier = Modifier.weight(1f),
                            singleLine = true,
                        )
                        OutlinedTextField(
                            value = state.objectId,
                            onValueChange = { viewModel.updateObjectId(it) },
                            label = { Text("Object ID") },
                            modifier = Modifier.weight(1f),
                            singleLine = true,
                        )
                    }

                    Spacer(modifier = Modifier.height(PegasusSpacing.sm))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                    ) {
                        OutlinedTextField(
                            value = state.patchKey,
                            onValueChange = { viewModel.updatePatchKey(it) },
                            label = { Text("Property Key") },
                            modifier = Modifier.weight(1f),
                            singleLine = true,
                        )
                        OutlinedTextField(
                            value = state.patchValue,
                            onValueChange = { viewModel.updatePatchValue(it) },
                            label = { Text("New Value") },
                            modifier = Modifier.weight(1f),
                            singleLine = true,
                        )
                    }

                    Spacer(modifier = Modifier.height(PegasusSpacing.md))

                    Button(
                        onClick = { viewModel.applyCowPatch() },
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Icon(Icons.Default.Add, contentDescription = null)
                        Spacer(modifier = Modifier.width(PegasusSpacing.sm))
                        Text("Apply Patch to Scenario")
                    }
                }
            }
        }

        if (state.appliedPatches.isNotEmpty()) {
            item {
                Text("Applied CoW Patches (${state.appliedPatches.size})", style = MaterialTheme.typography.titleMedium)
            }
            items(state.appliedPatches) { patch ->
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(PegasusSpacing.md),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column {
                            Text("${patch.objectType} : ${patch.objectId}", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.bodyMedium)
                            Text("${patch.key} → ${patch.value}", style = MaterialTheme.typography.bodySmall, fontFamily = FontFamily.Monospace)
                        }
                        Text("STAGED", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
                    }
                }
            }
        }

        if (state.conflicts.isNotEmpty()) {
            item {
                Text("Detected Concurrency Conflicts (409)", style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.error)
            }
            items(state.conflicts) { conflict ->
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer),
                ) {
                    Column(modifier = Modifier.padding(PegasusSpacing.md)) {
                        Text("${conflict.objectType} : ${conflict.objectId}", fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.onErrorContainer)
                        Text(
                            "Base Version: ${conflict.baseVersion} | Current Live Version: ${conflict.liveVersion}",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onErrorContainer,
                        )
                        Text("Conflict: Baseline changed concurrently. Rebase scenario before committing.", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.error)
                    }
                }
            }
        }

        item {
            Button(
                onClick = { viewModel.commitScenario() },
                enabled = !state.isCommitting,
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary),
                modifier = Modifier
                    .fillMaxWidth()
                    .height(50.dp),
            ) {
                if (state.isCommitting) {
                    CircularProgressIndicator(modifier = Modifier.size(24.dp), color = MaterialTheme.colorScheme.onPrimary)
                } else {
                    Text("Optimistic Commit to Live Layer")
                }
            }
        }
    }
}

@Composable
private fun LotRecallTab(
    state: FoundrySandboxUiState,
    viewModel: FoundrySandboxViewModel,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        item {
            Card(
                modifier = Modifier.fillMaxWidth(),
                elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
            ) {
                Column(modifier = Modifier.padding(PegasusSpacing.lg)) {
                    Text("Contaminated Lot Blast-Radius", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Text(
                        "Enter or scan a defective ingredient/raw material lot barcode to trace its lineage forward through work orders, finished goods, pallets, totes, and in-transit delivery blocks.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Spacer(modifier = Modifier.height(PegasusSpacing.md))
                    OutlinedTextField(
                        value = state.lotBarcode,
                        onValueChange = { viewModel.updateLotBarcode(it) },
                        label = { Text("Raw Material Lot Barcode") },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                    Spacer(modifier = Modifier.height(PegasusSpacing.md))
                    Button(
                        onClick = { viewModel.analyzeLotRecall() },
                        enabled = !state.isAnalyzingRecall,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        if (state.isAnalyzingRecall) {
                            CircularProgressIndicator(modifier = Modifier.size(24.dp), color = MaterialTheme.colorScheme.onPrimary)
                        } else {
                            Text("Trace Blast Radius")
                        }
                    }
                }
            }
        }

        state.recallReport?.let { report ->
            item {
                Text("Lineage Forward Blast Radius", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
            }

            item {
                BlastRadiusMetricCard(
                    title = "Affected Work Orders",
                    count = report.affectedWorkOrders.size,
                    items = report.affectedWorkOrders,
                )
            }

            item {
                BlastRadiusMetricCard(
                    title = "Affected Finished Lots",
                    count = report.affectedFinishedLots.size,
                    items = report.affectedFinishedLots,
                )
            }

            item {
                BlastRadiusMetricCard(
                    title = "Quarantined SSCC Pallets",
                    count = report.quarantinedSsccPallets.size,
                    items = report.quarantinedSsccPallets,
                    highlight = true,
                )
            }

            item {
                BlastRadiusMetricCard(
                    title = "Quarantined Warehouse Totes",
                    count = report.quarantinedWarehouseTotes.size,
                    items = report.quarantinedWarehouseTotes,
                )
            }

            item {
                BlastRadiusMetricCard(
                    title = "In-Transit Driver Drop Blocks",
                    count = report.inTransitDriverDropBlocks.size,
                    items = report.inTransitDriverDropBlocks,
                    highlight = true,
                )
            }

            item {
                BlastRadiusMetricCard(
                    title = "Impacted Soliq E-Facturas",
                    count = report.impactedSoliqFacturas.size,
                    items = report.impactedSoliqFacturas,
                )
            }

            item {
                Button(
                    onClick = { viewModel.executeQuarantineBlastRadius() },
                    enabled = !state.isQuarantining && !state.quarantineSuccess,
                    colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error),
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(52.dp),
                ) {
                    if (state.isQuarantining) {
                        CircularProgressIndicator(modifier = Modifier.size(24.dp), color = MaterialTheme.colorScheme.onError)
                    } else if (state.quarantineSuccess) {
                        Text("QUARANTINE ENFORCED")
                    } else {
                        Text("LOCKDOWN & QUARANTINE BLAST RADIUS")
                    }
                }
            }
        }
    }
}

@Composable
private fun BlastRadiusMetricCard(
    title: String,
    count: Int,
    items: List<String>,
    highlight: Boolean = false,
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = if (highlight) MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f) else MaterialTheme.colorScheme.surfaceVariant,
        ),
    ) {
        Column(modifier = Modifier.padding(PegasusSpacing.md)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text(title, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.bodyMedium)
                Text(
                    count.toString(),
                    fontWeight = FontWeight.Bold,
                    color = if (highlight) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary,
                )
            }
            if (items.isNotEmpty()) {
                Spacer(modifier = Modifier.height(PegasusSpacing.xs))
                Text(
                    items.joinToString(", "),
                    style = MaterialTheme.typography.bodySmall,
                    fontFamily = FontFamily.Monospace,
                    maxLines = 3,
                )
            }
        }
    }
}
