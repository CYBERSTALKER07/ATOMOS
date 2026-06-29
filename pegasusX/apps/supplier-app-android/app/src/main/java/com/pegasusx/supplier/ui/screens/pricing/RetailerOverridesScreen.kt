package com.pegasusx.supplier.ui.screens.pricing

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.CreateRetailerPriceOverrideRequest
import com.pegasusx.supplier.data.model.RetailerPriceOverride
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RetailerOverridesScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var overrides by remember { mutableStateOf<List<RetailerPriceOverride>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var showCreate by remember { mutableStateOf(false) }
    var retailerId by remember { mutableStateOf("") }
    var productId by remember { mutableStateOf("") }
    var price by remember { mutableStateOf("") }
    var notes by remember { mutableStateOf("") }
    var saving by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.listRetailerPriceOverrides()
                overrides = if (resp.isSuccessful) resp.body()?.overrides.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
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
                title = { Text("Retailer overrides") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    IconButton(onClick = { showCreate = true }) {
                        Icon(Icons.Default.Add, contentDescription = "Create")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading overrides…", "Per-retailer pricing")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Overrides unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            overrides.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No overrides",
                body = "Create retailer-specific product prices.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(overrides, key = { it.overrideId }) { row ->
                    ListItem(
                        headlineContent = { Text(row.productId.take(12)) },
                        supportingContent = {
                            Text("Retailer ${row.retailerId.take(8)} · price ${row.price}")
                        },
                        trailingContent = {
                            IconButton(onClick = {
                                scope.launch {
                                    val scopeId = SupplierIdempotencyKeys.supplierScopeId()
                                    ops.deleteRetailerPriceOverride(
                                        row.overrideId,
                                        SupplierIdempotencyKeys.retailerPriceOverrideDelete(scopeId, row.overrideId),
                                    )
                                    load()
                                }
                            }) {
                                Icon(Icons.Default.Delete, contentDescription = "Delete")
                            }
                        },
                    )
                    HorizontalDivider()
                }
            }
        }
    }

    if (showCreate) {
        AlertDialog(
            onDismissRequest = { showCreate = false },
            title = { Text("Create override") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    OutlinedTextField(retailerId, { retailerId = it }, label = { Text("Retailer ID") }, modifier = Modifier.fillMaxWidth())
                    OutlinedTextField(productId, { productId = it }, label = { Text("Product ID") }, modifier = Modifier.fillMaxWidth())
                    OutlinedTextField(
                        price,
                        { price = it.filter { ch -> ch.isDigit() } },
                        label = { Text("Price (minor)") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(notes, { notes = it }, label = { Text("Notes") }, modifier = Modifier.fillMaxWidth())
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val p = price.toLongOrNull() ?: return@TextButton
                        scope.launch {
                            saving = true
                            try {
                                val scopeId = SupplierIdempotencyKeys.supplierScopeId()
                                val resp = ops.createRetailerPriceOverride(
                                    CreateRetailerPriceOverrideRequest(
                                        retailerId = retailerId.trim(),
                                        productId = productId.trim(),
                                        price = p,
                                        notes = notes.ifBlank { null },
                                    ),
                                    SupplierIdempotencyKeys.retailerPriceOverrideCreate(
                                        scopeId,
                                        retailerId.trim(),
                                        productId.trim(),
                                        p,
                                    ),
                                )
                                if (resp.isSuccessful) {
                                    showCreate = false
                                    retailerId = ""
                                    productId = ""
                                    price = ""
                                    notes = ""
                                    load()
                                }
                            } finally {
                                saving = false
                            }
                        }
                    },
                    enabled = !saving,
                ) { Text("Create") }
            },
            dismissButton = { TextButton(onClick = { showCreate = false }) { Text("Cancel") } },
        )
    }
}
