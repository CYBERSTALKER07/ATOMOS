package com.pegasusx.driver.ui.components

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
import androidx.compose.material.icons.automirrored.filled.Login
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material.icons.filled.ErrorOutline
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Sync
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp

enum class DriverStateKind {
    Empty,
    Error,
    Offline,
    AuthFailure,
    Warning,
    Route,
    Delivery,
    Sync,
    Gps,
    Notifications,
}

@Composable
fun DriverLoadingState(
    title: String,
    body: String,
    modifier: Modifier = Modifier,
    compact: Boolean = false,
    shimmerLines: Boolean = false,
) {
    val transition = rememberInfiniteTransition(label = "driver-loading")
    val scale by transition.animateFloat(
        initialValue = 0.96f,
        targetValue = 1.04f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 900),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "driver-loading-scale",
    )

    val cardPadding = if (compact) DriverSpacing.lg else DriverSpacing.xl
    val iconSize = if (compact) 56.dp else 72.dp

    if (shimmerLines) {
        PegasusCard(modifier = modifier.fillMaxWidth()) {
            Column(
                modifier = Modifier.padding(horizontal = 20.dp, vertical = 24.dp),
                verticalArrangement = Arrangement.spacedBy(18.dp),
            ) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(14.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(28.dp),
                        strokeWidth = 3.dp,
                    )
                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text(
                            text = title,
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.SemiBold,
                        )
                        Text(
                            text = body,
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
                repeat(3) { index ->
                    Box(
                        modifier = Modifier
                            .fillMaxWidth(if (index == 2) 0.58f else 1f)
                            .size(height = 16.dp, width = 0.dp)
                            .clip(MaterialTheme.shapes.small)
                            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.32f))
                            .shimmer(),
                    )
                }
            }
        }
        return
    }

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
                verticalArrangement = Arrangement.spacedBy(DriverSpacing.md),
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
                    textAlign = TextAlign.Center,
                )
                Text(
                    text = body,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
}

@Composable
fun DriverStatePane(
    kind: DriverStateKind,
    headline: String,
    body: String,
    modifier: Modifier = Modifier,
    compact: Boolean = false,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
    iconOverride: ImageVector? = null,
    usePegasusCard: Boolean = false,
) {
    val transition = rememberInfiniteTransition(label = "driver-state")
    val scale by transition.animateFloat(
        initialValue = 0.98f,
        targetValue = 1.02f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 1200),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "driver-state-scale",
    )

    val palette = statePalette(kind, iconOverride)
    val cardPadding = if (compact) DriverSpacing.lg else DriverSpacing.xl
    val iconSize = if (compact) 56.dp else 72.dp

    val content: @Composable () -> Unit = {
        Column(
            modifier = Modifier.padding(cardPadding),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(DriverSpacing.md),
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
                textAlign = TextAlign.Center,
            )
            Text(
                text = body,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
            )
            if (actionLabel != null && onAction != null) {
                FilledTonalButton(onClick = onAction) {
                    Text(actionLabel)
                }
            }
        }
    }

    if (usePegasusCard) {
        StaggeredAppear(index = 0, modifier = modifier.fillMaxWidth()) {
            PegasusCard { content() }
        }
        return
    }

    Box(
        modifier = if (compact) modifier.fillMaxWidth() else modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        ElevatedCard(
            modifier = Modifier
                .fillMaxWidth()
                .widthIn(max = if (compact) 320.dp else 440.dp),
        ) {
            content()
        }
    }
}

private data class DriverStatePalette(
    val icon: ImageVector,
    val container: Color,
    val content: Color,
)

@Composable
private fun statePalette(kind: DriverStateKind, iconOverride: ImageVector?): DriverStatePalette {
    if (iconOverride != null) {
        return DriverStatePalette(
            icon = iconOverride,
            container = MaterialTheme.colorScheme.secondaryContainer,
            content = MaterialTheme.colorScheme.onSecondaryContainer,
        )
    }
    return when (kind) {
        DriverStateKind.Empty -> DriverStatePalette(
            icon = Icons.Default.Inbox,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        DriverStateKind.Error -> DriverStatePalette(
            icon = Icons.Default.ErrorOutline,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
        DriverStateKind.Offline -> DriverStatePalette(
            icon = Icons.Default.CloudOff,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        DriverStateKind.AuthFailure -> DriverStatePalette(
            icon = Icons.AutoMirrored.Filled.Login,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
        DriverStateKind.Warning -> DriverStatePalette(
            icon = Icons.Default.Warning,
            container = MaterialTheme.colorScheme.tertiaryContainer,
            content = MaterialTheme.colorScheme.onTertiaryContainer,
        )
        DriverStateKind.Route -> DriverStatePalette(
            icon = Icons.Default.LocalShipping,
            container = MaterialTheme.colorScheme.primaryContainer,
            content = MaterialTheme.colorScheme.onPrimaryContainer,
        )
        DriverStateKind.Delivery -> DriverStatePalette(
            icon = Icons.Default.LocalShipping,
            container = MaterialTheme.colorScheme.secondaryContainer,
            content = MaterialTheme.colorScheme.onSecondaryContainer,
        )
        DriverStateKind.Sync -> DriverStatePalette(
            icon = Icons.Default.Sync,
            container = MaterialTheme.colorScheme.primaryContainer,
            content = MaterialTheme.colorScheme.onPrimaryContainer,
        )
        DriverStateKind.Gps -> DriverStatePalette(
            icon = Icons.Default.MyLocation,
            container = MaterialTheme.colorScheme.errorContainer,
            content = MaterialTheme.colorScheme.onErrorContainer,
        )
        DriverStateKind.Notifications -> DriverStatePalette(
            icon = Icons.Default.Notifications,
            container = MaterialTheme.colorScheme.surfaceContainerHigh,
            content = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}
