package com.pegasusx.factory.ui.screens.dashboard.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.DashboardStats
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasusx.factory.ui.navigation.FactoryRoutes
import com.pegasusx.factory.ui.theme.PegasusSpacing
import com.pegasusx.factory.R

@Composable
fun DashboardHeroCard(
    stats: DashboardStats,
    onNavigate: (String) -> Unit,
) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.lg),
        ) {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                Text(
                    text = stringResource(R.string.mobile_factory_ui_outbound_floor_status),
                    style = MaterialTheme.typography.titleLarge,
                )
                Text(
                    text = stringResource(R.string.mobile_factory_ui_loadingtransfers_transfers_are_active_across_release_and_bay_lanes, stats.pendingTransfers + stats.loadingTransfers),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FactoryMetricTile(
                    label = stringResource(R.string.mobile_factory_ui_queued),
                    value = stats.pendingTransfers.toString(),
                    modifier = Modifier.weight(1f),
                )
                FactoryMetricTile(
                    label = stringResource(R.string.supplier_portal_dispatch_text_loading),
                    value = stats.loadingTransfers.toString(),
                    modifier = Modifier.weight(1f),
                )
                FactoryMetricTile(
                    label = stringResource(R.string.mobile_factory_ui_critical),
                    value = stats.criticalInsights.toString(),
                    modifier = Modifier.weight(1f),
                )
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FilledTonalButton(
                    onClick = { onNavigate(FactoryRoutes.LOADING_BAY) },
                    modifier = Modifier.weight(1f),
                ) {
                    Icon(Icons.Default.LocalShipping, contentDescription = null)
                    Spacer(Modifier.width(PegasusSpacing.sm))
                    Text("Open bay")
                }
                OutlinedButton(
                    onClick = { onNavigate(FactoryRoutes.TRANSFERS) },
                    modifier = Modifier.weight(1f),
                ) {
                    Icon(Icons.AutoMirrored.Filled.List, contentDescription = null)
                    Spacer(Modifier.width(PegasusSpacing.sm))
                    Text("View transfers")
                }
            }
        }
    }
}
