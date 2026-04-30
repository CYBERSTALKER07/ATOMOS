package com.thelab.retailer.ui.components

import android.view.HapticFeedbackConstants
import androidx.compose.animation.core.spring
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.draw.scale
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalView
import com.thelab.retailer.ui.theme.MotionTokens

// ── Animation Specs (MDC Motion Token-backed) ──

/** Fluid: MDC springDefaultSpatial — damping 0.9, stiffness 700 */
fun <T> fluidSpring() = spring<T>(
    dampingRatio = MotionTokens.springDefaultSpatial.dampingRatio,
    stiffness = MotionTokens.springDefaultSpatial.stiffness,
)

/** Express: MDC springFastSpatial — damping 0.9, stiffness 1400 */
fun <T> expressSpring() = spring<T>(
    dampingRatio = MotionTokens.springFastSpatial.dampingRatio,
    stiffness = MotionTokens.springFastSpatial.stiffness,
)

/** Bouncy: MDC springSlowEffects — damping 1.0, stiffness 800 */
fun <T> bouncySpring() = spring<T>(
    dampingRatio = MotionTokens.springSlowEffects.dampingRatio,
    stiffness = MotionTokens.springSlowEffects.stiffness,
)

// ── Press Scale Modifier (matches iOS .pressable()) ──

fun Modifier.pressScale(
    targetScale: Float = 0.96f,
    onClick: () -> Unit = {},
): Modifier = composed {
    var pressed by remember { mutableStateOf(false) }
    val scale = if (pressed) targetScale else 1f

    this
        .scale(scale)
        .pointerInput(Unit) {
            detectTapGestures(
                onPress = {
                    pressed = true
                    tryAwaitRelease()
                    pressed = false
                },
                onTap = { onClick() },
            )
        }
}

// ── Haptic Feedback ──

@Composable
fun rememberHaptic(): () -> Unit {
    val view = LocalView.current
    return remember {
        { view.performHapticFeedback(HapticFeedbackConstants.CLOCK_TICK) }
    }
}
