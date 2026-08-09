package com.pegasusx.retailer.ui.screens.autoorder

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.itemsIndexed

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.slideInVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width

import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AutoAwesome
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material.icons.rounded.Sync
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.ui.components.DemandSourceChips
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.ui.theme.SquircleShape

import com.pegasusx.retailer.ui.screens.autoorder.components.ForecastRow
import com.pegasusx.retailer.ui.screens.autoorder.components.GlobalToggleCard
import com.pegasusx.retailer.ui.screens.autoorder.components.HeaderCard
import com.pegasusx.retailer.ui.screens.autoorder.components.HowItWorksCard
import com.pegasusx.retailer.ui.screens.autoorder.components.OverrideRow
import com.pegasusx.retailer.ui.screens.autoorder.components.SectionHeader
import androidx.compose.material.icons.rounded.ShoppingCart
import androidx.compose.material3.OutlinedButton
import com.pegasusx.retailer.R

@Composable
fun AutoOrderScreen(
    viewModel: AutoOrderViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()

    if (uiState.placeConfirmOpen) {
        AlertDialog(
            onDismissRequest = viewModel::dismissPlaceConfirm,
            title = { Text("Create real supplier orders?") },
            text = {
                Text(
                    "Place mode creates real procurement orders (AUTO_ORDER). " +
                        "Requires primary location geo, place permission, and " +
                        "AUTO_ORDER_PLACE_ENABLED on the server.",
                )
            },
            confirmButton = {
                TextButton(
                    onClick = viewModel::runAutoOrderPlace,
                    enabled = !uiState.running,
                ) { Text("Confirm place") }
            },
            dismissButton = {
                TextButton(onClick = viewModel::dismissPlaceConfirm) { Text("Cancel") }
            },
        )
    }

    // History/Fresh dialog — shown for any entity that has existing order history
    val pendingTarget = uiState.pendingEnableTarget
    if (pendingTarget != null) {
        val entityLabel = when (pendingTarget) {
            is EnableTarget.Global -> "global auto-order"
            is EnableTarget.Supplier -> "this supplier"
            is EnableTarget.Category -> "this category"
            is EnableTarget.Product -> "this product"
            is EnableTarget.Variant -> "this variant / SKU"
        }
        AlertDialog(
            onDismissRequest = viewModel::dismissEnableDialog,
            title = { Text("Use Previous Analytics?") },
            text = { Text(stringResource(R.string.mobile_retailer_ui_use_existing_order_history_for_entitylabel_or_start_fresh_starting_fresh, entityLabel)) },
            confirmButton = {
                TextButton(onClick = { viewModel.confirmEnable(useHistory = true) }) {
                    Text("Use History")
                }
            },
            dismissButton = {
                Row {
                    TextButton(onClick = viewModel::dismissEnableDialog) {
                        Text("Cancel")
                    }
                    TextButton(onClick = { viewModel.confirmEnable(useHistory = false) }) {
                        Text("Start Fresh")
                    }
                }
            },
        )
    }

    if (uiState.isLoading && uiState.settings == null) {
        Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(color = MaterialTheme.colorScheme.primary)
        }
        return
    }

    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxSize(),
        horizontalArrangement = Arrangement.spacedBy(14.dp)
    ) {
        val syncMessage = when {
            uiState.loadIssue != null -> uiState.error ?: uiState.syncMessage.orEmpty()
            uiState.isLoading -> "Syncing auto-order settings..."
            else -> null
        }

        if (syncMessage != null) {
            item {
                val loadIssue = uiState.loadIssue
                val containerColor = when (loadIssue) {
                    AutoOrderLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
                    AutoOrderLoadIssue.OFFLINE -> MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.5f)
                    AutoOrderLoadIssue.ERROR -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.35f)
                    null -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
                }
                val contentColor = when (loadIssue) {
                    AutoOrderLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.onErrorContainer
                    AutoOrderLoadIssue.OFFLINE -> MaterialTheme.colorScheme.onTertiaryContainer
                    AutoOrderLoadIssue.ERROR -> MaterialTheme.colorScheme.onErrorContainer
                    null -> MaterialTheme.colorScheme.onPrimaryContainer
                }

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(12.dp))
                        .background(containerColor)
                        .padding(horizontal = 12.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = syncMessage,
                        modifier = Modifier.weight(1f),
                        style = MaterialTheme.typography.labelMedium,
                        color = contentColor,
                    )
                    if (loadIssue != null) {
                        TextButton(onClick = viewModel::loadAll) {
                            Text("Retry", color = contentColor)
                        }
                    }
                }
            }
        }

        // ── Header ──
        item {
            HeaderCard(
                supplierCount = uiState.settings?.supplierOverrides?.size ?: 0,
                categoryCount = uiState.settings?.categoryOverrides?.size ?: 0,
                productCount = uiState.settings?.productOverrides?.size ?: 0,
                predictionCount = uiState.forecasts.size,
            )
        }

        // ── Execution mode ──
        item {
            Card(
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceContainer,
                ),
            ) {
                Column(
                    modifier = Modifier.fillMaxWidth().padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Text("Execution mode", style = MaterialTheme.typography.titleMedium)
                    Text(
                        "Global aggressiveness. Scope toggles below choose which SKUs. " +
                            "Disable at any scope blocks even when global is on. Shadow recommended.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Row(
                        Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        listOf("off" to "Off", "shadow" to "Shadow", "draft" to "Draft", "place" to "Place").forEach { (mode, label) ->
                            val selected = uiState.executionMode == mode
                            OutlinedButton(
                                onClick = { viewModel.setExecutionMode(mode) },
                                enabled = !uiState.running,
                                modifier = Modifier.weight(1f),
                            ) {
                                Text(
                                    label,
                                    style = MaterialTheme.typography.labelSmall,
                                    color = if (selected) {
                                        MaterialTheme.colorScheme.primary
                                    } else {
                                        MaterialTheme.colorScheme.onSurface
                                    },
                                )
                            }
                        }
                    }
                    uiState.settings?.shadowStats?.let { st ->
                        Text(
                            stringResource(R.string.mobile_retailer_ui_30d_wape_toint_accept_toint_2_proposalcount_proposals, (st.wape * 100).toInt(), (st.unmodifiedAcceptRate * 100).toInt(), st.proposalCount),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }
        }

        // ── Shadow inbox ──
        item {
            Card(
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceContainer,
                ),
            ) {
                Column(
                    modifier = Modifier.fillMaxWidth().padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Text("Shadow inbox", style = MaterialTheme.typography.titleMedium)
                    if (uiState.shadowProposals.isEmpty()) {
                        Text(
                            "No shadow proposals yet. Set mode to Shadow and run Shadow now.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    } else {
                        uiState.shadowProposals.take(8).forEach { p ->
                            Text(
                                stringResource(R.string.mobile_retailer_ui_sku_qty_proposedqty_ip_toint_rop_toint_2, p.sku, p.proposedQty, p.ip.toInt(), p.reorderPoint.toInt()),
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                }
            }
        }

        // ── Run worker (shadow + draft + place) ──
        item {
            Card(
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceContainer,
                ),
            ) {
                Column(
                    modifier = Modifier.fillMaxWidth().padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Text("Auto-order worker", style = MaterialTheme.typography.titleMedium)
                    Text(
                        "Shadow records proposals only. Draft stages cart lines. " +
                            "Place creates real supplier orders when the server flag is on.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Row(
                        Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        OutlinedButton(
                            onClick = viewModel::runAutoOrderShadow,
                            enabled = !uiState.running && uiState.executionMode != "off",
                            modifier = Modifier.weight(1f),
                        ) {
                            if (uiState.running && uiState.runningMode == "shadow") {
                                CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                                Spacer(Modifier.width(6.dp))
                                Text("…")
                            } else {
                                Text("Shadow")
                            }
                        }
                        OutlinedButton(
                            onClick = viewModel::runAutoOrderNow,
                            enabled = !uiState.running && uiState.executionMode != "off",
                            modifier = Modifier.weight(1f),
                        ) {
                            if (uiState.running && uiState.runningMode == "draft") {
                                CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                                Spacer(Modifier.width(6.dp))
                                Text("Drafting…")
                            } else {
                                Icon(Icons.Rounded.PlayArrow, contentDescription = null)
                                Spacer(Modifier.width(6.dp))
                                Text("Draft")
                            }
                        }
                        Button(
                            onClick = viewModel::openPlaceConfirm,
                            enabled = !uiState.running && uiState.executionMode != "off",
                            modifier = Modifier.weight(1f),
                        ) {
                            if (uiState.running && uiState.runningMode == "place") {
                                CircularProgressIndicator(
                                    modifier = Modifier.size(16.dp),
                                    strokeWidth = 2.dp,
                                    color = MaterialTheme.colorScheme.onPrimary,
                                )
                                Spacer(Modifier.width(6.dp))
                                Text("Placing…")
                            } else {
                                Icon(Icons.Rounded.ShoppingCart, contentDescription = null)
                                Spacer(Modifier.width(6.dp))
                                Text("Place")
                            }
                        }
                    }
                    uiState.lastRun?.let { run ->
                        val placedBit = if (run.placedLines > 0) " · placed ${run.placedLines}" else ""
                        val viaBit = run.candidateSource?.let { " · via $it" } ?: ""
                        Text(
                            "Latest: ${run.mode} · draft ${run.draftLines}$placedBit · ${run.status}$viaBit",
                            style = MaterialTheme.typography.labelMedium,
                        )
                        run.message?.let {
                            Text(it, style = MaterialTheme.typography.labelSmall)
                        }
                        run.placedOrders.take(5).forEach { po ->
                            Text(
                                "${po.orderId}${po.supplierId?.let { " · $it" } ?: ""} · ${po.lineCount} lines",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.primary,
                            )
                        }
                    }
                    if (uiState.runs.isNotEmpty()) {
                        Text("Last runs", style = MaterialTheme.typography.labelLarge)
                        uiState.runs.take(8).forEach { run ->
                            Row(
                                Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                val pBit = if (run.placedLines > 0) " p${run.placedLines}" else ""
                                Text(stringResource(R.string.mobile_retailer_ui_take_mode_ddraftlinespbit, run.scheduleBucket ?: run.startedAt.take(10), run.mode, run.draftLines, pBit),
                                    style = MaterialTheme.typography.bodySmall,
                                )
                                Text(
                                    run.status,
                                    style = MaterialTheme.typography.labelSmall,
                                    color = if (run.status == "OK" || run.status == "PARTIAL") {
                                        MaterialTheme.colorScheme.primary
                                    } else {
                                        MaterialTheme.colorScheme.tertiary
                                    },
                                )
                            }
                        }
                    } else if (!uiState.runsLoading) {
                        Text(
                            "No runs yet. Enable auto-order and use Draft or Place.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }
        }

        // ── Reorder suggestions + source chips ──
        item {
            Card(
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceContainer,
                ),
            ) {
                Column(
                    modifier = Modifier.fillMaxWidth().padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Text("Reorder suggestions", style = MaterialTheme.typography.titleMedium)
                    Text(
                        "Sell-through aware OPEN suggestions (Store POS / Wholesale)",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    if (uiState.reorderSuggestions.isEmpty()) {
                        Text(
                            "No OPEN suggestions yet. POS sell-through and demand batch populate this list.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    } else {
                        uiState.reorderSuggestions.take(12).forEach { row ->
                            Column(
                                Modifier
                                    .fillMaxWidth()
                                    .padding(vertical = 4.dp),
                                verticalArrangement = Arrangement.spacedBy(4.dp),
                            ) {
                                Text(
                                    stringResource(R.string.mobile_retailer_ui_sku_qty_suggestedqty, row.sku, row.suggestedQty) +
                                        if (row.currentStock > 0) " · stock ${row.currentStock}" else "",
                                    style = MaterialTheme.typography.bodyMedium,
                                )
                                DemandSourceChips(sources = row.sources)
                                if (row.sellThroughVelocity > 0) {
                                    Text(stringResource(R.string.mobile_retailer_ui_pos_vel_format_d, "%.1f".format(row.sellThroughVelocity)),
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                }
                            }
                            HorizontalDivider()
                        }
                    }
                }
            }
        }

        // ── Global Toggle ──
        item {
            GlobalToggleCard(
                globalEnabled = uiState.globalEnabled,
                onToggle = viewModel::onGlobalToggle,
                analyticsStartDate = uiState.settings?.analyticsStartDate,
            )
        }

        // ── Supplier Overrides ──
        val supplierOverrides = uiState.settings?.supplierOverrides.orEmpty()
        if (supplierOverrides.isNotEmpty()) {
            item {
                SectionHeader("Supplier Overrides")
            }
            itemsIndexed(supplierOverrides, key = { _, it -> "s-${it.supplierId}" }) { _, override ->
                OverrideRow(
                    label = override.supplierName ?: override.supplierId,
                    subtitle = "Supplier-level override",
                    enabled = override.enabled,
                    onToggle = { viewModel.toggleSupplier(override.supplierId, it) },
                )
            }
        }

        // ── Category Overrides ──
        val categoryOverrides = uiState.settings?.categoryOverrides.orEmpty()
        if (categoryOverrides.isNotEmpty()) {
            item {
                SectionHeader("Category Overrides")
            }
            itemsIndexed(categoryOverrides, key = { _, it -> "c-${it.categoryId}" }) { _, override ->
                OverrideRow(
                    label = override.categoryId,
                    subtitle = "Category-level override",
                    enabled = override.enabled,
                    onToggle = { viewModel.toggleCategory(override.categoryId, it) },
                )
            }
        }

        // ── Product Overrides ──
        val productOverrides = uiState.settings?.productOverrides.orEmpty()
        if (productOverrides.isNotEmpty()) {
            item {
                SectionHeader("Product Overrides")
            }
            itemsIndexed(productOverrides, key = { _, it -> "p-${it.productId}" }) { _, override ->
                OverrideRow(
                    label = override.productName ?: override.productId,
                    subtitle = "Product-level override",
                    enabled = override.enabled,
                    onToggle = { viewModel.toggleProduct(override.productId, it) },
                )
            }
        }

        // ── Variant Overrides ──
        val variantOverrides = uiState.settings?.variantOverrides.orEmpty()
        if (variantOverrides.isNotEmpty()) {
            item {
                SectionHeader("Variant / SKU Overrides")
            }
            itemsIndexed(variantOverrides, key = { _, it -> "v-${it.skuId}" }) { _, override ->
                OverrideRow(
                    label = override.skuLabel ?: override.skuId,
                    subtitle = "Variant / SKU override",
                    enabled = override.enabled,
                    onToggle = { viewModel.toggleVariant(override.skuId, it) },
                )
            }
        }

        // ── Active Predictions ──
        if (uiState.forecasts.isNotEmpty()) {
            item {
                SectionHeader("Active Predictions")
            }
            itemsIndexed(uiState.forecasts, key = { _, it -> "f-${it.id}" }) { _, forecast ->
                ForecastRow(forecast)
            }
        }

        // ── How It Works ──
        item {
            HowItWorksCard()
        }

        // Bottom spacing
        item {
            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}
