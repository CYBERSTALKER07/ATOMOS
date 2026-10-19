package com.pegasusx.supplier.ui.screens.planning

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SeasonalOverrideRow
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun SeasonalOverridesList(overrides: List<SeasonalOverrideRow>) {
    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md), modifier = Modifier.fillMaxWidth()) {
        if (overrides.isEmpty()) {
            Text(
                "No custom seasonal overrides yet.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        } else {
            overrides.forEach { row ->
                SupplierOpsListCard(
                    headline = row.name?.ifBlank { row.templateId } ?: row.templateId,
                    supporting = "${row.startDate} → ${row.endDate} · ×${row.multiplier} · ${if (row.isActive) "Active" else "Inactive"}",
                )
            }
        }
    }
}
