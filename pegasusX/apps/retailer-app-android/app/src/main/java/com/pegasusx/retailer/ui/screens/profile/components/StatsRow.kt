package com.pegasusx.retailer.ui.screens.profile.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.retailer.ui.components.RetailerMetricTile
import com.pegasusx.retailer.ui.theme.PegasusSpacing
import com.pegasusx.retailer.R

@Composable
fun StatsRow(orderCount: Int, totalSpent: Long) {
    val spentDisplay = if (totalSpent >= 1000) "$${String.format("%.1f", totalSpent / 1000.0)}k" else "$$totalSpent"
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        RetailerMetricTile(
            label = stringResource(R.string.portal_nav_orders),
            value = "$orderCount",
            modifier = Modifier.weight(1f),
        )
        RetailerMetricTile(
            label = stringResource(R.string.mobile_retailer_ui_spent),
            value = spentDisplay,
            modifier = Modifier.weight(1f),
        )
    }
}
