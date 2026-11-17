package com.pegasusx.retailer.ui.screens.dashboard.components

import androidx.compose.ui.res.stringResource

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
import com.pegasusx.retailer.R

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
                    text = stringResource(R.string.mobile_retailer_ui_retail_operations),
                    style = MaterialTheme.typography.headlineSmall,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = stringResource(R.string.mobile_retailer_ui_track_deliveries_restock_faster_and_act_on_demand_signals_from_o),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                RetailerMetricTile(
                    label = stringResource(R.string.portal_page_orders_filter_active_tab),
                    value = activeOrderCount.toString(),
                    modifier = Modifier.weight(1f),
                )
                RetailerMetricTile(
                    label = stringResource(R.string.mobile_retailer_ui_suggestions),
                    value = predictionCount.toString(),
                    modifier = Modifier.weight(1f),
                )
                RetailerMetricTile(
                    label = stringResource(R.string.mobile_retailer_ui_recent_items),
                    value = recentProductCount.toString(),
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}
