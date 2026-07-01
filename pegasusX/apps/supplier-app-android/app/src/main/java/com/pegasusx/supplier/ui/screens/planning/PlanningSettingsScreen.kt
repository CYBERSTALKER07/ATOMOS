package com.pegasusx.supplier.ui.screens.planning

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.ui.Modifier
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SeasonalOverrideInput
import com.pegasusx.supplier.data.model.SeasonalOverrideRow
import com.pegasusx.supplier.data.model.SeasonalTemplatesResponse
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch

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
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbar) },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading seasonal overrides…", modifier = Modifier.padding(padding))
            error != null && data == null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
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
                item {
                    SupplierSectionTitle("Custom season")
                    Text(
                        "Date-range overrides for seasonal forecast baselines.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                item {
                    if ((data?.builtinTemplates?.size ?: 0) > 0) {
                        var expanded by remember { mutableStateOf(false) }
                        ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }) {
                            OutlinedTextField(
                                value = templateId.ifBlank { "Custom" },
                                onValueChange = {},
                                readOnly = true,
                                label = { Text("Template") },
                                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
                                modifier = Modifier.menuAnchor().fillMaxWidth(),
                            )
                            ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                                DropdownMenuItem(
                                    text = { Text("Custom") },
                                    onClick = { templateId = ""; expanded = false },
                                )
                                data?.builtinTemplates?.forEach { template ->
                                    DropdownMenuItem(
                                        text = { Text(template.name) },
                                        onClick = { templateId = template.id; expanded = false },
                                    )
                                }
                            }
                        }
                    }
                    OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name (optional)") }, modifier = Modifier.fillMaxWidth())
                    OutlinedTextField(value = startDate, onValueChange = { startDate = it }, label = { Text("Start (YYYY-MM-DD)") }, modifier = Modifier.fillMaxWidth())
                    OutlinedTextField(value = endDate, onValueChange = { endDate = it }, label = { Text("End (YYYY-MM-DD)") }, modifier = Modifier.fillMaxWidth())
                    formError?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
                    Button(
                        enabled = !saving,
                        onClick = {
                            if (startDate.isBlank() || endDate.isBlank()) {
                                formError = "Start and end dates are required"
                                return@Button
                            }
                            scope.launch {
                                saving = true
                                formError = null
                                val scopeId = SupplierIdempotencyKeys.supplierScopeId()
                                val key = SupplierIdempotencyKeys.seasonalOverrideCreate(scopeId, startDate, endDate)
                                val resp = ops.createSeasonalOverride(
                                    SeasonalOverrideInput(
                                        templateId = templateId.ifBlank { null },
                                        startDate = startDate.trim(),
                                        endDate = endDate.trim(),
                                        name = name.ifBlank { null },
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
                                    snackbar.showSnackbar("Override created")
                                } else {
                                    formError = "Create failed (${resp.code()})"
                                }
                                saving = false
                            }
                        },
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(if (saving) "Saving…" else "Create override")
                    }
                }
                item { SupplierSectionTitle("Active overrides") }
                val overrides = data?.overrides.orEmpty()
                if (overrides.isEmpty()) {
                    item {
                        Text(
                            "No custom seasonal overrides yet.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                } else {
                    items(overrides, key = SeasonalOverrideRow::overrideId) { row ->
                        SupplierOpsListCard(
                            headline = row.name?.ifBlank { row.templateId } ?: row.templateId,
                            supporting = "${row.startDate} → ${row.endDate} · ${if (row.isActive) "Active" else "Inactive"}",
                        )
                    }
                }
            }
        }
    }
}
