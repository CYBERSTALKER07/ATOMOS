package com.pegasus.driver.ui.theme

import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.unit.dp

/**
 * MDC Motion Tokens — aligned to Material Components Android 1.14 spec.
 *
 * Spring system: 3 speeds (fast/default/slow) × 2 types (spatial/effects).
 * Easing system: 7 interpolators + 16 duration slots.
 */
object MotionTokens {
    // ── Springs (MDC 1.14 spec) ──
    val springFastSpatial = spring<Float>(dampingRatio = 0.9f, stiffness = 1400f)
    val springFastEffects = spring<Float>(dampingRatio = 1f, stiffness = 3800f)
    val springDefaultSpatial = spring<Float>(dampingRatio = 0.9f, stiffness = 700f)
    val springDefaultEffects = spring<Float>(dampingRatio = 1f, stiffness = 1600f)
    val springSlowSpatial = spring<Float>(dampingRatio = 0.9f, stiffness = 300f)
    val springSlowEffects = spring<Float>(dampingRatio = 1f, stiffness = 800f)

    // ── Easing Curves (MDC 1.14 spec) ──
    val EasingStandard = CubicBezierEasing(0.2f, 0f, 0f, 1f)
    val EasingStandardDecelerate = CubicBezierEasing(0f, 0f, 0f, 1f)
    val EasingStandardAccelerate = CubicBezierEasing(0.3f, 0f, 1f, 1f)
    val EasingEmphasizedDecelerate = CubicBezierEasing(0.05f, 0.7f, 0.1f, 1f)
    val EasingEmphasizedAccelerate = CubicBezierEasing(0.3f, 0f, 0.8f, 0.15f)
    val EasingLinear = CubicBezierEasing(0f, 0f, 1f, 1f)

    // ── Duration Slots (MDC 1.14 spec, ms) ──
    const val DurationShort1 = 50
    const val DurationShort2 = 100
    const val DurationShort3 = 150
    const val DurationShort4 = 200
    const val DurationMedium1 = 250
    const val DurationMedium2 = 300
    const val DurationMedium3 = 350
    const val DurationMedium4 = 400
    const val DurationLong1 = 450
    const val DurationLong2 = 500
    const val DurationLong3 = 550
    const val DurationLong4 = 600
    const val DurationExtraLong1 = 700
    const val DurationExtraLong2 = 800
    const val DurationExtraLong3 = 900
    const val DurationExtraLong4 = 1000

    // ── Convenience tween builders ──
    fun <T> tweenStandard(durationMs: Int = DurationMedium2) =
        tween<T>(durationMillis = durationMs, easing = EasingStandard)

    fun <T> tweenEmphasizedDecelerate(durationMs: Int = DurationMedium4) =
        tween<T>(durationMillis = durationMs, easing = EasingEmphasizedDecelerate)

    fun <T> tweenEmphasizedAccelerate(durationMs: Int = DurationShort4) =
        tween<T>(durationMillis = durationMs, easing = EasingEmphasizedAccelerate)

    fun <T> tweenStandardDecelerate(durationMs: Int = DurationMedium1) =
        tween<T>(durationMillis = durationMs, easing = EasingStandardDecelerate)
}

// ── Legacy Anim bridge ──
object Anim {
    val snappy = MotionTokens.springFastSpatial
    val bouncy = MotionTokens.springDefaultSpatial
    val micro = spring<Float>(dampingRatio = 0.9f, stiffness = 1400f)
    val settle = MotionTokens.springSlowSpatial

    fun staggerDelay(index: Int): Long = (index * 50L)
}

// ── Spacing Tokens (4dp grid, M3 compliant) ──

object LabSpacing {
    val s4 = 4.dp
    val s8 = 8.dp
    val s12 = 12.dp
    val s16 = 16.dp
    val s20 = 20.dp
    val s24 = 24.dp
    val s32 = 32.dp
    val s48 = 48.dp
    val s64 = 64.dp
}

// ── Corner Radius Tokens (M3 shape scale) ──

object LabRadius {
    val none = 0.dp           // ShapeAppearance.M3.Corner.None
    val extraSmall = 4.dp     // ShapeAppearance.M3.Corner.ExtraSmall
    val small = 8.dp          // ShapeAppearance.M3.Corner.Small
    val medium = 12.dp        // ShapeAppearance.M3.Corner.Medium
    val large = 16.dp         // ShapeAppearance.M3.Corner.Large
    val extraLarge = 28.dp    // ShapeAppearance.M3.Corner.ExtraLarge
    val full = 50              // Percentage — use CircleShape or RoundedCornerShape(50%)

    // Legacy aliases
    val card = medium
    val button = small
    val pill = extraLarge
    val modal = extraLarge
    val sheet = extraLarge
}

// ── Pressable Modifier ──

fun Modifier.pressable(
    enabled: Boolean = true,
    scale: Float = 0.97f,
    onClick: () -> Unit
): Modifier = composed {
    if (!enabled) return@composed this
    var isPressed by remember { mutableStateOf(false) }
    this
        .graphicsLayer {
            scaleX = if (isPressed) scale else 1f
            scaleY = if (isPressed) scale else 1f
        }
        .pointerInput(Unit) {
            detectTapGestures(
                onPress = {
                    isPressed = true
                    tryAwaitRelease()
                    isPressed = false
                },
                onTap = { onClick() }
            )
        }
}

// ── Currency Formatter ──

fun Int.formattedAmount(): String {
    val formatted = String.format("%,d", this)
    return "$formatted"
}

fun Long.formattedAmount(): String {
    val formatted = String.format("%,d", this)
    return "$formatted"
}
