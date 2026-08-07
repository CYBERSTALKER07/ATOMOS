package com.pegasusx.factory.ui.screens.loadingbay.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.Transfer
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasusx.factory.ui.components.FactoryStatusChip
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun TransferCard(transfer: Transfer, onClick: () -> Unit) {
    ElevatedCard(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Row(
                verticalAlignment = Alignment.Top,
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                    Text(
                        text = transfer.warehouseName.ifBlank { transfer.warehouseId.take(8) },
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text(
                        text = stringResource(R.string.mobile_factory_ui_transfer_take, transfer.id.take(8)),
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Column(
                    horizontalAlignment = Alignment.End,
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                ) {
                    FactoryStatusChip(status = transfer.state)
                    FactoryStatusChip(status = transfer.priority.ifBlank { "STANDARD" })
                }
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FactoryMetricTile(
                    label = stringResource(R.string.warehouse_portal_supply_requests_id_text_items),
                    value = transfer.totalItems.toString(),
                    modifier = Modifier.weight(1f),
                )
                FactoryMetricTile(
                    label = stringResource(R.string.supplier_portal_promotions_text_volume),
                    value = "${String.format("%.0f", transfer.totalVolumeL)}L",
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}
