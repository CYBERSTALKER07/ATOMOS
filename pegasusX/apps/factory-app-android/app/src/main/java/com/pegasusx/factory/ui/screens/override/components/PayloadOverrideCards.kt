package com.pegasusx.factory.ui.screens.override.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.factory.data.model.Manifest
import com.pegasusx.factory.data.model.ManifestTransfer
import com.pegasusx.factory.ui.theme.PegasusSpacing
import com.pegasusx.factory.R

@Composable
fun OverrideSummaryCard(
    manifests: List<Manifest>,
    runtimeStatus: String,
    runtimeTone: PegasusRuntimeTone,
) {
    val transferCount = manifests.sumOf { it.transfers.size }
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
                text = stringResource(R.string.mobile_factory_ui_live_manifest_override),
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = stringResource(R.string.mobile_factory_ui_size_loading_manifests_transfercount_transfers_available_for_rebalance_o, manifests.size, transferCount),
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

@Composable
fun OverrideManifestCard(
    manifest: Manifest,
    hasMoveTargets: Boolean,
    actingKey: String?,
    onMove: (ManifestTransfer) -> Unit,
    onRemove: (ManifestTransfer) -> Unit,
    onCancelManifest: () -> Unit,
) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                ) {
                    Text(
                        text = manifest.truckPlate.ifBlank { manifest.truckId.take(8) },
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text(
                        text = stringResource(R.string.mobile_payload_ui_manifest_take, manifest.id.take(8)),
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                FilledTonalButton(
                    onClick = onCancelManifest,
                    enabled = actingKey == null,
                ) {
                    Text("Cancel manifest")
                }
            }

            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                LinearProgressIndicator(
                    progress = {
                        val capacity = manifest.maxCapacityVU.takeIf { it > 0 } ?: 1.0
                        (manifest.totalVolumeVU / capacity).coerceIn(0.0, 1.0).toFloat()
                    },
                    modifier = Modifier.fillMaxWidth(),
                )
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    OverrideMetric("Volume", "${trimDecimal(manifest.totalVolumeVU)} VU", Modifier.weight(1f))
                    OverrideMetric("Capacity", "${trimDecimal(manifest.maxCapacityVU)} VU", Modifier.weight(1f))
                    OverrideMetric("Transfers", manifest.transfers.size.toString(), Modifier.weight(1f))
                }
            }

            if (manifest.transfers.isEmpty()) {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = MaterialTheme.shapes.medium,
                    color = MaterialTheme.colorScheme.surfaceContainerLowest,
                ) {
                    Text(
                        text = stringResource(R.string.mobile_factory_ui_no_transfers_are_assigned_to_this_manifest),
                        modifier = Modifier.padding(PegasusSpacing.lg),
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            } else {
                manifest.transfers.forEach { transfer ->
                    OverrideTransferRow(
                        transfer = transfer,
                        canMove = hasMoveTargets,
                        busy = actingKey == transfer.transferId,
                        onMove = { onMove(transfer) },
                        onRemove = { onRemove(transfer) },
                    )
                }
            }
        }
    }
}

@Composable
fun OverrideTransferRow(
    transfer: ManifestTransfer,
    canMove: Boolean,
    busy: Boolean,
    onMove: () -> Unit,
    onRemove: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.surfaceContainerLowest,
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                ) {
                    Text(
                        text = transfer.productName.ifBlank { "Transfer ${transfer.transferId.take(8)}" },
                        style = MaterialTheme.typography.titleSmall,
                    )
                    Text(
                        text = transfer.transferId.take(8),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                OverrideStateTag(transfer.state)
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                OverrideMetric("Qty", transfer.quantity.toString(), Modifier.weight(1f))
                OverrideMetric("Volume", "${trimDecimal(transfer.volumeVU)} VU", Modifier.weight(1f))
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FilledTonalButton(
                    onClick = onMove,
                    enabled = canMove && !busy,
                    modifier = Modifier.weight(1f),
                ) {
                    Text(if (busy) "Working…" else "Move")
                }
                Button(
                    onClick = onRemove,
                    enabled = !busy,
                    modifier = Modifier.weight(1f),
                ) {
                    Text(if (busy) "Working…" else "Release")
                }
            }
        }
    }
}

@Composable
fun OverrideMetric(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier,
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.surfaceContainer,
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
        ) {
            Text(value, style = MaterialTheme.typography.titleSmall)
            Text(
                text = label,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
fun OverrideStateTag(
    text: String,
) {
    Surface(
        shape = MaterialTheme.shapes.small,
        color = MaterialTheme.colorScheme.secondaryContainer,
        contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelMedium,
            modifier = Modifier.padding(horizontal = PegasusSpacing.sm, vertical = PegasusSpacing.xs),
        )
    }
}
