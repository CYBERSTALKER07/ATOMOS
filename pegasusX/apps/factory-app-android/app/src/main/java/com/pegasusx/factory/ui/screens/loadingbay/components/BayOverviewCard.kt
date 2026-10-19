package com.pegasusx.factory.ui.screens.loadingbay.components

import androidx.compose.ui.res.stringResource

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
import com.pegasusx.factory.R

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
                text = stringResource(R.string.mobile_factory_ui_loading_bay_flow),
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = stringResource(R.string.mobile_factory_ui_track_approved_transfers_active_loading_work_and_dispatched_volu),
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
