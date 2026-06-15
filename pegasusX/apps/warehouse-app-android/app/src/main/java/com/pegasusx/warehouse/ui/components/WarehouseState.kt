package com.pegasusx.warehouse.ui.components

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material.icons.filled.ErrorOutline
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Login
import androidx.compose.material.icons.filled.SearchOff
import androidx.compose.material.icons.filled.Sync
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

enum class WarehouseStateKind {
    Empty,
    NoResults,
    Error,
    Offline,
    Restricted,
    AuthFailure,
}

enum class WarehouseRuntimeTone {
    Live,
    Refreshing,
    Warning,
    Offline,
}

@Composable
fun WarehouseLoadingState(
    title: String,
    body: String,
    modifier: Modifier = Modifier,
) {
    val transition = rememberInfiniteTransition(label = "warehouse-loading")
    val scale by transition.animateFloat(
        initialValue = 0.96f,
        targetValue = 1.04f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 900),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "warehouse-loading-scale",
    )

    Box(
        modifier = modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        ElevatedCard(
            modifier = Modifier
                .fillMaxWidth()
                .widthIn(max = 420.dp),
        ) {
            Column(
                modifier = Modifier.padding(PegasusSpacing.xl),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Box(
                    modifier = Modifier
                        .size(72.dp)
                        .scale(scale)
                        .background(MaterialTheme.colorScheme.surfaceContainerHigh, CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator()
                }
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleLarge,
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
fun WarehouseStatePane(
    kind: WarehouseStateKind,
    headline: String,
    body: String,
    modifier: Modifier = Modifier,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) {
    val transition = rememberInfiniteTransition(label = "warehouse-state")
    val scale by transition.animateFloat(
        initialValue = 0.98f,
        targetValue = 1.02f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 1200),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "warehouse-state-scale",
    )

    val palette = statePalette(kind)

    Box(
        modifier = modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        ElevatedCard(
            modifier = Modifier
                .fillMaxWidth()
                .widthIn(max = 440.dp),
        ) {
            Column(
                modifier = Modifier.padding(PegasusSpacing.xl),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Box(
                    modifier = Modifier
                        .size(72.dp)
                        .scale(scale)
                        .background(palette.container, CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = palette.icon,
                        contentDescription = null,
                        tint = palette.content,
                    )
                }
                Text(
                    text = headline,
                    style = MaterialTheme.typography.titleLarge,
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
fun WarehouseRuntimeBanner(
    tone: WarehouseRuntimeTone,
    message: String,
    modifier: Modifier = Modifier,
) {
    val palette = runtimePalette(tone)

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = MaterialTheme.shapes.medium,
        color = palette.container,
        contentColor = palette.content,
    ) {
        Row(
            modifier = Modifier.padding(horizontal = PegasusSpacing.md, vertical = PegasusSpacing.sm),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                imageVector = palette.icon,
                contentDescription = null,
                tint = palette.content,
            )
            Text(
                text = message,
                style = MaterialTheme.typography.labelMedium,
            )
        }
    }
}

private data class WarehouseStatePalette(
    val icon: ImageVector,
    val container: Color,
    val content: Color,
)

@Composable
private fun statePalette(kind: WarehouseStateKind): WarehouseStatePalette {
    return when (kind) {
        WarehouseStateKind.Empty -> WarehouseStatePalette(
            icon = Icons.Default.Inbox,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        WarehouseStateKind.NoResults -> WarehouseStatePalette(
            icon = Icons.Default.SearchOff,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        WarehouseStateKind.Error -> WarehouseStatePalette(
            icon = Icons.Default.ErrorOutline,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
        WarehouseStateKind.Offline -> WarehouseStatePalette(
            icon = Icons.Default.CloudOff,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        WarehouseStateKind.Restricted -> WarehouseStatePalette(
            icon = Icons.Default.Lock,
            container = MaterialTheme.colorScheme.secondaryContainer,
            content = MaterialTheme.colorScheme.onSecondaryContainer,
        )
        WarehouseStateKind.AuthFailure -> WarehouseStatePalette(
            icon = Icons.Default.Login,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
    }
}

@Composable
private fun runtimePalette(tone: WarehouseRuntimeTone): WarehouseStatePalette {
    return when (tone) {
        WarehouseRuntimeTone.Live -> WarehouseStatePalette(
            icon = Icons.Default.Sync,
            container = MaterialTheme.colorScheme.surfaceContainer,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        WarehouseRuntimeTone.Refreshing -> WarehouseStatePalette(
            icon = Icons.Default.Sync,
            container = MaterialTheme.colorScheme.primaryContainer,
            content = MaterialTheme.colorScheme.onPrimaryContainer,
        )
        WarehouseRuntimeTone.Warning -> WarehouseStatePalette(
            icon = Icons.Default.ErrorOutline,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
        WarehouseRuntimeTone.Offline -> WarehouseStatePalette(
            icon = Icons.Default.CloudOff,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}
