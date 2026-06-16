package com.pegasusx.supplier.ui.screens.returns

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.ResolveReturnRequest
import com.pegasusx.supplier.data.model.SupplierReturnRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch

private val RESOLUTIONS = listOf("RETURN_TO_STOCK", "WRITE_OFF")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReturnsScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var items by remember { mutableStateOf<List<SupplierReturnRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var resolvingId by remember { mutableStateOf<String?>(null) }
    var resolution by remember { mutableStateOf("RETURN_TO_STOCK") }
    var notes by remember { mutableStateOf("") }
    var actionLoading by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getReturns(status = "PENDING", limit = 100, offset = 0)
                items = if (resp.isSuccessful) resp.body()?.data.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun resolve(returnId: String) {
        scope.launch {
            actionLoading = returnId
            error = null
            try {
                val key = SupplierIdempotencyKeys.resolveReturn(returnId, resolution)
                val resp = ops.resolveReturn(
                    ResolveReturnRequest(
                        returnId = returnId,
                        lineItemId = returnId,
                        resolution = resolution,
                        notes = notes.trim(),
                    ),
                    key,
                )
                if (!resp.isSuccessful) throw IllegalStateException("resolve_failed (${resp.code()})")
                resolvingId = null
                notes = ""
                load()
            } catch (e: Exception) {
                error = e.message ?: "resolve_failed"
            } finally {
                actionLoading = null
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Returns") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    TextButton(onClick = { load() }) { Text("Refresh") }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading returns…", "Driver-rejected delivery lines")
            error != null && items.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Returns unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            items.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No open returns",
                body = "Rejected quantities will appear here after driver offload.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                if (error != null) {
                    item {
                        Text(error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                    }
                }
                items(items, key = { it.returnId }) { row ->
                    val gateResolved = row.physicalStatus == "RESTOCKED" || row.physicalStatus == "WRITTEN_OFF"
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text(row.productName, style = MaterialTheme.typography.titleMedium)
                            Text("Qty ${row.quantity} · ${row.reason}", style = MaterialTheme.typography.bodyMedium)
                            Text("Physical: ${row.physicalStatus}", style = MaterialTheme.typography.labelMedium)
                            if (row.driverName.isNotBlank()) {
                                Text("Driver: ${row.driverName}", style = MaterialTheme.typography.bodySmall)
                            }
                            if (row.receivedQty > 0) {
                                Text("Scanned: ${row.receivedQty}", style = MaterialTheme.typography.labelSmall)
                            }
                            if (resolvingId == row.returnId) {
                                var expanded by remember { mutableStateOf(false) }
                                ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }) {
                                    OutlinedTextField(
                                        value = resolution.replace('_', ' '),
                                        onValueChange = {},
                                        readOnly = true,
                                        label = { Text("Resolution") },
                                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded) },
                                        modifier = Modifier.menuAnchor().fillMaxWidth(),
                                    )
                                    ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                                        RESOLUTIONS.forEach { option ->
                                            DropdownMenuItem(
                                                text = { Text(option.replace('_', ' ')) },
                                                onClick = {
                                                    resolution = option
                                                    expanded = false
                                                },
                                            )
                                        }
                                    }
                                }
                                OutlinedTextField(
                                    value = notes,
                                    onValueChange = { notes = it },
                                    label = { Text("Notes (optional)") },
                                    modifier = Modifier.fillMaxWidth(),
                                    singleLine = true,
                                )
                                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                                    Button(
                                        onClick = { resolve(row.returnId) },
                                        enabled = actionLoading != row.returnId,
                                    ) {
                                        Text(if (actionLoading == row.returnId) "…" else "Confirm")
                                    }
                                    TextButton(onClick = { resolvingId = null }) { Text("Cancel") }
                                }
                            } else if (gateResolved) {
                                Text("Gate resolved", style = MaterialTheme.typography.labelSmall)
                            } else {
                                TextButton(onClick = {
                                    resolvingId = row.returnId
                                    resolution = "RETURN_TO_STOCK"
                                    notes = ""
                                }) {
                                    Text("Dispute / override")
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
