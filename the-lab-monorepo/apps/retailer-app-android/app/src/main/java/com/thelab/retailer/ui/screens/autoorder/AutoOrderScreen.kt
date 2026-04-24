package com.thelab.retailer.ui.screens.autoorder

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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
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
import com.thelab.retailer.data.model.DemandForecast
import com.thelab.retailer.ui.theme.SoftSquircleShape
import com.thelab.retailer.ui.theme.SquircleShape

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

    if (uiState.isLoading) {
        Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(color = MaterialTheme.colorScheme.primary)
        }
        return
    }

    LazyColumn(
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxSize(),
    ) {
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

@Composable
private fun HeaderCard(supplierCount: Int, categoryCount: Int, productCount: Int, predictionCount: Int) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.primary,
    ) {
        Column(modifier = Modifier.padding(20.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.AutoAwesome, contentDescription = null, tint = MaterialTheme.colorScheme.onPrimary, modifier = Modifier.size(24.dp))
                Spacer(modifier = Modifier.width(10.dp))
                Column {
                    Text("Empathy Engine", style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold), color = MaterialTheme.colorScheme.onPrimary)
                    Text("Auto-order intelligence with 5-level control", style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp), color = MaterialTheme.colorScheme.onPrimary.copy(alpha = 0.7f))
                }
            }
            Spacer(modifier = Modifier.height(16.dp))
            Row(modifier = Modifier.fillMaxWidth()) {
                HeaderStat(value = "$supplierCount", label = "Suppliers", modifier = Modifier.weight(1f))
                HeaderStat(value = "$categoryCount", label = "Categories", modifier = Modifier.weight(1f))
                HeaderStat(value = "$productCount", label = "Products", modifier = Modifier.weight(1f))
                HeaderStat(value = "$predictionCount", label = "Predictions", modifier = Modifier.weight(1f))
            }
        }
    }
}

@Composable
private fun HeaderStat(value: String, label: String, modifier: Modifier = Modifier) {
    Column(modifier = modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        Text(value, style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold), color = MaterialTheme.colorScheme.onPrimary)
        Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onPrimary.copy(alpha = 0.6f))
    }
}

@Composable
private fun GlobalToggleCard(
    globalEnabled: Boolean,
    onToggle: (Boolean) -> Unit,
    analyticsStartDate: String?,
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Box(
                    modifier = Modifier
                        .size(40.dp)
                        .clip(SquircleShape)
                        .background(
                            if (globalEnabled) MaterialTheme.colorScheme.primary.copy(alpha = 0.12f)
                            else MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f),
                        ),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        Icons.Rounded.Sync,
                        contentDescription = null,
                        modifier = Modifier.size(20.dp),
                        tint = if (globalEnabled) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                }
                Spacer(modifier = Modifier.width(12.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Text("Global Auto-Order", style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.SemiBold))
                    Text(
                        "Auto-order everything from all suppliers",
                        style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                    )
                }
                Switch(
                    checked = globalEnabled,
                    onCheckedChange = onToggle,
                    colors = SwitchDefaults.colors(
                        checkedTrackColor = MaterialTheme.colorScheme.primary,
                        checkedThumbColor = MaterialTheme.colorScheme.onPrimary,
                    ),
                )
            }

            AnimatedVisibility(visible = globalEnabled) {
                Column {
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        "Global auto-order active. Overrides all supplier/product settings.",
                        style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                        color = MaterialTheme.colorScheme.primary.copy(alpha = 0.7f),
                    )
                }
            }

            if (analyticsStartDate != null) {
                Spacer(modifier = Modifier.height(6.dp))
                Text(
                    "Analytics since: $analyticsStartDate",
                    style = MaterialTheme.typography.bodySmall.copy(fontSize = 10.sp),
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.35f),
                )
            }
        }
    }
}

@Composable
private fun SectionHeader(title: String) {
    Text(
        title,
        style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.Bold),
        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
        modifier = Modifier.padding(start = 4.dp, top = 4.dp),
    )
}

@Composable
private fun OverrideRow(
    label: String,
    subtitle: String,
    enabled: Boolean,
    onToggle: (Boolean) -> Unit,
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .shadow(2.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.04f), spotColor = Color.Black.copy(alpha = 0.04f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(label, style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium), maxLines = 1)
                Text(subtitle, style = MaterialTheme.typography.bodySmall.copy(fontSize = 10.sp), color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f))
            }
            Switch(
                checked = enabled,
                onCheckedChange = onToggle,
                colors = SwitchDefaults.colors(
                    checkedTrackColor = MaterialTheme.colorScheme.primary,
                    checkedThumbColor = MaterialTheme.colorScheme.onPrimary,
                ),
            )
        }
    }
}

@Composable
private fun ForecastRow(forecast: DemandForecast) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .shadow(2.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.04f), spotColor = Color.Black.copy(alpha = 0.04f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // Confidence ring
            Box(contentAlignment = Alignment.Center) {
                CircularProgressIndicator(
                    progress = { forecast.confidence.toFloat() },
                    modifier = Modifier.size(36.dp),
                    strokeWidth = 2.5.dp,
                    trackColor = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.2f),
                    color = when {
                        forecast.confidence >= 0.8 -> Color(0xFF4CAF50)
                        forecast.confidence >= 0.6 -> Color(0xFFFF9800)
                        else -> Color(0xFFF44336)
                    },
                )
                Text(
                    forecast.confidencePercent,
                    style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.Bold),
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                )
            }
            Spacer(modifier = Modifier.width(12.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(forecast.productName, style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium), maxLines = 1)
                Text(
                    "Order by ${forecast.suggestedOrderDate}",
                    style = MaterialTheme.typography.bodySmall.copy(fontSize = 10.sp),
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                )
            }
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text("${forecast.predictedQuantity}", style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold), color = MaterialTheme.colorScheme.primary)
                Text("units", style = MaterialTheme.typography.labelSmall.copy(fontSize = 8.sp), color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f))
            }
        }
    }
}

@Composable
private fun HowItWorksCard() {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .shadow(2.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.04f), spotColor = Color.Black.copy(alpha = 0.04f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.Info, contentDescription = null, modifier = Modifier.size(16.dp), tint = MaterialTheme.colorScheme.primary)
                Spacer(modifier = Modifier.width(8.dp))
                Text("How It Works", style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold))
            }
            Spacer(modifier = Modifier.height(12.dp))
            ExplainerStep(num = "1", text = "The AI analyzes your purchase patterns even when auto-order is off")
            ExplainerStep(num = "2", text = "When you enable, choose to use your history or start fresh")
            ExplainerStep(num = "3", text = "Starting fresh requires at least 2 orders per product")
            ExplainerStep(num = "4", text = "Overrides hierarchy: Variant > Product > Category > Supplier > Global")
        }
    }
}

@Composable
private fun ExplainerStep(num: String, text: String) {
    Row(modifier = Modifier.padding(vertical = 3.dp), verticalAlignment = Alignment.Top) {
        Box(
            modifier = Modifier
                .size(20.dp)
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.primary),
            contentAlignment = Alignment.Center,
        ) {
            Text(num, style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold, fontSize = 10.sp), color = MaterialTheme.colorScheme.onPrimary)
        }
        Spacer(modifier = Modifier.width(8.dp))
        Text(text, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
    }
}
