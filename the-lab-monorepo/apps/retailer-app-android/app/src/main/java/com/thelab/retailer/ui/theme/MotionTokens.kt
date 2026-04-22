package com.thelab.retailer.ui.theme

import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween

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
