package com.pegasusx.factory.ui.screens.supply.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun SupplySummaryCard(
    total: Int,
    visible: Int,
    runtimeStatus: String,
    runtimeTone: PegasusRuntimeTone,
) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Text(
                text = "Warehouse demand queue",
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = "$visible requests in view, $total total across the factory queue.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            PegasusRuntimeBanner(
                tone = runtimeTone,
                message = runtimeStatus,
            )
        }
    }
}
