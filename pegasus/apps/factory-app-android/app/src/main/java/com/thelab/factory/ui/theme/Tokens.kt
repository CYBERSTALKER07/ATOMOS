package com.thelab.factory.ui.theme

import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.unit.dp

// ── Motion Tokens ──
object MotionTokens {
    val SpringFast = spring<Float>(stiffness = Spring.StiffnessHigh)
    val SpringMedium = spring<Float>(stiffness = Spring.StiffnessMediumLow)
    val SpringSlow = spring<Float>(stiffness = Spring.StiffnessLow)
}

object Anim {
    val Short = 150
    val Medium = 300
    val Long = 500

    fun <T> short() = tween<T>(durationMillis = Short)
    fun <T> medium() = tween<T>(durationMillis = Medium)
    fun <T> long() = tween<T>(durationMillis = Long)
}

// ── Spacing ──
object LabSpacing {
    val xxs = 2.dp
    val xs = 4.dp
    val sm = 8.dp
    val md = 12.dp
    val lg = 16.dp
    val xl = 24.dp
    val xxl = 32.dp
    val xxxl = 48.dp
}

// ── Radius ──
object LabRadius {
    val xs = 4.dp
    val sm = 8.dp
    val md = 12.dp
    val lg = 16.dp
    val xl = 28.dp
    val full = 1000.dp
}

// ── Pressable Modifier ──
fun Modifier.pressable(onClick: () -> Unit): Modifier = composed {
    clickable(
        indication = null,
        interactionSource = remember { MutableInteractionSource() },
        onClick = onClick,
    )
}
