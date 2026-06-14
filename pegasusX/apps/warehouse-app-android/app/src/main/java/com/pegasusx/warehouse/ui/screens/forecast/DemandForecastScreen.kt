package com.pegasusx.warehouse.ui.screens.forecast

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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.DemandForecastDay
import com.pegasusx.warehouse.data.model.DemandForecastProduct
import com.pegasusx.warehouse.data.model.DemandForecastResponse
import com.pegasusx.warehouse.data.model.DemandForecastSources
import com.pegasusx.warehouse.data.model.ReplenishmentInsight
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

private enum class ForecastSegment { Products, Series }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DemandForecastScreen(
    api: WarehouseApi,
    onBack: (() -> Unit)? = null,
) {
    var horizon by remember { mutableIntStateOf(7) }
    var forecast by remember { mutableStateOf(DemandForecastResponse()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var segment by remember { mutableStateOf(ForecastSegment.Products) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getDemandForecast(days = horizon)
                if (resp.isSuccessful && resp.body() != null) {
                    var body = resp.body()!!
                    if (body.products.isEmpty()) {
                        val insightsResp = api.getReplenishmentInsights()
                        if (insightsResp.isSuccessful) {
                            val rows = insightsResp.body()?.resolved().orEmpty()
                            if (rows.isNotEmpty()) {
                                body = demandForecastFromInsights(rows, horizon)
                            }
                        }
                    }
                    forecast = body
                } else {
                    error = "Failed to load (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(horizon) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Demand Forecast") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back") } } },
                actions = {
                    var expanded by remember { mutableStateOf(false) }
                    Box {
                        TextButton(onClick = { expanded = true }) {
                            Text("${horizon}d")
                        }
                        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                            listOf(7, 14, 30).forEach { days ->
                                DropdownMenuItem(
                                    text = { Text("$days days") },
                                    onClick = {
                                        horizon = days
                                        expanded = false
                                    },
                                )
                            }
                        }
                    }
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            SingleChoiceSegmentedButtonRow(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
            ) {
                ForecastSegment.entries.forEachIndexed { index, entry ->
                    SegmentedButton(
                        selected = segment == entry,
                        onClick = { segment = entry },
                        shape = SegmentedButtonDefaults.itemShape(
                            index = index,
                            count = ForecastSegment.entries.size,
                        ),
                    ) {
                        Text(
                            when (entry) {
                                ForecastSegment.Products -> "Products"
                                ForecastSegment.Series -> "Series"
                            },
                        )
                    }
                }
            }

            when {
                loading -> Box(
                    Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center,
                ) { CircularProgressIndicator() }

                error != null -> Box(
                    Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center,
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(error!!, color = MaterialTheme.colorScheme.error)
                        Spacer(Modifier.height(PegasusSpacing.lg))
                        TextButton(onClick = { load() }) { Text("Retry") }
                    }
                }

                segment == ForecastSegment.Products -> ProductsForecastBody(
                    forecast = forecast,
                    fmt = fmt,
                )

                else -> SeriesForecastBody(
                    forecast = forecast,
                    fmt = fmt,
                )
            }
        }
    }
}

@Composable
private fun ProductsForecastBody(
    forecast: DemandForecastResponse,
    fmt: NumberFormat,
) {
    if (forecast.products.isEmpty()) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    "No product recommendations",
                    style = MaterialTheme.typography.titleMedium,
                )
                Spacer(Modifier.height(PegasusSpacing.xs))
                Text(
                    "Try another horizon or switch to Series for daily projection.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        return
    }

    val critical = forecast.products.count { it.priority.equals("CRITICAL", ignoreCase = true) }
    val urgent = forecast.products.count { it.priority.equals("URGENT", ignoreCase = true) }
    val normal = forecast.products.count { it.priority.equals("NORMAL", ignoreCase = true) }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        item {
            Text(
                "AI-powered stock recommendations from 4 data sources",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        item {
            Row(
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                modifier = Modifier.fillMaxWidth(),
            ) {
                ForecastSummaryCard(
                    title = "Critical",
                    count = critical,
                    subtitle = "< 2 days",
                    tint = MaterialTheme.colorScheme.error,
                    modifier = Modifier.weight(1f),
                )
                ForecastSummaryCard(
                    title = "Urgent",
                    count = urgent,
                    subtitle = "< 5 days",
                    tint = MaterialTheme.colorScheme.tertiary,
                    modifier = Modifier.weight(1f),
                )
                ForecastSummaryCard(
                    title = "Healthy",
                    count = normal,
                    subtitle = "5+ days",
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.weight(1f),
                )
            }
        }
        items(forecast.products, key = DemandForecastProduct::productId) { product ->
            ForecastProductCard(product = product, fmt = fmt)
        }
        forecast.generatedAt?.takeIf { it.isNotBlank() }?.let { generatedAt ->
            item {
                Text(
                    "Generated ${formatGeneratedAt(generatedAt)} · ${forecast.forecastDays}-day window",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun SeriesForecastBody(
    forecast: DemandForecastResponse,
    fmt: NumberFormat,
) {
    if (forecast.series.isEmpty()) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    "No series data",
                    style = MaterialTheme.typography.titleMedium,
                )
                Spacer(Modifier.height(PegasusSpacing.xs))
                Text(
                    "No daily demand projection for this window.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        return
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(forecast.series, key = DemandForecastDay::date) { day ->
            ElevatedCard(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(PegasusSpacing.lg)) {
                    Text(day.date, style = MaterialTheme.typography.titleMedium)
                    Spacer(Modifier.height(PegasusSpacing.xs))
                    Text("Projected units: ${fmt.format(day.projectedUnits)}")
                    Text(
                        "Committed: ${fmt.format(day.committedUnits)} · Pending: ${fmt.format(day.pendingConfirmationUnits)}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }
}

@Composable
private fun ForecastSummaryCard(
    title: String,
    count: Int,
    subtitle: String,
    tint: Color,
    modifier: Modifier = Modifier,
) {
    ElevatedCard(modifier = modifier) {
        Column(Modifier.padding(PegasusSpacing.md)) {
            Text(title, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Text(
                "$count",
                style = MaterialTheme.typography.headlineSmall,
                color = tint,
            )
            Text(subtitle, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
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
                    title = "Stock",
                    value = fmt.format(product.currentStock),
                    modifier = Modifier.weight(1f),
                )
                ForecastMetricColumn(
                    title = "Rec.",
                    value = fmt.format(product.recommendedQty),
                    modifier = Modifier.weight(1f),
                )
                ForecastMetricColumn(
                    title = "Stockout",
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

private fun demandForecastFromInsights(
    insights: List<ReplenishmentInsight>,
    horizon: Int,
): DemandForecastResponse {
    val products = insights.map { insight ->
        DemandForecastProduct(
            productId = insight.productId,
            productName = insight.productName,
            currentStock = insight.currentStock,
            recommendedQty = insight.reorderQuantity,
            daysUntilStockout = insight.daysUntilStockout.toDouble(),
            priority = mapInsightPriority(insight.urgency),
            unit = "VU",
            sources = DemandForecastSources(burnRate = insight.avgDailyVelocity),
        )
    }
    return DemandForecastResponse(
        forecastDays = horizon,
        products = products,
        generatedAt = insights.firstOrNull()?.createdAt,
    )
}

private fun mapInsightPriority(urgency: String): String {
    val u = urgency.uppercase(Locale.US)
    return when {
        u == "CRITICAL" || u == "HIGH" -> "CRITICAL"
        u == "URGENT" || u == "MEDIUM" -> "URGENT"
        else -> "NORMAL"
    }
}
