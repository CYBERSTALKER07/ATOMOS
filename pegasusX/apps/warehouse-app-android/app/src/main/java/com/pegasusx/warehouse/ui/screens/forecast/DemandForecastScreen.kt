package com.pegasusx.warehouse.ui.screens.forecast

import androidx.compose.ui.res.stringResource

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
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
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
import com.pegasusx.warehouse.ui.components.ForecastConfidenceView
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.parseForecastConfidence
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
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back)) } } },
                actions = {
                    var expanded by remember { mutableStateOf(false) }
                    Box {
                        TextButton(onClick = { expanded = true }) {
                            Text(stringResource(R.string.mobile_warehouse_ui_horizond, horizon))
                        }
                        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                            listOf(7, 14, 30).forEach { days ->
                                DropdownMenuItem(
                                    text = { Text(stringResource(R.string.mobile_warehouse_ui_days_days, days)) },
                                    onClick = {
                                        horizon = days
                                        expanded = false
                                    },
                                )
                            }
                        }
                    }
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = stringResource(R.string.portal_page_orders_action_refresh))
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

    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        item(span = { GridItemSpan(maxLineSpan) }) {
            Text(
                "AI-powered stock recommendations from 4 data sources",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        forecastChartPanel(critical, urgent, normal)
        forecastSkuTable(forecast.products, fmt, forecast.generatedAt, forecast.forecastDays)
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

    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        items(forecast.series, key = DemandForecastDay::date) { day ->
            ElevatedCard(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(PegasusSpacing.lg)) {
                    Text(day.date, style = MaterialTheme.typography.titleMedium)
                    Spacer(Modifier.height(PegasusSpacing.xs))
                    Text(stringResource(R.string.mobile_warehouse_ui_projected_units_format, fmt.format(day.projectedUnits)))
                    Text(
                        stringResource(R.string.mobile_warehouse_ui_committed_format_pending_format_2, fmt.format(day.committedUnits), fmt.format(day.pendingConfirmationUnits)),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }
}



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
            demandBreakdown = insight.demandBreakdown,
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
