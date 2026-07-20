package com.pegasusx.factory.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.SystemUpdate
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun ClientPolicyBanner(
    message: String?,
    modifier: Modifier = Modifier,
    force: Boolean = false,
    onUpdate: (() -> Unit)? = null,
    onDismiss: (() -> Unit)? = null,
) {
    if (message.isNullOrBlank()) return
    Surface(
        color = if (force) {
            MaterialTheme.colorScheme.errorContainer
        } else {
            MaterialTheme.colorScheme.tertiaryContainer
        },
        modifier = modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(PegasusSpacing.md)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(
                    if (force) Icons.Default.Warning else Icons.Default.SystemUpdate,
                    contentDescription = null,
                    tint = if (force) {
                        MaterialTheme.colorScheme.onErrorContainer
                    } else {
                        MaterialTheme.colorScheme.onTertiaryContainer
                    },
                )
                Text(
                    text = message,
                    style = MaterialTheme.typography.bodyMedium,
                    color = if (force) {
                        MaterialTheme.colorScheme.onErrorContainer
                    } else {
                        MaterialTheme.colorScheme.onTertiaryContainer
                    },
                    modifier = Modifier.padding(start = PegasusSpacing.sm),
                )
            }
            if (onUpdate != null) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = PegasusSpacing.sm),
                    horizontalArrangement = Arrangement.End,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    if (!force && onDismiss != null) {
                        TextButton(onClick = onDismiss) { Text("Later") }
                    }
                    Button(onClick = onUpdate) {
                        Text(if (force) "Update now" else "Update")
                    }
                }
            }
        }
    }
}
