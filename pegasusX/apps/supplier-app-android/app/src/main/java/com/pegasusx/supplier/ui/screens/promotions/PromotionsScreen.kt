package com.pegasusx.supplier.ui.screens.promotions

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
import com.pegasusx.supplier.data.model.SupplierPromotion
import com.pegasusx.supplier.data.model.SupplierPromotionUpsertRequest
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PromotionsScreen(api: SupplierApi) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var promotions by remember { mutableStateOf(emptyList<SupplierPromotion>()) }
    var showCreate by remember { mutableStateOf(false) }
    var showEdit by remember { mutableStateOf(false) }
    var editingPromotion by remember { mutableStateOf<SupplierPromotion?>(null) }
    var name by remember { mutableStateOf("") }
    var discountBps by remember { mutableStateOf("500") }
    val scope = rememberCoroutineScope()

    fun reload() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = api.getPromotions()
                promotions = if (resp.isSuccessful) resp.body()?.promotions.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { reload() }

    Scaffold(
        topBar = { TopAppBar(title = { Text("Promotions") }) },
        floatingActionButton = {
            FloatingActionButton(onClick = { showCreate = true }) {
                Icon(Icons.Default.Add, contentDescription = "Create promotion")
            }
        },
    ) { padding ->
        Box(Modifier.padding(padding).fillMaxSize()) {
            when {
                loading -> SupplierLoadingState("Loading promotions…", "Supplier promos")
                error != null -> SupplierStatePane(
                    kind = SupplierStateKind.Error,
                    headline = "Promotions unavailable",
                    body = error!!,
                )
                promotions.isEmpty() -> SupplierStatePane(
                    kind = SupplierStateKind.Empty,
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
                                Text(
                                    "${promo.discountBps / 100.0}% · ${promo.scopeType} · ${promo.retailerScope}" +
                                        if (promo.isActive) "" else " · inactive",
                                )
                            },
                            trailingContent = {
                                if (promo.isActive) {
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
