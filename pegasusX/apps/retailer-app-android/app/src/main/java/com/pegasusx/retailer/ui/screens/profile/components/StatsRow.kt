package com.pegasusx.retailer.ui.screens.profile.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.retailer.ui.components.RetailerMetricTile
import com.pegasusx.retailer.ui.theme.PegasusSpacing

@Composable
fun StatsRow(orderCount: Int, totalSpent: Long) {
    val spentDisplay = if (totalSpent >= 1000) "$${String.format("%.1f", totalSpent / 1000.0)}k" else "$$totalSpent"
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        RetailerMetricTile(
            label = "Orders",
            value = "$orderCount",
            modifier = Modifier.weight(1f),
        )
        RetailerMetricTile(
            label = "Spent",
            value = spentDisplay,
            modifier = Modifier.weight(1f),
        )
    }
}
