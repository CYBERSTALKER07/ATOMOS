package com.pegasusx.retailer.ui.screens.dashboard.components

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
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.ui.components.RetailerMetricTile
import com.pegasusx.retailer.ui.theme.PegasusSpacing

@Composable
fun DashboardOverviewCard(
    activeOrderCount: Int,
    predictionCount: Int,
    recentProductCount: Int,
) {
    Surface(
        shape = MaterialTheme.shapes.large,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
    ) {
        Column(
            modifier = Modifier.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(
                    text = "Retail operations",
                    style = MaterialTheme.typography.headlineSmall,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = "Track deliveries, restock faster, and act on demand signals from one place.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                RetailerMetricTile(
                    label = "Active orders",
                    value = activeOrderCount.toString(),
                    modifier = Modifier.weight(1f),
                )
                RetailerMetricTile(
                    label = "Suggestions",
                    value = predictionCount.toString(),
                    modifier = Modifier.weight(1f),
                )
                RetailerMetricTile(
                    label = "Recent items",
                    value = recentProductCount.toString(),
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}
