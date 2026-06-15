package com.pegasus.payload.ui.components

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Login
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material.icons.filled.ErrorOutline
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Sync
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

enum class PayloadStateKind {
    Empty,
    Error,
    Offline,
    AuthFailure,
    Warning,
    Truck,
    Manifest,
    Sync,
}

@Composable
fun PayloadLoadingState(
    title: String,
    body: String,
    modifier: Modifier = Modifier,
    compact: Boolean = false,
) {
    val transition = rememberInfiniteTransition(label = "payload-loading")
    val scale by transition.animateFloat(
        initialValue = 0.96f,
        targetValue = 1.04f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 900),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "payload-loading-scale",
    )

    val cardPadding = if (compact) PayloadSpacing.lg else PayloadSpacing.xl
    val iconSize = if (compact) 56.dp else 72.dp

    Box(
        modifier = if (compact) modifier.fillMaxWidth() else modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        ElevatedCard(
            modifier = Modifier
                .fillMaxWidth()
                .widthIn(max = if (compact) 320.dp else 420.dp),
        ) {
            Column(
                modifier = Modifier.padding(cardPadding),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(PayloadSpacing.md),
            ) {
                Box(
                    modifier = Modifier
                        .size(iconSize)
                        .scale(scale)
                        .background(MaterialTheme.colorScheme.surfaceContainerHigh, CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(if (compact) 24.dp else 32.dp),
                        strokeWidth = if (compact) 2.dp else 3.dp,
                    )
                }
                Text(
                    text = title,
                    style = if (compact) MaterialTheme.typography.titleMedium else MaterialTheme.typography.titleLarge,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = body,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
fun PayloadStatePane(
    kind: PayloadStateKind,
    headline: String,
    body: String,
    modifier: Modifier = Modifier,
    compact: Boolean = false,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) {
    val transition = rememberInfiniteTransition(label = "payload-state")
    val scale by transition.animateFloat(
        initialValue = 0.98f,
        targetValue = 1.02f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 1200),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "payload-state-scale",
    )

    val palette = statePalette(kind)
    val cardPadding = if (compact) PayloadSpacing.lg else PayloadSpacing.xl
    val iconSize = if (compact) 56.dp else 72.dp

    Box(
        modifier = if (compact) modifier.fillMaxWidth() else modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        ElevatedCard(
            modifier = Modifier
                .fillMaxWidth()
                .widthIn(max = if (compact) 320.dp else 440.dp),
        ) {
            Column(
                modifier = Modifier.padding(cardPadding),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(PayloadSpacing.md),
            ) {
                Box(
                    modifier = Modifier
                        .size(iconSize)
                        .scale(scale)
                        .background(palette.container, CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = palette.icon,
                        contentDescription = null,
                        tint = palette.content,
                        modifier = Modifier.size(if (compact) 24.dp else 32.dp),
                    )
                }
                Text(
                    text = headline,
                    style = if (compact) MaterialTheme.typography.titleMedium else MaterialTheme.typography.titleLarge,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = body,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                if (actionLabel != null && onAction != null) {
                    Button(onClick = onAction) {
                        Text(actionLabel)
                    }
                }
            }
        }
    }
}

@Composable
fun PayloadInlineLoading(modifier: Modifier = Modifier) {
    Box(
        modifier = modifier
            .fillMaxWidth()
            .padding(PayloadSpacing.xl),
        contentAlignment = Alignment.Center,
    ) {
        CircularProgressIndicator()
    }
}

private data class PayloadStatePalette(
    val icon: ImageVector,
    val container: Color,
    val content: Color,
)

@Composable
private fun statePalette(kind: PayloadStateKind): PayloadStatePalette {
    return when (kind) {
        PayloadStateKind.Empty -> PayloadStatePalette(
            icon = Icons.Default.Inbox,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        PayloadStateKind.Error -> PayloadStatePalette(
            icon = Icons.Default.ErrorOutline,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
        PayloadStateKind.Offline -> PayloadStatePalette(
            icon = Icons.Default.CloudOff,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        PayloadStateKind.AuthFailure -> PayloadStatePalette(
            icon = Icons.AutoMirrored.Filled.Login,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
        PayloadStateKind.Warning -> PayloadStatePalette(
            icon = Icons.Default.Warning,
            container = MaterialTheme.colorScheme.tertiaryContainer,
            content = MaterialTheme.colorScheme.onTertiaryContainer,
        )
        PayloadStateKind.Truck -> PayloadStatePalette(
            icon = Icons.Default.LocalShipping,
            container = MaterialTheme.colorScheme.primaryContainer,
            content = MaterialTheme.colorScheme.onPrimaryContainer,
        )
        PayloadStateKind.Manifest -> PayloadStatePalette(
            icon = Icons.Default.Inbox,
            container = MaterialTheme.colorScheme.secondaryContainer,
            content = MaterialTheme.colorScheme.onSecondaryContainer,
        )
        PayloadStateKind.Sync -> PayloadStatePalette(
            icon = Icons.Default.Sync,
            container = MaterialTheme.colorScheme.primaryContainer,
            content = MaterialTheme.colorScheme.onPrimaryContainer,
        )
    }
}
