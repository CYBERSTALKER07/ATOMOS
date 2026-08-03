package com.pegasusx.warehouse.ui.screens.forecast

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridScope
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

fun LazyGridScope.forecastChartPanel(
    critical: Int,
    urgent: Int,
    normal: Int
) {
    item(span = { GridItemSpan(maxLineSpan) }) {
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
