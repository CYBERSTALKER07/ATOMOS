package com.pegasusx.factory.ui.screens.loadingbay.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun BayOverviewCard(
    readyCount: Int,
    loadingCount: Int,
    dispatchedCount: Int,
) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                text = "Loading bay flow",
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = "Track approved transfers, active loading work, and dispatched volume from one queue.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FactoryMetricTile("Ready", readyCount.toString(), Modifier.weight(1f))
                FactoryMetricTile("Loading", loadingCount.toString(), Modifier.weight(1f))
                FactoryMetricTile("Out", dispatchedCount.toString(), Modifier.weight(1f))
            }
        }
    }
}
