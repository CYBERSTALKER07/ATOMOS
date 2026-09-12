package com.pegasusx.factory.ui.screens.staff.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasusx.factory.ui.theme.PegasusSpacing
import com.pegasusx.factory.R

@Composable
fun StaffSummaryCard(
    total: Int,
    onShift: Int,
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
                text = stringResource(R.string.mobile_factory_ui_staffing_snapshot),
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = stringResource(R.string.mobile_factory_ui_operators_currently_registered_and_active_on_the_factory_floor),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FactoryMetricTile("Total", total.toString(), Modifier.weight(1f))
                FactoryMetricTile("On shift", onShift.toString(), Modifier.weight(1f))
                FactoryMetricTile("Off shift", (total - onShift).toString(), Modifier.weight(1f))
            }
        }
    }
}
