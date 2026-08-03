package com.pegasusx.warehouse.ui.screens.analytics

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.DailyMetric
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import java.text.NumberFormat

@Composable
fun AnalyticsChartGrid(
    daily: List<DailyMetric>,
    formatter: NumberFormat,
) {
    if (daily.isNotEmpty()) {
        Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
            DailyRevenueChart(daily = daily, formatter = formatter)
        }
    }
}

@Composable
private fun DailyRevenueChart(
    daily: List<DailyMetric>,
    formatter: NumberFormat,
) {
    val maxRevenue = daily.maxOfOrNull { it.revenue }?.coerceAtLeast(1L) ?: 1L
    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
        Text("Daily Revenue", style = MaterialTheme.typography.titleMedium)
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(128.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.Bottom,
        ) {
            daily.forEach { day ->
                val fraction = day.revenue.toFloat() / maxRevenue.toFloat()
                Column(
                    modifier = Modifier.weight(1f),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height((fraction * 96f).dp.coerceAtLeast(4.dp))
                            .background(
                                MaterialTheme.colorScheme.primary,
                                MaterialTheme.shapes.extraSmall,
                            ),
                    )
                    Text(
                        text = day.date.takeLast(5),
                        style = MaterialTheme.typography.labelSmall,
                        maxLines = 1,
                    )
                }
            }
        }
        Text(
            text = "Peak day: ${formatter.format(daily.maxOf { it.revenue })} UZS",
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}
