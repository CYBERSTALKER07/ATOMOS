package com.pegasus.retailer.ui.screens.analytics

import androidx.compose.foundation.Canvas
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
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import com.pegasus.retailer.ui.theme.SoftSquircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.CalendarMonth
import androidx.compose.material.icons.outlined.ChevronLeft
import androidx.compose.material.icons.outlined.ChevronRight
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Edit
import androidx.compose.material.icons.rounded.Insights
import androidx.compose.material.icons.rounded.MoreVert
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.text.TextMeasurer
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
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
import com.pegasus.retailer.data.model.RetailerAnalytics
import com.pegasus.retailer.data.model.RetailerDayExpense
import com.pegasus.retailer.data.model.OrderStateCount
import com.pegasus.retailer.data.model.CategorySpend
import com.pegasus.retailer.data.model.DayOfWeekPattern
import com.pegasus.retailer.ui.components.PegasusEmptyState
import com.pegasus.retailer.ui.theme.SquircleShape
import java.text.NumberFormat
import java.util.Locale

// Health Connect palette
private val Purple600 = Color(0xFF6750A4)
private val Purple200 = Color(0xFFD0BCFF)
private val Green500 = Color(0xFF4CAF50)
private val GoalRed = Color(0xFFE91E63)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AnalyticsScreen(
    viewModel: AnalyticsViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    var selectedRange by remember { mutableStateOf("1M") }
    val ranges = listOf("7D", "1M", "Q1", "6M")

    PullToRefreshBox(
        isRefreshing = uiState.isLoading,
        onRefresh = viewModel::refresh,
        modifier = Modifier.fillMaxSize(),
    ) {
        val analytics = uiState.analytics
        if (analytics == null && !uiState.isLoading) {
            PegasusEmptyState(
                icon = Icons.Rounded.Insights,
                title = "No Analytics Data",
                message = "Complete a few orders and your expense insights will appear here",
            )
        } else if (analytics != null) {
            LazyColumn(
                contentPadding = PaddingValues(vertical = 16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                // ── Weekly Spend Tracker (Health Connect style) ──
                item {
                    WeeklySpendCard(
                        weekLabel = uiState.weekLabel,
                        avgPerDay = uiState.avgPerDayUzs,
                        daysOnBudget = uiState.daysOnBudget,
                        totalWeek = uiState.totalWeekUzs,
                        dailySpend = uiState.weeklySpend,
                        budgetGoal = uiState.weeklyBudgetUzs,
                        modifier = Modifier.padding(horizontal = 16.dp),
                    )
                }

                // Date Range Chips
                item {
                    LazyRow(
                        contentPadding = PaddingValues(horizontal = 16.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        items(ranges) { range ->
                            FilterChip(
                                selected = selectedRange == range,
                                onClick = { selectedRange = range },
                                label = {
                                    Text(range, style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold))
                                },
                                colors = FilterChipDefaults.filterChipColors(
                                    selectedContainerColor = MaterialTheme.colorScheme.onSurface,
                                    selectedLabelColor = MaterialTheme.colorScheme.surface,
                                ),
                            )
                        }
                    }
                }

                // KPI Cards
                item {
                    Row(
                        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        KpiCard(
                            title = "This Month",
                            value = formatAmount(analytics.totalThisMonth),
                            subtitle = "Amount",
                            modifier = Modifier.weight(1f),
                        )
                        val delta = if (analytics.totalLastMonth > 0)
                            ((analytics.totalThisMonth - analytics.totalLastMonth) * 100 / analytics.totalLastMonth).toInt()
                        else 0
                        KpiCard(
                            title = "vs Last Month",
                            value = if (delta >= 0) "+$delta%" else "$delta%",
                            subtitle = if (delta >= 0) "increase" else "decrease",
                            modifier = Modifier.weight(1f),
                        )
                    }
                }

                // Monthly Trend Chart (Line)
                if (analytics.monthlyExpenses.isNotEmpty()) {
                    item {
                        MonthlyTrendChart(analytics, modifier = Modifier.padding(horizontal = 16.dp))
                    }
                }

                // Top Suppliers (Bar)
                if (analytics.topSuppliers.isNotEmpty()) {
                    item {
                        TopSuppliersChart(analytics, modifier = Modifier.padding(horizontal = 16.dp))
                    }
                }

                // Top Products List
                if (analytics.topProducts.isNotEmpty()) {
                    item {
                        Surface(
                            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp),
                            shape = SquircleShape,
                            color = MaterialTheme.colorScheme.surface,
                            tonalElevation = 0.dp,
                        ) {
                            Column(modifier = Modifier.padding(16.dp)) {
                                Text(
                                    "Top Products",
                                    style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold),
                                )
                                Spacer(modifier = Modifier.height(12.dp))
                                analytics.topProducts.forEachIndexed { index, product ->
                                    Row(
                                        modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
                                        verticalAlignment = Alignment.CenterVertically,
                                    ) {
                                        Column(modifier = Modifier.weight(1f)) {
                                            Text(
                                                product.productName,
                                                style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium),
                                                maxLines = 1,
                                            )
                                            Text(
                                                "${product.quantity} units",
                                                style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                                            )
                                        }
                                        Text(
                                            formatAmount(product.total),
                                            style = MaterialTheme.typography.bodyMedium.copy(
                                                fontWeight = FontWeight.Medium,
                                                fontFamily = FontFamily.Monospace,
                                            ),
                                        )
                                    }
                                    if (index < analytics.topProducts.lastIndex) {
                                        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.3f))
                                    }
                                }
                            }
                        }
                    }
                }

                item { Spacer(modifier = Modifier.height(8.dp)) }

                // ── Advanced Analytics Section (from /v1/retailer/analytics/detailed) ──
                val detailed = uiState.detailed
                if (detailed != null) {
                    // Date range selector for detailed analytics
                    item {
                        val detailedRanges = listOf("7D", "14D", "30D", "90D", "6M", "1Y")
                        Column(modifier = Modifier.padding(horizontal = 16.dp)) {
                            Text(
                                "Advanced Insights",
                                style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                            )
                            Spacer(modifier = Modifier.height(8.dp))
                            LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                items(detailedRanges) { range ->
                                    FilterChip(
                                        selected = uiState.selectedRange == range,
                                        onClick = { viewModel.setRange(range) },
                                        label = {
                                            Text(range, style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold))
                                        },
                                        colors = FilterChipDefaults.filterChipColors(
                                            selectedContainerColor = MaterialTheme.colorScheme.primary,
                                            selectedLabelColor = MaterialTheme.colorScheme.onPrimary,
                                        ),
                                    )
                                }
                            }
                        }
                    }

                    // Summary KPIs
                    item {
                        Row(
                            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp),
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                        ) {
                            KpiCard(
                                title = "Total Spent",
                                value = formatAmount(detailed.totalSpent),
                                subtitle = "${detailed.totalOrders} orders",
                                modifier = Modifier.weight(1f),
                            )
                            KpiCard(
                                title = "Avg Order",
                                value = formatAmount(detailed.avgOrderValue),
                                subtitle = "per order",
                                modifier = Modifier.weight(1f),
                            )
                        }
                    }

                    // Daily Spending Line Chart
                    if (detailed.dailySpending.isNotEmpty()) {
                        item {
                            DailySpendingChart(
                                data = detailed.dailySpending,
                                modifier = Modifier.padding(horizontal = 16.dp),
                            )
                        }
                    }

                    // Orders by State (donut-style visualization)
                    if (detailed.ordersByState.isNotEmpty()) {
                        item {
                            OrdersByStateCard(
                                data = detailed.ordersByState,
                                modifier = Modifier.padding(horizontal = 16.dp),
                            )
                        }
                    }

                    // Category Breakdown (horizontal bars)
                    if (detailed.categoryBreakdown.isNotEmpty()) {
                        item {
                            CategoryBreakdownCard(
                                data = detailed.categoryBreakdown,
                                modifier = Modifier.padding(horizontal = 16.dp),
                            )
                        }
                    }

                    // Weekday Pattern
                    if (detailed.weekdayPattern.isNotEmpty()) {
                        item {
                            WeekdayPatternChart(
                                data = detailed.weekdayPattern,
                                modifier = Modifier.padding(horizontal = 16.dp),
                            )
                        }
                    }
                }

                item { Spacer(modifier = Modifier.height(16.dp)) }
            }
        }
    }
}

// ── Health Connect Style Weekly Spend Card ──

@Composable
private fun WeeklySpendCard(
    weekLabel: String,
    avgPerDay: Long,
    daysOnBudget: Int,
    totalWeek: Long,
    dailySpend: List<DailySpend>,
    budgetGoal: Long,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
        tonalElevation = 1.dp,
    ) {
        Column(modifier = Modifier.padding(20.dp)) {
            // Header row: icon + title + overflow
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    Icons.Rounded.Edit,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp),
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    "Spending",
                    style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.SemiBold),
                )
                Spacer(modifier = Modifier.weight(1f))
                Icon(
                    Icons.Rounded.MoreVert,
                    contentDescription = "More options",
                    modifier = Modifier.size(20.dp),
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Week navigation
            Row(
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    weekLabel,
                    style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.Medium),
                )
                Spacer(modifier = Modifier.weight(1f))
                IconButton(onClick = { }, modifier = Modifier.size(32.dp)) {
                    Icon(Icons.Outlined.ChevronLeft, "Previous week", modifier = Modifier.size(20.dp))
                }
                IconButton(onClick = { }, modifier = Modifier.size(32.dp)) {
                    Icon(Icons.Outlined.ChevronRight, "Next week", modifier = Modifier.size(20.dp))
                }
                IconButton(onClick = { }, modifier = Modifier.size(32.dp)) {
                    Icon(Icons.Outlined.CalendarMonth, "Calendar", modifier = Modifier.size(20.dp))
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            // Big KPI number
            Row(
                verticalAlignment = Alignment.Bottom,
            ) {
                Text(
                    formatCompact(avgPerDay),
                    style = MaterialTheme.typography.displaySmall.copy(
                        fontWeight = FontWeight.Bold,
                        letterSpacing = (-1).sp,
                    ),
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    "Per day (avg)",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(bottom = 4.dp),
                )
            }

            Text(
                "You stayed on budget $daysOnBudget days, and spent a total of ${formatAmount(totalWeek)}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            Spacer(modifier = Modifier.height(20.dp))

            // ── Bar Chart (Health Connect style) ──
            if (dailySpend.isNotEmpty()) {
                val textMeasurer = rememberTextMeasurer()
                HealthConnectBarChart(
                    dailySpend = dailySpend,
                    budgetGoal = budgetGoal,
                    textMeasurer = textMeasurer,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(220.dp),
                )
            }
        }
    }
}

@Composable
private fun HealthConnectBarChart(
    dailySpend: List<DailySpend>,
    budgetGoal: Long,
    textMeasurer: TextMeasurer,
    modifier: Modifier = Modifier,
) {
    val maxValue = (dailySpend.maxOf { it.amount } * 1.2f).toLong()
    val onSurfaceVariant = MaterialTheme.colorScheme.onSurfaceVariant

    Canvas(modifier = modifier) {
        val chartLeft = 40.dp.toPx()
        val chartRight = size.width - 8.dp.toPx()
        val chartTop = 8.dp.toPx()
        val chartBottom = size.height - 36.dp.toPx()
        val chartHeight = chartBottom - chartTop
        val chartWidth = chartRight - chartLeft

        val barCount = dailySpend.size
        val barSpacing = chartWidth / barCount
        val barWidth = barSpacing * 0.5f
        val cornerRadiusPx = 6.dp.toPx()

        // Y-axis labels
        val ySteps = listOf(0L, maxValue / 3, maxValue * 2 / 3, maxValue)
        for (step in ySteps) {
            val y = chartBottom - (step.toFloat() / maxValue * chartHeight)
            val label = "${step / 1_000}k"
            drawText(
                textMeasurer = textMeasurer,
                text = label,
                topLeft = Offset(0f, y - 6.dp.toPx()),
                style = TextStyle(
                    fontSize = 10.sp,
                    color = onSurfaceVariant.copy(alpha = 0.6f),
                ),
            )
        }

        // Budget goal line
        val goalY = chartBottom - (budgetGoal.toFloat() / maxValue * chartHeight)
        drawLine(
            color = GoalRed.copy(alpha = 0.7f),
            start = Offset(chartLeft, goalY),
            end = Offset(chartRight, goalY),
            strokeWidth = 1.5.dp.toPx(),
            pathEffect = PathEffect.dashPathEffect(floatArrayOf(8.dp.toPx(), 4.dp.toPx())),
        )

        // Goal label
        val goalLabel = "${budgetGoal / 1_000_000}M"
        drawText(
            textMeasurer = textMeasurer,
            text = goalLabel,
            topLeft = Offset(chartRight + 2.dp.toPx(), goalY - 8.dp.toPx()),
            style = TextStyle(
                fontSize = 10.sp,
                color = GoalRed,
                fontWeight = FontWeight.Bold,
            ),
        )

        // Bars + day labels
        for ((index, day) in dailySpend.withIndex()) {
            val barCenter = chartLeft + barSpacing * index + barSpacing / 2
            val barLeft = barCenter - barWidth / 2
            val barHeight = (day.amount.toFloat() / maxValue) * chartHeight
            val barTop = chartBottom - barHeight

            val onBudget = day.amount <= budgetGoal

            // Bar body — purple-ish below goal, green above goal
            if (onBudget) {
                // Solid purple bar
                drawRoundRect(
                    color = Purple600,
                    topLeft = Offset(barLeft, barTop),
                    size = Size(barWidth, barHeight),
                    cornerRadius = CornerRadius(cornerRadiusPx, cornerRadiusPx),
                )
            } else {
                // Purple portion (up to goal line)
                val goalHeight = (budgetGoal.toFloat() / maxValue) * chartHeight
                drawRoundRect(
                    color = Purple600,
                    topLeft = Offset(barLeft, chartBottom - goalHeight),
                    size = Size(barWidth, goalHeight),
                    cornerRadius = CornerRadius(0f, 0f),
                )
                // Green portion (above goal)
                val overHeight = barHeight - goalHeight
                drawRoundRect(
                    color = Green500,
                    topLeft = Offset(barLeft, barTop),
                    size = Size(barWidth, overHeight + cornerRadiusPx),
                    cornerRadius = CornerRadius(cornerRadiusPx, cornerRadiusPx),
                )
            }

            // Achievement badge (checkmark) for on-budget days
            if (onBudget) {
                val badgeRadius = 10.dp.toPx()
                val badgeCx = barCenter
                val badgeCy = barTop - badgeRadius - 4.dp.toPx()
                drawCircle(
                    color = Green500,
                    radius = badgeRadius,
                    center = Offset(badgeCx, badgeCy),
                )
                // Checkmark inside badge
                val checkSize = 7.dp.toPx()
                val path = androidx.compose.ui.graphics.Path().apply {
                    moveTo(badgeCx - checkSize * 0.35f, badgeCy + checkSize * 0.05f)
                    lineTo(badgeCx - checkSize * 0.05f, badgeCy + checkSize * 0.35f)
                    lineTo(badgeCx + checkSize * 0.4f, badgeCy - checkSize * 0.3f)
                }
                drawPath(
                    path = path,
                    color = Color.White,
                    style = androidx.compose.ui.graphics.drawscope.Stroke(
                        width = 1.8.dp.toPx(),
                        cap = androidx.compose.ui.graphics.StrokeCap.Round,
                        join = androidx.compose.ui.graphics.StrokeJoin.Round,
                    ),
                )
            }

            // Day label below bar
            drawText(
                textMeasurer = textMeasurer,
                text = day.dayLabel,
                topLeft = Offset(barCenter - 5.dp.toPx(), chartBottom + 8.dp.toPx()),
                style = TextStyle(
                    fontSize = 12.sp,
                    fontWeight = FontWeight.Medium,
                    color = onSurfaceVariant,
                    textAlign = TextAlign.Center,
                ),
            )
        }
    }
}

@Composable
private fun KpiCard(
    title: String,
    value: String,
    subtitle: String,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier,
        shape = SquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(
                title,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
            )
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                value,
                style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                maxLines = 1,
            )
            Spacer(modifier = Modifier.height(2.dp))
            Text(
                subtitle,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
            )
        }
    }
}

@Composable
private fun MonthlyTrendChart(analytics: RetailerAnalytics, modifier: Modifier = Modifier) {
    val modelProducer = remember { CartesianChartModelProducer() }
    val months = analytics.monthlyExpenses.map { it.shortMonth }

    LaunchedEffect(analytics) {
        modelProducer.runTransaction {
            lineSeries { series(analytics.monthlyExpenses.map { it.total }) }
        }
    }

    val bottomAxisFormatter = CartesianValueFormatter { _, value, _ ->
        months.getOrElse(value.toInt()) { "" }
    }

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = SquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(
                "Monthly Trend",
                style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold),
            )
            Spacer(modifier = Modifier.height(16.dp))
            CartesianChartHost(
                chart = rememberCartesianChart(
                    rememberLineCartesianLayer(
                        lineProvider = LineCartesianLayer.LineProvider.series(
                            LineCartesianLayer.rememberLine(
                                fill = LineCartesianLayer.LineFill.single(fill(Purple600)),
                            ),
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
private fun TopSuppliersChart(analytics: RetailerAnalytics, modifier: Modifier = Modifier) {
    val modelProducer = remember { CartesianChartModelProducer() }
    val names = analytics.topSuppliers.map { it.supplierName }

    LaunchedEffect(analytics) {
        modelProducer.runTransaction {
            columnSeries { series(analytics.topSuppliers.map { it.total }) }
        }
    }

    val bottomAxisFormatter = CartesianValueFormatter { _, value, _ ->
        names.getOrElse(value.toInt()) { "" }
    }

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = SquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(
                "Top Suppliers",
                style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold),
            )
            Spacer(modifier = Modifier.height(16.dp))
            CartesianChartHost(
                chart = rememberCartesianChart(
                    rememberColumnCartesianLayer(
                        columnProvider = ColumnCartesianLayer.ColumnProvider.series(
                            rememberLineComponent(
                                fill = fill(Purple600),
                                shape = CorneredShape.rounded(allPercent = 20),
                            ),
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

private fun formatAmount(value: Long): String {
    val formatter = NumberFormat.getNumberInstance(Locale.US)
    return "${formatter.format(value)}"
}

private fun formatCompact(value: Long): String {
    return when {
        value >= 1_000_000 -> "${value / 1_000_000}.${(value % 1_000_000) / 100_000}M"
        value >= 1_000 -> "${value / 1_000},${(value % 1_000) / 100}00"
        else -> "$value"
    }
}

// ── Advanced Analytics Composables ──

private val StateColors = mapOf(
    "COMPLETED" to Color(0xFF4CAF50),
    "ARRIVED" to Color(0xFF2196F3),
    "IN_TRANSIT" to Color(0xFFFF9800),
    "PENDING" to Color(0xFFFFC107),
    "LOADED" to Color(0xFF9C27B0),
    "CANCELLED" to Color(0xFFE91E63),
    "CANCELLED_BY_ADMIN" to Color(0xFFF44336),
)

@Composable
private fun DailySpendingChart(data: List<RetailerDayExpense>, modifier: Modifier = Modifier) {
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
private fun OrdersByStateCard(data: List<OrderStateCount>, modifier: Modifier = Modifier) {
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
private fun CategoryBreakdownCard(data: List<CategorySpend>, modifier: Modifier = Modifier) {
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
private fun WeekdayPatternChart(data: List<DayOfWeekPattern>, modifier: Modifier = Modifier) {
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
