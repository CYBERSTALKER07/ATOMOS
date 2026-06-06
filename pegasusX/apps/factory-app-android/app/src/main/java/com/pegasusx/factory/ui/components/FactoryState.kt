package com.pegasusx.factory.ui.components

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
import com.pegasusx.factory.ui.theme.PegasusSpacing

enum class FactoryStateKind {
    Empty,
    NoResults,
    Error,
    Offline,
    Restricted,
    AuthFailure,
}

enum class FactoryRuntimeTone {
    Live,
    Refreshing,
    Warning,
    Offline,
}

@Composable
fun FactoryLoadingState(
    title: String,
    body: String,
    modifier: Modifier = Modifier,
) {
    val transition = rememberInfiniteTransition(label = "factory-loading")
    val scale by transition.animateFloat(
        initialValue = 0.96f,
        targetValue = 1.04f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 900),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "factory-loading-scale",
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
fun FactoryStatePane(
    kind: FactoryStateKind,
    headline: String,
    body: String,
    modifier: Modifier = Modifier,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) {
    val transition = rememberInfiniteTransition(label = "factory-state")
    val scale by transition.animateFloat(
        initialValue = 0.98f,
        targetValue = 1.02f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 1200),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "factory-state-scale",
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
fun FactoryRuntimeBanner(
    tone: FactoryRuntimeTone,
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

private data class FactoryStatePalette(
    val icon: ImageVector,
    val container: Color,
    val content: Color,
)

@Composable
private fun statePalette(kind: FactoryStateKind): FactoryStatePalette {
    return when (kind) {
        FactoryStateKind.Empty -> FactoryStatePalette(
            icon = Icons.Default.Inbox,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        FactoryStateKind.NoResults -> FactoryStatePalette(
            icon = Icons.Default.SearchOff,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        FactoryStateKind.Error -> FactoryStatePalette(
            icon = Icons.Default.ErrorOutline,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
        FactoryStateKind.Offline -> FactoryStatePalette(
            icon = Icons.Default.CloudOff,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        FactoryStateKind.Restricted -> FactoryStatePalette(
            icon = Icons.Default.Lock,
            container = MaterialTheme.colorScheme.secondaryContainer,
            content = MaterialTheme.colorScheme.onSecondaryContainer,
        )
        FactoryStateKind.AuthFailure -> FactoryStatePalette(
            icon = Icons.Default.Login,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
    }
}

@Composable
private fun runtimePalette(tone: FactoryRuntimeTone): FactoryStatePalette {
    return when (tone) {
        FactoryRuntimeTone.Live -> FactoryStatePalette(
            icon = Icons.Default.Sync,
            container = MaterialTheme.colorScheme.surfaceContainer,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        FactoryRuntimeTone.Refreshing -> FactoryStatePalette(
            icon = Icons.Default.Sync,
            container = MaterialTheme.colorScheme.primaryContainer,
            content = MaterialTheme.colorScheme.onPrimaryContainer,
        )
        FactoryRuntimeTone.Warning -> FactoryStatePalette(
            icon = Icons.Default.ErrorOutline,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
        FactoryRuntimeTone.Offline -> FactoryStatePalette(
            icon = Icons.Default.CloudOff,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}