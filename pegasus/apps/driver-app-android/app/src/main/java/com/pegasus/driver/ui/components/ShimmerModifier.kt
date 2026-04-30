package com.pegasus.driver.ui.components

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.graphics.Brush
import com.pegasus.driver.ui.theme.MotionTokens

/**
 * Reusable shimmer modifier — M3-themed horizontal gradient sweep.
 * Apply to any placeholder Box/Surface for loading skeletons.
 */
fun Modifier.shimmer(): Modifier = composed {
    val transition = rememberInfiniteTransition(label = "shimmer")
    val alpha by transition.animateFloat(
        initialValue = 0.15f,
        targetValue = 0.45f,
        animationSpec = infiniteRepeatable(
            animation = tween(MotionTokens.DurationExtraLong4),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "shimmer_alpha",
    )
    val surfaceVariant = MaterialTheme.colorScheme.surfaceContainerHighest
    this.background(surfaceVariant.copy(alpha = alpha))
}
