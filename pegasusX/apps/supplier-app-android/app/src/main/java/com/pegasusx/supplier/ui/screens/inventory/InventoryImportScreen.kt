package com.pegasusx.supplier.ui.screens.inventory

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.ImportSessionCreateRequest
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.TokenHolder
import com.pegasusx.supplier.ui.components.SupplierRuntimeBanner
import com.pegasusx.supplier.ui.components.SupplierRuntimeTone
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody

private enum class ImportWizardStep { CREATE, INGEST, MAPPING, APPROVE, APPLY }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InventoryImportScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var step by remember { mutableStateOf(ImportWizardStep.CREATE) }
    var fileName by remember { mutableStateOf("import.csv") }
    var csvBody by remember { mutableStateOf("") }
    var sessionId by remember { mutableStateOf("") }
    var mappingJson by remember { mutableStateOf<JsonElement?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val scopeId = TokenHolder.supplierId.orEmpty().ifBlank { "supplier" }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Inventory import") },
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
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                "Step ${step.ordinal + 1} of 5 — ${step.name.lowercase().replaceFirstChar { it.uppercase() }}",
                style = MaterialTheme.typography.titleSmall,
                color = MaterialTheme.colorScheme.primary,
            )
            when (step) {
                ImportWizardStep.CREATE -> {
                    OutlinedTextField(
                        value = fileName,
                        onValueChange = { fileName = it },
                        label = { Text("File name") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Button(
                        onClick = {
                            scope.launch {
                                busy = true
                                error = null
                                try {
                                    val bytes = csvBody.ifBlank { "product_id,quantity\n" }.toByteArray()
                                    val resp = ops.createImportSession(
                                        SupplierIdempotencyKeys.importCreate(scopeId, fileName.trim(), bytes.size),
                                        ImportSessionCreateRequest(fileName.trim(), bytes.size),
                                    )
                                    if (resp.isSuccessful) {
                                        sessionId = resp.body()?.sessionId.orEmpty()
                                        statusMessage = "Session created: $sessionId"
                                        step = ImportWizardStep.INGEST
                                    } else {
                                        error = "Create failed (${resp.code()})"
                                    }
                                } catch (e: Exception) {
                                    error = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        },
                        enabled = !busy && fileName.isNotBlank(),
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Create session") }
                }
                ImportWizardStep.INGEST -> {
                    Text("Session: $sessionId", style = MaterialTheme.typography.bodySmall)
                    OutlinedTextField(
                        value = csvBody,
                        onValueChange = { csvBody = it },
                        label = { Text("CSV body") },
                        minLines = 6,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Button(
                        onClick = {
                            scope.launch {
                                busy = true
                                error = null
                                try {
                                    val body = csvBody.toRequestBody("text/csv".toMediaType())
                                    val resp = ops.ingestImportSession(
                                        sessionId,
                                        SupplierIdempotencyKeys.importIngest(sessionId, csvBody),
                                        body,
                                    )
                                    if (resp.isSuccessful) {
                                        statusMessage = "Ingest complete"
                                        step = ImportWizardStep.MAPPING
                                    } else {
                                        error = "Ingest failed (${resp.code()})"
                                    }
                                } catch (e: Exception) {
                                    error = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        },
                        enabled = !busy && sessionId.isNotBlank() && csvBody.isNotBlank(),
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Ingest CSV") }
                }
                ImportWizardStep.MAPPING -> {
                    LaunchedEffect(sessionId) {
                        if (sessionId.isNotBlank()) {
                            val resp = ops.getImportMapping(sessionId)
                            if (resp.isSuccessful) mappingJson = resp.body()
                        }
                    }
                    Text(
                        mappingJson?.toString() ?: "Loading mapping…",
                        style = MaterialTheme.typography.bodySmall,
                    )
                    Button(
                        onClick = { step = ImportWizardStep.APPROVE },
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Continue to approve") }
                }
                ImportWizardStep.APPROVE -> {
                    Button(
                        onClick = {
                            scope.launch {
                                busy = true
                                error = null
                                try {
                                    val resp = ops.approveImportSession(
                                        sessionId,
                                        SupplierIdempotencyKeys.importApprove(sessionId),
                                    )
                                    if (resp.isSuccessful) {
                                        statusMessage = "Approved"
                                        step = ImportWizardStep.APPLY
                                    } else {
                                        error = "Approve failed (${resp.code()})"
                                    }
                                } catch (e: Exception) {
                                    error = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        },
                        enabled = !busy,
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Approve import") }
                }
                ImportWizardStep.APPLY -> {
                    Button(
                        onClick = {
                            scope.launch {
                                busy = true
                                error = null
                                try {
                                    val resp = ops.applyImportSession(
                                        sessionId,
                                        SupplierIdempotencyKeys.importApply(sessionId),
                                    )
                                    if (resp.isSuccessful) {
                                        statusMessage = "Applied: ${resp.body()}"
                                    } else {
                                        error = "Apply failed (${resp.code()})"
                                    }
                                } catch (e: Exception) {
                                    error = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        },
                        enabled = !busy,
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Apply import") }
                }
            }
            statusMessage?.let {
                SupplierRuntimeBanner(
                    tone = SupplierRuntimeTone.Live,
                    message = it,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            error?.let {
                SupplierRuntimeBanner(
                    tone = SupplierRuntimeTone.Warning,
                    message = it,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }
    }
}
