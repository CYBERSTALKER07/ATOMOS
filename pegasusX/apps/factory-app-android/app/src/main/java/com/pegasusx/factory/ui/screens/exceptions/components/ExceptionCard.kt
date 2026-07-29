package com.pegasusx.factory.ui.screens.exceptions.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import com.pegasusx.factory.data.model.ManifestException
import com.pegasusx.factory.ui.theme.PegasusSpacing
import java.text.DateFormat
import java.util.Date

@Composable
fun ExceptionCard(exception: ManifestException) {
    val isDlq = exception.attemptCount >= 3
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = if (isDlq) {
                MaterialTheme.colorScheme.errorContainer
            } else {
                MaterialTheme.colorScheme.surfaceContainer
            },
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text(exception.reason, style = MaterialTheme.typography.titleSmall)
                if (exception.escalated) {
                    Surface(
                        shape = MaterialTheme.shapes.small,
                        color = MaterialTheme.colorScheme.error,
                    ) {
                        Text(
                            text = "Escalated",
                            modifier = Modifier.padding(horizontal = PegasusSpacing.sm, vertical = PegasusSpacing.xs),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onError,
                        )
                    }
                }
            }
            Text(
                text = "Transfer ${shortId(exception.transferId)} · Manifest ${shortId(exception.manifestId)}",
                style = MaterialTheme.typography.bodyMedium,
                fontFamily = FontFamily.Monospace,
            )
            Text(
                text = buildString {
                    append("Attempts: ${exception.attemptCount}")
                    if (isDlq) append(" — DLQ")
                },
                style = MaterialTheme.typography.bodySmall,
                color = if (isDlq) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                text = formatTimestamp(exception.createdAt),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

private fun shortId(id: String): String = if (id.length > 12) "${id.take(8)}…" else id

private fun formatTimestamp(raw: String): String {
    if (raw.isBlank()) return "—"
    return runCatching {
        DateFormat.getDateTimeInstance(DateFormat.SHORT, DateFormat.SHORT).format(Date.from(java.time.Instant.parse(raw)))
    }.getOrDefault(raw)
}
