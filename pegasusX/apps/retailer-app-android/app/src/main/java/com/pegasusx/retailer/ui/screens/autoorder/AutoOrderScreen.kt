package com.pegasusx.retailer.ui.screens.autoorder

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
import androidx.compose.material.icons.rounded.Sync
import androidx.compose.material3.AlertDialog
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
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.ui.theme.SquircleShape

import com.pegasusx.retailer.ui.screens.autoorder.components.ForecastRow
import com.pegasusx.retailer.ui.screens.autoorder.components.GlobalToggleCard
import com.pegasusx.retailer.ui.screens.autoorder.components.HeaderCard
import com.pegasusx.retailer.ui.screens.autoorder.components.HowItWorksCard
import com.pegasusx.retailer.ui.screens.autoorder.components.OverrideRow
import com.pegasusx.retailer.ui.screens.autoorder.components.SectionHeader

@Composable
fun AutoOrderScreen(
    viewModel: AutoOrderViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()

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
            text = { Text("Use existing order history for $entityLabel, or start fresh? Starting fresh requires at least 2 orders.") },
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
