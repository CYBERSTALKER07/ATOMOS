package com.pegasus.payload.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasus.payload.data.model.HandoffCardMetadata

@Composable
fun HandoffInboxCard(
    metadata: HandoffCardMetadata,
    onAction: ((String) -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(10.dp),
        color = MaterialTheme.colorScheme.surfaceContainerHigh,
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Text(metadata.title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
            metadata.subtitle?.takeIf { it.isNotBlank() }?.let {
                Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            metadata.fields?.forEach { (key, value) ->
                Text(
                    "${key.replace('_', ' ')}: $value",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            metadata.primaryLink?.takeIf { it.isNotBlank() }?.let { link ->
                OutlinedButton(onClick = { onAction?.invoke(link) }) {
                    Text(metadata.primaryCta ?: "Open")
                }
            }
        }
    }
}
