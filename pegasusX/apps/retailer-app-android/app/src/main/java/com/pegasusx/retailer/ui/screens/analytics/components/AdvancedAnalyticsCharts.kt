package com.pegasusx.retailer.ui.screens.analytics.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.patrykandpatrick.vico.compose.cartesian.CartesianChartHost
import com.patrykandpatrick.vico.compose.cartesian.axis.rememberBottom
import com.patrykandpatrick.vico.compose.cartesian.axis.rememberStart
import com.patrykandpatrick.vico.compose.cartesian.layer.rememberColumnCartesianLayer
import com.patrykandpatrick.vico.compose.cartesian.layer.rememberLine
import com.patrykandpatrick.vico.compose.cartesian.layer.rememberLineCartesianLayer
import com.patrykandpatrick.vico.compose.cartesian.rememberCartesianChart
import com.patrykandpatrick.vico.compose.common.component.rememberLineComponent
import com.patrykandpatrick.vico.compose.common.fill
import com.patrykandpatrick.vico.core.cartesian.axis.HorizontalAxis
import com.patrykandpatrick.vico.core.cartesian.axis.VerticalAxis
import com.patrykandpatrick.vico.core.cartesian.data.CartesianChartModelProducer
import com.patrykandpatrick.vico.core.cartesian.data.CartesianValueFormatter
import com.patrykandpatrick.vico.core.cartesian.data.columnSeries
import com.patrykandpatrick.vico.core.cartesian.data.lineSeries
import com.patrykandpatrick.vico.core.cartesian.layer.ColumnCartesianLayer
import com.patrykandpatrick.vico.core.cartesian.layer.LineCartesianLayer
import com.patrykandpatrick.vico.core.common.shape.CorneredShape
import com.pegasusx.retailer.data.model.CategorySpend
import com.pegasusx.retailer.data.model.DayOfWeekPattern
import com.pegasusx.retailer.data.model.OrderStateCount
import com.pegasusx.retailer.data.model.RetailerDayExpense
import com.pegasusx.retailer.ui.theme.SquircleShape

@Composable
fun DailySpendingChart(data: List<RetailerDayExpense>, modifier: Modifier = Modifier) {
    val modelProducer = remember { CartesianChartModelProducer() }
    val dates = data.map { it.date.takeLast(5) }
    LaunchedEffect(data) {
        modelProducer.runTransaction { lineSeries { series(data.map { it.total }) } }
    }
    val bottomAxisFormatter = CartesianValueFormatter { _, value, _ -> dates.getOrElse(value.toInt()) { "" } }

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = SquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text("Daily Spending", style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold))
            Spacer(modifier = Modifier.height(16.dp))
            CartesianChartHost(
                chart = rememberCartesianChart(
                    rememberLineCartesianLayer(
                        lineProvider = LineCartesianLayer.LineProvider.series(
                            LineCartesianLayer.rememberLine(fill = LineCartesianLayer.LineFill.single(fill(Purple600)))
                        ),
                    ),
                    startAxis = VerticalAxis.rememberStart(),
                    bottomAxis = HorizontalAxis.rememberBottom(valueFormatter = bottomAxisFormatter),
                ),
                modelProducer = modelProducer,
                modifier = Modifier.fillMaxWidth().height(200.dp),
            )
        }
    }
}

@Composable
fun OrdersByStateCard(data: List<OrderStateCount>, modifier: Modifier = Modifier) {
    val total = data.sumOf { it.count }.toFloat()

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = SquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text("Orders by Status", style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold))
            Spacer(modifier = Modifier.height(12.dp))
            data.forEach { item ->
                val fraction = if (total > 0) item.count / total else 0f
                val color = StateColors[item.state] ?: MaterialTheme.colorScheme.outlineVariant
                Row(
                    modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Box(
                        modifier = Modifier.size(10.dp).clip(CircleShape).background(color),
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        item.state.replace("_", " "),
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.weight(1f),
                    )
                    // Progress bar
                    Box(
                        modifier = Modifier.weight(2f).height(8.dp)
                            .clip(RoundedCornerShape(4.dp))
                            .background(MaterialTheme.colorScheme.surfaceVariant),
                    ) {
                        Box(
                            modifier = Modifier.fillMaxWidth(fraction).height(8.dp)
                                .clip(RoundedCornerShape(4.dp))
                                .background(color),
                        )
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        "${item.count}",
                        style = MaterialTheme.typography.labelSmall.copy(fontFamily = FontFamily.Monospace),
                    )
                }
            }
        }
    }
}

@Composable
fun CategoryBreakdownCard(data: List<CategorySpend>, modifier: Modifier = Modifier) {
    val maxTotal = data.maxOfOrNull { it.total } ?: 1L
    val categoryColors = listOf(
        Color(0xFF6750A4), Color(0xFF4CAF50), Color(0xFF2196F3),
        Color(0xFFFF9800), Color(0xFFE91E63), Color(0xFF9C27B0),
        Color(0xFF00BCD4), Color(0xFFFF5722), Color(0xFF607D8B), Color(0xFF795548),
    )

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = SquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text("Spending by Category", style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold))
            Spacer(modifier = Modifier.height(12.dp))
            data.forEachIndexed { index, item ->
                val fraction = item.total.toFloat() / maxTotal
                val color = categoryColors[index % categoryColors.size]
                Row(
                    modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        item.category,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        modifier = Modifier.width(80.dp),
                    )
                    Box(
                        modifier = Modifier.weight(1f).height(12.dp)
                            .clip(RoundedCornerShape(6.dp))
                            .background(MaterialTheme.colorScheme.surfaceVariant),
                    ) {
                        Box(
                            modifier = Modifier.fillMaxWidth(fraction).height(12.dp)
                                .clip(RoundedCornerShape(6.dp))
                                .background(color),
                        )
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        formatAmount(item.total),
                        style = MaterialTheme.typography.labelSmall.copy(fontFamily = FontFamily.Monospace),
                    )
                }
            }
        }
    }
}

@Composable
fun WeekdayPatternChart(data: List<DayOfWeekPattern>, modifier: Modifier = Modifier) {
    val modelProducer = remember { CartesianChartModelProducer() }
    val days = data.map { it.weekday.take(3) }
    LaunchedEffect(data) {
        modelProducer.runTransaction { columnSeries { series(data.map { it.count }) } }
    }
    val bottomAxisFormatter = CartesianValueFormatter { _, value, _ -> days.getOrElse(value.toInt()) { "" } }

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = SquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text("Ordering Pattern by Day", style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold))
            Spacer(modifier = Modifier.height(16.dp))
            CartesianChartHost(
                chart = rememberCartesianChart(
                    rememberColumnCartesianLayer(
                        columnProvider = ColumnCartesianLayer.ColumnProvider.series(
                            rememberLineComponent(fill = fill(Purple600), shape = CorneredShape.rounded(allPercent = 20))
                        ),
                    ),
                    startAxis = VerticalAxis.rememberStart(),
                    bottomAxis = HorizontalAxis.rememberBottom(valueFormatter = bottomAxisFormatter),
                ),
                modelProducer = modelProducer,
                modifier = Modifier.fillMaxWidth().height(200.dp),
            )
            Spacer(modifier = Modifier.height(8.dp))
            // Avg per day
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly,
            ) {
                data.forEach { d ->
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(d.weekday.take(3), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                        Text(formatCompact(d.avg), style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold))
                    }
                }
            }
        }
    }
}
