package com.pegasusx.warehouse.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.util.ForecastConfidenceData
import com.pegasusx.warehouse.util.formatSourceBadge
import java.util.Locale

@Composable
fun ForecastConfidenceView(
    confidence: ForecastConfidenceData,
    modifier: Modifier = Modifier,
    compact: Boolean = false,
) {
    if (compact) {
        Row(
            modifier = modifier,
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            if (confidence.blocked) {
                Text(
                    "Insufficient history",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.tertiary,
                )
            } else {
                val low = confidence.lowUnits ?: 0L
                val high = confidence.highUnits ?: low
                Text(
                    "${formatUnits(low)} – ${formatUnits(high)}",
                    style = MaterialTheme.typography.labelSmall,
                    fontFamily = FontFamily.Monospace,
                )
            }
            confidence.baselineSource?.takeIf { it.isNotBlank() }?.let { source ->
                SourceBadge(formatSourceBadge(source))
            }
            if (confidence.seasonalActive) {
                Text(
                    "Seasonal",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.tertiary,
                )
            }
        }
        return
    }

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = MaterialTheme.shapes.medium,
        tonalElevation = 1.dp,
    ) {
        Column(
            Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text("Forecast confidence", style = MaterialTheme.typography.labelMedium)
                confidence.baselineSource?.takeIf { it.isNotBlank() }?.let { source ->
                    SourceBadge(formatSourceBadge(source))
                }
            }
            if (confidence.blocked) {
                Text(
                    "Insufficient history — predictive forecast blocked",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.tertiary,
                )
            } else {
                val low = confidence.lowUnits ?: 0L
                val high = confidence.highUnits ?: low
                Text(
                    "${formatUnits(low)} – ${formatUnits(high)} units",
                    style = MaterialTheme.typography.titleSmall,
                    fontFamily = FontFamily.Monospace,
                )
            }
            confidence.confidencePct?.takeIf { !confidence.blocked }?.let { pct ->
                Text(
                    "$pct% confidence",
                    style = MaterialTheme.typography.bodySmall,
                    color = confidenceColor(pct),
                )
            }
            if (confidence.seasonalActive) {
                Text(
                    "Seasonal template active",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.tertiary,
                )
            }
        }
    }
}

@Composable
private fun SourceBadge(label: String) {
    Surface(
        shape = MaterialTheme.shapes.extraSmall,
        color = MaterialTheme.colorScheme.surfaceVariant,
    ) {
        Text(
            label.uppercase(Locale.US),
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
            style = MaterialTheme.typography.labelSmall,
        )
    }
}

@Composable
private fun confidenceColor(pct: Int) = when {
    pct >= 80 -> MaterialTheme.colorScheme.primary
    pct >= 60 -> MaterialTheme.colorScheme.tertiary
    else -> MaterialTheme.colorScheme.error
}

private fun formatUnits(value: Long): String = String.format(Locale.US, "%,d", value)
