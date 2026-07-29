package com.pegasusx.factory.ui.screens.override.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.Manifest
import com.pegasusx.factory.data.model.ManifestTransfer
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun MoveTransferDialog(
    sourceManifestId: String,
    transfer: ManifestTransfer,
    manifests: List<Manifest>,
    selectedTargetManifestId: String,
    onTargetSelected: (String) -> Unit,
    actingKey: String?,
    onConfirm: (String) -> Unit,
    onDismiss: () -> Unit
) {
    val targetOptions = manifests.filter { it.id != sourceManifestId }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Move transfer") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                Text("Select the loading manifest that should receive transfer ${transfer.transferId.take(8)}.")
                targetOptions.forEach { manifest ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onTargetSelected(manifest.id) }
                            .padding(vertical = PegasusSpacing.xs),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                    ) {
                        RadioButton(
                            selected = selectedTargetManifestId == manifest.id,
                            onClick = { onTargetSelected(manifest.id) },
                        )
                        Column {
                            Text(manifest.truckPlate.ifBlank { manifest.truckId.take(8) })
                            Text(
                                text = "${trimDecimal(manifest.totalVolumeVU)} / ${trimDecimal(manifest.maxCapacityVU)} VU",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
                if (targetOptions.isEmpty()) {
                    Text(
                        text = "Create or keep another loading manifest active before moving this transfer.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        },
        confirmButton = {
            TextButton(
                onClick = { onConfirm(selectedTargetManifestId) },
                enabled = selectedTargetManifestId.isNotBlank() && actingKey == null,
            ) {
                Text("Move")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("Cancel") }
        },
    )
}

@Composable
fun CancelTransferDialog(
    transfer: ManifestTransfer,
    actingKey: String?,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Remove transfer") },
        text = { Text("Release transfer ${transfer.transferId.take(8)} back to APPROVED so it can be reassigned.") },
        confirmButton = {
            TextButton(
                onClick = onConfirm,
                enabled = actingKey == null,
            ) { Text("Release") }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("Keep") }
        },
    )
}

@Composable
fun CancelManifestDialog(
    manifest: Manifest,
    actingKey: String?,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Cancel manifest") },
        text = { Text("Cancel manifest ${manifest.id.take(8)} and return all linked transfers to APPROVED.") },
        confirmButton = {
            TextButton(
                onClick = onConfirm,
                enabled = actingKey == null,
            ) { Text("Cancel manifest") }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("Keep") }
        },
    )
}

internal fun trimDecimal(value: Double): String =
    if (value % 1.0 == 0.0) value.toInt().toString() else String.format("%.1f", value)
