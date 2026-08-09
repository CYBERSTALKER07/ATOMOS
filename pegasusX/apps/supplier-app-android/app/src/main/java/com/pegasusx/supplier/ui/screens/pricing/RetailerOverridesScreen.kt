package com.pegasusx.supplier.ui.screens.pricing

import androidx.compose.ui.res.stringResource

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
import com.pegasusx.supplier.data.model.RetailerOverridePreview
import com.pegasusx.supplier.data.model.RetailerOverridePreviewRequest
import com.pegasusx.supplier.data.model.RetailerPriceOverride
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

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
    var preview by remember { mutableStateOf<RetailerOverridePreview?>(null) }
    var previewLoading by remember { mutableStateOf(false) }
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

    LaunchedEffect(retailerId, productId, price, showCreate) {
        if (!showCreate) {
            preview = null
            previewLoading = false
            return@LaunchedEffect
        }
        val product = productId.trim()
        val proposed = price.toLongOrNull()
        if (product.isEmpty() || proposed == null || proposed <= 0) {
            preview = null
            previewLoading = false
            return@LaunchedEffect
        }
        previewLoading = true
        kotlinx.coroutines.delay(400)
        try {
            val resp = ops.previewRetailerPriceOverride(
                RetailerOverridePreviewRequest(
                    retailerId = retailerId.trim().ifBlank { null },
                    productId = product,
                    proposedPrice = proposed,
                ),
            )
            preview = if (resp.isSuccessful) resp.body() else null
        } catch (_: Exception) {
            preview = null
        } finally {
            previewLoading = false
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Retailer overrides") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                actions = {
                    IconButton(onClick = { showCreate = true }) {
                        Icon(Icons.Default.Add, contentDescription = stringResource(R.string.mobile_supplier_ui_create))
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading overrides…", "Per-retailer pricing")
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Overrides unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            overrides.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
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
                            Text(stringResource(R.string.mobile_supplier_ui_retailer_take_price_price, row.retailerId.take(8), row.price))
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
                                Icon(Icons.Default.Delete, contentDescription = stringResource(R.string.mobile_supplier_ui_delete))
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
                    when {
                        previewLoading -> Text("Calculating impact preview…", style = MaterialTheme.typography.bodySmall)
                        preview != null -> {
                            val p = preview!!
                            ElevatedCard(Modifier.fillMaxWidth()) {
                                Column(Modifier.padding(PegasusSpacing.md), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                                    Text("Impact preview", style = MaterialTheme.typography.titleSmall)
                                    Text(stringResource(R.string.mobile_supplier_ui_retailers_on_sku_retailersonskucount, p.retailersOnSkuCount))
                                    Text(stringResource(R.string.mobile_supplier_ui_active_overrides_activeoverridecount, p.activeOverrideCount))
                                    Text(stringResource(R.string.mobile_supplier_ui_catalog_list_price_cataloglistprice, p.catalogListPrice))
                                    Text(stringResource(R.string.mobile_supplier_ui_margin_delta_unit_margindeltaperunit_marginestimatelabel, p.marginDeltaPerUnit, p.marginEstimateLabel))
                                }
                            }
                        }
                    }
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
                                    preview = null
                                    load()
                                }
                            } finally {
                                saving = false
                            }
                        }
                    },
                    enabled = !saving,
                ) { Text(stringResource(R.string.mobile_supplier_ui_create)) }
            },
            dismissButton = { TextButton(onClick = { showCreate = false }) { Text("Cancel") } },
        )
    }
}
