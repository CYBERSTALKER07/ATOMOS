package com.pegasus.design

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

enum class PegasusStateKind {
    Empty,
    NoResults,
    Error,
    Offline,
    Restricted,
    AuthFailure,
}

enum class PegasusRuntimeTone {
    Live,
    Refreshing,
    Warning,
    Offline,
}

@Composable
fun PegasusLoadingState(
    title: String,
    body: String,
    modifier: Modifier = Modifier,
) {
    val transition = rememberInfiniteTransition(label = "pegasus-loading")
    val scale by transition.animateFloat(
        initialValue = 0.96f,
        targetValue = 1.04f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 900),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "pegasus-loading-scale",
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
                Text(text = title, style = MaterialTheme.typography.titleLarge)
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
fun PegasusStatePane(
    kind: PegasusStateKind,
    headline: String,
    body: String,
    modifier: Modifier = Modifier,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) {
    val transition = rememberInfiniteTransition(label = "pegasus-state")
    val scale by transition.animateFloat(
        initialValue = 0.98f,
        targetValue = 1.02f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 1200),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "pegasus-state-scale",
    )

    val palette = pegasusStatePalette(kind)

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
                    Icon(imageVector = palette.icon, contentDescription = null, tint = palette.content)
                }
                Text(text = headline, style = MaterialTheme.typography.titleLarge)
                Text(
                    text = body,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                if (actionLabel != null && onAction != null) {
                    Button(onClick = onAction) { Text(actionLabel) }
                }
            }
        }
    }
}

@Composable
fun PegasusRuntimeBanner(
    tone: PegasusRuntimeTone,
    message: String,
    modifier: Modifier = Modifier,
) {
    val palette = pegasusRuntimePalette(tone)

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
            Icon(imageVector = palette.icon, contentDescription = null, tint = palette.content)
            Text(text = message, style = MaterialTheme.typography.labelMedium)
        }
    }
}

private data class PegasusStatePalette(
    val icon: ImageVector,
    val container: Color,
    val content: Color,
)

@Composable
private fun pegasusStatePalette(kind: PegasusStateKind): PegasusStatePalette {
    return when (kind) {
        PegasusStateKind.Empty -> PegasusStatePalette(
            Icons.Default.Inbox,
            MaterialTheme.colorScheme.surfaceContainerHigh,
            MaterialTheme.colorScheme.onSurfaceVariant,
        )
        PegasusStateKind.NoResults -> PegasusStatePalette(
            Icons.Default.SearchOff,
            MaterialTheme.colorScheme.surfaceContainerHigh,
            MaterialTheme.colorScheme.onSurfaceVariant,
        )
        PegasusStateKind.Error -> PegasusStatePalette(
            Icons.Default.ErrorOutline,
            MaterialTheme.colorScheme.errorContainer,
            MaterialTheme.colorScheme.onErrorContainer,
        )
        PegasusStateKind.Offline -> PegasusStatePalette(
            Icons.Default.CloudOff,
            MaterialTheme.colorScheme.surfaceContainerHigh,
            MaterialTheme.colorScheme.onSurfaceVariant,
        )
        PegasusStateKind.Restricted -> PegasusStatePalette(
            Icons.Default.Lock,
            MaterialTheme.colorScheme.secondaryContainer,
            MaterialTheme.colorScheme.onSecondaryContainer,
        )
        PegasusStateKind.AuthFailure -> PegasusStatePalette(
            Icons.Default.Login,
            MaterialTheme.colorScheme.errorContainer,
            MaterialTheme.colorScheme.onErrorContainer,
        )
    }
}

@Composable
private fun pegasusRuntimePalette(tone: PegasusRuntimeTone): PegasusStatePalette {
    return when (tone) {
        PegasusRuntimeTone.Live -> PegasusStatePalette(
            Icons.Default.Sync,
            MaterialTheme.colorScheme.surfaceContainer,
            MaterialTheme.colorScheme.onSurfaceVariant,
        )
        PegasusRuntimeTone.Refreshing -> PegasusStatePalette(
            Icons.Default.Sync,
            MaterialTheme.colorScheme.primaryContainer,
            MaterialTheme.colorScheme.onPrimaryContainer,
        )
        PegasusRuntimeTone.Warning -> PegasusStatePalette(
            Icons.Default.ErrorOutline,
            MaterialTheme.colorScheme.errorContainer,
            MaterialTheme.colorScheme.onErrorContainer,
        )
        PegasusRuntimeTone.Offline -> PegasusStatePalette(
            Icons.Default.CloudOff,
            MaterialTheme.colorScheme.surfaceContainerHigh,
            MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}
