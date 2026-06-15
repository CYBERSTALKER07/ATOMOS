package com.pegasus.payload.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasus.payload.data.model.Manifest

/**
 * Tactical KPI header row — mirrors iOS ManifestWorkflow volume/stops/zone cards.
 */
@Composable
fun ManifestKpiGrid(manifest: Manifest, modifier: Modifier = Modifier) {
    val total = manifest.totalVolumeVu
    val cap = manifest.maxVolumeVu.coerceAtLeast(0.001)
    val pct = (total / cap).coerceIn(0.0, 1.0).toFloat()

    Column(modifier = modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(PayloadSpacing.md)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(PayloadSpacing.md),
            verticalAlignment = Alignment.Top,
        ) {
            PayloadStatusChip(status = manifest.state)
            PayloadKpiTile(
                label = "PAYLOAD VOLUME",
                value = "%.1f / %.1f VU".format(total, manifest.maxVolumeVu),
                modifier = Modifier.weight(1f),
                footer = {
                    LinearProgressIndicator(
                        progress = { pct },
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(6.dp),
                    )
                },
            )
        }
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(PayloadSpacing.md),
        ) {
            PayloadKpiTile(
                label = "TARGET STOPS",
                value = "${manifest.stopCount} UNITS",
                modifier = Modifier.weight(1f),
            )
            if (manifest.regionCode.isNotBlank()) {
                PayloadKpiTile(
                    label = "DEPLOYMENT ZONE",
                    value = manifest.regionCode.uppercase(),
                    modifier = Modifier.weight(1f),
                )
            }
        }
        Text(
            "Manifest ${manifest.manifestId.take(8)}",
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}
