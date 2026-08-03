package com.pegasus.payload.ui.home

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasus.payload.data.model.TruckRecommendation

@Composable
fun RecommendationCard(
    rec: TruckRecommendation,
    enabled: Boolean,
    onPickComplete: () -> Unit,
    onPickPartial: () -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceContainerHigh,
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                Text(
                    rec.driverName.ifBlank { rec.driverId.take(8) },
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    "score %.2f".format(rec.score),
                    style = MaterialTheme.typography.labelMedium,
                    fontFamily = FontFamily.Monospace,
                )
            }
            Text(
                listOfNotNull(
                    rec.licensePlate.takeIf { it.isNotBlank() },
                    rec.vehicleClass.takeIf { it.isNotBlank() },
                    rec.truckStatus.takeIf { it.isNotBlank() },
                ).joinToString(" • "),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                "%.1f km • free %.1f VU • %d orders".format(rec.distanceKm, rec.freeVolumeVu, rec.orderCount),
                style = MaterialTheme.typography.bodySmall,
            )
            if (rec.recommendation.isNotBlank()) {
                Text(
                    rec.recommendation,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.primary,
                )
            }
            Row(
                modifier = Modifier.fillMaxWidth().padding(top = 4.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp, Alignment.End)
            ) {
                OutlinedButton(
                    onClick = onPickPartial,
                    enabled = enabled,
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp),
                ) {
                    Text("Partial", style = MaterialTheme.typography.labelMedium)
                }
                Button(
                    onClick = onPickComplete,
                    enabled = enabled,
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp),
                ) {
                    Text("Complete", style = MaterialTheme.typography.labelMedium)
                }
            }
        }
    }
}
