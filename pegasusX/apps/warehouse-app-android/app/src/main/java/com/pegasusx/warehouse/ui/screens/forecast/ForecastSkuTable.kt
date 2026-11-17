package com.pegasusx.warehouse.ui.screens.forecast

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridScope
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextOverflow
import com.pegasusx.warehouse.data.model.DemandForecastProduct
import com.pegasusx.warehouse.ui.components.ForecastConfidenceView
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.parseForecastConfidence
import java.text.NumberFormat
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale
import com.pegasusx.warehouse.R

fun LazyGridScope.forecastSkuTable(
    products: List<DemandForecastProduct>,
    fmt: NumberFormat,
    generatedAt: String?,
    forecastDays: Int
) {
    items(products, key = DemandForecastProduct::productId) { product ->
        ForecastProductCard(product = product, fmt = fmt)
    }
    generatedAt?.takeIf { it.isNotBlank() }?.let { generated ->
        item(span = { GridItemSpan(maxLineSpan) }) {
            Text(
                stringResource(R.string.mobile_warehouse_ui_generated_formatgeneratedat_forecastdays_day_window, formatGeneratedAt(generated), forecastDays),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun ForecastProductCard(
    product: DemandForecastProduct,
    fmt: NumberFormat,
) {
    val displayName = product.productName.ifBlank {
        product.productId.take(8)
    }
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(
            Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    displayName,
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    product.priority,
                    style = MaterialTheme.typography.labelMedium,
                    color = priorityColor(product.priority),
                )
            }
            Row(modifier = Modifier.fillMaxWidth()) {
                ForecastMetricColumn(
                    title = stringResource(R.string.portal_nav_stock),
                    value = fmt.format(product.currentStock),
                    modifier = Modifier.weight(1f),
                )
                ForecastMetricColumn(
                    title = stringResource(R.string.mobile_warehouse_ui_rec),
                    value = fmt.format(product.recommendedQty),
                    modifier = Modifier.weight(1f),
                )
                ForecastMetricColumn(
                    title = stringResource(R.string.warehouse_portal_forecast_forecast_sku_table_text_stockout),
                    value = String.format(Locale.US, "%.1fd", product.daysUntilStockout),
                    valueColor = stockoutColor(product.daysUntilStockout),
                    modifier = Modifier.weight(1f),
                )
            }
            Row(modifier = Modifier.fillMaxWidth()) {
                SourceChip("In", fmt.format(product.sources.incomingOrders), Modifier.weight(1f))
                SourceChip("AI", fmt.format(product.sources.aiPrediction), Modifier.weight(1f))
                SourceChip("Pre", fmt.format(product.sources.preOrders), Modifier.weight(1f))
                SourceChip(
                    "Burn",
                    String.format(Locale.US, "%.1f", product.sources.burnRate),
                    Modifier.weight(1f),
                )
            }
            parseForecastConfidence(product.demandBreakdown)?.let { confidence ->
                ForecastConfidenceView(confidence = confidence, compact = true)
            }
        }
    }
}

@Composable
private fun ForecastMetricColumn(
    title: String,
    value: String,
    modifier: Modifier = Modifier,
    valueColor: Color = MaterialTheme.colorScheme.onSurface,
) {
    Column(modifier = modifier) {
        Text(title, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, style = MaterialTheme.typography.bodyMedium, fontFamily = FontFamily.Monospace, color = valueColor)
    }
}

@Composable
private fun SourceChip(label: String, value: String, modifier: Modifier = Modifier) {
    Column(modifier = modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, style = MaterialTheme.typography.labelMedium, fontFamily = FontFamily.Monospace)
    }
}

@Composable
private fun priorityColor(priority: String): Color = when (priority.uppercase(Locale.US)) {
    "CRITICAL" -> MaterialTheme.colorScheme.error
    "URGENT" -> MaterialTheme.colorScheme.tertiary
    else -> MaterialTheme.colorScheme.onSurfaceVariant
}

@Composable
private fun stockoutColor(days: Double): Color = when {
    days < 2 -> MaterialTheme.colorScheme.error
    days < 5 -> MaterialTheme.colorScheme.tertiary
    else -> MaterialTheme.colorScheme.onSurface
}

private fun formatGeneratedAt(raw: String): String = runCatching {
    val instant = Instant.parse(raw)
    DateTimeFormatter.ofPattern("MMM d, HH:mm")
        .withZone(ZoneId.systemDefault())
        .format(instant)
}.getOrDefault(raw)
