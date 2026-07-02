package com.pegasusx.supplier.ui.screens.planning

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import com.pegasusx.supplier.data.model.ForecastConfidence
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.formatBaselineSourceLabel

@Composable
fun ForecastConfidenceView(
    confidence: ForecastConfidence,
    updatedAt: String? = null,
    stale: Boolean = false,
    modifier: Modifier = Modifier,
) {
    val blocked = confidence.blockedReason != null || confidence.label == "insufficient_history"
    val low = confidence.lowUnits ?: 0L
    val high = confidence.highUnits ?: low
    val pct = confidence.confidencePct

    ElevatedCard(modifier.fillMaxWidth()) {
        Column(
            Modifier.padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
        ) {
            Text("Forecast confidence", style = MaterialTheme.typography.titleSmall)
            confidence.baselineSource?.let {
                Text(
                    formatBaselineSourceLabel(it),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.primary,
                )
            }
            if (blocked) {
                Text(
                    "Insufficient history — predictive forecast blocked",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.tertiary,
                )
            } else {
                Text(
                    "%,d – %,d units".format(low, high),
                    style = MaterialTheme.typography.titleMedium,
                )
            }
            if (pct != null && !blocked) {
                Text(
                    "$pct% confidence",
                    style = MaterialTheme.typography.bodySmall,
                    color = confidenceColor(pct),
                )
            }
            updatedAt?.let {
                Text(
                    buildString {
                        if (stale) append("Stale · ")
                        append("Updated $it")
                    },
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun confidenceColor(pct: Int): Color = when {
    pct >= 80 -> MaterialTheme.colorScheme.primary
    pct >= 60 -> MaterialTheme.colorScheme.tertiary
    else -> MaterialTheme.colorScheme.error
}
