package com.pegasusx.retailer.ui.screens.analytics.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.CalendarMonth
import androidx.compose.material.icons.outlined.ChevronLeft
import androidx.compose.material.icons.outlined.ChevronRight
import androidx.compose.material.icons.rounded.Edit
import androidx.compose.material.icons.rounded.MoreVert
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import com.pegasusx.retailer.ui.screens.analytics.DailySpend
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.text.TextMeasurer
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.R

// ── Health Connect Style Weekly Spend Card ──

@Composable
fun WeeklySpendCard(
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
                    contentDescription = stringResource(R.string.retailer_desktop_orders_text_more_options),
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
                stringResource(R.string.mobile_retailer_ui_you_stayed_on_budget_daysonbudget_days_and_spent_a_total_of_formatamount, daysOnBudget, formatAmount(totalWeek)),
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
    val ySteps = listOf(0L, maxValue / 3, maxValue * 2 / 3, maxValue)
    val yLabels = ySteps.map { stringResource(R.string.mobile_retailer_ui_step_1_000k, it / 1_000) }

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
        ySteps.forEachIndexed { index, step ->
            val y = chartBottom - (step.toFloat() / maxValue * chartHeight)
            val label = yLabels[index]
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
