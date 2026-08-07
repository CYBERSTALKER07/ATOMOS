package com.pegasusx.retailer.ui.screens.analytics

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.itemsIndexed

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

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
import androidx.compose.foundation.lazy.LazyRow

import androidx.compose.foundation.shape.CircleShape
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
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
import androidx.compose.material3.TextButton
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
import com.pegasusx.retailer.ui.screens.analytics.components.CategoryBreakdownCard
import com.pegasusx.retailer.ui.screens.analytics.components.DailySpendingChart
import com.pegasusx.retailer.ui.screens.analytics.components.KpiCard
import com.pegasusx.retailer.ui.screens.analytics.components.MonthlyTrendChart
import com.pegasusx.retailer.ui.screens.analytics.components.OrdersByStateCard
import com.pegasusx.retailer.ui.screens.analytics.components.TopSuppliersChart
import com.pegasusx.retailer.ui.screens.analytics.components.WeekdayPatternChart
import com.pegasusx.retailer.ui.screens.analytics.components.WeeklySpendCard
import com.pegasusx.retailer.ui.screens.analytics.components.formatAmount
import com.pegasusx.retailer.ui.screens.analytics.components.formatCompact
import com.pegasusx.retailer.ui.components.PegasusEmptyState
import com.pegasusx.retailer.ui.screens.orders.components.AiPlannedCard
import com.pegasusx.retailer.ui.theme.SquircleShape

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
                title = when (uiState.loadIssue) {
                    AnalyticsLoadIssue.RESTRICTED -> "Analytics Access Restricted"
                    AnalyticsLoadIssue.OFFLINE -> "Analytics Offline"
                    AnalyticsLoadIssue.ERROR -> "Analytics Unavailable"
                    null -> "No Analytics Data"
                },
                message = uiState.error ?: "Complete a few orders and your expense insights will appear here",
            )
        } else if (analytics != null) {
            LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                contentPadding = PaddingValues(vertical = 16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp)
    ) {
                if (uiState.loadIssue != null || uiState.isLoading) {
                    item {
                        val loadIssue = uiState.loadIssue
                        val syncMessage = when {
                            loadIssue != null -> uiState.error ?: uiState.syncMessage.orEmpty()
                            else -> "Syncing analytics..."
                        }
                        val containerColor = when (loadIssue) {
                            AnalyticsLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
                            AnalyticsLoadIssue.OFFLINE -> MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.5f)
                            AnalyticsLoadIssue.ERROR -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.35f)
                            null -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
                        }
                        val contentColor = when (loadIssue) {
                            AnalyticsLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.onErrorContainer
                            AnalyticsLoadIssue.OFFLINE -> MaterialTheme.colorScheme.onTertiaryContainer
                            AnalyticsLoadIssue.ERROR -> MaterialTheme.colorScheme.onErrorContainer
                            null -> MaterialTheme.colorScheme.onPrimaryContainer
                        }

                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp)
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
                                TextButton(onClick = viewModel::refresh) {
                                    Text("Retry", color = contentColor)
                                }
                            }
                        }
                    }
                }

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

                if (uiState.predictions.isNotEmpty()) {
                    item {
                        Text(
                            "AI Demand Signals",
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.Bold,
                            modifier = Modifier.padding(horizontal = 16.dp),
                        )
                    }
                    items(uiState.predictions, key = { it.id }) { forecast ->
                        Box(modifier = Modifier.padding(horizontal = 16.dp)) {
                            AiPlannedCard(
                                forecast = forecast,
                                onPreorder = { },
                                onCorrect = { },
                                onReject = { viewModel.dismissPrediction(forecast.id) },
                            )
                        }
                    }
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
                            title = stringResource(R.string.mobile_retailer_ui_this_month),
                            value = formatAmount(analytics.totalThisMonth),
                            subtitle = "Amount",
                            modifier = Modifier.weight(1f),
                        )
                        val delta = if (analytics.totalLastMonth > 0)
                            ((analytics.totalThisMonth - analytics.totalLastMonth) * 100 / analytics.totalLastMonth).toInt()
                        else 0
                        KpiCard(
                            title = stringResource(R.string.mobile_retailer_ui_vs_last_month),
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
                                                stringResource(R.string.mobile_retailer_ui_quantity_units, product.quantity),
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

                item(span = { GridItemSpan(maxLineSpan) }) { Spacer(modifier = Modifier.height(8.dp)) }

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
                                title = stringResource(R.string.mobile_retailer_ui_total_spent),
                                value = formatAmount(detailed.totalSpent),
                                subtitle = "${detailed.totalOrders} orders",
                                modifier = Modifier.weight(1f),
                            )
                            KpiCard(
                                title = stringResource(R.string.mobile_retailer_ui_avg_order),
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

                item(span = { GridItemSpan(maxLineSpan) }) { Spacer(modifier = Modifier.height(16.dp)) }
            }
        }
    }
}


