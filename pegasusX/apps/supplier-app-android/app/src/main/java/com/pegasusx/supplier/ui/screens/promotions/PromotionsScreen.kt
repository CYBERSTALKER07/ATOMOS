package com.pegasusx.supplier.ui.screens.promotions

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.pegasus.design.network.RealtimeRefreshEffect
import com.pegasus.design.network.showFullScreenLoading
import com.pegasusx.supplier.data.model.PromoSimulateInput
import com.pegasusx.supplier.data.model.PromoSimulateResult
import com.pegasusx.supplier.data.model.SupplierPromotion
import com.pegasusx.supplier.data.model.SupplierPromotionUpsertRequest
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PromotionsScreen(api: SupplierApi, realtimeSignals: SupplierRealtimeSignals) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var promotions by remember { mutableStateOf(emptyList<SupplierPromotion>()) }
    var simResults by remember { mutableStateOf<Map<String, PromoSimulateResult>>(emptyMap()) }
    var simulatingId by remember { mutableStateOf<String?>(null) }
    var showCreate by remember { mutableStateOf(false) }
    var showEdit by remember { mutableStateOf(false) }
    var editingPromotion by remember { mutableStateOf<SupplierPromotion?>(null) }
    var name by remember { mutableStateOf("") }
    var discountBps by remember { mutableStateOf("500") }
    val scope = rememberCoroutineScope()

    fun reload(silent: Boolean = false) {
        scope.launch {
            if (!silent) {
                loading = true
                error = null
            }
            try {
                val resp = api.getPromotions()
                if (resp.isSuccessful) {
                    promotions = resp.body()?.promotions.orEmpty()
                } else if (!silent) {
                    error = "Failed (${resp.code()})"
                    promotions = emptyList()
                }
            } catch (e: Exception) {
                if (!silent) error = e.message
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(Unit) { reload() }

    RealtimeRefreshEffect(
        refreshTick = realtimeSignals.refreshTick,
        reconnectTick = realtimeSignals.reconnectTick,
        onRefresh = { reload(silent = it) },
    )

    Scaffold(
        topBar = { TopAppBar(title = { Text("Promotions") }) },
        floatingActionButton = {
            FloatingActionButton(onClick = { showCreate = true }) {
                Icon(Icons.Default.Add, contentDescription = stringResource(R.string.mobile_supplier_ui_create_promotion))
            }
        },
    ) { padding ->
        Box(Modifier.padding(padding).fillMaxSize()) {
            when {
                showFullScreenLoading(loading, promotions.isNotEmpty()) -> PegasusLoadingState("Loading promotions…", "Supplier promos")
                error != null -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Promotions unavailable",
                    body = error!!,
                )
                promotions.isEmpty() -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No promotions",
                    body = "Create a sale for products, categories, or your full catalog.",
                )
                else -> LazyColumn(
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    items(promotions, key = { it.promotionId }) { promo ->
                        ListItem(
                            headlineContent = { Text(promo.name) },
                            supportingContent = {
                                val sim = simResults[promo.promotionId]
                                Column {
                                    Text(stringResource(R.string.mobile_supplier_ui_n_0_scopetype_retailerscope, promo.discountBps / 100.0, promo.scopeType, promo.retailerScope) +
                                            if (promo.isActive) "" else " · inactive",
                                    )
                                    sim?.let {
                                        Text(
                                            stringResource(R.string.mobile_supplier_ui_p_l_sandbox_projectedvolume_units_margin_n_0_margindeltapct, it.projectedVolume, it.projectedMarginMinor / 100.0, it.marginDeltaPct),
                                            style = MaterialTheme.typography.bodySmall,
                                        )
                                    }
                                }
                            },
                            trailingContent = {
                                if (promo.isActive) {
                                    TextButton(
                                        enabled = simulatingId != promo.promotionId,
                                        onClick = {
                                            scope.launch {
                                                simulatingId = promo.promotionId
                                                runCatching {
                                                    api.simulatePromotionPandL(
                                                        PromoSimulateInput(
                                                            promotionId = promo.promotionId,
                                                            discountPct = promo.discountBps / 100.0,
                                                            expectedUnits = 500,
                                                            avgUnitMarginMinor = 1000,
                                                        ),
                                                    ).body()
                                                }.onSuccess { result ->
                                                    result?.let { sim ->
                                                        simResults = simResults + (promo.promotionId to sim)
                                                    }
                                                }
                                                simulatingId = null
                                            }
                                        },
                                    ) {
                                        Text(if (simulatingId == promo.promotionId) "…" else "P&L")
                                    }
                                    TextButton(onClick = {
                                        editingPromotion = promo
                                        name = promo.name
                                        discountBps = promo.discountBps.toString()
                                        showEdit = true
                                    }) { Text("Edit") }
                                    TextButton(onClick = {
                                        scope.launch {
                                            api.deactivatePromotion(promo.promotionId)
                                            reload()
                                        }
                                    }) { Text("Deactivate") }
                                }
                            },
                        )
                    }
                }
            }
        }
    }

    if (showCreate) {
        AlertDialog(
            onDismissRequest = { showCreate = false },
            title = { Text("New promotion") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    OutlinedTextField(
                        value = name,
                        onValueChange = { name = it },
                        label = { Text("Name") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = discountBps,
                        onValueChange = { discountBps = it.filter { ch -> ch.isDigit() } },
                        label = { Text("Discount (bps)") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    val bps = discountBps.toLongOrNull() ?: return@TextButton
                    if (name.isBlank()) return@TextButton
                    scope.launch {
                        api.createPromotion(
                            SupplierPromotionUpsertRequest(
                                name = name.trim(),
                                discountBps = bps,
                            ),
                        )
                        showCreate = false
                        name = ""
                        discountBps = "500"
                        reload()
                    }
                }) { Text("Create") }
            },
            dismissButton = {
                TextButton(onClick = { showCreate = false }) { Text("Cancel") }
            },
        )
    }

    val editing = editingPromotion
    if (showEdit && editing != null) {
        AlertDialog(
            onDismissRequest = {
                showEdit = false
                editingPromotion = null
            },
            title = { Text("Edit promotion") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    OutlinedTextField(
                        value = name,
                        onValueChange = { name = it },
                        label = { Text("Name") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = discountBps,
                        onValueChange = { discountBps = it.filter { ch -> ch.isDigit() } },
                        label = { Text("Discount (bps)") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    val bps = discountBps.toLongOrNull() ?: return@TextButton
                    if (name.isBlank()) return@TextButton
                    scope.launch {
                        api.updatePromotion(
                            editing.promotionId,
                            SupplierPromotionUpsertRequest(
                                name = name.trim(),
                                discountBps = bps,
                                scopeType = editing.scopeType,
                                retailerScope = editing.retailerScope,
                            ),
                        )
                        showEdit = false
                        editingPromotion = null
                        reload()
                    }
                }) { Text("Save") }
            },
            dismissButton = {
                TextButton(onClick = {
                    showEdit = false
                    editingPromotion = null
                }) { Text("Cancel") }
            },
        )
    }
}
